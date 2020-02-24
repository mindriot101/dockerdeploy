NOW := $(shell date +'%Y-%m-%dT%T')
BINARY := dockerdeploy

build:
	go build -o ${BINARY} -ldflags "-X main.sha1ver=$(shell git rev-parse HEAD) -X main.buildTime=${NOW}" main.go

.PHONY: clean
clean:
	rm -f ${BINARY}
