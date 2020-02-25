// Manage docker containers and trigger refreshes on command
package main

import (
	"fmt"
	"log"

	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/mindriot101/dockerdeploy/config"
	"github.com/mindriot101/dockerdeploy/controller"
	"github.com/spf13/cobra"
)

var (
	sha1ver   string
	buildTime string

	// Used for flags
	configFilename string
)

func run(cmd *cobra.Command, args []string) {
	cfg, err := config.Parse(configFilename)
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

func main() {
	// Set up command line arguments
	rootCmd := &cobra.Command{
		Use:     "dockerdeploy",
		Short:   "Deploy and manage docker containers",
		Run:     run,
		Version: fmt.Sprintf("Binary sha %s built on %s\n", sha1ver, buildTime),
	}

	rootCmd.Flags().StringVarP(&configFilename, "config", "c", "", "FILENAME")
	rootCmd.MarkFlagRequired("config")
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}

}
