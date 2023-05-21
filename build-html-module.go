package godev

import (
	"fmt"
	"os"
	"strings"
)

func (u ui) buildHtmlModule(required_area byte) string {

	sb := strings.Builder{}

	for _, module := range u.Modules() {

		for _, module_area := range module.Areas {
			if module_area == required_area {

				// obtener index.html del modulo
				content, err := os.ReadFile("modules/" + module.Name + "/index.html")
				if err == nil && len(content) != 0 {
					sb.WriteString(fmt.Sprintf(u.ModuleHtmlTemplate()+"\n", module.Name, string(content)))
				}

				break
			}
		}
	}

	return sb.String()
}
