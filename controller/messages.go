package controller

import (
	"fmt"

	"github.com/mindriot101/dockerdeploy/config"
	"github.com/xanzy/go-gitlab"
)

type MessageType interface {
	Validate() error
}

type Poll struct{}

func (p Poll) Validate() error {
	// Nothing to do
	return nil
}

type Trigger struct {
	Command       []string       `json:"command"`
	ImageName     string         `json:"image_name"`
	ImageTag      string         `json:"image_tag"`
	ContainerName string         `json:"container_name"`
	Ports         []config.Port  `json:"ports"`
	Mounts        []config.Mount `json:"mounts"`
}

func (p Trigger) Validate() error {
	// Check that the details are all non-empty
	if p.ImageName == "" {
		return fmt.Errorf("validation error: empty image name")
	}

	if p.ImageTag == "" {
		return fmt.Errorf("validation error: empty image tag")
	}

	if p.ContainerName == "" {
		return fmt.Errorf("validation error: empty container name")
	}

	return nil
}

type WebHook struct {
	Event gitlab.PipelineEvent
}

func (p WebHook) Validate() error {
	return nil
}
