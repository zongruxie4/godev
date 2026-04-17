# PLAN: Corrección de inconsistencias en tinywasm/app

## Development Rules
- **SRP:** Cada archivo debe tener una única responsabilidad bien definida.
- **DI obligatoria:** Sin estado global. Interfaces para dependencias externas.
- **Estructura plana:** Sin subdirectorios en librerías Go.
- **Máximo 500 líneas por archivo.** Los que superen ese límite deben subdividirse.

---

## Inconsistencias encontradas

### 1. `daemon.go` supera 500 líneas (545 líneas)

**Problema:** `daemon.go` mezcla en un solo archivo: HTTP server setup, herramienta `start_development`, lifecycle del proyecto (`startProject`, `stopProject`, `restartCurrentProject`, `runProjectLoop`), y la función utilitaria `unquote`. Viola SRP y el límite de 500 líneas.

**Corrección sugerida:** Dividir en:
- `daemon.go` — solo `runDaemon()` + setup HTTP (~200 líneas)
- `daemon_provider.go` — `daemonToolProvider`, `startProject`, `stopProject`, `restartCurrentProject`, `runProjectLoop`
- `utils.go` (nuevo o existente) — función `unquote`

---

### 2. Lógica de acción duplicada: `POST /mcp` vs `POST /tinywasm/action`

**Archivo:** `daemon.go` líneas 157–195 y 222–261

**Problema:** El handler de `POST /mcp` intercepta los métodos `tinywasm/state` y `tinywasm/action` antes de pasarlos a `mcpServer.HandleMessage`. El mismo dispatch de acciones (`start`, `stop`, `restart`, `DispatchAction`) está duplicado en el endpoint `POST /tinywasm/action`. Cualquier cambio en la lógica de acciones debe hacerse en dos lugares.

**Corrección sugerida:** Extraer el dispatch a una función privada `dispatchAction(key, value string)` que ambos handlers llamen. Considerar eliminar el interceptor `tinywasm/action` en `/mcp` y redirigir siempre a `/tinywasm/action`.

---

### 3. Bug en detección de puerto al reiniciar proyecto

**Archivo:** `daemon.go` línea 434

```go
conn, err := net.Dial("tcp", "localhost:"+os.Getenv("PORT"))
if err != nil {
    conn8080, err8080 := net.Dial("tcp", "localhost:8080")
```

**Problema:** Si `PORT` está vacío, `net.Dial("tcp", "localhost:")` falla con error de dirección inválida, no porque el puerto esté libre. El código entonces cae al fallback de `8080`, lo que puede causar que `portLoop` salga prematuramente pensando que el puerto está libre cuando en realidad nunca lo verificó correctamente.

**Corrección sugerida:**
```go
port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}
conn, err := net.Dial("tcp", "localhost:"+port)
if err != nil {
    break portLoop // puerto libre
}
conn.Close()
```

---

### 4. Nombre de archivo no estándar: `mcp-tools.go`

**Archivo:** `mcp-tools.go`

**Problema:** Go convenciona nombres de archivo con guiones bajos (`_`), no guiones (`-`). Herramientas como `go vet` y algunos linters reportan advertencias. Los otros archivos MCP usan `mcp_ide.go`, `mcp_registry.go`.

**Corrección sugerida:** Renombrar a `mcp_tools.go`.

---

### 5. Comentario obsoleto en `mcp_registry.go`

**Archivo:** `mcp_registry.go` líneas 56–62

```go
// Re-map tools from old to new if necessary.
// We know devbrowser.GetMCPTools() returns []mcp.Tool but probably the OLD mcp.Tool struct.
// HOWEVER, since devbrowser v0.3.19 ALREADY depends on mcp v0.1.1, its GetMCPTools
// should already return the NEW mcp.Tool struct if it compiles.
// Let's assume it returns the new struct but with old method name.
```

**Problema:** Comentario de migración temporal que ya no aplica. Ensucia el código y puede confundir a futuros lectores.

**Corrección sugerida:** Eliminar el comentario. Si hay dudas sobre la compatibilidad, resolverlas en código (type assertion explícita) o en un test.

---

### 6. `start_development` usa `Action: 'c'` pero el SKILL documentaba `'u'`

**Archivo:** `daemon.go` línea 340

**Problema:** La acción `'c'` (create) es semánticamente incorrecta para `start_development`, que inicia/cambia un proyecto ya existente. Debería ser `'u'` (update) ya que modifica el estado del entorno activo. El SKILL.md ya fue corregido para reflejar `'c'`.

**Decisión pendiente:** Confirmar si el Authorizer personalizado del proyecto usa la acción para filtrar permisos. Si `Can()` siempre devuelve `true` (como en los built-ins), la acción es solo semántica. Aún así, usar `'u'` es más preciso.

---

### 7. `mcp_ide.go` usa `encoding/json` y `fmt` stdlib

**Archivo:** `mcp_ide.go` líneas 1–10

**Problema:** Este archivo usa `encoding/json` y `fmt` del stdlib de Go en lugar de `tinywasm/json` y `tinywasm/fmt`. No tiene build tag `//go:build !wasm`, por lo que si alguna vez se incluye en una compilación WASM romperá la build.

**Corrección sugerida:** Agregar `//go:build !wasm` al archivo (es código server-only: escribe ficheros de configuración de IDEs). Documentar explícitamente que este archivo no es WASM-compatible.

---

## Prioridad de corrección

| # | Severidad | Cambio |
|---|-----------|--------|
| 3 | Alta | Bug en detección de puerto → puede causar arranques prematuros |
| 2 | Media | Duplicación de lógica de acciones → mantenibilidad |
| 7 | Media | Falta build tag `!wasm` en `mcp_ide.go` → riesgo de compilación |
| 1 | Baja | Dividir `daemon.go` → cumplir regla de 500 líneas |
| 4 | Baja | Renombrar `mcp-tools.go` → convención Go |
| 5 | Baja | Eliminar comentario obsoleto en `mcp_registry.go` |
| 6 | Info | Confirmar semántica de `Action` en `start_development` |
