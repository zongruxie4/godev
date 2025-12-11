# Refactorización Implementada - Opción A

## Fecha: October 17, 2025
## Estado: ✅ IMPLEMENTADO - Esperando validación de tests

---

## Cambios Realizados

### 1. tinywasm/builderInit.go

**Método modificado:** `OutputRelativePath()`

**Antes:**
```go
// returns the full path to the final output file eg: web/build/main.wasm
func (w *TinyWasm) OutputRelativePath() string {
	return w.activeBuilder.FinalOutputPath()
}
```

**Después:**
```go
// OutputRelativePath returns the RELATIVE path to the final output file
// eg: "deploy/edgeworker/app.wasm" (relative to AppRootDir)
// This is used by file watchers to identify output files that should be ignored.
func (w *TinyWasm) OutputRelativePath() string {
	// FinalOutputPath() returns absolute path like: /tmp/test/deploy/edgeworker/app.wasm
	// We need to extract the relative portion: deploy/edgeworker/app.wasm
	fullPath := w.activeBuilder.FinalOutputPath()
	
	// Remove AppRootDir prefix to get relative path
	if strings.HasPrefix(fullPath, w.Config.AppRootDir) {
		relPath := strings.TrimPrefix(fullPath, w.Config.AppRootDir)
		// Remove leading separator (/ or \)
		relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
		relPath = strings.TrimPrefix(relPath, "/") // Handle Unix paths
		return relPath
	}
	
	// Fallback: construct from config values (which are already relative)
	return filepath.Join(w.Config.OutputDir, w.Config.OutputName+".wasm")
}
```

**Imports agregados:**
```go
import (
	"path"
	"path/filepath"  // NUEVO
	"strings"        // NUEVO
	"time"
	"github.com/tinywasm/gobuild"
)
```

---

### 2. tinywasm/output_path_test.go (NUEVO)

**Test unitario creado:** `TestOutputRelativePath`

Verifica que:
- ✅ El método retorna rutas RELATIVAS (no absolutas)
- ✅ Funciona con diferentes AppRootDir (Unix, Windows, temp dirs)
- ✅ No tiene separadores iniciales (/, \)
- ✅ Produce salida consistente

**Test adicional:** `TestOutputRelativePathConsistency`

Verifica que:
- ✅ Retorna la misma ruta relativa en todos los modos (coding, debug, production)
- ✅ No cambia al cambiar de compilador

---

### 3. golite/deploy_unobserved_files_test.go

**Actualizado para esperar rutas relativas:**

```go
// Verify UnobservedFiles contains the expected files (both should be RELATIVE paths)
expectedFiles := []string{
	"deploy/edgeworker/app.wasm",
	"deploy/edgeworker/_worker.js",
}

for _, expectedFile := range expectedFiles {
	found := false
	for _, actual := range unobservedFiles {
		// Normalize paths for comparison (handle / vs \)
		normalizedActual := filepath.ToSlash(actual)
		normalizedExpected := filepath.ToSlash(expectedFile)
		if normalizedActual == normalizedExpected {
			found = true
			break
		}
	}
	require.True(t, found, "UnobservedFiles should contain: %s", expectedFile)
}
```

---

## Impacto de los Cambios

### Archivos Modificados
1. ✅ `tinywasm/builderInit.go` - Lógica de OutputRelativePath
2. ✅ `tinywasm/output_path_test.go` - Nuevo test unitario
3. ✅ `golite/deploy_unobserved_files_test.go` - Test actualizado

### Paquetes Afectados
- ✅ `tinywasm` - Fix principal
- ✅ `goflare` - Beneficiario del fix (no requiere cambios)
- ✅ `golite` - Test actualizado

---

## Resultado Esperado

### Antes del Fix
```go
goflare.UnobservedFiles() = []string{
	"/tmp/test/deploy/edgeworker/app.wasm",    // ❌ ABSOLUTO
	"deploy/edgeworker/_worker.js",            // ✅ RELATIVO
}
```

### Después del Fix
```go
goflare.UnobservedFiles() = []string{
	"deploy/edgeworker/app.wasm",     // ✅ RELATIVO
	"deploy/edgeworker/_worker.js",   // ✅ RELATIVO
}
```

---

## Tests a Ejecutar

### 1. Test de tinywasm
```bash
cd /home/cesar/Dev/Pkg/Mine/tinywasm
go test -v -run TestOutputRelativePath
go test -v -run TestOutputRelativePathConsistency
```

**Expectativa:** ✅ Ambos tests deben PASAR

---

### 2. Tests existentes de tinywasm
```bash
cd /home/cesar/Dev/Pkg/Mine/tinywasm
go test ./...
```

**Expectativa:** ✅ Todos los tests deben PASAR (no debe haber regresión)

---

### 3. Test de golite (bug reproduction)
```bash
cd /home/cesar/Dev/Pkg/Mine/golite
go test -v -run TestDeployUnobservedFilesNotProcessedByAssetmin
```

**Expectativa:** ✅ Test debe PASAR (el bug está arreglado)

---

### 4. Suite completa de golite
```bash
cd /home/cesar/Dev/Pkg/Mine/golite
go test ./...
```

**Expectativa:** ✅ Todos los tests deben PASAR

---

### 5. Test manual en golite/example
```bash
cd /home/cesar/Dev/Pkg/Mine/golite/example
golite
```

**Verificar en logs:**
- ❌ NO debería aparecer: `ASSETS .js create ... deploy/edgeworker/_worker.js`
- ✅ Solo deberían procesarse archivos de `src/web/ui/`

**Verificar en src/web/public/main.js:**
```bash
cat src/web/public/main.js | grep -i "worker\|fetch\|export default"
```
**Expectativa:** ❌ NO debe contener contenido de _worker.js

---

## Notas de Implementación

### Por qué este approach funciona:

1. **Extrae la porción relativa:** Usa `strings.TrimPrefix()` para remover `AppRootDir`
2. **Limpia separadores:** Remueve `/` o `\` iniciales
3. **Fallback seguro:** Si la extracción falla, construye desde config
4. **Cross-platform:** Usa `filepath.Separator` para compatibilidad Windows/Unix

### Consideraciones:

- ✅ No rompe API existente (método sigue siendo público)
- ✅ Mejora semántica (el nombre del método ahora coincide con su comportamiento)
- ✅ Beneficia a todos los consumidores de tinywasm
- ✅ Sin breaking changes (solo corrige comportamiento incorrecto)

---

## Próximos Pasos (Esperando Tu Decisión)

1. **Ejecutar tests** - Tú harás esto manualmente
2. **Revisar resultados** - Verificar que todos los tests pasen
3. **Decidir:**
   - ✅ Si tests pasan → Commit y push
   - ❌ Si tests fallan → Ajustar según errores
   - 🔄 Si hay regresión → Revisar approach

---

## Estado Actual

**Implementación:** ✅ COMPLETA
**Tests:** ⏳ PENDIENTE (esperando tu ejecución)
**Documentación:** ✅ COMPLETA
**Commit:** ⏳ PENDIENTE (esperando validación)

---

**Esperando tu decisión después de ejecutar los tests.**
