package controller

import (
	"testing"

	"github.com/mindriot101/dockerdeploy/internal/config"
	"github.com/stretchr/testify/assert"
)

func dummyController() Controller {
	cfg := config.Config{}
	c := NewController(&cfg)
	return c
}

func TestHandlePollInstruction(t *testing.T) {
	c := dummyController()

	err := c.handle(Poll)
	assert.Nil(t, err)
}
