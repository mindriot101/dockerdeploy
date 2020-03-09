# Docker deploy

[![mindriot101](https://circleci.com/gh/mindriot101/dockerdeploy.svg?style=svg)](https://github.com/mindriot101/dockerdeploy)

This process is a daemon that:

- exposes a web server that can trigger a docker pull
- refreshes (replaces) a running container
- remembers which containers it's managing and checks on the state of managed
  containers

## API endpoints

- `/webhook` - let gitlab pipeline updates trigger a container refresh
- `/trigger` - manually trigger a container refresh

### Webhook

Add this into the gitlab webhook interface

### Trigger

`curl -X POST -H 'Content-Type: application/json' <server ip>:<server port>/trigger
