package godev

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

const js_wasm_format = `const go = new Go();
WebAssembly.instantiateStreaming(fetch("static/app.wasm"), go.importObject).then((result) => {
	go.run(result.instance);
});`

var with_tinyGo bool

func (u ui) addWasmJS(out_js *bytes.Buffer) {

	if u.AppInProduction() { // si existen los archivos js wasm agregamos la llamada a estos
		err := readFile("ui/theme/wasm/wasm_exec_tinygo.js", out_js)
		if err == nil {
			// fmt.Println("wasm_exec_tinygo.js")
			out_js.WriteString(js_wasm_format)
			with_tinyGo = true

		} else {

			err := readFile("ui/theme/wasm/wasm_exec.js", out_js)
			if err == nil {
				// fmt.Println("WASM EN PRODUCCIÓN JS: wasm_exec.js")
				out_js.WriteString(js_wasm_format)
			}
		}

	} else {
		err := readFile("ui/theme/wasm/wasm_exec.js", out_js)
		if err == nil {
			out_js.WriteString(js_wasm_format)
		}
	}

}

func (u ui) BuildWASM() {

	err := u.buildWASM("ui/theme/wasm/main.go", "ui/built/static/app.wasm")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

}

func (u ui) buildWASM(input_go_file string, out_wasm_file string) error {

	var cmd *exec.Cmd

	// Ajustamos los parámetros de compilación según la configuración
	if u.AppInProduction() && with_tinyGo {
		cmd = exec.Command("tinygo", "build", "-o", out_wasm_file, "-target", "wasm", input_go_file)

	} else {
		// compilación normal...
		cmd = exec.Command("go", "build", "-o", out_wasm_file, "-tags", "dev", "-ldflags", "-s -w", "-v", input_go_file)
		cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, string(output))
		return fmt.Errorf("al compilar a WebAssembly: %v", err)
	}

	// Verificamos si el archivo wasm se creó correctamente
	if _, err := os.Stat(out_wasm_file); err != nil {
		return fmt.Errorf("el archivo WebAssembly no se creó correctamente: %v", err)
	}

	// fmt.Printf("WebAssembly compilado correctamente y guardado en %s\n", out_wasm_file)

	return nil
}
