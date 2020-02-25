NOW := $(shell date +'%Y-%m-%dT%T')
BINARY := dockerdeploy

build:
	go build -o ${BINARY} -ldflags "-X main.commit=$(shell git rev-parse HEAD) -X main.date=${NOW}" main.go

.PHONY: clean
clean:
	rm -f ${BINARY}

.PHONY: test
test:
	go test ./...
