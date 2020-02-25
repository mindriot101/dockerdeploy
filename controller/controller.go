package controller

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/mindriot101/dockerdeploy/config"
	"github.com/xanzy/go-gitlab"
)

// Controller that reconciles and manages containers
type Controller struct {
	inbox  chan MessageType
	client DockerClient
	cfg    *config.Config
	cancel chan interface{}
}

// NewControllerOptions specifies options for creating new controllers
type NewControllerOptions struct {
	Cfg    *config.Config
	Client DockerClient
}

// NewController creates a new controller from arguments
func NewController(opts NewControllerOptions) (*Controller, error) {
	// Validate options
	if opts.Cfg == nil {
		return nil, fmt.Errorf("config argument not valid")
	}

	inbox := make(chan MessageType)
	cancel := make(chan interface{})

	// Set up the polling loop
	go func() {
		// Spawn an initial poll message
		// TODO: make this a configuration parameter?
		inbox <- Poll{}

		t := time.Tick(time.Duration(opts.Cfg.Heartbeat.SleepTime) * time.Second)

		for {
			select {
			case <-t:
				log.Println("sending poll message")
				inbox <- Poll{}
			case <-cancel:
				log.Println("cancelling polling loop")
				break
			}
		}
	}()

	return &Controller{
		inbox:  inbox,
		cfg:    opts.Cfg,
		client: opts.Client,
		cancel: cancel,
	}, nil
}

// HandleTrigger handles trigger web requests
func (c *Controller) HandleTrigger(ctx *gin.Context) {
	c.inbox <- c.createTrigger()

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

func (c *Controller) StopPolling() {
	c.cancel <- nil
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

func (c *Controller) Close() error {
	c.StopPolling()
	return nil
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
			return c.refreshImage(c.createTrigger())

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

	// Do not perform the work if any of the builds were unsuccessful
	for _, build := range w.Event.Builds {
		if !c.cfg.Branch.BuildOnFailure && build.Status != string(Successful) {
			log.Printf("found unsuccessful build: %+v, skipping deploy", build)
			return nil
		}
	}

	return c.refreshImage(c.createTrigger())
}

func (c *Controller) createTrigger() Trigger {
	return Trigger{
		ImageName:     c.cfg.Image.Name,
		ImageTag:      c.cfg.Image.Tag,
		ContainerName: c.cfg.Container.Name,
		Mounts:        c.cfg.Container.Mounts,
		Ports:         c.cfg.Container.Ports,
	}
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

	// Drain the ReadCloser. When this completes then the image pull is complete
	io.Copy(ioutil.Discard, rc)

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

	// Finally run the container
	opts := RunContainerOptions{
		Name: ref,
	}
	created, err := RunContainer(ctx, c.client, t, opts)
	if err != nil {
		log.Printf("error running container: %v", err)
		return err
	}
	log.Printf("created container with id %s", created.ID)

	return nil
}
