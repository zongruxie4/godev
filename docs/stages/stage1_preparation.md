# Stage 1 — Preparation & Dependencies

### Prerequisites
- Confirm that `tinywasm/mcp` Stages 1-4 are completed and the new API is available.
- Confirm `RegisterMethod` status from mcp (Stage 5).
- Confirm `mcp` now requires explicit routing (Stage 7 - `HTTPEngine`).

### Steps
- [ ] Pull latest `tinywasm/mcp` showing the new API signatures.
- [ ] Update `go.mod` in `tinywasm/app` to point to the latest `mcp` commit.
