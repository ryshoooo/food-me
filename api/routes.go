package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	foodme "github.com/ryshoooo/food-me/internal"
	"github.com/sirupsen/logrus"
)

func HandleErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(&ApiError{Detail: message})
	if err != nil {
		panic(err)
	}
}

func CreateNewConnection(logger *logrus.Logger, usernameLifetime int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.WithFields(logrus.Fields{"component": "api"}).Infof("[%p] %s %s %s", r, r.Method, r.URL, r.RemoteAddr)
		defer r.Body.Close()
		data := &NewConnectionData{}
		err := json.NewDecoder(r.Body).Decode(data)
		if err != nil {
			logger.WithFields(logrus.Fields{"component": "api"}).Errorf("[%p] %s", r, err)
			HandleErrorResponse(w, http.StatusBadRequest, "Failed to parse request")
			return
		}
		id := uuid.New().String()
		foodme.GlobalState.AddConnection(id, data.AccessToken, data.RefreshToken, usernameLifetime)
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(&NewConnectionResponse{Username: id})
		if err != nil {
			panic(err)
		}
	}
}
