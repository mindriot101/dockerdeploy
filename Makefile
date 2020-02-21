NOW := $(shell date +'%Y-%m-%dT%T')
BINARY := dockerdeploy

.PHONY: build
build: ${BINARY}

dockerdeploy: $(wildcard *.go)
	go build -o $@ -ldflags "-X main.sha1ver=$(shell git rev-parse HEAD) -X main.buildTime=${NOW}" .

.PHONY: clean
clean:
	rm -f ${BINARY}
