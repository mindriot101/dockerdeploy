# Docker deploy

![Go](https://github.com/mindriot101/dockerdeploy/workflows/Go/badge.svg?branch=master)

This process is a daemon that:

- exposes a web server that can trigger a docker pull
* refreshes (replaces) a running container
* remembers which containers it's managing and knows the state

## TODO

* [ ] read config file
* [ ] decide on which docker arguments to support

## API endpoints

* `/api/v1/refresh` - refresh a single container
	* method: `POST`
	* data: `{"image": "string", "name": "string", "force": "boolean"}"
* `/api/v1/list` - list status of known containers
	* method: `GET`
* `/api/v1/status/<containername>` - get the status of a single container
	* method: `GET`
