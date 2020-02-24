package controller

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/mindriot101/dockerdeploy/config"
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
	if err != nil {
		t.Fatalf("error creating controller: %v", err)
	}

	err = c.handle(Poll{})
	if err != nil {
		t.Fatalf("error handling poll instruction: %v", err)
	}
}

type MockDockerClient struct {
	instructions []string
}

func (d *MockDockerClient) pushInstruction(msg string) {
}

type MockPullResponse struct{}

func (m *MockPullResponse) Read(p []byte) (int, error) {
	return 0, io.EOF
}

func (m *MockPullResponse) Close() error {
	return nil
}

func (d *MockDockerClient) ImagePull(ctx context.Context, ref string, options types.ImagePullOptions) (io.ReadCloser, error) {
	d.instructions = append(d.instructions, fmt.Sprintf("pulling image %s with options %v", ref, options))
	return &MockPullResponse{}, nil
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

func (d *MockDockerClient) ContainerStart(ctx context.Context, containerID string, opts types.ContainerStartOptions) error {
	d.instructions = append(d.instructions, fmt.Sprintf("starting container %s", containerID))
	return nil
}

func (d *MockDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return types.ContainerJSON{}, nil
}

func TestHandleWebhookRequest(t *testing.T) {
	c, err := dummyController()
	if err != nil {
		t.Fatalf("error creating dummy controller: %v", err)
	}

	event := gitlab.PipelineEvent{}
	msg := WebHook{
		Event: event,
	}

	err = c.handle(msg)
	if err != nil {
		t.Fatalf("error handling WebHook message: %v", err)
	}

	client, ok := c.client.(*MockDockerClient)
	if !ok {
		t.Fatalf("error casting docker client to concrete type")
	}

	if len(client.instructions) != 4 {
		t.Fatalf("expected 4 instructions, found %d", len(client.instructions))
	}

	if !strings.Contains(client.instructions[0], "pulling image") {
		t.Fatalf("first instruction should be pulling the new image")
	}

	if !strings.Contains(client.instructions[1], "removing container") {
		t.Fatalf("second instruction should be removing the old container")
	}

	if !strings.Contains(client.instructions[2], "creating container") {
		t.Fatalf("third instruction should be creating the new container")
	}

	if !strings.Contains(client.instructions[3], "starting container") {
		t.Fatalf("fourth instruction should be starting the new container")
	}
}

func TestHandleNonConfiguredBranch(t *testing.T) {
	c, err := dummyController()
	if err != nil {
		t.Fatalf("error creating dummy controller: %v", err)
	}

	// Set the config parameters
	c.cfg.Branch.Name = "master"

	// Set the event to have a different branch
	event := gitlab.PipelineEvent{}
	event.ObjectAttributes.Ref = "canary"
	msg := WebHook{
		Event: event,
	}

	err = c.handle(msg)
	if err != nil {
		t.Fatalf("error handling WebHook message: %v", err)
	}

	client, ok := c.client.(*MockDockerClient)
	if !ok {
		t.Fatalf("error casting docker client to concrete type")
	}

	if len(client.instructions) != 0 {
		t.Fatalf("client should have not run deployment, found %d instructions", len(client.instructions))
	}
}

func TestNonSuccessfulBuild(t *testing.T) {
	type Build struct {
		ID         int    `json:"id"`
		Stage      string `json:"stage"`
		Name       string `json:"name"`
		Status     string `json:"status"`
		CreatedAt  string `json:"created_at"`
		StartedAt  string `json:"started_at"`
		FinishedAt string `json:"finished_at"`
		When       string `json:"when"`
		Manual     bool   `json:"manual"`
		User       struct {
			Name      string `json:"name"`
			Username  string `json:"username"`
			AvatarURL string `json:"avatar_url"`
		} `json:"user"`
		Runner struct {
			ID          int    `json:"id"`
			Description string `json:"description"`
			Active      bool   `json:"active"`
			IsShared    bool   `json:"is_shared"`
		} `json:"runner"`
		ArtifactsFile struct {
			Filename string `json:"filename"`
			Size     int    `json:"size"`
		} `json:"artifacts_file"`
	}

	c, err := dummyController()
	if err != nil {
		t.Fatalf("error creating dummy controller: %v", err)
	}

	// Set the event to have a different branch
	event := gitlab.PipelineEvent{}

	// Set up some unsuccessful builds
	event.Builds = append(event.Builds, Build{
		Status: "success",
	})
	event.Builds = append(event.Builds, Build{
		Status: "failure",
	})

	msg := WebHook{
		Event: event,
	}

	err = c.handle(msg)
	if err != nil {
		t.Fatalf("error handling WebHook message: %v", err)
	}

	client, ok := c.client.(*MockDockerClient)
	if !ok {
		t.Fatalf("error casting docker client to concrete type")
	}

	// The build should not have run as one of the builds was not successful
	if len(client.instructions) != 0 {
		t.Fatalf("client should have not run deployment, found %d instructions", len(client.instructions))
	}

	// Update the config to allow build on success
	c.cfg.Branch.BuildOnFailure = true

	err = c.handle(msg)
	if err != nil {
		t.Fatalf("error handling WebHook message: %v", err)
	}

	// This time the build should occur because we have enabled BuildOnFailure
	if len(client.instructions) == 0 {
		t.Fatalf("client should have not run deployment, found %d instructions", len(client.instructions))
	}

}
