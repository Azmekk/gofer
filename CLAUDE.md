# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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
```

## Quick Smoke Test

```bash
cd src && go build -o bin/gofer
bin/gofer init              # creates starter gofer.json in current directory
bin/gofer hello             # runs the hello task from the starter config
```

## Tests

No test suite exists yet. If tests are added, run with `cd src && go test ./...`.

## Architecture

Gofer is a JSON-based task runner (~700 lines of Go) built with Cobra. All source is in `src/`.

**Packages:**

- `main` (`main.go`) — entry point, calls `cmd.Execute()`
- `cmd` — CLI commands (Cobra): root command + `init`, `list`, `validate` subcommands. Task execution logic lives in `gofer.go`.
- `config` — loads and parses `gofer.json` into `GoferConfig`/`Task`/`Step`/`Param` structs. `ResolveTask("task")` looks up a task by name in the flat `Tasks` map.
- `executor` — runs tasks: resolves parameters via Go `text/template`, executes shell commands (`sh -c` on unix, `cmd /C` on windows), handles `ref` steps recursively with circular reference detection, and runs `concurrent` steps with goroutines + `errors.Join()`.
- `schema` — validates config structure programmatically. Also embeds `gofer_schema.json` (JSON Schema Draft 7) via `//go:embed`.
- `env` — loads `.env.gofer` files (KEY=VALUE format), merges on top of host environment.

**Execution flow:** CLI parses args → config loaded & validated → task resolved by name → parameters filled (positional args → `-p` flags → defaults) → env loaded → executor runs steps sequentially (cmd/ref/concurrent).

**Key patterns:**
- Steps are exactly one of: `cmd` (shell command), `ref` (task reference), or `concurrent` (parallel sub-steps)
- OS filtering per step: `linux`, `darwin`, `windows`, or `*`
- Template resolution uses `missingkey=error` to fail on undefined parameters
- Circular reference detection via a `running map[string]bool` on the Executor struct
