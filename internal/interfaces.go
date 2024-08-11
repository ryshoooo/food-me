package foodme

import (
	"net"
	"net/http"
)

type IHttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type IServer interface {
	Start() error
	Listen(listener net.Listener) error
}

type IHandler interface {
	Handle(client net.Conn) error
}

type IUpstreamHandler interface {
	Connect() (net.Conn, error)
}

type ISQLHandler interface {
	Handle(sql string, userInfo map[string]interface{}) (string, error)
	SetDDL(userInfo map[string]interface{}) error
}

type IPermissionAgent interface {
	SelectFilters(tableName, tableAlias string, userInfo map[string]interface{}) (*SelectFilters, error)
	CreateAllowed() bool
	UpdateAllowed() bool
	DeleteAllowed() bool
	SetCreateAllowed(userInfo map[string]interface{}) error
	SetUpdateAllowed(userInfo map[string]interface{}) error
	SetDeleteAllowed(userInfo map[string]interface{}) error
}
