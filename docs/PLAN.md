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
        return // no recargar browser si el módulo falló
    }
    // ReloadSSRModule actualiza slots en memoria; RefreshAsset regenera el cache HTTP
    h.AssetsHandler.RefreshAsset(".css")
    h.AssetsHandler.RefreshAsset(".js")
    h.AssetsHandler.RefreshAsset(".html")
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
- Modo disco (`buildOnDisk=true`) respetado: `RefreshAsset` llama `processAsset` que escribe al disco si aplica

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

### 4a — Añadir `ImageHandler` al struct `Handler` en `handler.go`

```go
// Build dependencies
AssetsHandler *assetmin.AssetMin
ImageHandler  *imagemin.Handler   // NUEVO
```

### 4b — Añadir `Logger()` público a `imagemin.Handler`

`imagemin.Handler` necesita el mismo patrón que `assetmin.AssetMin`: un método
`Logger()` público que delegue al `log` interno. Sin esto el código de `app` no
puede llamar `h.ImageHandler.Logger(...)`.

```go
func (h *Handler) Logger(messages ...any) {
    h.log(messages...)
}
```

### 4c — Crear el handler y conectarlo en `InitBuildHandlers()`

Debe ejecutarse **después del paso 5** (Watcher ya creado) para que `h.Watcher.Logger` no sea nil.

```go
// Inicializar DESPUÉS de crear h.Watcher (paso 5)
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

// Encadenar sobre el callback ya wired de assetmin (Cambio 3)
// No sobreescribir: assetmin + imagemin deben ejecutarse ambos
existingSSRCallback := h.GoModHandler.OnSSRFileChange
h.GoModHandler.OnSSRFileChange = func(moduleDir string) {
    if existingSSRCallback != nil {
        existingSSRCallback(moduleDir) // assetmin: reload SSR + RefreshAsset + Browser.Reload
    }
    // imagemin: re-procesar imágenes del módulo; Browser.Reload ya fue llamado arriba
    if err := h.ImageHandler.ReloadModule(moduleDir); err != nil {
        h.ImageHandler.Logger("Image hot reload error:", err)
    }
}
```

### 4d — Excluir carpeta de salida de imágenes en devwatch

En `section-build.go`, añadir dentro del closure `UnobservedFiles`:

```go
uf = append(uf, h.ImageHandler.UnobservedFiles()...)
```

Sin esto devwatch observa `/public/img` y dispara eventos espurios cada vez que
imagemin escribe un `.webp`.

### 4e — Añadir `imagemin` a `tinywasm/imagemin/imagemin.go`

Añadir `Logger()` antes de publicar nueva versión de imagemin.

## Orden de implementación

1. Implementar `tinywasm/assetmin` PLAN completo
2. Añadir `Logger()` a `imagemin.Handler` + publicar nueva versión (Cambio 4b, 4e)
3. Actualizar `go.mod` de `tinywasm/app` para las nuevas versiones
4. Añadir `ImageHandler *imagemin.Handler` al struct `Handler` (Cambio 4a)
5. Añadir `RootDir` en `section-build.go` al crear `AssetsHandler` (Cambio 1)
6. Añadir `LoadSSRModules()` en goroutine (Cambio 2)
7. Añadir wire del callback SSR para assetmin (Cambio 3)
8. Añadir `ImageHandler` + encadenamiento de callback + `UnobservedFiles` (Cambio 4c, 4d)
9. Correr tests de integración
10. Verificar hot reload manualmente con `clinical_encounter/ssr.go`

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

    am := assetmin.NewAssetMin(&assetmin.Config{OutputDir: t.TempDir()})
    am.SetListModulesFn(func(root string) ([]string, error) {
        return []string{moduleDir}, nil
    })

    // LoadSSRModules es síncrono en tests — WaitForSSRLoad es un no-op aquí.
    // Solo usar WaitForSSRLoad cuando LoadSSRModules corre en goroutine (producción).
    am.LoadSSRModules()

    // Verificar CSS inicial presente en bundle
    // css, _ := am.GetCSS(); assert contains ".v1"

    // Simular hot reload: reescribir ssr.go + llamar ReloadSSRModule directamente
    os.WriteFile(filepath.Join(moduleDir, "ssr.go"), []byte(`
//go:build !wasm
package mypkg
func RenderCSS() string { return ".v2 { color: blue; }" }
`), 0644)
    am.ReloadSSRModule(moduleDir)
    am.RefreshAsset(".css")

    // Verificar CSS actualizado
    // css, _ = am.GetCSS(); assert contains ".v2", not ".v1"
}
```

## Estado de dependencias

| Módulo | Estado | Versión |
|---|---|---|
| `tinywasm/devflow` | ✅ Completado | v0.4.16 — `OnSSRFileChange` implementado |
| `tinywasm/assetmin` | Pendiente | requiere implementar PLAN.md de assetmin |
| `tinywasm/app` | Pendiente | requiere assetmin publicado |
