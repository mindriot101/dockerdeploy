package controller

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

type RunContainerOptions struct {
	Name string
}

func RunContainer(ctx context.Context, client DockerClient, t Trigger, opts RunContainerOptions) (*container.ContainerCreateCreatedBody, error) {
	// Set up the initial data that the requests need
	// Include port bindings
	ports := make(nat.PortMap)
	exposedPorts := make(nat.PortSet)
	for _, portDef := range t.Ports {
		// TODO: support udp ports?

		// Host or target?
		port := fmt.Sprintf("%d/tcp", portDef.Target)
		portBinding := nat.PortBinding{
			HostIP:   "127.0.0.1",
			HostPort: fmt.Sprintf("%d", portDef.Host),
		}
		ports[nat.Port(port)] = []nat.PortBinding{
			portBinding,
		}

		exposedPorts[nat.Port(port)] = struct{}{}
	}

	log.Printf("starting container %s with image %s", t.ContainerName, opts.Name)
	containerConfig := container.Config{
		Cmd:          t.Command,
		Image:        opts.Name,
		ExposedPorts: exposedPorts,
	}

	// Configure mounts
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	mounts := []mount.Mount{}
	for _, m := range t.Mounts {
		source := strings.Replace(m.Host, "$PWD", cwd, 1)
		mount := mount.Mount{
			Type:   "bind",
			Source: source,
			Target: m.Target,
		}
		_ = m
		mounts = append(mounts, mount)
	}

	hostConfig := container.HostConfig{
		RestartPolicy: container.RestartPolicy{
			Name: "always",
		},
		PortBindings: ports,
		Mounts:       mounts,
		AutoRemove:   false,
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
