# PLAN: tinywasm/app — Orquestación de SSR Module Extraction

## Objetivo

`tinywasm/app` orquesta la carga de assets SSR desde módulos externos al arrancar y
coordina el hot reload cuando un `ssr.go` cambia en un módulo local. La lógica de
extracción y registro vive en `tinywasm/assetmin` — aquí solo se configura y dispara.

## Dependencia

Este plan requiere que `tinywasm/assetmin/docs/PLAN.md` esté implementado primero.
Los nuevos métodos que se usan: `assetmin.LoadSSRModules()`, `assetmin.ReloadSSRModule()`.

## Cambio 1 — `assetmin.Config`: añadir `RootDir`

En `section-build.go`, `InitBuildHandlers()`, al crear `AssetsHandler`:

```go
h.AssetsHandler = assetmin.NewAssetMin(&assetmin.Config{
    OutputDir:          publicDir,
    RootDir:            h.RootDir,   // NUEVO
    GetSSRClientInitJS: func() (string, error) { ... },
    AppName:            h.FrameworkName,
    DevMode:            h.DevMode,
})
```

Sin este campo, `LoadSSRModules` no puede ejecutar `go list` en el directorio correcto.

## Cambio 2 — Carga inicial en `InitBuildHandlers()`

Después de crear el `Watcher` (paso 5) y configurar `GoModHandler` (paso 6), añadir:

```go
// 6b. SSR MODULE EXTRACTION — cargar assets de todos los módulos al arrancar
go func() {
    if err := h.AssetsHandler.LoadSSRModules(); err != nil {
        h.AssetsHandler.Logger("SSR load error:", err)
    }
}()
```

Corre en goroutine para no bloquear el arranque. El servidor queda disponible
inmediatamente; los assets SSR se inyectan en segundos.

## Cambio 3 — Hot reload via callback en `GoModHandler`

`devwatch` usa `depFinder.ThisFileIsMine` para eventos `.go` — assetmin con
`MainInputFileRelativePath()=""` nunca recibiría eventos `.go`. En lugar de cambiar
devwatch, se usa `GoModHandler` como relay (ya observa todos los replace-local paths).

Añadir en `InitBuildHandlers()` después de configurar `GoModHandler` (paso 6):

```go
// Wire SSR hot reload: GoModHandler notifica a assetmin cuando ssr.go cambia
h.GoModHandler.OnSSRFileChange = func(moduleDir string) {
    if err := h.AssetsHandler.ReloadSSRModule(moduleDir); err != nil {
        h.AssetsHandler.Logger("SSR hot reload error:", err)
    }
    // ReloadSSRModule ya llama processAsset internamente — no llamar RefreshAsset aquí
    if err := h.Browser.Reload(); err != nil {
        h.AssetsHandler.Logger("Browser reload error:", err)
    }
}
```

**Estado:** `OnSSRFileChange` ya existe en `GoModHandler` — implementado en
`tinywasm/devflow v0.4.16`. Solo falta wire el callback aquí.

## Sin cambios en comportamiento actual

- El watcher sigue funcionando igual para `.go`, `.js`, `.css`, `.html`, `.svg`
- `WasmClient` y `Server` siguen recibiendo sus eventos `.go` sin interferencia
- `assetmin` no intercepta eventos `.go` — su `SupportedExtensions` y `NewFileEvent` no cambian
- El hot reload de `ssr.go` llega por el callback `GoModHandler.OnSSRFileChange`, no por devwatch→assetmin
- Modo disco (`buildOnDisk=true`) respetado: `ReloadSSRModule` llama `processAsset` internamente

## Tests requeridos en `tinywasm/app`

### `ssr_integration_test.go` (test de integración)

| Test | Qué verifica | Prioridad |
|---|---|---|
| `TestSSRLoadOnInit` | `InitBuildHandlers` → CSS de módulos visible en bundle al arrancar | Alta |
| `TestSSRHotReload` | cambio en `ssr.go` de replace local → nuevo CSS en bundle sin reiniciar | Alta |
| `TestSSRNoBlockOnStartup` | servidor responde HTTP antes de que `LoadSSRModules` termine | Media |
| `TestSSRProxyModulesLoaded` | módulos de proxy (`$GOPATH/pkg/mod`) tienen CSS en bundle | Media |

Patrón: `t.TempDir()` + `os.WriteFile` para fixtures. `SetListModulesFn` para evitar red.

## Tests requeridos en `tinywasm/devflow`

Añadir a `test/gomod_handler_test.go`:

| Test | Qué verifica | Prioridad |
|---|---|---|
| `TestOnSSRFileChangeTriggered` | callback dispara solo para `ssr.go`, no para `model.go` ni `client.go` | Alta |
| `TestOnSSRFileChangeNilSafe` | `OnSSRFileChange == nil` → no panic al recibir evento en `ssr.go` | Alta |

## Cambio 4 — Integración de `imagemin` en `InitBuildHandlers()`

```go
h.ImageHandler = imagemin.New(&imagemin.Config{
    RootDir:   h.RootDir,
    OutputDir: filepath.Join(h.RootDir, h.Config.WebPublicDir(), "img"),
    Quality:   82,
})
h.ImageHandler.SetLog(h.Watcher.Logger)

// Carga inicial en goroutine — no bloquea arranque
go func() {
    if err := h.ImageHandler.LoadImages(); err != nil {
        h.ImageHandler.Logger("Image load error:", err)
    }
}()

// Encadenar callback SSR — GoModHandler ya notifica a assetmin (Cambio 3)
// imagemin se añade a la cadena sin sobreescribir el callback existente
existingSSRCallback := h.GoModHandler.OnSSRFileChange
h.GoModHandler.OnSSRFileChange = func(moduleDir string) {
    if existingSSRCallback != nil {
        existingSSRCallback(moduleDir)
    }
    if err := h.ImageHandler.ReloadModule(moduleDir); err != nil {
        h.ImageHandler.Logger("Image hot reload error:", err)
    }
}
```

## Orden de implementación

1. Implementar `tinywasm/assetmin` PLAN completo
2. Implementar `tinywasm/imagemin` PLAN completo
3. Actualizar `go.mod` de `tinywasm/app` para las nuevas versiones
4. Añadir `RootDir` en `section-build.go` al crear `AssetsHandler`
5. Añadir `LoadSSRModules()` en goroutine (Cambio 2)
6. Añadir wire del callback SSR para assetmin (Cambio 3)
7. Añadir `ImageHandler` + encadenamiento de callback (Cambio 4)
8. Correr tests de integración
9. Verificar hot reload manualmente con `clinical_encounter/ssr.go`

## Decisiones de diseño incorporadas

| Decisión | Opción elegida | Justificación |
|---|---|---|
| Hot reload entrega | `GoModHandler.OnSSRFileChange` callback | devwatch bloquea `.go` en assetmin |
| Arranque bloqueante | Goroutine + `WaitForSSRLoad` solo en tests | Servidor listo inmediatamente |
| Fallo de `go list` | Degradar a replace-locals + warning log | No bloquear dev offline |
| Pattern de tests | `t.TempDir()` + `os.WriteFile` + `listModulesFn` inyectable | Sin red, reproducible |

## Patrón de test de integración

```go
func TestSSRHotReload(t *testing.T) {
    moduleDir := t.TempDir()
    os.WriteFile(filepath.Join(moduleDir, "ssr.go"), []byte(`
//go:build !wasm
package mypkg
func RenderCSS() string { return ".v1 { color: red; }" }
`), 0644)

    am := assetmin.NewAssetMin(&assetmin.Config{...})
    am.SetListModulesFn(func(root string) ([]string, error) {
        return []string{moduleDir}, nil
    })
    am.LoadSSRModules()
    am.WaitForSSRLoad(2 * time.Second)

    // Verificar CSS inicial
    // Simular cambio en ssr.go y verificar que CSS se actualiza
}
```

## Estado de dependencias

| Módulo | Estado | Versión |
|---|---|---|
| `tinywasm/devflow` | ✅ Completado | v0.4.16 — `OnSSRFileChange` implementado |
| `tinywasm/assetmin` | Pendiente | requiere implementar PLAN.md de assetmin |
| `tinywasm/app` | Pendiente | requiere assetmin publicado |
