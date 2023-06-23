package godev

import (
	"github.com/cdvelop/model"
)

type ui struct {
	app
	theme_folder string
	//módulos registrados
	modules []*model.Module
	//componentes registrados
	registered    map[string]struct{}
	components    []model.Component
	folders_watch []string // ej: "modules", "ui\\theme"

	wasm_build  bool
	with_tinyGo bool
}

const WorkFolder = "frontend"
const BuiltFolder = "frontend/built"
const StaticFolder = "frontend/built/static"

var ui_store = ui{
	app:           nil,
	theme_folder:  "",
	modules:       []*model.Module{},
	registered:    map[string]struct{}{},
	components:    []model.Component{},
	folders_watch: []string{},
	with_tinyGo:   false,
}
