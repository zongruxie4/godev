package setting

import (
	"github.com/cdvelop/godev/test/modules/info_test"
	"github.com/cdvelop/godev/test/modules/product_test"
	"github.com/cdvelop/model"
	"github.com/cdvelop/platform"
)

type app struct {
	platform.Theme
}

func App() *app {
	return &app{}
}

func (app) HotReload() bool {
	return false
}

func (app) AppName() string {
	return "myapp"
}

func (app) AppPort() string {
	return "8080"
}

func (app) AppVersion() string {
	return "0.0.0"
}

func (app) AppInProduction() bool {
	return false
}

func (a app) Areas() map[byte]string {
	return map[byte]string{
		'v': "Area Ventas",
		'a': "Area Administrativa",
		't': "Area TI Soporte",
	}
}

func (a app) Modules() []model.Module {
	return []model.Module{
		product_test.Add,
		info_test.Add,
	}
}
