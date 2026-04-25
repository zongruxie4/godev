# PLAN: tinywasm/app — Master Plan

## Execution Order

| Stage | File | Status | Dependency |
|-------|------|--------|------------|
| 1 | [PLAN_stage1_webclient.md](PLAN_stage1_webclient.md) | pending | tinywasm/client Stage 3 |
| 2 | [PLAN_stage2_logs.md](PLAN_stage2_logs.md) | pending | none |
| 3 | [PLAN_stage3_env.md](PLAN_stage3_env.md) | pending | Stage 2, tinywasm/fmt PLAN |

Stage 2 and Stage 3 both modify `cmd/tinywasm/main.go` — they cannot be applied in parallel
by the same agent without merge conflicts. Complete Stage 2 first, then Stage 3.
Stage 3 also requires the tinywasm/fmt PLAN to be executed before it (dictionary words dependency).

