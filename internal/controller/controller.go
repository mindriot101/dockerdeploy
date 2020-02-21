package controller

import (
	"github.com/mindriot101/dockerdeploy/internal/config"
)

type MessageType string

const (
	Poll MessageType = "Poll"
)

type Controller struct {
	inbox chan MessageType
}

func NewController(cfg *config.Config) Controller {
	return Controller{
		inbox: make(chan MessageType),
	}
}

func (c Controller) Run() error {
	for {
		msg := <-c.inbox
		err := c.handle(msg)
		if err != nil {
			return err
		}
	}
}

func (c Controller) handle(msg MessageType) error {
	return nil
}
