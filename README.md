# GoDEV

⚠️ **Warning: Development in Progress**
This project is currently under active development and may contain unstable features. NOT USE.

A live reload development environment for full stack web applications with Go and WebAssembly in frontend (PWA). When detecting file changes, it automatically reloads the browser and recompiles the application.


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

## Motivation
This project is designed to bring **Go** to the frontend compiled to **WebAssembly**, taking advantage of its static typing on the client side (app domain) for better maintenance (avoiding **"only read code"**). It uses vanilla JavaScript and CSS without dependencies on libraries or frameworks.

## Features
- Chrome browser automation
- File watching with hot reload
- Code compilation (HTML, CSS, JS, and WebAssembly)
- Similar functionality to Webpack but specifically designed for Go web fullstack projects

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
