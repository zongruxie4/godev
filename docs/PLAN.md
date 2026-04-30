# PLAN: Fix CSS Hot-Reload — SSR Mode Not Active in In-Memory Dev

## Problem 1 — `SetExternalSSRCompiler` nunca se llama en modo in-memory

En `InitBuildHandlers()`, `SetExternalSSRCompiler` solo se llama dentro del callback
`SetOnExternalModeExecution`, que se dispara únicamente cuando el servidor cambia a
modo externo. En desarrollo in-memory normal **nunca se llama**.

Consecuencia: `assetmin.isSSRMode()` = `false` → cambios de CSS van por el path
non-SSR → `UpdateFileContentInMemory` usa full file path como clave → no hace match
con la clave por nombre de módulo registrada en startup → entrada duplicada →
`RegenerateCache` produce CSS stale + CSS nuevo combinados.

Test que falla en `tinywasm/assetmin`:
`tests/css_ssr_hotreload_test.go::TestCSSHotReload_NonSSRMode_KeyMismatchDuplicatesCSS`

## Problem 2 — `RefreshAsset` expuesta a `app` viola SRP

`app` llama `RefreshAsset(".css")`, `RefreshAsset(".js")`, `RefreshAsset(".html")`
directamente — conoce los tipos de asset internos de assetmin. `RefreshAsset` pasará
a privada en `tinywasm/assetmin` (ver `assetmin/docs/PLAN.md`). Este plan elimina
todos los usos desde `app` y usa la nueva API encapsulada.

---

## Changes — `section-build.go`

### Fix 1 — Activar SSR mode desde el inicio (~línea 52)

Después de crear `h.AssetsHandler`, antes del bloque `SetOnExternalModeExecution`:

```go
// Activate SSR hot-reload path from startup so CSS changes update the correct
// module-name keyed slot instead of appending a duplicate full-path entry.
h.AssetsHandler.SetExternalSSRCompiler(func() error { return nil }, false)
```

El callback `SetOnExternalModeExecution` ya existente puede llamarlo de nuevo con
`isExternal=true` para el modo disco — la segunda llamada sobreescribe limpiamente.

### Fix 2 — Eliminar `RefreshAsset` en `SetOnSSRFileChange` (líneas 208-210)

`ReloadSSRModule` ahora encapsula el refresh internamente. Las tres llamadas se
eliminan. El `return` en error se mantiene: es responsabilidad de `app` decidir si
recargar el browser cuando el módulo falla — assetmin no conoce el browser.

```go
// ANTES
if err := h.AssetsHandler.ReloadSSRModule(moduleDir); err != nil {
    h.AssetsHandler.Logger("SSR hot reload error:", err)
    return
}
h.AssetsHandler.RefreshAsset(".css")   // ← eliminar
h.AssetsHandler.RefreshAsset(".js")    // ← eliminar
h.AssetsHandler.RefreshAsset(".html")  // ← eliminar

// DESPUÉS
if err := h.AssetsHandler.ReloadSSRModule(moduleDir); err != nil {
    h.AssetsHandler.Logger("SSR hot reload error:", err)
    return // evita browser reload con assets inconsistentes
}
```

### Fix 3 — Reemplazar `RefreshAsset` en `OnWasmExecChange` (líneas 243-244)

```go
// ANTES
h.AssetsHandler.RefreshAsset(".js")
h.AssetsHandler.RefreshAsset(".html")

// DESPUÉS
h.AssetsHandler.RefreshJSAssets()
```

---

---

## Tests

### `TestInitBuildHandlers_SSRMode_InMemory`
Crea un handler real, llama `InitBuildHandlers()`, registra un módulo SSR con CSS
inicial, modifica el CSS en disco, dispara el evento de archivo, verifica que:
- El cache contiene el CSS actualizado
- El CSS stale del módulo anterior **no** está en el cache (sin duplicado)
- Ningún archivo fue escrito en disco (`buildOnDisk=false`)

**Por qué:** valida el wiring completo de la cadena
`InitBuildHandlers → SetExternalSSRCompiler(fn, false) → SSR path activo → cache
en memoria actualizado`. Sin este test, eliminar la línea de init rompe el bug
original sin que ningún test en app lo detecte.

### `TestInitBuildHandlers_SSRMode_Disk`
Misma secuencia pero activa el modo disco llamando `SetOnExternalModeExecution(true)`
tras el init. Dispara el mismo evento de archivo y verifica que:
- El archivo CSS de salida en disco existe y contiene el CSS actualizado
- El cache en memoria también es consistente con el disco

**Por qué:** `SetExternalSSRCompiler` tiene dos ramas de comportamiento —
`buildOnDisk=false` (in-memory) y `buildOnDisk=true` (disco). Si solo se testea uno,
un refactor puede romper el otro silenciosamente. Son contratos distintos de la misma
función de inicialización.

---

## Dependency

Requiere que `tinywasm/assetmin` implemente `RefreshJSAssets()` y haga `refreshAsset`
privada antes de aplicar este plan.

## Files Affected

| File | Change |
|------|--------|
| `section-build.go` | `SetExternalSSRCompiler(fn, false)` al init; eliminar `RefreshAsset`; usar `RefreshJSAssets()` |
| `test/css_hotreload_integration_test.go` | 2 tests de integración nuevos |
