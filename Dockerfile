FROM golang:1.13-alpine as builder

RUN apk add --no-cache git

RUN mkdir /app
WORKDIR /app
COPY go.mod go.sum /app/
RUN go get -v -t -d ./...


COPY . /app
RUN go build -o dockerdeploy \
  -ldflags "-X main.sha1ver=$(git rev-parse HEAD) -X main.buildTime=$(date +%Y-%m-%dT%T)" \
  cmd/dockerdeploy/*.go

FROM scratch

COPY --from=builder /app/dockerdeploy /

ENTRYPOINT ["./dockerdeploy"]
