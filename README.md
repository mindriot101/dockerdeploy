# Docker deploy

This process is a daemon that:

- exposes a web server that can trigger a docker pull
- refreshes (replaces) a running container
- remembers which containers it's managing and knows the state

## API endpoints

- `/api/v1/refresh` - refresh a single container
	- method: `POST`
	- data: `{"image": "string", "name": "string", "force": "boolean"}"
- `/api/v1/list` - list status of known containers
	- method: `GET`
- `/api/v1/status/<containername>` - get the status of a single container
	- method: `GET`
