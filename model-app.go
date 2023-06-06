package godev

import "github.com/cdvelop/model"

type app interface {
	AppName() string
	AppVersion() string
	AppPort() string
	AppInProduction() bool
	Areas() map[byte]string
	Modules() []model.Module
	theme
}

type theme interface {
	FolderPath() string
	// PathTemplateIndexHTML with:
	// {{.StyleSheet}} {{.AppName}} {{.AppVersion}} {{.Menu}}
	// {{.Message}} {{.UserName}} {{.UserArea}} {{.Modules}} {{.Script}}
	PathTemplateIndexHTML() string
	// ej: nombre del module html y el contenido
	//<div id="%v" class="slider_panel">%v</div>
	ModuleHtmlTemplate() string
	MenuButtonTemplate() string
	ModuleJsTemplate() string
}
