package godev

import (
	"bytes"
	"log"
	"text/template"
)

type parseJS struct {
	ModuleName string
	FieldName  string
}

func parseModuleJS(p parseJS, data []byte) (js_out bytes.Buffer) {

	t, err := template.New("").Parse(string(data))
	if err != nil {
		log.Println(err)
		return
	}

	err = t.Execute(&js_out, p)
	if err != nil {
		log.Println(err)
		return
	}
	return
}
