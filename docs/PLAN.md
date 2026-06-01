# PLAN: tinywasm/app — Migrar imagemin → image/min e inyectar en assetmin

## Repositorio
`github.com/tinywasm/app` — path local: `tinywasm/app/`

## Dependencias de ejecución
```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

## Prerequisito
- `github.com/tinywasm/image` publicado con el subpaquete `min/` (ver `tinywasm/image/docs/PLAN.md`).
- `github.com/tinywasm/assetmin` publicado con `SSRFileWatcher`, `ImageProcessor` y
  `SetImageProcessor` (ver `tinywasm/assetmin/docs/PLAN.md`).

---

## Contexto y decisión

`tinywasm/app` es el composition root. Hoy usa `tinywasm/imagemin` (archivado) y cablea el
hotreload SSR vía un callback en `GoModHandler` (`ssrRelay`).

Cambios:
1. **imagemin → image/min** (renombre de import/paquete).
2. **image deja de ser un mecanismo aparte** — se **inyecta** en assetmin como `ImageProcessor`.
3. **El hotreload SSR pasa a `assetmin.SSRFileWatcher`** — se elimina el bloque `ssrRelay`.

**Archivos afectados:** `app/handler.go`, `app/section-build.go`, `app/go.mod`

---

## Cambio 1: `app/go.mod`

```
# eliminar:
github.com/tinywasm/imagemin v0.0.5

# agregar:
github.com/tinywasm/image v<nueva-version>
```

```bash
go get github.com/tinywasm/image@latest
go mod tidy
```

---

## Cambio 2: `app/handler.go`

```go
// import:
"github.com/tinywasm/imagemin"   →   "github.com/tinywasm/image/min"

// campo del struct Handler:
ImageHandler  *imagemin.Handler   →   ImageHandler  *min.Handler
```

`ImageHandler` se mantiene como campo (app lo construye), pero ahora se **inyecta** en
assetmin en vez de cablearse por separado.

---

## Cambio 3: `app/section-build.go`

### 3a: Import
```go
"github.com/tinywasm/imagemin"   →   "github.com/tinywasm/image/min"
```

### 3b: Construcción del handler de imagen
```go
// ANTES:
h.ImageHandler = imagemin.New(&imagemin.Config{
    RootDir:   h.RootDir,
    OutputDir: filepath.Join(h.RootDir, h.Config.WebPublicDir(), "img"),
    Quality:   82,
})
h.ImageHandler.SetLog(h.Watcher.Logger)

// DESPUÉS:
h.ImageHandler = min.New(&min.Config{
    RootDir:   h.RootDir,
    OutputDir: filepath.Join(h.RootDir, h.Config.WebPublicDir(), "img"),
    Quality:   82,
})
h.ImageHandler.SetLog(h.Watcher.Logger)
// Loader: prod usa InitDefaultLoader (go list); tests inyectan ListModulesFn.
// OBLIGATORIO antes de inyectar — assetmin llamará LoadImages() y sin loader retorna
// "listModulesFn not set". (Bug latente actual: en prod no se llamaba ninguno.)
if h.ListModulesFn != nil {
    h.ImageHandler.SetListModulesFn(h.ListModulesFn)
} else {
    h.ImageHandler.InitDefaultLoader()
}

// Inyectar en assetmin — assetmin reconoce image.go y delega a este processor.
h.AssetsHandler.SetImageProcessor(h.ImageHandler)
```

### 3c: UnobservedFiles — eliminar el merge manual de imagen
assetmin ahora incluye los outputs del `ImageProcessor` en su propio `UnobservedFiles()`
(ver assetmin PLAN Cambio 2d). Eliminar de app:
```go
// ELIMINAR:
if h.ImageHandler != nil {
    uf = append(uf, h.ImageHandler.UnobservedFiles()...)
}
```
`uf = append(uf, h.AssetsHandler.UnobservedFiles()...)` ya los cubre.

### 3d: Carga inicial de imágenes — eliminar de app
La hace assetmin durante su escaneo SSR (`imageProcessor.LoadImages()`, ver assetmin PLAN
Cambio 2c). Eliminar la goroutine y el wiring de loader duplicado (ya movido a 3b):
```go
// ELIMINAR (el loader ahora se cablea en 3b, antes de la inyección):
if h.ListModulesFn != nil {
    h.ImageHandler.SetListModulesFn(h.ListModulesFn)
}

// ELIMINAR (la carga inicial la dispara assetmin):
go func() {
    if err := h.ImageHandler.LoadImages(); err != nil {
        h.ImageHandler.Logger("Image load error:", err)
    }
}()
```
> Asegurar el orden: `SetImageProcessor(...)` **antes** de `ReloadSSRModule`/`LoadSSRModules`,
> para que el escaneo inicial incluya las imágenes.

### 3e: Reemplazar el bloque `ssrRelay` por `SSRFileWatcher`
```go
// ELIMINAR todo el bloque:
type ssrRelay interface {
    SetOnSSRFileChange(fn func(string))
}
if g, ok := h.GoModHandler.(ssrRelay); ok {
    g.SetOnSSRFileChange(func(moduleDir string) {
        if err := h.AssetsHandler.ReloadSSRModule(moduleDir); err != nil { ... }
        if err := h.ImageHandler.ReloadModule(moduleDir); err != nil { ... }
        if err := h.Browser.Reload(); err != nil { ... }
    })
}

// REEMPLAZAR con — assetmin enruta css/svg/html.go (texto) e image.go (pipeline) internamente:
ssrWatcher := h.AssetsHandler.NewSSRFileWatcher(func() error {
    return h.Browser.Reload()
})
h.Watcher.AddFilesEventHandlers(ssrWatcher)
```

### 3f: TUI log handler (opcional, recomendado mantener)
`h.Tui.AddHandler(h.ImageHandler, colorTealMedium, h.SectionBuild)` puede mantenerse para
que los logs del pipeline tengan su propio canal en la TUI. `image/min.Handler` satisface
la interfaz (`Name()` → `"IMAGE"`, `Logger()`).

### 3g: Construir e inyectar el extractor SSR (`tinywasm/ssr`)
La extracción SSR (codegen + `go run`) se delegó a `tinywasm/ssr` (ver assetmin PLAN Cambio 7).
app la construye y la inyecta, igual que el `ImageProcessor`:
```go
import "github.com/tinywasm/ssr"

ssrExtractor := ssr.New(h.RootDir)
ssrExtractor.SetLog(h.Watcher.Logger)
if h.ListModulesFn != nil {
    ssrExtractor.SetListModulesFn(h.ListModulesFn)
}
h.AssetsHandler.SetSSRExtractor(ssrExtractor)  // ANTES de ReloadSSRModule/LoadSSRModules
```
`go.mod`: agregar `github.com/tinywasm/ssr`.

**Eliminar** la llamada obsoleta `h.AssetsHandler.SetListModulesFn(h.ListModulesFn)`
(section-build.go ~línea 64): la discovery de módulos ahora la hace el extractor SSR, no assetmin
(`assetmin.SetListModulesFn` se elimina, ver assetmin Cambio 7b). El `listModulesFn` se pasa al
`ssrExtractor` (arriba).

> Orden: `SetSSRExtractor` y `SetImageProcessor` **antes** de `ReloadSSRModule`/`LoadSSRModules`
> (que ahora delegan en lo inyectado). Si el extractor es nil, assetmin no extrae.

---

## Cambio 4: Test de integración para `image.go` (gate `.go` end-to-end)

Los unit tests del `SSRFileWatcher` (en assetmin) llaman `NewFileEvent` directo y **saltan**
`depFinder.ThisFileIsMine`, por lo que NO detectan errores en `MainInputFileRelativePath()`.
Hace falta un test de integración que ejercite el camino completo de devwatch.

Reutilizar el patrón de `app/css_hotreload_integration_test.go` / `app/ssr_subpackage_hotreload_test.go`,
agregando un caso para `image.go`:
- Crear un módulo temporal con `image.go` (`RenderImages() []image.Asset`) + una imagen en testdata.
- Levantar el watcher real, escribir/modificar `image.go`.
- Verificar que el `ImageProcessor` inyectado recibió `ReloadModule(dir)` y que se disparó el
  browser reload. (Usar un mock `ImageProcessor` para observar la llamada.)

Esto blinda el fix de `MainInputFileRelativePath() == "go.mod"`: si alguien lo rompe, el test
de integración falla (el unit test no lo haría).

---

## Verificación

```bash
cd tinywasm/app
go build ./...
gotest
```

Verificación manual (ver también assetmin PLAN):
1. Modificar `css.go` → recarga con nuevo CSS.
2. Modificar `image.go` (agregar imagen) → se genera el WebP y recarga.
3. Modificar `image.go` sin tocar el archivo de imagen → no reprocesa (mtime: el `.webp` ya está al día).

Ver `tinywasm/docs/MASTER_PLAN.md` para el orden global de ejecución.
