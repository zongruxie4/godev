# PLAN: Adaptar tinywasm/app tras refactor de sprite inline en assetmin

## Contexto

`tinywasm/assetmin` está siendo refactorizado para que el sprite SVG **solo exista inline en el HTML** — se elimina la ruta HTTP `/assets/icons.svg` y el `spriteSvgHandler` deja de registrarse en `RegisterRoutes`.

Ver detalles del refactor: `tinywasm/assetmin/docs/PLAN.md`.

Este documento describe los cambios necesarios en `tinywasm/app` tras ese refactor, y el bug que hace que el sprite quede vacío en entornos de desarrollo de componentes individuales.

## Análisis del flujo actual en tinywasm/app

### Ciclo de vida del sprite en `InitBuildHandlers` (`section-build.go`)

```
NewAssetMin(Config{RootDir, ...})       // assetmin inicializado, sprite vacío
    ↓
RegisterRoutes(AssetsHandler.RegisterRoutes)  // registra /, style.css, script.js, icons.svg (← eliminar en refactor)
    ↓
LoadSSRModules()                        // goroutine async: escanea ssr.go de módulos,
                                        //   llama IconSvg() de cada módulo,
                                        //   registra iconos con InjectSpriteIcon()
    ↓
AddDynamicContent(fn)                   // ya configurado en NewAssetMin:
                                        //   el HTML sirve el sprite en memoria al momento del request
```

### Problema: el sprite puede estar vacío en el primer request

`LoadSSRModules()` corre en un goroutine. Si el browser recarga antes de que termine, el HTML se sirve con el sprite vacío (`<svg><defs></defs></svg>`). Esto es especialmente visible en entornos dev de componentes individuales (`tinywasm/components/*/web/`).

### Problema específico del entorno dev de componentes

En `tinywasm/components/selectsearch/web/`, el servidor dev arranca y el `ListModulesFn` (si se inyecta) apunta a los módulos del proyecto raíz. El `ssr.go` del propio componente que está siendo desarrollado (que contiene `IconSvg()`) **puede no estar incluido** en la lista de módulos devuelta por `ListModulesFn`, dependiendo de cómo esté configurado.

Además, `LoadSSRModules` es asíncrono — si el browser carga la página antes de que termine el escaneo, el sprite se sirve vacío.

## Cambios en tinywasm/app

### 1. `section-build.go` — eliminar `RefreshAsset(".svg")` del hot reload

En la callback `SetOnSSRFileChange`, el sprite ya no necesita un refresh explícito como asset HTTP separado, porque vive inline en el HTML. El `.html` refresh es suficiente:

```go
// ANTES
h.AssetsHandler.RefreshAsset(".css")
h.AssetsHandler.RefreshAsset(".js")
h.AssetsHandler.RefreshAsset(".html")
// (no había .svg explícito, pero si se añadía era redundante)

// DESPUÉS — sin cambio funcional; solo verificar que no se llame RefreshAsset(".svg")
// ya que esa extensión ya no corresponde a un asset HTTP registrado
```

> **Nota:** En el código actual NO hay `RefreshAsset(".svg")` en el hot reload. Este punto es preventivo para cuando el refactor de assetmin elimine esa rama del switch en `RefreshAsset`.

### 2. `section-build.go` — verificar que `RefreshAsset(".svg")` no crashee tras el refactor

En `assetmin.go`, el switch de `RefreshAsset` tiene una rama `case ".svg"`. Tras el refactor, esa rama ya no tendrá un handler HTTP, pero el sprite sigue siendo un `*asset` interno. Verificar que no se llame externamente o que la rama sea eliminada / marcada como no-op.

### 3. Fix del bug: sprite vacío en primer request del entorno dev de componentes

**Causa raíz:** `LoadSSRModules()` es asíncrono. El browser puede recibir el HTML antes de que `IconSvg()` sea registrado.

**Solución A — `WaitForSSRLoad` antes de servir el primer request HTML (recomendada para dev):**

El `AssetMin` ya expone `WaitForSSRLoad(timeout)`. En el entorno dev de componentes, llamar a `WaitForSSRLoad` antes de abrir el browser garantiza que el sprite esté poblado:

```go
// En InitBuildHandlers, después de LoadSSRModules():
h.AssetsHandler.LoadSSRModules()
if h.DevMode {
    h.AssetsHandler.WaitForSSRLoad(5 * time.Second)
}
```

> Esto bloquea brevemente el arranque pero garantiza que el HTML inicial tenga el sprite completo. En producción no aplica porque SSR se compila por separado.

**Solución B — Inyección directa del módulo local antes de LoadSSRModules:**

Para el entorno dev de un componente individual, el `ssr.go` del componente que se está desarrollando es siempre conocido. Se puede inyectar directamente sin esperar al escaneo:

```go
// Antes de LoadSSRModules, inyectar el módulo raíz directamente:
if err := h.AssetsHandler.ReloadSSRModule(h.RootDir); err != nil {
    h.AssetsHandler.Logger("Initial SSR load error:", err)
}
h.AssetsHandler.LoadSSRModules() // continúa con el resto de módulos
```

**Solución recomendada: A + B combinadas** — inyectar el módulo raíz sincrónicamente, luego cargar el resto en background.

### 4. `ssr_integration_test.go` — no hay cambios necesarios

Los tests existentes usan `UpdateSSRModule` directamente (inyección manual) y no dependen de la ruta HTTP del sprite. Todos siguen pasando tras el refactor.

No obstante, agregar un test que verifique que `IconSvg()` de un módulo SSR real registra iconos en el sprite sería útil:

```go
// TestSSRIconInjection — verifica que IconSvg() de un módulo se registra en el sprite inline
func TestSSRIconInjection(t *testing.T) {
    // ...
    h.AssetsHandler.UpdateSSRModule("mod", "", "", "", map[string]string{
        "test-icon": `<path fill="currentColor" d="M1 2h3"/>`,
    })
    if !h.AssetsHandler.HasIcon("test-icon") {
        t.Error("icon should be registered in sprite")
    }
    // Verificar que el HTML contiene el símbolo
    html, _ := h.AssetsHandler.GetMinifiedHTML()
    if !strings.Contains(string(html), `id="test-icon"`) {
        t.Error("HTML should contain sprite symbol inline")
    }
}
```

## Orden de implementación

1. Aplicar refactor en `tinywasm/assetmin` (eliminar ruta HTTP del sprite)
2. Verificar que `tinywasm/app` compila sin cambios — `RegisterRoutes` de assetmin simplemente registrará una ruta menos, sin cambio de firma
3. Aplicar **Solución A**: añadir `WaitForSSRLoad` condicional en `InitBuildHandlers`
4. Aplicar **Solución B**: inyectar `ReloadSSRModule(h.RootDir)` sincrónicamente antes de `LoadSSRModules()`
5. Añadir `TestSSRIconInjection` en `ssr_integration_test.go`

## Archivos a modificar

| Archivo | Cambio |
|---------|--------|
| `section-build.go` | Añadir `ReloadSSRModule(h.RootDir)` + `WaitForSSRLoad` tras `LoadSSRModules()` |
| `ssr_integration_test.go` | Añadir `TestSSRIconInjection` |
