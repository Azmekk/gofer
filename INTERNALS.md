# Internals

A developer-facing guide to how Gofer works under the hood. For usage and configuration, see [README.md](README.md).

## Execution flow

```
CLI args → config load → schema validate → task resolve → params fill → env load → execute steps
```

This is orchestrated by `runTask` in `src/cmd/gofer.go`.

## Package by package

### `config` — the data model

The structs (`GoferConfig`, `Task`, `Step`, `Param`) mirror the JSON 1:1.

- **`Step.Name` is an optional display label.** When set, it is used in output formatting as the step's label. When absent, labels are derived automatically (truncated command, ref name, or `step-N` fallback).
- **`Param.Default` is `*string`, not `string`.** A nil pointer means the parameter is required. The `list` command uses this same distinction to render `<name>` vs `name=default`.
- **`Load` returns both the parsed struct AND the raw bytes.** The raw bytes go to schema validation (which works on raw JSON), while the struct goes to execution. This avoids parsing twice and keeps validation decoupled from the Go type system.
- **`LoadFromURL` fetches config over HTTP.** Makes a GET request, validates a 200 status, reads the body, and parses identically to `Load`. Returns the same `(*GoferConfig, []byte, error)` tuple.
- **`LoadAuto` dispatches between `Load` and `LoadFromURL`.** Checks if the path starts with `http://` or `https://` and delegates accordingly. Used by the CLI layer so `--config` accepts both local paths and URLs.
- **`ResolveTask` rejects dots in task names.** A forward-looking guard, probably reserving dot notation for future namespacing.

### `schema` — hand-rolled validation

Despite shipping a `gofer_schema.json` (JSON Schema Draft 7), **the validator does not use a JSON Schema library**. The schema file exists purely for editor autocomplete (VS Code, etc.). Validation is done programmatically in `schema/schema.go`.

- **Duplicate task key detection** — Go's `json.Unmarshal` silently takes the last value when there are duplicate keys. The validator works around this by manually tokenizing the JSON with `json.NewDecoder` and tracking seen keys. This is the most intricate piece of code in the project.
- **Step validation enforces "exactly one of cmd/ref/concurrent".** This is the core structural invariant.
- **OS values are validated against a hardcoded allowlist:** `linux`, `darwin`, `windows`, `*`.
- **The schema JSON is embedded via `//go:embed`** so the `init` command can write it to disk without bundling a separate file.

### `executor` — the runtime engine

Three step types, handled by a switch in `executeStep`:

1. **`cmd`** — template-resolve the string, then shell out via `sh -c` (or `cmd /C` on Windows).
2. **`ref`** — recursively call `RunTask` on another task.
3. **`concurrent`** — fan out with goroutines, collect errors with a mutex, join with `errors.Join`.

- **`Stdout` and `Stderr` writer fields** on the Executor default to `os.Stdout`/`os.Stderr`. Sequential steps use these for command output and status messages. Concurrent steps create child Executors with `PrefixWriter` wrappers so each sub-step's output is labeled with `[stepLabel]`. This plumbing means even `ref` steps inside concurrent blocks get prefixed output.
- **Circular reference detection** uses a `running map[string]bool` on the Executor struct. When a task starts it is marked; when it finishes it is deleted via `defer`. Re-entering a marked task produces a cycle error.
- **Parameter resolution is per-task, not global.** When `RunTask` is called (including via `ref`), it copies the shared params map and fills in defaults for the current task's params. A `ref` step inherits the caller's params, but the referred task's own defaults fill in anything not already provided.
- **`missingkey=error`** on the template means `{{.foo}}` with no `foo` in params is a hard error, not an empty string.
- **Concurrent steps all run to completion.** One failure does not cancel the others. Errors are collected behind a mutex and joined.
- **`os.Stdin` is connected** so commands can be interactive.

### `env` — environment loading

- **A missing env file is silently ignored.** The default `.env.gofer` might not exist and that is fine.
- **Env file values override host variables.** The host env is loaded first, then env file values are written on top. This is the opposite of what some tools do (where host takes precedence).

### `output` — formatting utilities

Uses `github.com/fatih/color` for terminal colors (respects `NO_COLOR` env var).

- **`StepLabel(step, index)`** derives a display label: explicit `Name` field → truncated `Cmd` (40 chars) → `Ref` name → `step-N` fallback.
- **`PrintStepStart`/`PrintStepDone`/`PrintStepFail`** print status lines with `▸`/`✓`/`✗` indicators to the given writer. Start is bold, done is green, fail is red.
- **`PrefixWriter`** is a thread-safe `io.Writer` that prepends a colored `[label] ` prefix to every line. It buffers partial lines internally and flushes on newline. The `Flush()` method writes any remaining buffered content.
- **Color cycling** — a palette of 6 distinct colors is cycled across concurrent sub-steps via `LabelColor(index)`.

### `cmd` — the CLI layer

- **The root command doubles as the task runner.** There is no `run` subcommand — `gofer <taskname>` directly hits `runTask`. Cobra's `Args: cobra.ArbitraryArgs` makes this work. No args shows help.
- **Positional args fill params in declaration order.** Named `-p` flags override by name.
- **`init` refuses to overwrite.** If `gofer.json` already exists it errors. `--no-schema` and `--remote-schema` are mutually exclusive.

## Versioning & self-update

### Version injection

`cmd.Version` is a package-level `var` string defaulting to `"dev"`. A normal `go build` bakes in `"dev"`. Release builds override it at link time:

```
go build -ldflags="-X github.com/Azmekk/gofer/cmd.Version=v1.0.0"
```

The `-X` flag tells the Go linker to patch a string variable by its full package path. This only works on package-level `var` strings (not `const`). The full path (`github.com/Azmekk/gofer/cmd.Version`) is required because the linker operates on compiled symbols, not source code. Cobra's `rootCmd.Version` is set to this variable, which wires up `--version` automatically.

### The `--update` flag

`--update` is a persistent flag (not a subcommand) to avoid colliding with user-defined task names. It is checked in `PersistentPreRunE` on the root command — if set, `selfUpdate()` runs and the process exits before any task logic.

### Self-update logic (`cmd/update.go`)

1. Queries the GitHub releases API for the latest tag.
2. Compares the tag against `cmd.Version` (simple string equality — both are git tags like `v0.1.0`).
3. Constructs a download URL based on `runtime.GOOS` and `runtime.GOARCH` (pattern: `gofer-{os}-{arch}[.exe]`).
4. Downloads the binary to a temp file next to the current executable (via `os.Executable()` + `filepath.EvalSymlinks`).
5. Downloads `checksums.txt` and verifies SHA256. If checksums are unavailable, it warns and continues.
6. Replaces the binary:
   - **Linux/macOS:** `os.Rename` (atomic on same filesystem).
   - **Windows:** Renames current exe to `.old`, then renames temp to the original path. The `.old` file lingers until the next update.

### Release pipeline (`.github/workflows/release.yml`)

Triggered on `v*` tag pushes. Two jobs:
- **`build`** — matrix of 6 targets (linux/darwin/windows × amd64/arm64). Cross-compiles with `CGO_ENABLED=0` and injects the tag as `cmd.Version` via `-ldflags`.
- **`release`** — downloads all artifacts, generates `checksums.txt` via `sha256sum`, and publishes a GitHub release with `softprops/action-gh-release@v2`.

### Installer scripts

- **`install.sh`** — POSIX shell, curl-pipeable. Detects OS/arch via `uname`, fetches latest release tag from GitHub API (parsed with `grep`/`cut`, no `jq`), downloads binary, best-effort SHA256 verification (tries `sha256sum` then `shasum`), installs to `$GOFER_INSTALL_DIR` or a platform-appropriate default.
- **`install.ps1`** — PowerShell 5.1+. Uses `Invoke-RestMethod` for native JSON parsing, `Get-FileHash` for SHA256 verification, installs to `$env:LOCALAPPDATA\gofer` by default, adds to user PATH via registry if needed.

## Examples

The `examples/` directory at the repo root contains sample `gofer.json` configs that demonstrate key features:

- **`coffee-break.json`** — Params with defaults, sequential steps, concurrent steps with interleaving output, groups.
- **`rubber-duck.json`** — Ref steps (task composition), per-platform OS filtering (`linux`/`darwin`/`windows`), groups.
- **`hackathon.json`** — Ref composition (chaining multiple refs), concurrent steps with labeled output, groups.

All commands are `echo`-only and cross-platform. You can run them remotely too:

```
gofer -c https://raw.githubusercontent.com/Azmekk/gofer/main/examples/coffee-break.json list
```

## Things to keep in mind

1. **Tests exist.** Run with `cd src && go test ./...`. Covers config loading, schema validation, template resolution, executor behavior (params, refs, cycles, concurrency, OS filtering), env file parsing, and output formatting.

2. **The schema JSON and the Go validator are separate truths.** Adding a new config field requires updating `gofer_schema.json` (editor support), `schema.go` (runtime validation), and the config structs. New step fields also need `output.StepLabel` updated if they affect display labels.

3. **Concurrent steps share the Executor's `running` map without synchronization.** The mutex in `executeConcurrent` only protects error collection, not the `running` map. If a concurrent step uses `ref`, multiple goroutines could read/write `running` simultaneously. This is a latent race condition.

4. **Params are stringly typed.** Everything is `map[string]string` with no type coercion or validation on values.

5. **No shell escaping.** Param values are interpolated directly into shell commands via Go templates. This is expected for a local task runner (you run your own commands), but worth being conscious of.

6. **The `output` package depends on `fatih/color`.** This is the only non-Cobra external dependency. It respects the `NO_COLOR` environment variable automatically.
