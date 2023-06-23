package godev

import (
	"bytes"
	"strings"

	"github.com/cdvelop/model"
)

func attachJsFromGoComponentCodeToModule(m *model.Module, funtions, listener_add, listener_rem *bytes.Buffer) {

	for _, comp := range m.Components {

		if comp.JsPrivate != nil {

			parsed_js := parseModuleJS(parseJS{
				ModuleName: m.Name,
				FieldName:  "",
			}, []byte(comp.JsPrivate.JsPrivate()))

			funtions.WriteString(parsed_js.String() + "\n")

		}

		if comp.JsListeners != nil {
			parsed_js := parseModuleJS(parseJS{
				ModuleName: m.Name,
				FieldName:  "",
			}, []byte(comp.JsListeners.JsListeners()))

			listener_add.WriteString(parsed_js.String() + "\n")

			// reemplazar todas las ocurrencias de "addEventListener" por "removeEventListener"
			rem_listeners := strings.ReplaceAll(parsed_js.String(), "addEventListener", "removeEventListener")

			listener_rem.WriteString(rem_listeners + "\n")

		}

	}

}
