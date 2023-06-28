package godev

import (
	"bytes"
	"strings"

	"github.com/cdvelop/model"
)

func (u ui) attachJsToModuleFromGoCode(m *model.Module, comp *model.Object, funtions, listener_add, listener_rem *bytes.Buffer) {

	if comp.JsFunctions != nil {

		parsed_js := parseModuleJS(parseJS{
			ModuleName: m.MainName,
			FieldName:  "",
		}, []byte(comp.JsFunctions.JsFunctions()))

		funtions.WriteString(parsed_js.String() + "\n")

	}

	if comp.JsListeners != nil {
		parsed_js := parseModuleJS(parseJS{
			ModuleName: m.MainName,
			FieldName:  "",
		}, []byte(comp.JsListeners.JsListeners()))

		listener_add.WriteString(parsed_js.String() + "\n")

		// reemplazar todas las ocurrencias de "addEventListener" por "removeEventListener"
		rem_listeners := strings.ReplaceAll(parsed_js.String(), "addEventListener", "removeEventListener")

		listener_rem.WriteString(rem_listeners + "\n")

	}

}
