# PLAN: tinywasm/app — Fix tests greet (tracker frágil de logs)

## Problema

`TestGreetFileEventTriggersWasmCompilation` y `TestGreetFileRepeatedEdits`
fallan reportando 0 compilaciones WASM.

La compilación SÍ ocurre (`[CLIENT]  [mem|1.8 MB]` aparece en los logs).
El tracker de los tests busca strings de log que no coinciden con el output
real de `WasmClient.LogSuccessState()`.

## Solución: usar `WasmClient.SetOnCompile` (disponible desde client v0.6.2)

---

## Cambio 1 — `test/greet_file_event_test.go`

**Eliminar** el tracker basado en logs y la variable `browserReloads` que
dependía de él. Reemplazar por `SetOnCompile` después de `startTestApp`.

Eliminar estas líneas (aprox. 104-126):
```go
// Track what happens
var wasmCompilations int32
var browserReloads int32

tracker := func(messages ...any) {
    msg := strings.Join(func() []string {
        s := make([]string, len(messages))
        for i, m := range messages {
            s[i] = fmt.Sprint(m)
        }
        return s
    }(), " ")

    lowerMsg := strings.ToLower(msg)
    if (strings.Contains(msg, "WASM") && strings.Contains(lowerMsg, "compil")) || strings.Contains(msg, "WASM In-Memory") {
        atomic.AddInt32(&wasmCompilations, 1)
    }
    if strings.Contains(lowerMsg, "reload") {
        atomic.AddInt32(&browserReloads, 1)
    }
}

ctx := startTestApp(t, tmp, tracker)
```

Reemplazar por:
```go
var wasmCompilations int32

ctx := startTestApp(t, tmp)
```

Después del bloque de espera `"Listening for File Changes"`, añadir:
```go
h := app.GetActiveHandler()
if h == nil || h.WasmClient == nil {
    t.Fatal("WasmClient not initialized")
}
h.WasmClient.SetOnCompile(func(err error) {
    atomic.AddInt32(&wasmCompilations, 1)
})
```

Eliminar también las referencias a `browserReloads` e `initialReloads` del
resto del test, y la variable `initialReloads` basada en `ctx.Browser`.

---

## Cambio 2 — `test/greet_repeated_edits_test.go`

Eliminar estas líneas (aprox. 96-113):
```go
// Track compilations
var compilationCount int32
tracker := func(messages ...any) {
    msg := strings.Join(func() []string {
        s := make([]string, len(messages))
        for i, m := range messages {
            s[i] = fmt.Sprint(m)
        }
        return s
    }(), " ")

    if strings.Contains(msg, "Compiling WASM") || strings.Contains(msg, "WASM In-Memory") {
        atomic.AddInt32(&compilationCount, 1)
    }
}

ctx := startTestApp(t, tmp, tracker)
```

Reemplazar por:
```go
var compilationCount int32

ctx := startTestApp(t, tmp)
```

Después de `app.WaitWatcherReady`, añadir:
```go
h := app.GetActiveHandler()
if h == nil || h.WasmClient == nil {
    t.Fatal("WasmClient not initialized")
}
h.WasmClient.SetOnCompile(func(err error) {
    atomic.AddInt32(&compilationCount, 1)
})
```

Eliminar también los imports `fmt` y `strings` si quedan sin uso tras los
cambios anteriores.

---

## Orden de ejecución

| # | Tarea | Archivo | Estado |
|---|-------|---------|--------|
| 1 | `OnCompile` + `SetOnCompile` en `client` | `client/` | ✅ Publicado v0.6.2 |
| 2 | `go.mod` apunta a client v0.6.2 | `app/go.mod` | ✅ Actualizado por codejob |
| 3 | Reemplazar tracker en `greet_file_event_test.go` | `app/test/` | Pendiente |
| 4 | Reemplazar tracker en `greet_repeated_edits_test.go` | `app/test/` | Pendiente |
| 5 | Verificar `TestGreetFileEventTriggersWasmCompilation` pasa | — | Pendiente |
| 6 | Verificar `TestGreetFileRepeatedEdits` pasa | — | Pendiente |
