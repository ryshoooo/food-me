package api

import (
	"fmt"
	"net/http"
	"time"

	foodme "github.com/ryshoooo/food-me/internal"
	"github.com/sirupsen/logrus"
)

func Cleaner(logger *logrus.Logger, period int) {
	logger.WithFields(logrus.Fields{"component": "cleaner"}).Infof("Starting the Cleaner")
	t := time.NewTicker(time.Duration(period) * time.Second)
	go func() {
		for {
			<-t.C
			logger.WithFields(logrus.Fields{"component": "cleaner"}).Infof("Cleaning up expired connections")
			usernames := foodme.GlobalState.GetExpiredUsernames()
			for _, username := range usernames {
				logger.WithFields(logrus.Fields{"component": "cleaner", "username": username}).Infof("Cleaning up expired connection")
				foodme.GlobalState.DeleteConnection(username)
			}
		}
	}()
}

func Start(logger *logrus.Logger, port, usernameLifetime, gcPeriod int) {
	logger.WithFields(logrus.Fields{"component": "api"}).Infof("Starting the API")

	go Cleaner(logger, gcPeriod)
	server := http.NewServeMux()
	server.HandleFunc("POST /connection", CreateNewConnection(logger, usernameLifetime))
	logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), server))
}
