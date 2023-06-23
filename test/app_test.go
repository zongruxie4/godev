package test

import (
	"github.com/cdvelop/godev/test/modules/module_info"
	"github.com/cdvelop/godev/test/modules/module_product"
	"github.com/cdvelop/model"
)

type app struct {
}

func App() *app {
	return &app{}
}

func (app) AppName() string {
	return "myapp"
}

func (app) AppVersion() string {
	return "0.0.0"
}

func (app) AppInProduction() bool {
	return false
}

// registrar módulos
var modules = []*model.Module{
	module_info.Add(),
	module_product.Add(),
}
