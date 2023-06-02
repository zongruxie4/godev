package godev

import "fmt"

func (a *Args) ProcessProgramOutput() {
	for a.Scanner.Scan() {
		line := a.Scanner.Text()

		fmt.Println(line)

		// switch {
		// case strings.Contains(line, "setting_app"):
		// Esperar 100 milisegundos antes de utilizar la dirección de memoria
		// time.Sleep(500 * time.Millisecond)

		// Asigna el puntero a la variable a.app si es necesario

		// var pPointer uint64
		// extractArgumentIn64(line, &pPointer)

		// fmt.Println("Puntero a la aplicación: ", pPointer)
		// Convertir el puntero a un entero ej: 0x7ff735103380
		// la cadena comienza con "0x", que indica una representación hexadecimal.
		// Por lo tanto, debes especificar la base 16 al convertir la cadena en un entero.
		// pointerValue, err := strconv.ParseUint(pPointer, 16, 64)
		// if err != nil {
		// 	fmt.Println("Error: No se pudo convertir el puntero a entero")
		// 	return
		// }

		// Convertir el número entero en un puntero
		// convertedPtr := unsafe.Pointer(uintptr(pPointer))

		// Convertir la cadena hexadecimal a un número entero
		// pointerValue, err := strconv.ParseUint(pPointer, 0, 64)
		// if err != nil {
		// 	fmt.Printf("Error: No se pudo convertir el puntero %v a entero", pPointer)
		// 	return
		// }

		// Convertir el uintptr en un puntero
		// ptr := unsafe.Pointer(uintptr(pointerValue))

		// Obtener el puntero deseado
		// a.App = (*model.App)(convertedPtr)

		// fmt.Println("Puntero obtenido:", a.App.AppName)

		// }

	}
}
