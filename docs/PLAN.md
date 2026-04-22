# PLAN: app — actualizar dependencias tras fix MCP content array

## Contexto

El paquete `tinywasm/mcp` tiene un bug donde el campo `content` de las respuestas MCP se serializa
como objeto en lugar de array, violando el protocolo MCP. El fix está en `tinywasm/mcp/docs/PLAN.md`.

`tinywasm/app` es el servidor principal que registra el MCP y no requiere cambios de lógica,
solo actualizar sus dependencias una vez publicados los fixes.

## Orden de ejecución

Este plan debe ejecutarse **después** de:
1. `tinywasm/mcp/docs/PLAN.md` — fix del bug y publicación
2. `tinywasm/devbrowser/docs/PLAN.md` — actualización de dependencia y publicación

## Pasos

### Paso 1 — Actualizar dependencias

```bash
cd /home/cesar/Dev/Project/tinywasm/app
go get github.com/tinywasm/mcp@latest
go get github.com/tinywasm/devbrowser@latest
go mod tidy
```

### Paso 2 — Compilar y verificar

```bash
go build ./...
```

### Paso 3 — Iniciar el servidor y probar con Claude Code

```bash
go run ./cmd/...
# o el comando habitual de inicio del proyecto
```

Desde Claude Code verificar:
- `browser_get_console` → sin error `expected array, received object`
- `browser_screenshot` → captura correcta
- `browser_get_content` → contenido HTML
- `app_get_logs` → logs del servidor

### Paso 4 — Publicar si es librería

```bash
gopush
```

## Archivos relevantes

- `mcp_registry.go` — registra providers (no requiere cambios)
- `mcp-tools.go` — tools del Handler (no requiere cambios)
- `go.mod` — versiones de dependencias a actualizar
