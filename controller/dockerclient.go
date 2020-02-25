package controller

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

type RunContainerOptions struct {
	Name string
}

func RunContainer(ctx context.Context, client DockerClient, t Trigger, opts RunContainerOptions) (*container.ContainerCreateCreatedBody, error) {

	log.Printf("starting container %s with image %s", t.ContainerName, opts.Name)
	containerConfig := container.Config{
		Cmd:   []string{"sleep", "86400"},
		Image: opts.Name,
	}

	// Include port bindings
	ports := make(nat.PortMap)
	for _, portDef := range t.Ports {
		// TODO: support udp ports?

		// Host or target?
		port := fmt.Sprintf("%d/tcp", portDef.Target)
		portBinding := nat.PortBinding{
			HostPort: fmt.Sprintf("%d", portDef.Host),
		}
		ports[nat.Port(port)] = []nat.PortBinding{
			portBinding,
		}
	}

	hostConfig := container.HostConfig{
		RestartPolicy: container.RestartPolicy{
			Name: "always",
		},
		// TODO
		PortBindings: ports,
		// TODO
		Mounts:     nil,
		AutoRemove: false,
	}

	log.Printf("host ports: %+v", hostConfig.PortBindings)

	networkConfig := network.NetworkingConfig{}

	created, err := client.ContainerCreate(
		ctx,
		&containerConfig,
		&hostConfig,
		&networkConfig,
		t.ContainerName,
	)
	if err != nil {
		log.Printf("error creating container: %v", err)
		return nil, err
	}

	// Inspect the `created` object to get information about the container creation process
	for _, warning := range created.Warnings {
		log.Printf("WARNING: %s", warning)
	}

	if err := client.ContainerStart(ctx, created.ID, types.ContainerStartOptions{}); err != nil {
		log.Printf("error starting container: %v", err)
		return nil, err
	}

	return &created, nil
}

// Interface that defines the API surface of the Docker interactions
type DockerClient interface {
	ImagePull(ctx context.Context, ref string, options types.ImagePullOptions) (io.ReadCloser, error)
	ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig,
		networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error)
	ContainerStart(ctx context.Context, containerID string, opts types.ContainerStartOptions) error
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
}
