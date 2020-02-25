package config

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var config = `
api_version: "1"
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
  name: foobar
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

	if err = cfg.Validate(); err != nil {
		t.Fatalf("test config is not valid: %v", err)
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
		Name: "foobar",
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
	cfg, err := parseString([]byte(config))
	if err != nil {
		t.Fatalf("error parsing config: %v", err)
	}

	{
		orig := cfg.Container.Name
		cfg.Container.Name = ""
		err = cfg.Validate()
		if err == nil {
			t.Fatalf("validation should not pass with empty container name")
		}

		if !strings.Contains(err.Error(), "container name can not be empty") {
			t.Fatalf("error validating empty container name")
		}
		cfg.Container.Name = orig
	}

	{
		orig := cfg.Image.Name
		cfg.Image.Name = ""
		err = cfg.Validate()
		if err == nil {
			t.Fatalf("validation should not pass with empty image name")
		}

		if !strings.Contains(err.Error(), "image name can not be empty") {
			t.Fatalf("error validating empty image name")
		}
		cfg.Image.Name = orig
	}

	{
		orig := cfg.Branch.Name
		cfg.Branch.Name = ""
		err = cfg.Validate()
		if err == nil {
			t.Fatalf("validation should not pass with empty branch name")
		}

		if !strings.Contains(err.Error(), "branch name can not be empty") {
			t.Fatalf("error validating empty branch name")
		}
		cfg.Branch.Name = orig
	}
}

func TestSaneDefaults(t *testing.T) {
	cfg, err := parseString([]byte(config))
	if err != nil {
		t.Fatalf("error parsing config: %v", err)
	}

	{
		cfg.Image.Tag = ""
		err = cfg.Validate()
		if err != nil {
			t.Fatalf("validation should pass")
		}

		if cfg.Image.Tag != "latest" {
			t.Fatalf("validation method did not set default tag")
		}
	}

}

func TestApiVersionRequired(t *testing.T) {
	cfg, err := parseString([]byte(config))
	if err != nil {
		t.Fatalf("error parsing config: %v", err)
	}

	{
		orig := cfg.ApiVersion
		cfg.ApiVersion = ""
		err = cfg.Validate()
		if err == nil {
			t.Fatalf("validation should not pass with blank api_version key")
		}
		if !strings.Contains(err.Error(), "api version can not be empty") {
			t.Fatalf("error validating empty api version")
		}
		cfg.ApiVersion = orig
	}
}
