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

func Start(logger *logrus.Logger, conf *foodme.Configuration) {
	logger.WithFields(logrus.Fields{"component": "api"}).Infof("Starting the API")

	go Cleaner(logger, conf.ApiGarbageCollectionPeriod)
	server := http.NewServeMux()
	httpClient := &http.Client{}
	server.HandleFunc("POST /connection", CreateNewConnection(logger, conf.ApiUsernameLifetime))
	server.HandleFunc("POST /permissionapply", ApplyPermissionAgent(logger, conf, httpClient))
	if conf.APITLSEnabled {
		logger.Fatal(http.ListenAndServeTLS(fmt.Sprintf(":%v", conf.ApiPort), conf.ServerTLSCertificateFile, conf.ServerTLSCertificateKeyFile, server))
	} else {
		logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", conf.ApiPort), server))
	}

}
