package foodme

func NewPermissionAgent(conf *Configuration, httpClient IHttpClient) IPermissionAgent {
	switch conf.PermissionAgentType {
	case "opa":
		return NewOPASQL(conf.PermissionAgentOPAURL, conf.PermissionAgentOPAQueryTemplate, conf.PermissionAgentOPAStringEscapeCharacter, httpClient)
	default:
		return nil
	}
}
