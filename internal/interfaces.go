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
}

type IPermissionAgent interface {
	GetFilters(tableName, tableAlias string, userInfo map[string]interface{}) (string, error)
}
