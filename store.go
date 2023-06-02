package godev

import (
	"net/http"

	"github.com/cdvelop/model"
)

type ui struct {
	app
	http_server_mux *http.ServeMux
	//componentes registrados
	registered map[string]struct{}
	components []model.Component
}

var ui_store = ui{
	app:             nil,
	http_server_mux: nil,
	registered:      map[string]struct{}{},
	components:      []model.Component{},
}
