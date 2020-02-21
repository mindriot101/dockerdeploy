package controller

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/mindriot101/dockerdeploy/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
)

func dummyController() (*Controller, error) {
	cfg := config.Config{}
	client := MockDockerClient{}
	return NewController(NewControllerOptions{
		Cfg:    &cfg,
		Client: &client,
	})
}

func TestHandlePollInstruction(t *testing.T) {
	c, err := dummyController()
	assert.Nil(t, err)

	err = c.handle(Poll{})
	assert.Nil(t, err)
}

type MockDockerClient struct {
	instructions []string
}

func (d *MockDockerClient) pushInstruction(msg string) {
}

func (d *MockDockerClient) ImagePull(ctx context.Context, ref string, options types.ImagePullOptions) (io.ReadCloser, error) {
	d.instructions = append(d.instructions, fmt.Sprintf("pulling image %s with options %v", ref, options))
	return nil, nil
}

func (d *MockDockerClient) ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error {
	d.instructions = append(d.instructions, fmt.Sprintf("removing container %s with options %v", containerID, options))
	return nil
}

func (d *MockDockerClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
	d.instructions = append(d.instructions, fmt.Sprintf("creating container %s", containerName))
	body := container.ContainerCreateCreatedBody{}
	return body, nil
}

func TestHandleWebhookRequest(t *testing.T) {
	c, err := dummyController()
	assert.Nil(t, err)
	event := gitlab.PipelineEvent{}
	msg := WebHook{
		Event: event,
	}

	err = c.handle(msg)

	assert.Nil(t, err)
	client, ok := c.client.(*MockDockerClient)
	assert.True(t, ok)
	assert.Len(t, client.instructions, 3)
	assert.Contains(t, client.instructions[0], "pulling image")
	assert.Contains(t, client.instructions[1], "removing container")
	assert.Contains(t, client.instructions[2], "creating container")
}
