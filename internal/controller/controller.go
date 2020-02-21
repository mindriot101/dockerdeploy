package controller

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/gin-gonic/gin"
	"github.com/mindriot101/dockerdeploy/internal/config"
	"github.com/xanzy/go-gitlab"
)

type MessageType interface {
	Name() string
}

type Poll struct{}

func (p Poll) Name() string {
	return "Poll"
}

type Trigger struct{}

func (p Trigger) Name() string {
	return "Trigger"
}

type WebHook struct {
	Event gitlab.PipelineEvent
}

func (p WebHook) Name() string {
	return "WebHook"
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
		inbox <- Trigger{}
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

		inbox <- WebHook{
			Event: event,
		}
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
	return nil
}

func (c *Controller) webhook(w WebHook) error {
	// Implementation is to pull the previous image, then remove the current
	// image and run the new image in its place
	ref := fmt.Sprintf("%s:%s", c.cfg.Image.Name, c.cfg.Image.Tag)
	_, err := c.client.ImagePull(context.Background(), ref, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	return nil
}

type DockerClient interface {
	ImagePull(ctx context.Context, ref string, options types.ImagePullOptions) (io.ReadCloser, error)
}
