# Client Lifecycle: `tinywasm` Invocation Flow

```mermaid
flowchart TD
    A[User runs `tinywasm` in /some/dir] --> B{Port 3030 open?}

    B -- No --> C[startDaemonProcess /some/dir]
    C --> D[waitForPortReady 3030]
    D --> E[runClient /some/dir]

    B -- Yes --> F{Daemon version current?}
    F -- No --> G[killDaemon]
    G --> H[startDaemonProcess /some/dir]
    H --> I[waitForPortReady 3030]
    I --> E

    F -- Yes --> E

    E --> J[POST /action?key=start&value=/some/dir]
    E --> K[Start TUI in ClientMode<br/>connect SSE /logs]

    J --> L[Daemon: startProject /some/dir]
    L --> M[runProjectLoop: server + watcher + WASM]
    M --> N[Server ready → OpenBrowser]

    K --> O[User presses Ctrl+C]
    O --> P[POST /action?key=stop]
    O --> Q[close ExitChan]
    O --> R[ExitAltScreen → Quit]

    P --> S[Daemon: stopProject]
    Q --> T[SSE client goroutine exits]
    R --> U[Terminal restored cleanly]

    S --> V[Project loop cancelled<br/>Server + Watcher stop]

    V --> W[Daemon still running on :3030]
    U --> W

    W --> X[Next `tinywasm` run → connects as client<br/>POSTs start again]
```

## Key Invariants

| Invariant | Test |
|-----------|------|
| Every `tinywasm` invocation POSTs `start` with cwd to daemon | `TestRunClient_PostsStartAction` |
| Ctrl+C sends `stop` to daemon before closing TUI | `TestClientMode_CtrlC_SendsStop` |
| Ctrl+C exits alt-screen before quit (terminal cleaned) | `TestCtrlC_ExitsAltScreen` |
| Daemon `start` action calls `startProject(value)` | `TestDaemon_StartAction_StartsProject` |
| Daemon `stop` action calls `stopProject()` | `TestDaemon_StopAction_StopsProject` |
| Daemon survives client disconnect | (integration, not automated) |
