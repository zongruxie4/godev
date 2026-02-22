# App Refactoring Plan (MCP Headless Daemon vs TUI Client)

## Development Rules
<!-- START_SECTION:CORE_PRINCIPLES -->
- **Single Responsibility Principle (SRP):** Every file (CSS, Go, JS) must have a single, well-defined purpose. This must be reflected in both the file's content and its naming convention.

- **Mandatory Dependency Injection (DI):**
    - **No Global State:** Avoid direct system calls (OS, Network) in logic.
    - **Interfaces:** Define interfaces for external dependencies (`Downloader`, `ProcessManager`).
    - **Composition:** Main structs must hold these interfaces.
    - **Injection:** `cmd/<app_name>/main.go` is the ONLY place where "Real" implementations are injected.

- **Framework-less Development:** For HTML/Web projects, use only the **Standard Library**. No external frameworks or libraries are allowed.

- **Strict File Structure:**
    - **Flat Hierarchy:** Go libraries must avoid subdirectories. Keep files in the root.
    - **Max 500 lines:** Files exceeding 500 lines MUST be subdivided and renamed by domain.
    - **Test Organization:** If >5 test files exist in root, move **ALL** tests to `tests/`.
<!-- END_SECTION:CORE_PRINCIPLES -->

## 1. Objective
Refactor the `tinywasm/app` package to support the MCP Client-Server architecture using `tinywasm/sse` while preserving a single binary `cmd/tinywasm/main.go`.

**IMPORTANT RECOVERY PROCEDURE**: Before implementing these changes, you MUST create a git recovery branch (e.g., `git checkout -b refactor-mcp-daemon`).

## 2. Sequence Flow
See [MCP_REFACTOR_FLOW.md](diagrams/MCP_REFACTOR_FLOW.md) for precise execution paths.

## 3. Precise Code Changes

### 3.1. `app/start.go` and Orchestration
- **Injectable Orchestration Function**: Export a new function (e.g., `Bootstrap(cfg Config)`) containing the logic currently embedded in `cmd/tinywasm/main.go`. This ensures that the actual `main()` only parses the port or minor CLI options and passes control, allowing orchestration to be tested cleanly with fake HTTP dependencies (`httptest`).
- **Logic Branching (`Bootstrap`)**:
  - Parse a `Dialer` interface or check port connectivity (e.g., `3030`).
  - **IF port open and `headless` is false**: Start the `devtui.NewTUI` instance in **Consumer Client Mode** connecting to the SSE server. Do NOT call `app.Start()`.
  - **IF port closed**: Start the MCP Daemon in the background natively. Wait until the port opens, then run the TUI in Client Mode. Inform the MCP daemon (via an HTTP call) that the TUI wants to "Attach" to the `$PWD` project.
  - **IF `-mcp` daemon mode requested directly**: Validate the port, then only run the `mcpserve` loop without the `devtui` or an immediate `app.Start()`.

### 3.2. `headless` Parameter in `app.Start()`
- Move the logic in `app.Start()` to accept a dynamic `headless bool` parameter. If called in this mode, skip initializing the console UI renderer.

### 3.3. Integration with `devtui` and `mcpserve`
- Extract all outputs currently going directly to `h.Logger` so they also feed into a centralized `tinywasm/sse` hub (if required by `mcpserve`).
- Modify the `mcpserve.NewHandler` call in `start.go` to provide a way to restart the `app.Start()` process when the `start_development` tool is invoked.

## 4. Diagram-Driven Testing (DDT)
As mandated by the `DEFAULT_LLM_SKILL.md`, the branching network defined in the new sequence diagram ([diagrams/MCP_REFACTOR_FLOW.md](diagrams/MCP_REFACTOR_FLOW.md)) **MUST** be covered.
- **DDT Restrictions (Never block the CI)**: Do NOT test the raw `main.go` nor test OS process spawning (`exec.Command`).
- You MUST test the pure `Bootstrap()` method by injecting a fake `net.Listener` or `httptest.Server` simulating whether the `3030` port is occupied or empty.
- Verify through mocks or return channels that `Bootstrap` calls `StartTUI_As_Client` when the port answers, and `Start_As_Daemon` when it is empty.
