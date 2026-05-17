# What TinyWasm actually contributes to software development

A deliberate, non-marketing perspective. The aim is to be honest about what this
tool moves forward in the industry, what it does not, and where the contribution
is incremental vs. genuinely novel.

## TL;DR

TinyWasm is a small but coherent experiment in three orthogonal hypotheses:

1. **Single-language full-stack with Go+WASM is viable for a real class of apps.**
2. **The dev loop is the right place to integrate LLMs as first-class participants,
   not just code generators.**
3. **Convention over configuration applied to Go+WASM removes a missing rung from
   the ecosystem.**

None of the three is unprecedented. Together, packaged into a single TUI-driven
tool, they form a useful concrete data point — not a paradigm shift.

---

## The contributions, honestly

### 1. A working datapoint in the "single-language full-stack" experiment

The TypeScript ecosystem proved an industry-wide point: sharing types and logic
between client and server saves real engineering time. The proof is the adoption
curve, not the language itself.

TinyWasm runs the same experiment in Go. **The contribution is not the conclusion
("Go wins") but the experiment itself**: every additional credible attempt at
unified-language full-stack development gives the industry more data on which
language traits actually matter (GC behavior under WASM, compile-time vs.
runtime safety, package ecosystem density, tooling maturity).

Concretely useful for: people choosing a stack for internal tooling, dashboards,
admin panels, technical demos. Less useful for: public-facing SPAs where WASM
payload size still dominates the tradeoff.

**Honest counterweight:** Vugu, Vecty, go-app, Hugo+HTMX hybrids, and others have
explored adjacent territory. TinyWasm is not the first; it is one more credible
attempt with a different design center (TUI + LLM + module-based SSR).

### 2. LLM as a first-class actor in the dev loop, not a code-suggestion appliance

Most current LLM dev tools live at the *editing layer* — autocomplete, refactors,
PR review. They watch the developer work. TinyWasm puts the LLM **inside the
running development environment** via MCP: the model can take a screenshot, read
the wasm compile log, switch compilation modes, reload the browser, query
project state. The development feedback loop ceases to be human-only.

This is genuinely interesting because it inverts a common assumption: instead of
the LLM helping the developer drive the tooling, the tooling exposes itself as a
first-class API for the LLM. The developer becomes the supervisor of a loop the
model can partially close on its own.

**Where the real contribution sits:** not in the MCP protocol itself (that is
upstream from Anthropic), but in the demonstration that a small, opinionated dev
environment can expose just enough surface area for an LLM to make autonomous
decisions inside it without spiralling. The lesson is portable to other stacks.

**Honest counterweight:** this is a research-grade contribution at the moment.
The MCP-tooling integration is novel enough to learn from, but the workflow has
not been validated by a large user base. It might turn out that the right
granularity for LLM-in-the-loop is different from what TinyWasm exposes. The
value is the experiment, not (yet) the verdict.

### 3. Filling a real gap: convention-over-configuration for Go+WASM

JavaScript has Vite, Next, Remix, Astro — opinionated toolchains that turn
"compile, bundle, serve, hot-reload" into one verb. Go's WASM story did not have
this. Building a Go+WASM dev experience meant assembling fsnotify watchers,
tinygo invocations, custom asset pipelines, browser reload via websockets,
yourself. Every new project replicated the same plumbing.

TinyWasm consolidates the plumbing. That is not a research contribution; it is
**ecosystem hygiene**. Ecosystems mature when this kind of consolidation
happens. Webpack was not novel computer science; it was missing infrastructure
that, once present, freed people to think about their app instead of their
build.

**Honest counterweight:** the consolidation is currently opinionated to one
author's workflow. Whether the conventions generalize across the Go-WASM
community remains to be seen.

### 4. A small but useful UX primitive: compilation mode as a runtime toggle

Most build systems treat the compiler choice as a build-time, project-wide
decision. TinyWasm exposes Go-stdlib vs. TinyGo-debug vs. TinyGo-release as a
runtime toggle inside the TUI, so a developer (or an LLM) can flip modes mid-
session to investigate "is the bug a stdlib vs. tinygo divergence?".

This is a minor contribution, but it is the kind of small UX move that, once
seen, is hard to unsee. Build systems that surface compiler-choice as a
debug-time experiment rather than a CI-time decision are rare.

### 5. Pressure on the "you must use JS" assumption for browser interactivity

The pragmatic web consensus is htmx + alpine + a sprinkle of vanilla JS. TinyWasm
deliberately pushes harder: minimal JS, Go-via-WASM for behavior. The result is
sometimes awkward (WASM bundle costs, DOM round-trips) and sometimes liberating
(true type safety client-to-server).

Independent of whether the approach wins, **publicly running the experiment
matters**. The industry needs people willing to take an extreme position so the
moderate position has data to reference.

---

## What TinyWasm does NOT contribute (so we are clear)

- **It is not a paradigm shift in language, runtime, or compilation.**
- **It is not solving a problem the industry has no other solution for.**
  TypeScript, htmx+Go, React+Go-API, Phoenix LiveView, Rails+Hotwire all exist
  and work.
- **It is not yet production-validated at scale.** It is a focused tool by a
  small team. Treat claims accordingly.
- **It does not invent MCP, Go, WebAssembly, or hot reload.** It composes
  existing pieces with a specific opinion.
- **It is not a replacement for mainstream toolchains for most teams today.**
  The cost of leaving the JS ecosystem is real and project-specific.

---

## Who benefits, concretely

| Audience                              | Concrete benefit                                                                 |
|---------------------------------------|----------------------------------------------------------------------------------|
| Go developers building internal tools | Skip the JS toolchain entirely; ship a single binary.                            |
| Teams experimenting with LLM-driven dev | Working example of MCP-as-dev-loop, not a slide deck.                          |
| Framework / tooling authors            | Reference for what "LLM-friendly dev environment" can look like at code level.  |
| Educators                              | A self-contained project that demonstrates WASM, MCP, and full-stack Go.        |
| The Go+WASM ecosystem                  | One more credible attempt to standardize the dev loop in this niche.            |

## Who probably does NOT benefit

- Teams shipping consumer-facing SPAs where bundle size dominates: WASM is still
  costly here.
- Teams deeply invested in the React/Vue/Svelte ecosystem with working tooling.
- Anyone needing rich, mature ecosystem libraries (charts, editors, complex form
  widgets) — JS still leads here.
- Projects where the team's expertise is already a JS or TS stack: switching
  costs outweigh single-language wins.

---

## The meta-contribution

The most defensible thing TinyWasm contributes is not any single feature but a
**design point** in the space of "what a modern dev environment can look like
when you start from three assumptions":

1. The developer has an LLM at hand and the tooling should treat it as a peer.
2. The infrastructure should disappear; conventions replace configuration.
3. One language should reach from server to browser unless there is a hard
   reason it cannot.

You can disagree with any of the three. The value is that the codebase makes the
disagreement concrete — you can read the implementation and point at where each
assumption pays off or fails.

That is what early-stage, opinionated tools contribute to an industry: not
verdicts, but well-formed experiments other people can learn from.

---

*This document is intentionally conservative. If the project's claims later
exceed what is written here, the document should be updated, not the other way
around.*
