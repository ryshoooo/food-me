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

type PermissionData struct {
	Username string `json:"username"`
	Database string `json:"database"`
	SQL      string `json:"sql"`
}

type PermissionApplyResponse struct {
	SQL    string `json:"sql"`
	NewSQL string `json:"new_sql"`
}
