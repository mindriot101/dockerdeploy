package controller

import (
	"testing"

	"github.com/mindriot101/dockerdeploy/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestHandlePollInstruction(t *testing.T) {
	cfg := config.Config{}
	c := NewController(&cfg)

	err := c.handle(Poll)
	assert.Nil(t, err)
}
