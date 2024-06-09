package main

import (
	"fmt"
	"os"

	"github.com/ryshoooo/food-me/api"
	foodme "github.com/ryshoooo/food-me/internal"
)

func main() {
	conf, err := foodme.NewConfiguration(os.Args)
	if err != nil {
		fmt.Printf("Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	logger := foodme.NewLogger(conf)
	handler, err := foodme.GetHandler(conf, logger)
	if err != nil {
		logger.Errorf("Failed to establish a database handler: %s", err)
		os.Exit(1)
	}
	server := foodme.NewServer(conf.ServerPort, logger, handler)
	go api.Start(logger, conf.ApiPort)
	logger.Fatal(server.Start())
}
