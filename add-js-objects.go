package godev

import (
	"bytes"
	"strings"

	"github.com/cdvelop/model"
)

func (u ui) attachJsToModuleFromFieldsObject(module_name string, object *model.Object, funtions, listener_add, listener_rem *bytes.Buffer) {

	var input_registered = make(map[string]struct{}, 0)

	for _, field := range object.Fields {

		if _, exist := input_registered[field.Input.Name]; !exist && field.JsFunctions != nil {

			parsed_js := parseModuleJS(parseJS{
				ModuleName: module_name,
			}, []byte(field.JsFunctions.JsFunctions()))

			funtions.WriteString(parsed_js.String() + "\n")

			input_registered[field.Input.Name] = struct{}{}
		}

		if field.JsListeners != nil {

			parsed_js := parseModuleJS(parseJS{
				ModuleName: module_name,
				FieldName:  field.Name,
			}, []byte(field.JsListeners.JsListeners()))

			listener_add.WriteString(parsed_js.String() + "\n")

			// reemplazar todas las ocurrencias de "addEventListener" por "removeEventListener"
			rem_listeners := strings.ReplaceAll(parsed_js.String(), "addEventListener", "removeEventListener")

			listener_rem.WriteString(rem_listeners + "\n")

		}
	}

}
