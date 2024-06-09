package api

type ApiError struct {
	Detail string `json:"detail"`
}

type NewConnectionData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type NewConnectionResponse struct {
	Username string `json:"username"`
}
