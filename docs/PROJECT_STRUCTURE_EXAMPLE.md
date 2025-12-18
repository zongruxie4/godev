### 📂 **Project Structure Example**
```plaintext
projectName/                  # ⚠️ MANDATORY STRUCTURE
├── .env                      # 🔐 Environment variables
├── .gitignore                # 🙈 Git ignore rules
├── go.mod                    # 📦 Go Module
├── docs/                     # 📚 Documentation
├── deploy/                   # 🚀 Deployment scripts and configurations
│   ├── docker/
│   │   └── Dockerfile        # 🐳 Docker for server
│   └── cloudflare/
│       └── wrangler.toml     # ☁️ Cloudflare configuration
│
│
├── modules/                  # 🔒 Business logic (not importable)
│   ├── modules.go            # 🔌 Module Registry (Init() []any)
│   ├── billing/              # 💰 Billing
│   ├── medical/              # 🏥 Medical
│   └── users/                # 👥 Users
│
├── pkg/                      # 📦 Shared code (safe to import)
│   ├── greet/                # 👋 Greeting
│   │   └── greet.go
│   └── router/               # 🛣️ Router
│       └── router.go
│
└── web/                      # 🌐 Frontend & Backend logic
    ├── client.go             # 🌐 Web client (//go:build wasm)
    ├── server.go             # 🔙 Go server (//go:build !wasm)
    ├── public/               # 📁 Static resources (HTML, CSS, JS, WASM, images)
    └── ui/                   # 🎨 Visual components, theme or layouts
```

**Why This Structure?**
- **Native Go Build Tags** - Uses `//go:build wasm` and `//go:build !wasm` (pure Go, no magic)
- **Single Directory** - All application code in `web/`, no unnecessary folder nesting
- **Zero Config Files** - No `package.json`, `webpack.config.js`, or `tsconfig.json`
- **LLM-Friendly** - Less directory jumping, clearer context for AI assistants
- **Go Idiomatic** - Build tags are standard Go practice

### 🔌 **Modules Loading Strategy**
The `modules/modules.go` file serves as the central entry point. It must implement an `Init() []any` function that returns all application handlers. This allows to:
1. **Single Pass Loading** - Initialize all modules once.
2. **Interface Based Registration** - Handlers are registered automatically based on the interfaces they implement (e.g., typically used in `routes.go` to register endpoints).
