# GoDEV


Entorno de desarrollo [TUI](https://en.wikipedia.org/wiki/Text-based_user_interface) full stack con recarga en vivo, test, despliegue, ci/cd para aplicaciones web (PWA) con Go, WebAssembly y TinyGo.

⚠️ **Advertencia: Desarrollo en Progreso**
Este proyecto está actualmente en desarrollo activo y puede contener características inestables. NO USAR.

![vista previa de godev tui](docs/tui.JPG)

## Tabla de Contenidos
- [Motivación](#motivación)
- [Características](#características)
- [Instalación](#instalación)
  - [Prerrequisitos](#prerrequisitos)
  - [Instalación con go install](#instalación-con-go-install)
- [Uso](#uso)
- [Estructura del Proyecto](#estructura-del-proyecto)
- [Configuración](#configuración)
- [Hoja de ruta](#-hoja-de-ruta)
- [Agradecimientos](#prerrequisitos)
- [Contribuir](#contribuir)

## Motivación  

el principal problema que pretende resolver este framework es el facilitar el desarrollo de aplicaciones web de pila completa con Go, utilizando WebAssembly en el frontend y minimizando el uso de JavaScript.

el problema de otras implementaciones de webAssembly y go que desean escribir todo en go inclusive el css, ese enfoque de este framework quiere evitar ya que busca un equilibrio entre javascript y go, dejando el uso de javascript (syscall/js) para el manejo de la interfaz de usuario y el uso de go para la lógica de negocio.

otros proyectos de go en la web generan un único fichero webAssembly en la salida, generando un resultado de un archivo wasm muy grande y difícil de optimizar. el enfoque de este framework es que el desarrollo sea en módulos y estos ser compilados y optimizados por separado ya se a usando el compilador go o tinygo.

en este framework se quiero evitar en lo posible configuraciones interminables para iniciar un proyecto dado que su único lenguaje es go eso lo facilita.

¿Cansado de configuraciones complejas para desarrollar aplicaciones web? ¿Frustrado por depender de múltiples herramientas para compilar, recargar, desplegar, configurar Docker y VPS?  

**Godev** es una herramienta diseñada para compilar y desplegar proyectos **full stack con Go**, utilizando **WebAssembly en el frontend** y minimizando el uso de JavaScript. Su objetivo es ofrecer un flujo de trabajo integrado, eliminando la necesidad de configuraciones externas y facilitando el desarrollo con **hot reload, automatización de navegador y empaquetado optimizado**.  

## Características  

- **Automatización del navegador:** Recarga automática del navegador cuando hay cambios en archivos **Go (WebAssembly), HTML, CSS o JavaScript**. Se puede activar o desactivar presionando la tecla `W` en la interfaz TUI.

- **Hot Reload con ejecución de servidor:**  
  - Si el proyecto incluye un servidor, **Godev** lo recompila y reinicia automáticamente cuando detecta cambios.  
  - Si el proyecto es solo frontend con **Go/WebAssembly**, se ejecuta con un servidor integrado sin necesidad de configuración adicional.  

- **Compilación y empaquetado optimizado:**  
  - Minificación y unión automática de archivos **CSS y JavaScript**, generando un solo archivo optimizado para cada uno.  
  - No transpila TypeScript, Vue u otros frameworks, ya que está pensado para usar **JavaScript nativo en caso de ser necesario**.  
  - **Soporte automático para HTML**, donde el único archivo servido será `build/index.html`.

- **WebAssembly + Interoperabilidad con JavaScript:**  
  - Permite usar **Go y JavaScript en conjunto**.  
  - Un framework adicional proporcionará integración avanzada, pero **Godev** solo se encarga de empaquetar y desplegar. 
  - soporte con tinygo para la compilación de WebAssembly.

- **Despliegue automatizado:**  
  - **Soporte para Docker** (en desarrollo), permitiendo desplegar con un solo comando.  
  - Facilita la configuración de entornos de producción sin pasos manuales.  

- **Alternativa ligera a Webpack:**  
  - Similar a Webpack en el empaquetado, pero sin dependencias de JavaScript o CSS externas.  
  - Se enfoca en **Go como lenguaje principal** y minimiza los tiempos de carga optimizando los archivos generados. 

- **Uso de fichero de configuración mínimo**
  - para desarrollo no es necesario crear un fichero de configuración, este se creara automáticamente si cambias algún setting. 

## Instalación

### Prerrequisitos
 **Instalar Go**  
   Descarga e instala Go desde el [sitio web oficial de Go](https://go.dev/dl/).
   Verifica la instalación con:
   
   go version

### Instalación con go install
	
go install -v github.com/cdvelop/godev/cmd/godev@latest


## Uso
Ejecuta desde tu terminal preferida:

godev


## Arquitectura
![arquitectura godev](docs/godev.arq.svg)

## Estructura del Proyecto

dentro del directorio modules al modificar y guardar archivos go con prefijo: 
- **b.** (backend) el servidor se reiniciara y el navegador se recargará
- **f.** (frontend) se compilara a webAssembly y recargará el navegador

si el archivo no tiene prefijo se reiniciara el servidor, se compilara a webAssembly y 
se recargará el navegador
```md
miProyecto/
├── env              # Variables de entorno
├── .gitignore       # Archivos ignorados por git
├── modules/
│   ├── modules.go   # Registro de módulos en main.server.go, main.wasm.go
│   │
│   ├── auth/
│   │   ├── auth.go             # Estructuras y lógica compartida
│   │   ├── b.back.api.go       # API endpoints (// go: build !wasm)
│   │   ├── handlers.go         # Handlers compartidos
│   │   └── wasm/
│   │       └── auth.wasm.go    # modulo wasm (// go: build wasm)
│   │
│   ├── users/
│   │   ├── user.go             # Definición de estructuras y modelos
│   │   ├── b.api.go            # API endpoints
│   │   ├── f.events.go         # Definición de eventos pub/sub
│   │   └── wasm/
│   │       └── users.wasm.go   # modulo wasm (// go: build wasm)
│   │
│   └── medical/
│       ├── b.api.go            # API endpoints
│       ├── patient.go          # Modelo de paciente
│       ├── handlers.go         # Handlers compartidos
│       └── wasm/
│           └── medical.wasm.go # modulo wasm (// go: build wasm)
│
├── web/                        # servidor y Archivos web
│   ├── theme/                  # Archivos de tema
│   │   ├── css/                # Archivos CSS sin procesar
│   │   └── js/                 # Archivos JavaScript sin procesar
│   ├── public/                 # Archivos públicos
│   │   ├── img/                # Imágenes optimizadas y comprimidas
│   │   ├── icons.svg           # Iconos SVG
│   │   ├── main.js             # JavaScript minificado y concatenado
│   │   ├── style.css           # CSS minificado y concatenado
│   │   ├── wasm/               # Archivos WebAssembly compilados
│   │   │   ├── medical.wasm    # módulo medical
│   │   │   ├── users.wasm      # módulo users
│   │   │   ├── auth.wasm       # módulo auth
│   │   │   └── main.wasm       # main compilado de la aplicación principal
│   │   └── index.html          # HTML principal generado
│   ├── main.server.exe         # Ejecutable del servidor compilado
│   ├── main.server.go          # si existe el proyecto ya tiene servidor principal
│   └── main.wasm.go            # si existe el proyecto es WebAssembly
|
└── go.mod
```



## Configuración
- Puerto predeterminado: 8080 (http)

## 📌 Hoja de Ruta  

### ✅ MVP (Versión Mínima Viable)  
### Frontend
- [x] Unificación y minificación de archivos CSS y JavaScript 
- [ ] no compilar automáticamente js,css etc. al iniciar el servidor
- [ ] cargar assets del directorio `web/theme` primero (assets handler)
- [ ] Generación automática de `web/public/index.html` si este no existe  
- [ ] Compilar iconos svg módulos en sprite único en `web/public/icons.svg`

### Servidor de Desarrollo
- [ ] Servidor de desarrollo integrado para servir archivos estáticos en `web/public`
- [ ] https integrado en desarrollo local
- [x] cerrar navegador al cerrar aplicación 
- [x] Ejecución navegador Chrome (tecla `w`)  
- [x] cambiar el tamaño de la ventana del navegador desde la tui

### Hot Reload
- [x] Detección de cambios en archivos HTML, CSS, y JS  
- [x] detección de cambios en archivos GO frontend para webAssembly y servidor backend
- [ ] detectar cambios en archivos SVG
- [ ] Recarga en caliente del navegador (Hot Reload)

### Backend
- [x] Detección de cambios en archivos del servidor  
- [ ] Reinicio automático si hay modificaciones  

### Configuración
- [x] Interfaz TUI mejorada con más opciones de configuración  
- [x] Soporte para configuración mediante archivo `godev.yml`  
- [ ] agregar gitignore de forma automática
- [ ] crear env de forma automática (variables de entorno)

### 🚀 Mejoras Futuras  
- [ ] Integración de git  
- [ ] Modo producción: Generación de artefactos optimizados y listos para deploy  
- [ ] Compatibilidad con servidores VPS para despliegue automatizado  
- [ ] Compatibilidad con Docker para despliegue automatizado
- [ ] Integrar ayudante IA



### uses case
- [x] cuando se ejecuta el archivo servidor main.server.go y este tiene errores si este modifica en vivo, tiene que arrancar.

## Agradecimientos
Este proyecto no sería posible sin:
- github.com/fsnotify
- github.com/chromedp
- github.com/tdewolff/minify
- github.com/fstanis/screenresolution

Para problemas o soporte, por favor visita [GitHub Issues](https://github.com/cdvelop/godev/issues).

## Participar
si quieres participar en el proyecto puedes contactarme con un mensaje privado 


## Contribuir

Si encuentras útil este proyecto y te gustaría apoyarlo, puedes hacer una donación [aquí con paypal](https://paypal.me/cdvelop?country.x=CL&locale.x=es_XC)

Cualquier contribución, por pequeña que sea, es muy apreciada. 🙌