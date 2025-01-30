# GoDEV

⚠️ **Warning: Development in Progress**
This project is currently under active development and may contain unstable features. NOT USE.

A live reload development environment for full stack web applications with Go and WebAssembly (PWA). When detecting file changes, it automatically reloads the browser and recompiles the application.


![godev tui preview](docs/tui.JPG)

## Table of Contents
- [Motivation](#motivation)
- [Features](#features)
- [Installation](#installation)
  - [Prerequisites](#prerequisites)
  - [Installing with go install](#installing-with-go-install)
- [Usage](#usage)
- [Project Structure](#project-structure)
- [Configuration](#configuration)
- [Acknowledgments](#acknowledgments)

## Motivación  

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


## Installation

### Prerequisites
 **Install Go**  
   Download and install Go from the [official Go website](https://go.dev/dl/).
   Verify installation with:
   
   go version

### Installing with go install
```bash	
go install -v github.com/cdvelop/godev/cmd/godev@latest
```

## Usage
Run the basic command:

godev


For help and available options:

godev

## Architecture
![godev architecture](docs/godev.arq.svg)


## Project Structure
godev uses `go.mod` as the reference point for your project:


project  
└── go.mod


### Module Structure
```
Module  
├── js  
│    ├── 1xFun.js
│    ├── func.js
│    ├── Help.js
│    └── main.js
├── jsTest
│    └── test.js
├── css  
│    ├── 1xStyle.css
│    ├── Help.css
│    └── main.css
└── Load.js
```


### JavaScript Loading Order
1. Root files starting with uppercase
2. Files in the `js` folder (alphabetically)
3. Files in the `jsTest` folder

### CSS Loading Order
Similar to JavaScript, but using the `css` folder.

## Configuration
- Default port: 8080 (http)
- HTTPS is used when port contains "44" (e.g., 4433)
- Module directories can be configured in `godev.yml`



## 📌 Roadmap  

### ✅ MVP (Versión Mínima Viable)  
- [ ] **Compilación y empaquetado básico:**  
  - [ ] Unificación y minificación de archivos **CSS** y **JavaScript** en `build/`  
  - [ ] Generación automática de `build/index.html` si este no existe  
- [ ] **Soporte para Go en frontend con WebAssembly**  
- [ ] **Servidor de desarrollo integrado** para servir archivos estáticos y WebAssembly  
- [ ] **Ejecución automática del navegador Chrome** (opcional con tecla `W`)  
- [ ] **Recarga en caliente (Hot Reload):**  
  - [ ] Detección de cambios en archivos Go, HTML, CSS y JS  
  - [ ] Recarga del navegador automáticamente  
- [ ] **Soporte para backend en Go:**  
  - [ ] Detección de cambios en archivos del servidor  
  - [ ] Reinicio automático si hay modificaciones  

---

### 🚀 Mejoras Futuras  
- [ ] **Compatibilidad con Docker para despliegue automatizado**  
- [x] **Interfaz TUI mejorada** con más opciones de configuración  
- [ ] **Modo producción:** Generación de artefactos optimizados y listos para deploy  
- [x] **Soporte para configuración mediante archivo `godev.yml`**  
- [ ] **Integración con framework de interoperabilidad Go ↔ JavaScript**  
- [ ] **Optimización en la carga de WebAssembly para mejorar rendimiento**  
- [ ] **Compatibilidad con servidores VPS para despliegue automatizado**  







## Acknowledgments
This project wouldn't be possible without:
- github.com/fsnotify
- github.com/chromedp
- github.com/tdewolff/minify
- github.com/fstanis/screenresolution
- github.com/lxn/win
- github.com/dustin/go-humanize
- github.com/mailru/easyjson
- github.com/gobwas/
- github.com/orisano/pixelmatch
- github.com/ledongthuc/pdf
- github.com/osharian/intern

For issues or support, please visit [GitHub Issues](https://github.com/cdvelop/godev/issues).
