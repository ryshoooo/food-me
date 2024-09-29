package foodme

func NewPermissionAgent(conf *Configuration, httpClient IHttpClient) IPermissionAgent {
	switch conf.PermissionAgentType {
	case "opa":
		return NewOPASQL(
			conf.PermissionAgentOPAURL,
			conf.PermissionAgentOPASelectQueryTemplate,
			conf.PermissionAgentOPACreateQuery,
			conf.PermissionAgentOPAUpdateQuery,
			conf.PermissionAgentOPADeleteQuery,
			conf.PermissionAgentOPAStringEscapeCharacter,
			httpClient,
		)
	case "http":
		return NewHTTPPermissionAgent(conf.PermissionAgentHTTPDDLEndpoint, conf.PermissionAgentHTTPSelectEndpoint, httpClient)
	default:
		return nil
	}
}
