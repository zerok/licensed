<h1 align="center">licensed</h1>

<p align="center"><em>licensed</em> helps you include license information of
all your dependencies into your Go tool.</p>

-------------

**Note:** This is just a very naive approach to including license information
for dependency libraries. It will most likely be refined in the future.

Probably the easiest way to get started is by going into the folder where your
main package is located and adding `// go:generate licensed` as the first line
to your source code.

When you now execute `$ go generate ./...` a new file named
`licenses_generated.go` will appear which includes function `getLicenseInfos()
[]licenseInfo`.

Now add the flag `--licenses` and render the output of this function if the
user sets it:

```go
var showLicenses bool

flag.BoolVar(&showLicenses, "licenses", false, "Show information about all used dependencies")
flag.Parse()

if showLicenses {
    fmt.Println("The following third-party libraries are used:\n")
    for _, l := range getLicenseInfos() {
        fmt.Printf("> %s\n\n%s\n\n", l.ProjectRoot, l.LicenseText)
    }
    os.Exit(0)
}
```

## Installation

You can install *licensed* using `go get`:

```sh
$ go get -u github.com/zerok/licensed
```

## Configuration

In order to avoid name clashes with types or functions already defined inside
your code, you can customize both the name of the type as well as the name of
the `getLicenseInfos` function with the following parameters:

- `-func string`: Name of the function that should be generated (default
  `getLicenseInfos`)

- `-output string`: Path of the file that should be generated (default
  `licenses_generated.go`)

- `-type string`: Name of the type that should be generated (default
  `licenseInfo`)
