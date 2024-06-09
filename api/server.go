package api

import (
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

func Start(logger *logrus.Logger, port int) {
	logger.WithFields(logrus.Fields{"component": "api"}).Infof("Starting the API")

	server := http.NewServeMux()
	server.HandleFunc("POST /connection", CreateNewConnection(logger))
	logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), server))
}
