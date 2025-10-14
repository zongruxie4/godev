### 📂 **Project Structure Example**
```plaintext
projectName/                        # ⚠️ MANDATORY STRUCTURE
├── go.mod                          # 📦 Go Module
├── docs/                           # 📚 Documentation
├── deploy/                         # 🚀 Deployment scripts and configurations
│   ├── appserver/
│   │   └── Dockerfile              # 🐳 Docker for server
│   └── edgeworker/
│       └── wrangler.toml           # ☁️ Cloudflare configuration
│
└── src/                            # 📁 Source code
    ├── cmd/                        # 🚀 Entry points: appserver, edgeworker, webclient
    │   ├── appserver/
    │   │   └── main.go             # 🔙 Go server
    │   ├── edgeworker/
    │   │   └── main.go             # ☁️ Edge worker
    │   └── webclient/
    │       └── main.go             # 🌐 Web client
    │
    ├── internal/                   # 🔒 Business logic (not importable)
    │   ├── billing/                # 💰 Billing
    │   ├── medical/                # 🏥 Medical
    │   └── users/                  # 👥 Users
    │
    ├── pkg/                        # 📦 Shared code (safe to import)
    │   ├── greet/                  # 👋 Greeting
    │   │   └── greet.go
    │   └── router/                 # 🛣️ Router
    │       └── router.go
    │
    └── web/                        # 🌐 Frontend assets
        ├── public/                 # 📁 Static resources (HTML, CSS, JS, WASM, images)
        └── ui/                     # 🎨 Visual components, theme or layouts
```