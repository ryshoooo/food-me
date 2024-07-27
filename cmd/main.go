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

	server := foodme.NewServer(conf, logger)
	go api.Start(logger, conf)
	logger.Fatal(server.Start())
}
