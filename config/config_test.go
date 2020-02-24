package config

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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
	cfg, err := parseString([]byte(config))

	if err != nil {
		t.Fatalf("error parsing config string: %v", err)
	}

	expectedImage := Image{
		Name: "ubuntu",
		Tag:  "latest",
	}
	if cfg.Image != expectedImage {
		t.Fatalf("failure to parse image tag, %v != %v", cfg.Image, expectedImage)
	}

	expectedContainer := Container{
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
	}

	if !cmp.Equal(cfg.Container, expectedContainer) {
		t.Fatalf("failure to parse container, %v != %v", cfg.Container, expectedContainer)
	}

	expectedBranch := Branch{
		Name:           "master",
		BuildOnFailure: false,
	}
	if cfg.Branch != expectedBranch {
		t.Fatalf("failure to parse branch, %v != %v", cfg.Branch, expectedBranch)
	}

	expectedHeartbeat := Heartbeat{
		SleepTime: 10,
		Endpoint:  "/heartbeat",
	}
	if cfg.Heartbeat != expectedHeartbeat {
		t.Fatalf("failure to parse heartbeat, %v != %v", cfg.Heartbeat, expectedHeartbeat)
	}
}

func TestValidation(t *testing.T) {
}
