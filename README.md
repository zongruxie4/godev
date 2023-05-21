# godev

librería para trabajar con código: go para WebAssembly, css, js y compilarlo
según sea necesario en la ruta ui/built 

tiene incluido un observador de cambios de las rutas:
> modules
> ui/theme

observa cambios en archivos tipo js,css, html y ficheros go que contengan en el nombre wasm los compilara a WebAssembly si el tema contiene el fichero ui/theme/wasm/main.go

ejemplo de uso en la carpeta test/

saludos.