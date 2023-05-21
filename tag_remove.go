package godev

import "strings"

// quita solo 2 etiquetas principio y final <scrip></script>
func RemoveTagStartdEnd(in *string, tagRemove string) {
	lines := strings.Split(string(*in), "\n")
	for i, line := range lines { //desde el inicio
		if strings.Contains(line, tagRemove) {
			lines = lines[i+1:]
			break //salgo del for
		}
	}

	for i := len(lines) - 1; i > 0; i-- { //desde el final
		if strings.Contains(lines[i], tagRemove) {
			lines = lines[:i]
			break //salgo del for
		}
	}
	*in = strings.Join(lines, "\n")
}
