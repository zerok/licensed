all: licensed

licensed: $(shell find . -name '*.go')
	cd cmd/licensed && go build -o ../../licensed

clean:
	rm -f licensed

install:
	cd cmd/licensed && go install
