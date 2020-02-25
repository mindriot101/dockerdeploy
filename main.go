// Manage docker containers and trigger refreshes on command
package main

import (
	"flag"
	"log"

	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/mindriot101/dockerdeploy/config"
	"github.com/mindriot101/dockerdeploy/controller"
)

var (
	sha1ver   string
	buildTime string
)

func main() {
	// Set up command line arguments
	version := flag.Bool("version", false, "Print the program version")
	configFilename := flag.String("config", "", "Config filename")

	flag.Parse()

	if *version {
		log.Printf("Binary sha %s built on %s\n", sha1ver, buildTime)
		return
	}

	// Validate the arguments
	if *configFilename == "" {
		log.Fatalf("config file argument `-config` not passed")
	}

	cfg, err := config.Parse(*configFilename)
	if err != nil {
		log.Fatal(err)
	}

	if err = cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	// Create the docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	controller, err := controller.NewController(controller.NewControllerOptions{
		Cfg:    cfg,
		Client: cli,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer controller.Close()

	// Create the web server
	r := gin.Default()
	r.POST("/trigger", controller.HandleTrigger)
	r.POST("/webhook", controller.HandleWebHook)

	controller.Listen()

	log.Fatal(r.Run())
}
