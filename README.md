# godev

librería para trabajar con código: go para WebAssembly, css, js y compilarlo
según sea necesario en la ruta frontend/built 

tiene incluido un observador de cambios de la ruta por defecto:
> modules


observa cambios en archivos tipo js,css, html y ficheros go que contengan en el nombre wasm los compilara a WebAssembly si el proyecto en su ruta raíz contiene el fichero frontend/main.go

ejemplo de uso en la carpeta test/

saludos.