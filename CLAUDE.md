# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Code Style (IMPORTANT)

**Use clear, descriptive variable names.** Do not use short or abbreviated variable names like `p`, `s`, `n`, `cfg`, `err` (except `err` for errors, which is idiomatic Go). Variable names should be immediately understandable without context. For example:
- `data` instead of `p`
- `step` instead of `s`
- `config` instead of `cfg`
- `count` instead of `n`
- `index` instead of `i` (in most cases)

This rule is non-negotiable. Readable code is more important than saving keystrokes.

## Build

All source code is under `src/`. There is no Makefile.

```bash
cd src && go build -o bin/gofer
```

## Run

```bash
src/bin/gofer init                          # create starter gofer.json
src/bin/gofer <task> [args...]              # run a task
src/bin/gofer list                           # list available tasks
src/bin/gofer validate                       # validate gofer.json
src/bin/gofer --version                     # print version
src/bin/gofer --update                      # self-update to latest release
```

## Quick Smoke Test

```bash
cd src && go build -o bin/gofer
bin/gofer init              # creates starter gofer.json in current directory
bin/gofer hello             # runs the hello task from the starter config
```

## Tests

```bash
cd src && go test ./...
```

See `examples/` at the repo root for sample configs demonstrating params, refs, concurrent steps, OS filtering, and groups.

## Architecture

Gofer is a JSON-based task runner (~700 lines of Go) built with Cobra. All source is in `src/`.

**Packages:**

- `main` (`main.go`) — entry point, calls `cmd.Execute()`
- `cmd` — CLI commands (Cobra): root command + `init`, `list`, `validate` subcommands. Task execution logic lives in `gofer.go`.
- `config` — loads and parses `gofer.json` into `GoferConfig`/`Task`/`Step`/`Param` structs. `ResolveTask("task")` looks up a task by name in the flat `Tasks` map.
- `executor` — runs tasks: resolves parameters via Go `text/template`, executes shell commands (`sh -c` on unix, `cmd /C` on windows), handles `ref` steps recursively with circular reference detection, and runs `concurrent` steps with goroutines + `errors.Join()`.
- `schema` — validates config structure programmatically. Also embeds `gofer_schema.json` (JSON Schema Draft 7) via `//go:embed`.
- `env` — loads `.env.gofer` files (KEY=VALUE format), merges on top of host environment.
- `output` — formatting utilities for step execution output. `PrefixWriter` for labeled concurrent output, status indicators (▸/✓/✗). Uses `fatih/color` (respects `NO_COLOR`).

**Execution flow:** CLI parses args → config loaded (local or remote URL) & validated → task resolved by name → parameters filled (positional args → `-p` flags → defaults) → env loaded → executor runs steps with formatted status output (cmd/ref/concurrent).

**Key patterns:**
- Steps are exactly one of: `cmd` (shell command), `ref` (task reference), or `concurrent` (parallel sub-steps)
- OS filtering per step: `linux`, `darwin`, `windows`, or `*`
- Template resolution uses `missingkey=error` to fail on undefined parameters
- Circular reference detection via a `running map[string]bool` on the Executor struct
- `Stdout`/`Stderr` writer fields on Executor allow output redirection; concurrent steps use `PrefixWriter` wrappers for labeled output
- Config loading supports both local paths and HTTP(S) URLs via `LoadAuto`

## Versioning & Releases

- `cmd.Version` defaults to `"dev"`. CI injects the real version via `-ldflags "-X github.com/Azmekk/gofer/cmd.Version=..."`. See [INTERNALS.md](INTERNALS.md) for how this works.
- `--update` flag triggers self-update via GitHub releases. `--version` prints the version.
- `.github/workflows/release.yml` builds 6 cross-platform binaries on `v*` tag pushes.
- `install.sh` and `install.ps1` are curl/irm-pipeable installer scripts in the project root.
