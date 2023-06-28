package test

import (
	"github.com/cdvelop/godev/test/modules/module_info"
	"github.com/cdvelop/godev/test/modules/module_product"
	"github.com/cdvelop/model"
	"github.com/cdvelop/platform"
)

var theme = platform.Theme{}

var info_module = module_info.Add(theme)
var product_module = module_product.Add(theme)

// registrar módulos
var modules = []*model.Module{
	info_module,
	product_module,
}
