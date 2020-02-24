package controller

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

type DockerClient interface {
	ImagePull(ctx context.Context, ref string, options types.ImagePullOptions) (io.ReadCloser, error)
	ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig,
		networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error)
	ContainerStart(ctx context.Context, containerID string, opts types.ContainerStartOptions) error
	ContainerWait(ctx context.Context, containerID string, wc container.WaitCondition) (<-chan container.ContainerWaitOKBody, <-chan error)
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
}
