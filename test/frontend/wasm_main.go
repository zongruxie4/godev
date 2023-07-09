package main

import "syscall/js"

func main() {
	js.Global().Get("console").Call("log", "¡Hi 3 Go y WebAssembly!")

	select {}

}
