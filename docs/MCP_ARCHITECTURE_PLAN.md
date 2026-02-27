# Architecture Plan: Persistent MCP Server Startup and `tinywasm` CLI

## 1. Current Problem
Currently, the `tinywasm` command starts the project's development server and the **MCP Server on port 3030** in a coupled manner.
This causes the IDE (LLM) to lose access to the MCP unless the developer has first started the server in the terminal. Additionally, if the developer switches projects, they must manually stop and restart.

## 2. Proposed Solution
It is proposed to separate the **MCP Server** (which will be a single, global process) from each project's **Web Server/Compiler**, using **Server-Sent Events (SSE)** instead of WebSockets for UI communication, thus maintaining minimal dependencies.
All of this will live in a **single binary** `tinywasm` (compiled in `cmd/tinywasm/main.go`).

### 2.1. The Global MCP Server (`tinywasm -mcp`)
- The `tinywasm -mcp` command starts the process on port `3030`.
- If the IDE (or LLM) starts and detects nothing on 3030, it should execute `tinywasm -mcp`.
- This global MCP server will wait for requests. Its main objective is to expose tools (read logs, issue commands, `start_development`).

### 2.2. Project Management (Single-Project Mode)
- There can only be **one active project at a time** in development (managed by the MCP).
- If the LLM calls the `start_development(ide_name, project_path)` tool:
  1. The global MCP will stop the `app` process (compiler/watcher) that was previously running.
  2. It will start the new environment in `project_path` in *headless* mode (without rendering the visual TUI to STDOUT).

### 2.3. Terminal Interface (TUI) Client
- When the **developer** opens their terminal and types `tinywasm`, the binary skips starting a new backend. It first pings port `3030`.
- **If the MCP exists**: It connects to `http://localhost:3030/logs` via SSE (`tinywasm/sse`). The local process renders the iterative graphical interface of `devtui` reading those logs, acting merely as a *viewer* or client.
- **Headless Flow**: The LLM will use `start_development`, which starts the compiler in the background silently. If the human wants to see the TUI, they just type `tinywasm` and it magically attaches.

## 3. "Ctrl+C" Behavior and Exit
- **Detach (`Ctrl+C`)**: If the developer presses `Ctrl+C` in their terminal acting as a TUI client (i.e., the `tinywasm` command with no flags), the terminal will close and return to the OS shell. However, **the project will continue compiling and running in the background** via the MCP. The LLM can continue working without interruption.
- **Full Shutdown (`q`)**: If the developer wants to shut down the web server and watcher completely (releasing the port, e.g., 8080), they must press `q` in the TUI. This will send an HTTP POST request to `http://localhost:3030/action?key=q` to the MCP to stop the active project.

## 4. Flow Diagrams
- [Current Flow (TUI-dependent)](diagrams/MCP_CURRENT_FLOW.md)
- [Proposed Flow (Persistent Daemon)](diagrams/MCP_PROPOSED_FLOW.md)

## 5. Technical Refactoring Plans
The refactoring will be divided by libraries (each with its own detailed `MCP_REFACTOR.md` guiding the sub-agent). You can use the following list to track which parts have already been implemented:
- [ ] **`app`**: Refactor for *daemon* and *client* modes. ([app/docs/MCP_REFACTOR.md](../app/docs/MCP_REFACTOR.md))
- [ ] **`devtui`**: Adaptation to read from SSE and post HTTP actions. ([devtui/docs/MCP_REFACTOR.md](../devtui/docs/MCP_REFACTOR.md))
- [ ] **`mcpserve`**: SSE endpoints to emit logs from the project in progress. ([mcpserve/docs/MCP_REFACTOR.md](../mcpserve/docs/MCP_REFACTOR.md))
