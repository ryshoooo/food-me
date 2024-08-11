package api

import (
	"encoding/json"
	"fmt"
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

func ApplyPermissionAgent(logger *logrus.Logger, conf *foodme.Configuration, httpClient foodme.IHttpClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.WithFields(logrus.Fields{"component": "api"}).Infof("[%p] %s %s %s", r, r.Method, r.URL, r.RemoteAddr)

		// Parse body
		defer r.Body.Close()
		data := &PermissionData{}
		err := json.NewDecoder(r.Body).Decode(data)
		if err != nil {
			logger.WithFields(logrus.Fields{"component": "api"}).Errorf("[%p] %s", r, err)
			HandleErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse request: %s", err))
			return
		}

		if data.Username == "" {
			logger.WithFields(logrus.Fields{"component": "api"}).Errorf("[%p] No username provided", r)
			HandleErrorResponse(w, http.StatusBadRequest, "No username provided")
			return
		}

		if data.SQL == "" {
			logger.WithFields(logrus.Fields{"component": "api"}).Errorf("[%p] No SQL provided", r)
			HandleErrorResponse(w, http.StatusBadRequest, "No SQL provided")
			return
		}

		// Validate data and state
		if !conf.OIDCEnabled {
			logger.WithFields(logrus.Fields{"component": "api"}).Errorf("[%p] OIDC is disabled", r)
			HandleErrorResponse(w, http.StatusFailedDependency, "OIDC is disabled")
			return
		}

		if !conf.PermissionAgentEnabled {
			logger.WithFields(logrus.Fields{"component": "api"}).Errorf("[%p] Permission agent is disabled", r)
			HandleErrorResponse(w, http.StatusFailedDependency, "Permission agent is disabled")
			return
		}

		at, rt := foodme.GlobalState.GetTokens(data.Username)
		if at == "" || rt == "" {
			logger.WithFields(logrus.Fields{"component": "api"}).Errorf("[%p] No tokens found for user %s", r, data.Username)
			HandleErrorResponse(w, http.StatusNotFound, "No tokens found for user "+data.Username)
			return
		}

		// Get userinfo
		cspec, ok := conf.OIDCDatabaseClients[data.Database]
		if !ok && !conf.OIDCDatabaseFallBackToBaseClient {
			logger.WithFields(logrus.Fields{"component": "api"}).Errorf("[%p] No client found for database %s", r, data.Database)
			HandleErrorResponse(w, http.StatusNotFound, "No client found for database "+data.Database)
			return
		} else if !ok {
			cspec = &foodme.OIDCDatabaseClientSpec{ClientID: conf.OIDCClientID, ClientSecret: conf.OIDCClientSecret}
		}

		oidcClient := foodme.NewOIDCClient(httpClient, cspec.ClientID, cspec.ClientSecret, conf.OIDCTokenURL, conf.OIDCUserInfoURL, at, rt)
		if !oidcClient.IsAccessTokenValid() {
			err = oidcClient.RefreshAccessToken()
			if err != nil {
				logger.WithFields(logrus.Fields{"component": "api"}).Errorf("[%p] %s", r, err)
				HandleErrorResponse(w, http.StatusUnauthorized, "Failed to refresh access token: "+err.Error())
				return
			}
		}

		uinfo, err := oidcClient.GetUserInfo()
		if err != nil {
			logger.WithFields(logrus.Fields{"component": "api"}).Errorf("[%p] %s", r, err)
			HandleErrorResponse(w, http.StatusUnauthorized, "Failed to get user info: "+err.Error())
			return
		}

		// Establish sql handler
		agent := foodme.NewPermissionAgent(conf, httpClient)
		if agent == nil {
			logger.WithFields(logrus.Fields{"component": "api"}).Errorf("[%p] Failed to create permission agent", r)
			HandleErrorResponse(w, http.StatusInternalServerError, "Failed to create permission agent")
			return
		}

		sqlHandler, err := foodme.NewSQLHandler(conf.DestinationDatabaseType, logger, agent)
		if err != nil {
			logger.WithFields(logrus.Fields{"component": "api"}).Errorf("[%p] %s", r, err)
			HandleErrorResponse(w, http.StatusInternalServerError, "Failed to create SQL handler: "+err.Error())
			return
		}

		err = sqlHandler.SetDDL(uinfo)
		if err != nil {
			logger.WithFields(logrus.Fields{"component": "api"}).Errorf("[%p] %s", r, err)
			HandleErrorResponse(w, http.StatusInternalServerError, "Failed to set DDL: "+err.Error())
			return
		}

		newSQL, err := sqlHandler.Handle(data.SQL, uinfo)
		if err != nil {
			logger.WithFields(logrus.Fields{"component": "api"}).Errorf("[%p] %s", r, err)
			HandleErrorResponse(w, http.StatusInternalServerError, "Failed to handle SQL: "+err.Error())
			return
		}

		// Respond
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(&PermissionApplyResponse{SQL: data.SQL, NewSQL: newSQL})
		if err != nil {
			panic(err)
		}
	}
}
