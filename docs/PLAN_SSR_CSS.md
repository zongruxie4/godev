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
    h.AssetsHandler.RefreshAsset(".css")
    h.AssetsHandler.RefreshAsset(".js")
    if err := h.Browser.Reload(); err != nil {
        h.AssetsHandler.Logger("Browser reload error:", err)
    }
}
```

**Requiere:** campo `OnSSRFileChange func(moduleDir string)` en `GoModHandler`
(`tinywasm/devflow/go_mod.go`) y que `NewFileEvent` lo dispare cuando `fileName == "ssr.go"`.

## Sin cambios en comportamiento actual

- El watcher sigue funcionando igual para `.go`, `.js`, `.css`, `.html`, `.svg`
- `WasmClient` y `Server` siguen recibiendo sus eventos `.go` sin interferencia
- `assetmin` procesa primero el check `fileName == "ssr.go"` y retorna early — los
  demás `.go` nunca llegan a la lógica de assets normal
- Modo disco (`buildOnDisk=true`) respetado: `ReloadSSRModule` usa `processAsset`
  existente que ya maneja el switch memoria/disco

## Tests requeridos en `tinywasm/app`

### `ssr_integration_test.go` (test de integración)

| Test | Qué verifica |
|---|---|
| `TestSSRLoadOnInit` | `InitBuildHandlers` → `LoadSSRModules` llamado → CSS de módulos en memoria |
| `TestSSRHotReload` | cambio en `ssr.go` de replace local → nuevo CSS sin reiniciar |
| `TestSSRNoBlockOnStartup` | servidor responde antes de que `LoadSSRModules` termine |
| `TestSSRProxyModulesLoaded` | módulos de proxy (`$GOPATH/pkg/mod`) tienen CSS en bundle |

Los tests de integración pueden usar el módulo `example/` existente en `tinywasm/app`
o crear fixtures mínimos con go.mod + ssr.go de prueba.

## Orden de implementación

1. Implementar `tinywasm/assetmin` PLAN completo
2. Actualizar `go.mod` de `tinywasm/app` para la nueva versión de assetmin
3. Añadir `RootDir` en `section-build.go` al crear `AssetsHandler`
4. Añadir llamada `LoadSSRModules()` en goroutine en `InitBuildHandlers`
5. Correr tests de integración
6. Verificar hot reload manualmente con `clinical_encounter/ssr.go`

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

## Módulo adicional afectado: `tinywasm/devflow`

Requiere añadir en `GoModHandler`:
```go
OnSSRFileChange func(moduleDir string)
```
Y en `NewFileEvent` de `go_mod.go`:
```go
if g.OnSSRFileChange != nil && fileName == "ssr.go" {
    g.OnSSRFileChange(filepath.Dir(filePath))
}
```
Este cambio es menor y no rompe el comportamiento actual.
