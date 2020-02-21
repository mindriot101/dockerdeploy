package controller

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mindriot101/dockerdeploy/internal/config"
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

type WebHook struct{}

func (p WebHook) Name() string {
	return "WebHook"
}

type Controller struct {
	inbox  chan MessageType
	server *gin.Engine
}

func NewController(cfg *config.Config) Controller {
	inbox := make(chan MessageType)

	// Set up the polling loop
	go func() {
		for {
			log.Println("sending poll message")
			inbox <- Poll{}
			time.Sleep(time.Duration(cfg.Heartbeat.SleepTime) * time.Second)
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
		inbox <- WebHook{}
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	return Controller{
		inbox:  inbox,
		server: r,
	}
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
	log.Printf("handling %s message", msg)
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

func (c Controller) poll(p Poll) error {
	return nil
}

func (c Controller) trigger(t Trigger) error {
	return nil
}

func (c Controller) webhook(w WebHook) error {
	return nil
}
