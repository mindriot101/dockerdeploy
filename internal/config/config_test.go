package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var config = `
image:
  name: ubuntu
  tag: latest
container:
  ports:
    - host: 8080
      target: 80
  mounts:
    - host: $PWD/data
      target: /data
branch:
  name: master
  build_on_failure: false
heartbeat:
  sleep_time: 10
  endpoint: /heartbeat
`

func TestParseConfig(t *testing.T) {
	assert := assert.New(t)
	cfg, err := parseString([]byte(config))

	assert.Nil(err)

	assert.Equal(cfg.Image, Image{
		Name: "ubuntu",
		Tag:  "latest",
	})

	assert.Equal(cfg.Container, Container{
		Ports: []Port{
			Port{
				Host:   8080,
				Target: 80,
			},
		},
		Mounts: []Mount{
			Mount{
				Host:   "$PWD/data",
				Target: "/data",
			},
		},
	})

	assert.Equal(cfg.Branch, Branch{
		Name:           "master",
		BuildOnFailure: false,
	})

	assert.Equal(cfg.Heartbeat, Heartbeat{
		SleepTime: 10,
		Endpoint:  "/heartbeat",
	})
}
