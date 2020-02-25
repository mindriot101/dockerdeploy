package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Overall config object
type Config struct {
	ApiVersion string    `yaml:"api_version"`
	Image      Image     `yaml:"image"`
	Container  Container `yaml:"container"`
	Branch     Branch    `yaml:"branch"`
	Heartbeat  Heartbeat `yaml:"heartbeat"`
}

// Configuration for the image
type Image struct {
	Name string `yaml:"name"`
	Tag  string `yaml:"tag"`
}

// Configuration for the container to run
type Container struct {
	Ports   []Port   `yaml:"ports"`
	Mounts  []Mount  `yaml:"mounts"`
	Name    string   `yaml:"name"`
	Command []string `yaml:"command"`
}

type Port struct {
	Host   int `yaml:"host"`
	Target int `yaml:"target"`
}

type Mount struct {
	Host   string `yaml:"host"`
	Target string `yaml:"target"`
}

// Configuration for the git branch
type Branch struct {
	Name           string `yaml:"name"`
	BuildOnFailure bool   `yaml:"build_on_failure"`
}

// Configuration for the heartbeat functionality
type Heartbeat struct {
	SleepTime int    `yaml:"sleep_time"`
	Endpoint  string `yaml:"endpoint"`
}

func parseString(contents []byte) (*Config, error) {
	var c Config
	err := yaml.Unmarshal(contents, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func Parse(filename string) (*Config, error) {
	t, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return parseString(t)
}

// Validate the configuration value to make sure any required parameters are not missing
func (c *Config) Validate() error {
	// Errors
	if c.ApiVersion == "" {
		return fmt.Errorf("validation error: api version can not be empty")
	}

	if c.Container.Name == "" {
		return fmt.Errorf("validation error: container name can not be empty")
	}

	if c.Image.Name == "" {
		return fmt.Errorf("validation error: image name can not be empty")
	}

	if c.Branch.Name == "" {
		return fmt.Errorf("validation error: branch name can not be empty")
	}

	// Defaults
	if c.Image.Tag == "" {
		c.Image.Tag = "latest"
	}

	return nil
}
