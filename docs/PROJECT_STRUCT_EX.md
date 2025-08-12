### 📂 **Estructura del Proyecto**
```plaintext
AppName/                        # ⚠️ ESTRUCTURA OBLIGATORIA
├── cmd/                        # 📋 Aplicación de consola (opcional)
│   └── AppName/
│       └── main.go             # Punto de entrada CLI
│
├── modules/                    # 🔧 Lógica modular (obligatorio)
│   ├── modules.go              # Registro de módulos
│   │
│   ├── home/                   # 🏠 Módulo home con autenticación
│   │   ├── auth.go             # Estructuras y lógica de autenticación
│   │   ├── api.go              # 🔙 Backend API (// +build !wasm)
│   │   ├── auth.go             # 🌐 Frontend autenticación (// +build wasm)
│   │   └── handlers.go         # Handlers compartidos
│   │
│   ├── users/                  # 👥 Módulo de usuarios
│   │   ├── user.go             # Modelos de datos
│   │   ├── api.go              # 🔙 Backend API endpoints
│   │   ├── users.go            # 🌐 Frontend usuarios (// +build wasm)
│   │   └── events.go           # 🌐 Frontend eventos pub/sub
│   │
│   └── medical/                # 🏥 Módulo médico (ejemplo)
│       ├── patient.go          # Modelo de paciente
│       ├── api.go              # 🔙 Backend API
│       ├── medical.go          # 🌐 Frontend médico (// +build wasm)
│       └── handlers.go         # Handlers HTTP
│
├── pwa/                        # 📱 Progressive Web App (una de las 3)
│   ├── theme/                  # 🎨 Assets de desarrollo
│   │   ├── css/                # CSS sin procesar
│   │   └── js/                 # JavaScript sin procesar
│   │
│   ├── public/                 # � Assets finales (generados)
│   │   ├── img/                # Imágenes optimizadas
│   │   ├── icons.svg           # Sprite de iconos SVG
│   │   ├── main.js             # JavaScript minificado
│   │   ├── style.css           # CSS minificado
│   │   ├── AppName.wasm        # 🎯 WebAssembly compilado
│   │   ├── manifest.json       # Manifiesto PWA
│   │   ├── sw.js               # Service Worker
│   │   ├── icons/              # Iconos PWA
│   │   │   ├── icon-192x192.png
│   │   │   └── icon-512x512.png
│   │   ├── offline.html        # Página offline
│   │   └── index.html          # HTML principal generado
│   │
│   ├── main.server.go          # 🔙 Servidor Go (opcional)
│   └── main.wasm.go            # 🌐 Entry point WebAssembly (opcional)
│
├── spa/                        # 🌐 Single Page Application (alternativa)
│   ├── theme/                  # 🎨 Assets de desarrollo
│   ├── public/                 # 📁 Assets finales
│   ├── main.server.go          # 🔙 Servidor Go (opcional)
│   └── main.wasm.go            # 🌐 Entry point WebAssembly (opcional)
│
├── mpa/                        # 🌐 Multi-Page Application (alternativa)
│   ├── theme/                  # 🎨 Assets de desarrollo
│   ├── public/                 # 📁 Assets finales
│   ├── main.server.go          # 🔙 Servidor Go (opcional)
│   └── main.wasm.go            # 🌐 Entry point WebAssembly (opcional)
│
├── go.mod                      # 📦 Módulo Go
├── env                         # 🔧 Variables de entorno
└── .gitignore                  # 📋 Archivos ignorados por git
```