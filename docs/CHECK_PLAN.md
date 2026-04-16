# PLAN: Eliminar encoding/json de daemon.go

## Problema

`daemon.go` importa `encoding/json` para parsear el envelope JSON-RPC antes de delegarlo al MCP server:

```go
import "encoding/json"

var rpcEnvelope struct {
    ID     string          `json:"id"`
    Method string          `json:"method"`
    Params json.RawMessage `json:"params"`
}
if err := json.Unmarshal(msg, &rpcEnvelope); err != nil {
    w.WriteHeader(http.StatusBadRequest)
    return
}
```

Dos problemas:
1. Viola la regla del ecosistema: solo `tinywasm/json` debe usarse.
2. `ID string` falla cuando Claude Code envía `"id":1` (integer válido por spec MCP) → **400 Bad Request**.

## Fix

`daemon.go` solo necesita el valor de `"method"` (y opcionalmente `"id"` y `"params"`) para el switch. `tinywasm/mcp` ya exporta `ExtractJSONValue` que extrae valores de bytes raw sin parsing completo ni imports externos:

```go
// Sin import de encoding/json
methodBytes := mcp.ExtractJSONValue(msg, "method")
method := string(methodBytes)

switch method {
case "tinywasm/state":
    _, token := getAuthCtx(r)
    ...
case "tinywasm/action":
    _, token := getAuthCtx(r)
    paramsBytes := mcp.ExtractJSONValue(msg, "params")
    // parsear key/value directamente de paramsBytes
    key := string(mcp.ExtractJSONValue(paramsBytes, "key"))
    value := string(mcp.ExtractJSONValue(paramsBytes, "value"))
    ...
default:
    ctx, _ := getAuthCtx(r)
    resp := mcpServer.HandleMessage(ctx, msg)
    ...
}
```

## Prerequisito

`tinywasm/mcp` debe tener `ExtractJSONValue` exportado y accesible (ya lo está en `request_handler.go`).

## Resultado

- Sin `encoding/json` en el ecosistema tinywasm/app
- Claude Code puede enviar `"id":1` sin recibir 400
- El switch de métodos sigue funcionando igual
