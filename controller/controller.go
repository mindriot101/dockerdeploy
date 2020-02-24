package controller

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/mindriot101/dockerdeploy/config"
	"github.com/xanzy/go-gitlab"
)

type Controller struct {
	inbox  chan MessageType
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

	return &Controller{
		inbox:  inbox,
		cfg:    opts.Cfg,
		client: opts.Client,
	}, nil
}

// Web route handlers
func (c *Controller) HandleTrigger(ctx *gin.Context) {
	var t Trigger

	if err := ctx.ShouldBind(&t); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":     "error",
			"error":      err.Error(),
			"error_type": "decoding",
		})
		return
	}

	if err := t.Validate(); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":     "error",
			"error":      err.Error(),
			"error_type": "validation",
		})
		return
	}

	c.inbox <- t

	ctx.JSON(200, gin.H{
		"status": "ok",
	})
}

func (c *Controller) HandleWebHook(ctx *gin.Context) {
	var event gitlab.PipelineEvent

	if err := ctx.ShouldBind(&event); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	msg := WebHook{
		Event: event,
	}

	if err := msg.Validate(); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":     "error",
			"error":      err.Error(),
			"error_type": "validation",
		})
		return
	}

	c.inbox <- msg

	ctx.JSON(200, gin.H{
		"status": "ok",
	})
}

func (c *Controller) Listen() {
	go func() {
		for {
			msg := <-c.inbox
			err := c.handle(msg)
			if err != nil {
				log.Printf("error in handle function: %v", err)
			}
		}
	}()
}

func (c *Controller) handle(msg MessageType) error {
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
	// Check on the current container. If it is not running, restart it
	// Outline:
	// - get the container by name
	// - if it exists: continue
	// - if it doesn't exist, start the container

	ctx := context.Background()

	_, err := c.client.ContainerInspect(ctx, c.cfg.Container.Name)
	if err != nil {
		if client.IsErrNotFound(err) {
			log.Printf("cannot find running container with name %s, restarting", c.cfg.Container.Name)
			trigger := Trigger{
				ImageName:     c.cfg.Image.Name,
				ImageTag:      c.cfg.Image.Tag,
				ContainerName: c.cfg.Container.Name,
			}
			return c.refreshImage(trigger)

		} else {
			log.Printf("unknown error occurred: %v", err)
			return err
		}

	}
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
		if !client.IsErrNotFound(err) {
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
		RestartPolicy: container.RestartPolicy{
			Name: "always",
		},
		// TODO
		PortBindings: nil,
		// TODO
		AutoRemove: false,
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
