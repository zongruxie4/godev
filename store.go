package godev

import (
	"github.com/cdvelop/model"
)

type ui struct {
	app
	//componentes registrados
	registered    map[string]struct{}
	components    []model.Component
	build_folder  string   // ej: "ui/built"
	folders_watch []string // ej: "modules", "ui\\theme"

	with_tinyGo bool
}

var ui_store = ui{
	app:           nil,
	registered:    map[string]struct{}{},
	components:    []model.Component{},
	build_folder:  "ui/built",
	folders_watch: []string{},

	with_tinyGo: false,
}
