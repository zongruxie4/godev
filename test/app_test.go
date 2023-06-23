package test

import (
	"github.com/cdvelop/godev/test/modules/module_info"
	"github.com/cdvelop/godev/test/modules/module_product"
	"github.com/cdvelop/model"
	"github.com/cdvelop/platform"
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

var theme = platform.Theme{}

var info_module = module_info.Add(theme)
var product_module = module_product.Add(theme)

// registrar módulos
var modules = []*model.Module{
	info_module,
	product_module,
}
