package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type templateContext struct {
	PackageName  string
	NumPackages  int
	Packages     []*depEntry
	FunctionName string
	TypeName     string
}

const fileTmpl = `package {{ .PackageName }}
type {{ .TypeName }} struct {
	Package string
	LicenseText string
}

func {{ .FunctionName }}() []{{.TypeName}} {
	result := make([]{{.TypeName}}, 0, {{ .NumPackages }})
	{{ range .Packages -}}
	result = append(result, {{$.TypeName}}{Package: "{{ .ProjectRoot }}", LicenseText: ""})
	{{ end }}
	return result
}
`

type depEntry struct {
	ProjectRoot string
	// LicenseFile is not provided by dep itself but we are simply reusing
	// the data structure.
	LicenseFile string
}

func main() {
	var outputFilePath string
	var functionName string
	var typeName string
	flag.StringVar(&outputFilePath, "output", "licenses_generated.go", "Path of the file that should be generated")
	flag.StringVar(&functionName, "func", "getLicenseInfos", "Name of the function that should be generated")
	flag.StringVar(&typeName, "type", "licenseInfo", "Name of the type that should be generated")
	flag.Parse()

	depPath, err := exec.LookPath("dep")
	if err != nil {
		log.Fatalf("dep not installed.")
	}

	rootDir, err := findRootFolder(".")
	if err != nil {
		log.Fatalf(err.Error())
	}
	dependencies, err := getDependencies(rootDir, depPath)
	pkgMap := make(map[string]*depEntry)

	for _, dep := range dependencies {
		pkgMap[dep.ProjectRoot] = dep
		licenseSearchPath := filepath.Join(rootDir, "vendor", dep.ProjectRoot, "LICENSE*")
		licenseCandidates, err := filepath.Glob(licenseSearchPath)
		if err != nil {
			log.Fatalf("Failed to find license candidates for %s: %s", dep.ProjectRoot, err.Error())
		}
		if len(licenseCandidates) > 0 {
			dep.LicenseFile = licenseCandidates[0]
		}
	}

	fileset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fileset, ".", nil, 0)
	if err != nil {
		log.Fatalf("Failed to parse current package: %s", err.Error())
	}
	var targetPkg *ast.Package
	for pkgName, pkg := range pkgs {
		if strings.HasSuffix(pkgName, "_testing") {
			continue
		}
		targetPkg = pkg
	}
	if targetPkg == nil {
		log.Fatalf("Could not determine target package")
	}

	tmplContext := templateContext{
		FunctionName: functionName,
		TypeName:     typeName,
		PackageName:  targetPkg.Name,
		NumPackages:  len(dependencies),
		Packages:     dependencies,
	}

	tmpl := template.Must(template.New("root").Parse(fileTmpl))
	var fileTmplBuffer bytes.Buffer
	if err := tmpl.Execute(&fileTmplBuffer, tmplContext); err != nil {
		log.Fatalf("Failed to generate target file: %s", err.Error())
	}

	file, err := parser.ParseFile(fileset, "", fileTmplBuffer.String(), 0)
	if err != nil {
		log.Fatalf("Failed to generate target file: %s", err.Error())
	}
	obj := file.Scope.Lookup(functionName)
	if obj == nil {
		log.Fatalf("Failed to find generated function")
	}
	for _, stmt := range obj.Decl.(*ast.FuncDecl).Body.List {
		switch st := stmt.(type) {
		case *ast.AssignStmt:
			for _, e := range st.Rhs {
				switch rhs := e.(type) {
				case *ast.CallExpr:
					if rhs.Fun.(*ast.Ident).Name == "append" {
						info := rhs.Args[len(rhs.Args)-1].(*ast.CompositeLit)
						projectRoot, err := strconv.Unquote(info.Elts[0].(*ast.KeyValueExpr).Value.(*ast.BasicLit).Value)
						if err != nil {
							log.Fatalf("Failed to unquote package name")
						}
						data, err := ioutil.ReadFile(pkgMap[projectRoot].LicenseFile)
						if err != nil {
							log.Fatalf("Failed to read license of %s: %s", projectRoot, err.Error())
						}
						info.Elts[1].(*ast.KeyValueExpr).Value = &ast.BasicLit{
							Kind:  token.STRING,
							Value: strconv.Quote(string(data)),
						}
					}
				}
			}
		}
	}
	if outputFilePath == "-" {
		printer.Fprint(os.Stdout, fileset, file)
	} else {
		fp, err := os.OpenFile(outputFilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if err != nil {
			log.Fatalf("Failed to open output file for writing: %s", err.Error())
		}
		defer fp.Close()
		printer.Fprint(fp, fileset, file)
	}
}

func getDependencies(rootDir string, depPath string) ([]*depEntry, error) {
	var buffer bytes.Buffer
	cmd := exec.Command(depPath, "status", "--json")
	cmd.Stdout = &buffer
	cmd.Dir = rootDir
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	var dependencies []*depEntry
	if err := json.NewDecoder(&buffer).Decode(&dependencies); err != nil {
		return nil, err
	}
	return dependencies, nil
}

func findRootFolder(path string) (string, error) {
	p, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	for {
		if exists(filepath.Join(p, "vendor")) {
			return p, nil
		}
		parent := filepath.Dir(p)
		if p == parent {
			break
		}
		p = parent
	}
	return "", fmt.Errorf("no root directory found")
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
