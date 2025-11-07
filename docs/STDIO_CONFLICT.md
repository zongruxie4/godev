# Conflicto stdio: DevTUI vs MCP Server

## Problema Identificado

Al intentar ejecutar el servidor MCP junto con la interfaz DevTUI, la aplicación se congelaba, presentaba lags y requería múltiples `Ctrl+C` para salir.

## Causa Raíz

**Ambos sistemas compiten por los mismos recursos de entrada/salida estándar (stdio)**:

### DevTUI (vía Bubble Tea)
```go
tea.NewProgram(tui, tea.WithAltScreen())
```
- **Lee de `stdin`**: Captura eventos de teclado del usuario
- **Escribe a `stdout`**: Renderiza la interfaz TUI
- **Usa terminal alternativa**: Modo pantalla completa

### MCP Server  
```go
server.ServeStdio(s)  // Bloquea esperando en stdio
```
- **Lee de `stdin`**: Espera mensajes JSON-RPC del cliente MCP
- **Escribe a `stdout`**: Envía respuestas JSON-RPC
- **Protocolo stdio**: Comunicación sincrónica bidireccional

### El Conflicto
Cuando ambos intentan usar stdio simultáneamente:
1. **Race condition**: Ambos leen stdin, causando pérdida de datos
2. **Output mezclado**: TUI render + JSON responses en mismo stdout
3. **Bloqueos**: ServeStdio() bloquea, impide que TUI procese eventos
4. **Terminal corrupto**: Escape sequences del TUI rompen JSON del MCP

## Investigación DevTUI

```bash
# Búsqueda de uso de stdio en DevTUI
grep -r "stdin|stdout|stderr" devtui/*.go
# Resultado: No uso directo

# Verificación de dependencia
cat devtui/go.mod
# Resultado: Usa charmbracelet/bubbletea v1.3.7
```

**Bubble Tea** (la librería base de DevTUI) por diseño usa:
- `os.Stdin` para input del usuario
- `os.Stdout` para renderizar
- No expone opciones de redirección de IO sin cambios mayores

## Solución Implementada

### MCP sobre HTTP en lugar de stdio

La solución definitiva es usar **HTTP transport para MCP** en lugar de stdio:

```bash
golite                    # UI + MCP HTTP server (puerto 3100)
golite --mcp-port 8080    # UI + MCP HTTP server (puerto 8080)
```

#### Arquitectura Final

```
┌─────────────────────────────────────────┐
│           GoLite Process                │
├─────────────────────────────────────────┤
│                                         │
│  ┌──────────────┐  ┌─────────────────┐ │
│  │   DevTUI     │  │  MCP HTTP       │ │
│  │   (stdio)    │  │  Server         │ │
│  │              │  │  Port 3100      │ │
│  │  stdin/out   │  │  /mcp endpoint  │ │
│  └──────────────┘  └─────────────────┘ │
│         ▲                   ▲          │
└─────────┼───────────────────┼──────────┘
          │                   │
     User Input         LLM Clients
      (Terminal)      (HTTP Requests)
```

#### Beneficios

1. ✅ **No hay conflicto**: stdio para UI, HTTP para MCP
2. ✅ **Concurrencia real**: Ambos funcionan simultáneamente
3. ✅ **Simple**: Un solo modo de operación
4. ✅ **Estándar**: HTTP es ampliamente soportado
5. ✅ **Flexible**: Puede conectarse desde cualquier cliente HTTP

## Alternativas Evaluadas

### ✅ 1. MCP sobre HTTP (IMPLEMENTADO)
**Solución elegida**: Usar StreamableHTTP transport de mcp-go
**Pros**: 
- Elimina conflicto de stdio completamente
- Compatible con especificación MCP
- Permite operación concurrente de UI + MCP
- No requiere cambios en clientes MCP HTTP
**Contras**: 
- Requiere configurar puerto adicional
- Cliente debe usar HTTP en lugar de stdio

### ❌ 2. Ejecutar ambos sobre stdio simultáneamente
**Imposible** - Conflicto fundamental de recursos compartidos

### ❌ 3. Separación por flags (modo stdio exclusivo)
**Descartado**: Requiere elegir entre UI o MCP, no permite ambos
**Problemas**:
- Usuario debe decidir entre desarrollo interactivo o integración LLM
- No permite usar asistente IA mientras se desarrolla

### ❌ 4. Redirigir IO de Bubble Tea
**Contras**:
- Bubble Tea no expone fácil redirección de IO
- Requiere fork de la librería
- Complejidad innecesaria

## Beneficios de la Solución

1. ✅ **Claridad**: Modos de operación explícitos
2. ✅ **Simplicidad**: Flag único, sin configuración compleja
3. ✅ **Compatibilidad**: Mantiene ambos casos de uso
4. ✅ **Rendimiento**: Sin overhead de sincronización
5. ✅ **Mantenibilidad**: No requiere cambios en librerías

## Uso en Producción

### Para desarrollo normal con asistente IA
```bash
cd my-project
golite
# UI interactiva + MCP HTTP en http://localhost:3100/mcp
```

### Para cambiar puerto MCP
```bash
golite --mcp-port 8080
# MCP disponible en http://localhost:8080/mcp
```

### Configuración en Claude Desktop
```json
{
  "mcpServers": {
    "golite": {
      "transport": "http",
      "url": "http://localhost:3100/mcp"
    }
  }
}
```

### Usando con otro cliente MCP HTTP
```bash
curl -X POST http://localhost:3100/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "clientInfo": {"name": "test", "version": "1.0"}
    }
  }'
```

## Lecciones Aprendidas

1. **stdio es un recurso exclusivo** - No puede ser compartido por múltiples protocolos
2. **Bubble Tea asume control total** - No diseñado para compartir terminal
3. **MCP soporta múltiples transports** - HTTP, SSE, stdio, in-process
4. **HTTP es la mejor opción** - Permite concurrencia sin conflictos
5. **Simplicidad > Complejidad** - Un solo modo de operación es más mantenible

## Referencias

- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [MCP Specification - stdio transport](https://spec.modelcontextprotocol.io)
- [DevTUI Implementation](https://github.com/cdvelop/devtui)
