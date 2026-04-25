# PLAN: tinywasm/app — Master Plan

## Execution Order

| Stage | File | Status | Dependency |
|-------|------|--------|------------|
| 1 | [PLAN_stage1_webclient.md](PLAN_stage1_webclient.md) | pending | tinywasm/client Stage 3 |
| 2 | [PLAN_stage2_logs.md](PLAN_stage2_logs.md) | completed | none |
| 3 | [PLAN_stage3_env.md](PLAN_stage3_env.md) | pending | Stage 2, tinywasm/fmt PLAN |
| 4 | [PLAN_stage4_shutdown.md](PLAN_stage4_shutdown.md) | pending | Stage 2, devtui PLAN_clean_shutdown |

Stage 2 and Stage 3 both modify `cmd/tinywasm/main.go` — they cannot be applied in parallel
by the same agent without merge conflicts. Complete Stage 2 first, then Stage 3.
Stage 3 also requires the tinywasm/fmt PLAN to be executed before it (dictionary words dependency).
Stage 4 depends on Stage 2 (log-to-file required for silent shutdown) and on devtui's
PLAN_clean_shutdown.md being applied and published first (new devtui contract: no ExitChan
in TuiConfig, Shutdown() method on TuiInterface).

