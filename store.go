package godev

import (
	"github.com/cdvelop/model"
)

type ui struct {
	app
	//componentes registrados
	registered    map[string]struct{}
	components    []model.Component
	folders_watch []string // ej: "modules", "ui\\theme"

	with_tinyGo bool
}

const WorkFolder = "frontend"
const BuiltFolder = "frontend/built"
const StaticFolder = "frontend/built/static"

var ui_store = ui{
	app:           nil,
	registered:    map[string]struct{}{},
	components:    []model.Component{},
	folders_watch: []string{},
	with_tinyGo:   false,
}
