# PLAN: Simplify Server Run Args — Only Pass `-server_port`

## Problem

`app/section-build.go` builds the run args for the external server and appends:
- `-public-dir=...`
- `-port=...`
- `-dev` (when DevMode)
- everything from `h.WasmClient.ArgumentsForServer()` (e.g. `-wasmsize_mode`)

The server template uses Go's `flag` package, which crashes on unknown flags.
Any arg `app` or `client` adds silently breaks all existing generated servers.

## Solution — Pass only `-server_port`, drop the rest

After the server template is simplified (see `server/docs/PLAN.md`):

- `-server_port` replaces `-port` (new agreed name)
- `-public-dir` is dropped — server always serves from its working dir (`web/public`)
- `-dev` is dropped — server doesn't need to know about dev mode
- `h.WasmClient.ArgumentsForServer()` is **not passed to the server** — wasm size mode
  is a compiler concern, not a server concern

### New `SetRunArgs` in `section-build.go`

```go
srv.SetRunArgs(func() []string {
    return []string{"-server_port=" + h.Config.ServerPort()}
})
```

That's it. One arg. The server ignores everything it doesn't recognize.

## Files to Change

| File | Change |
|------|--------|
| `app/section-build.go` | Simplify `SetRunArgs` to only `-server_port` |

## Dependency

Requires `server` to be published first with the new `lookupArg`-based template
(see `server/docs/PLAN.md`).

## Stage Checklist

- [ ] `server` published with simplified template
- [ ] Update `SetRunArgs` in `section-build.go`
- [ ] Run `gotest` in `tinywasm/app`
- [ ] Publish with `gopush`

---

# PLAN — JS API Migration (orchestration side)

> Auto-contenido. Independiente del PLAN superior de `-server_port`.

## Problema

`tinywasm/app` orquesta `client → assetmin` para producir el bundle JS de la
página. La cadena actual:

- [section-build.go:50](../section-build.go#L50) inyecta el callback
  `GetSSRClientInitJS: func() { return h.WasmClient.GetSSRClientInitJS() }`
  a assetmin.
- `WasmClient.GetSSRClientInitJS()` compone header + `wasm_exec.js` + footer
  con `WebAssembly.instantiateStreaming(fetch("/client.wasm"))`.
- assetmin bundlea el resultado en `/script.js`.

Cambios externos a este PLAN que ya están publicados como precondición
(verificar antes de empezar):

- `tinywasm/js v0.2.0` expone:
  ```go
  package js
  type Runtime int
  const (RuntimeGo Runtime = iota; RuntimeTinyGo)
  func SetRuntime(r Runtime)                   // write-once-at-boot
  func PageBootstrap() *Script                  // Name="" → /script.js
  func ServiceWorker(h ServiceWorkerHandler) *Script
  func WebWorker(name string, h WebWorkerHandler) *Script
  type Script struct { Name, Content string }
  ```
- `tinywasm/client` ya migró a build-only: eliminados `Javascript` struct,
  `GetSSRClientInitJS`, embeds de `wasm_exec.js`. `client` ya **no** compone JS.

Responsabilidad de `app` en esta migración:
1. **Escribir** el estado global `js.SetRuntime(...)` al boot y en cada
   cambio de modo de compilación.
2. **Reemplazar** el callback opaco `GetSSRClientInitJS` por registro directo
   de un `js.PageBootstrap()` Script con assetmin.

## Cambios en `tinywasm/app`

### 1. Inyectar runtime al boot y en cambio de modo

`section-build.go` debe llamar `js.SetRuntime(...)` **antes** de que assetmin
componga cualquier Script (porque `js.PageBootstrap()` lee el runtime activo).

```go
import "github.com/tinywasm/js"

// helper local
func syncJSRuntime(c *client.WasmClient) {
    if c.TinyGoCompilerFlag {
        js.SetRuntime(js.RuntimeTinyGo)
    } else {
        js.SetRuntime(js.RuntimeGo)
    }
}

// llamadas:
syncJSRuntime(h.WasmClient)            // tras la detección inicial
h.WasmClient.OnWasmExecChange = func() {
    syncJSRuntime(h.WasmClient)         // cualquier cambio de runtime
    // ... resto del callback existente
}
```

### 2. Reemplazar callback `GetSSRClientInitJS` por `js.PageBootstrap()`

[section-build.go:50](../section-build.go#L50) hoy hace:
```go
h.AssetsHandler = assetmin.NewAssetMin(&assetmin.Config{
    OutputDir: publicDir,
    RootDir:   h.RootDir,
    GetSSRClientInitJS: func() (string, error) {
        return h.WasmClient.GetSSRClientInitJS()
    },
    // ...
})
```

`client.WasmClient.GetSSRClientInitJS` ya **no existe** tras la migración de
client. Reemplazo:

- Eliminar el campo `GetSSRClientInitJS` de `assetmin.Config` (coordinar con
  assetmin/PLAN; o dejarlo deprecado retornando string vacío).
- En `section-build.go`, después de `syncJSRuntime(h.WasmClient)` y antes de
  cualquier `FlushToDisk`, registrar el Script con assetmin:
  ```go
  // El runtime ya está sincronizado → js.PageBootstrap() compone con el
  // wasm_exec correcto y el bootstrap apuntando a /client.wasm.
  h.AssetsHandler.RegisterScript(js.PageBootstrap())
  ```
- El método exacto de registro (`RegisterScript`, `UpdateSSRModule`, etc.)
  depende de la API que exponga assetmin tras su propia migración. Verificar
  en `assetmin` antes de implementar; alternativa: tratar el bootstrap como
  un módulo SSR sintético con `RenderJS() []*js.Script` que devuelve
  `[]*js.Script{js.PageBootstrap()}`.

### 3. Sin reexports, full breaking change

Búsqueda de consumidores rotos al inicio del PR:
```bash
grep -rn "GetSSRClientInitJS\|WasmExecGoSignatures\|WasmExecTinyGoSignatures\|\.Javascript\b" app/
```

Hits conocidos al redactar este PLAN:
- `app/section-build.go:50-52` — campo callback (eliminar/reemplazar según §2).

Cualquier hit adicional se migra a la API equivalente en `tinywasm/js`.

## Tests

| Archivo | Test | Verifica |
|---|---|---|
| `app/section-build_js_runtime_test.go` (nuevo) | `TestSyncJSRuntime_GoMode` | Tras boot en modo Go, `js.PageBootstrap().Content` contiene firmas runtime Go (declarar inline: `"runtime.scheduleTimeoutEvent"`) |
| idem | `TestSyncJSRuntime_TinyGoMode` | Tras boot en modo M/S, `js.PageBootstrap().Content` contiene firmas TinyGo (`"tinygo_js"`) |
| idem | `TestSyncJSRuntime_ChangeOnHotReload` | Simular cambio de modo → siguiente llamada a `js.PageBootstrap()` refleja el nuevo runtime |
| `app/test/bundle_regression_test.go` (nuevo o anexo) | `TestPageBundle_ContainsBootstrap` | El bundle `/script.js` producido por assetmin contiene `WebAssembly.instantiateStreaming` y `fetch("/client.wasm")` — sin regresión |

Ejecución: `gotest ./...` en `tinywasm/app`.

## Dependencia

Requiere `tinywasm/js v0.2.0` **y** `tinywasm/client` (build-only) **y**
`tinywasm/assetmin` actualizado con el nuevo método de registro de Script.
Las tres precondiciones deben estar publicadas antes de iniciar este PLAN.

## Stage Checklist — JS API Migration

- [ ] `tinywasm/js v0.2.0` publicado con `PageBootstrap`, `SetRuntime`, etc.
- [ ] `tinywasm/client` migrado y publicado (build-only, sin `Javascript`/`GetSSRClientInitJS`)
- [ ] `tinywasm/assetmin` migrado y publicado (sin campo `GetSSRClientInitJS` en Config, con método de registro de Script disponible)
- [ ] Añadir `require github.com/tinywasm/js` en `app/go.mod` (si no entra transitivamente)
- [ ] Implementar `syncJSRuntime` y llamarlo en boot + `OnWasmExecChange`
- [ ] Reemplazar callback `GetSSRClientInitJS` por registro directo de `js.PageBootstrap()` Script
- [ ] Migrar consumidores rotos en `app/` (búsqueda §3)
- [ ] Crear los 4 tests listados en §Tests
- [ ] `gotest ./...` verde en `tinywasm/app`
- [ ] Validación E2E con PWA real (out of scope para este PLAN — fase posterior)
- [ ] Publish con `gopush`
