# PLAN: tinywasm/app — Correcciones de Tests Fallidos

## Resumen

Seis bugs identificados. Tres ya aplicados en este módulo (2, 3, 4). Tres pendientes
de dependencias externas o corrección de tests (1, 5, 6).

---

## Bug 1 — Tests SSR en `ssr_integration_test.go` (tests mal escritos) — Pendiente

### Síntomas
```
TestSSRLoadOnInit:         Expected CSS to contain '.my-class'
TestSSRHotReload:          Expected CSS to contain '.v1'
TestSSRProxyModulesLoaded: Expected CSS from proxy module to be loaded
```

### Causa

Los tests crean módulos con `ssr.go` pero no crean archivos `.go` en `RootDir` que los
importen. `ScanProjectImports` devuelve vacío → el CSS nunca se inyecta. No es un bug
de assetmin; es el comportamiento documentado.

### Corrección (en `ssr_integration_test.go`)

En cada test que cree un módulo externo con `ssr.go`, añadir:

```go
os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testapp\ngo 1.21\n"), 0644)
os.WriteFile(filepath.Join(root, "main.go"), []byte(
    "package main\nimport _ \"testapp/mymodule\"\nfunc main() {}",
), 0644)
h.ListModulesFn = func(rootDir string) ([]string, error) {
    return []string{moduleDir}, nil
}
```

**Alternativa**: usar `h.AssetsHandler.UpdateSSRModule(name, css, js, html, icons)` para
inyectar assets directamente en memoria y testear solo el wiring de `InitBuildHandlers`.

---

## Bug 2 — `GoModHandler` no recibe `SetRootDir` ✅ Aplicado

`start.go`: añadida línea `goModHandler.SetRootDir(startDir)`.

---

## Bug 3 — Editar `greet.go` no dispara compilación WASM ✅ Resuelto (depfind publicado)

Fallback en `ThisFileIsMine` cuando `go list` falla implementado y publicado en
`tinywasm/depfind`. Actualizar la dependencia en este módulo para cerrar.

---

## Bug 4 — Logs del daemon van a pestaña BUILD en lugar de MCP ✅ Aplicado

`sse_publisher.go`: `PublishLog` apunta a tab `"MCP"` con `colorOrangeLight`.

---

## Bug 5 — Terminal no se limpia al cerrar en modo standalone — Pendiente (requiere devtui)

`devtui.Start()` debe aceptar `chan bool` y cerrarlo al terminar. Ver
`tinywasm/devtui/docs/PLAN.md`.

En `app/start.go` (bloque standalone), una vez devtui publicado:
```go
go h.Tui.Start(&wg, ExitChan)
```

Verificación:
```bash
tinywasm  # Ctrl+C → terminal limpio, sin proceso zombie
```

---

## Bug 6 — Data race `WaitForSSRLoad` / `LoadSSRModules` ✅ Aplicado

`section-build.go`: eliminado `go func(){}` wrapper de `LoadSSRModules()` (assetmin ya
lanza su propio goroutine internamente). Corrección de assetmin publicada.

---

## Orden de ejecución

| # | Tarea | Archivo | Estado |
|---|-------|---------|--------|
| 1 | Corregir tests SSR: crear estructura de proyecto válida | `ssr_integration_test.go` | Pendiente |
| 2 | `goModHandler.SetRootDir(startDir)` | `start.go` | ✅ Aplicado |
| 3 | Actualizar dependencia depfind (fallback ya publicado) | `go.mod` | Pendiente |
| 4 | `PublishLog` apunta a tab `"MCP"` con `colorOrangeLight` | `sse_publisher.go` | ✅ Aplicado |
| 5 | Pasar `ExitChan` a `h.Tui.Start()` en bloque standalone | `start.go` | Pendiente (requiere devtui) |
| 6 | Quitar `go func(){}` de `LoadSSRModules` en `section-build.go` | `section-build.go` | ✅ Aplicado |
