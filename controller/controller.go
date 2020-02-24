package controller

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/gin-gonic/gin"
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
	ImageName     string `json:"image_name"`
	ImageTag      string `json:"image_tag"`
	ContainerName string `json:"container_name"`
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

type Controller struct {
	inbox  chan MessageType
	server *gin.Engine
	client DockerClient
	cfg    *config.Config
}

type NewControllerOptions struct {
	Cfg    *config.Config
	Client DockerClient
}

func NewController(opts NewControllerOptions) (*Controller, error) {
	// Validate options
	if opts.Cfg == nil {
		return nil, fmt.Errorf("config argument not valid")
	}

	inbox := make(chan MessageType)

	// Set up the polling loop
	go func() {
		for {
			log.Println("sending poll message")
			inbox <- Poll{}
			time.Sleep(time.Duration(opts.Cfg.Heartbeat.SleepTime) * time.Second)
		}
	}()

	// Set up web server
	r := gin.Default()
	r.POST("/trigger", func(c *gin.Context) {
		var t Trigger

		if err := c.ShouldBind(&t); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":     "error",
				"error":      err.Error(),
				"error_type": "decoding",
			})
			return
		}

		if err := t.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":     "error",
				"error":      err.Error(),
				"error_type": "validation",
			})
			return
		}

		inbox <- t
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})
	r.POST("/webhook", func(c *gin.Context) {
		var event gitlab.PipelineEvent

		if err := c.ShouldBind(&event); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

		msg := WebHook{
			Event: event,
		}

		if err := msg.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":     "error",
				"error":      err.Error(),
				"error_type": "validation",
			})
			return
		}

		inbox <- msg

		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	return &Controller{
		inbox:  inbox,
		cfg:    opts.Cfg,
		server: r,
		client: opts.Client,
	}, nil
}

func (c Controller) Run() error {
	go func() {
		for {
			msg := <-c.inbox
			err := c.handle(msg)
			if err != nil {
				log.Printf("error in handle function: %v", err)
			}
		}
	}()

	// Start the web server
	return c.server.Run()
}

func (c Controller) handle(msg MessageType) error {
	switch msg.(type) {
	case Poll:
		m, _ := msg.(Poll)
		return c.poll(m)
	case Trigger:
		m, _ := msg.(Trigger)
		return c.trigger(m)
	case WebHook:
		m, _ := msg.(WebHook)
		return c.webhook(m)
	default:
		return fmt.Errorf("unhandled message type: %s\n", msg)
	}
}

func (c *Controller) poll(p Poll) error {
	return nil
}

func (c *Controller) trigger(t Trigger) error {
	return c.refreshImage(t)
}

func (c *Controller) webhook(w WebHook) error {

	// Check that the pipeline event is from the branch that we care about
	eventBranch := w.Event.ObjectAttributes.Ref
	watchedBranch := c.cfg.Branch.Name
	if eventBranch != watchedBranch {
		log.Printf("pipeline event found for branch that is not being monitored, found %s expected %s", eventBranch, watchedBranch)
		return nil
	}

	trigger := Trigger{
		ImageName:     c.cfg.Image.Name,
		ImageTag:      c.cfg.Image.Tag,
		ContainerName: c.cfg.Container.Name,
	}

	return c.refreshImage(trigger)
}

func (c *Controller) refreshImage(t Trigger) error {
	// Implementation is to pull the previous image, then remove the current
	// image and run the new image in its place
	ctx := context.Background()

	ref := fmt.Sprintf("%s:%s", t.ImageName, t.ImageTag)
	log.Printf("refreshing image %s", ref)
	rc, err := c.client.ImagePull(ctx, ref, types.ImagePullOptions{})
	if err != nil {
		log.Printf("error pulling image %s: %v", ref, err)
		return err
	}
	defer rc.Close()
	io.Copy(os.Stderr, rc)

	log.Printf("removing container %s if exists", t.ContainerName)
	// Remove the currently running container
	err = c.client.ContainerRemove(ctx, t.ContainerName, types.ContainerRemoveOptions{
		Force: true,
	})
	if err != nil {
		// XXX there does not seem to be a good way to get whether the
		// container could not be found, or that another error occurs. Despite
		// all of my attempts, `client.IsErrNotFound` does not return the
		// correct thing, even though the error message clearly states that the
		// container cannot be found. We therefore have to resort to this
		// string checking which feels dirty
		if !strings.Contains(err.Error(), "No such container") {
			return err
		}
	}

	log.Printf("starting container %s with image %s", t.ContainerName, ref)
	containerConfig := container.Config{
		Cmd:   []string{"sleep", "86400"},
		Image: ref,
	}
	hostConfig := container.HostConfig{
		// TODO
		PortBindings: nil,
		// TODO
		AutoRemove: true,
		// TODO
		Mounts: nil,
	}
	networkConfig := network.NetworkingConfig{}

	created, err := c.client.ContainerCreate(
		ctx,
		&containerConfig,
		&hostConfig,
		&networkConfig,
		t.ContainerName,
	)
	if err != nil {
		log.Printf("error creating container: %v", err)
		return err
	}

	if err := c.client.ContainerStart(ctx, created.ID, types.ContainerStartOptions{}); err != nil {
		log.Printf("error starting container: %v", err)
		return err
	}

	// Inspect the `created` object to get information about the container creation process
	for _, warning := range created.Warnings {
		log.Printf("WARNING: %s", warning)
	}

	log.Printf("created container with id %s", created.ID)

	return nil
}

type DockerClient interface {
	ImagePull(ctx context.Context, ref string, options types.ImagePullOptions) (io.ReadCloser, error)
	ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig,
		networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error)
	ContainerStart(ctx context.Context, containerID string, opts types.ContainerStartOptions) error
}
