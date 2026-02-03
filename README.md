# Gofer

A JSON-based task runner for defining and executing parameterized shell commands across platforms.

## Features

- Flat task map with optional display groups for organization in `gofer list`
- Parameterized commands using Go template syntax (`{{.param}}`)
- Parameters via positional args, named `-p key=value` flags, or JSON defaults
- Sequential and concurrent step execution
- Task composition through `ref` steps (call one task from another)
- Per-step OS filtering (`linux`, `darwin`, `windows`, `*`)
- Environment variable loading from `.env.gofer` (or custom path)
- Circular reference detection
- Built-in config validation
- Configurable shell per step (`sh`, `bash`, `cmd`, `powershell`, `pwsh`)
- Step output formatting with status indicators (▸/✓/✗) and colored `[label]` prefixes for concurrent output
- Remote configs via `--config https://...`

## Installation

### Quick install

**macOS / Linux:**

```sh
curl -sSL https://raw.githubusercontent.com/Azmekk/gofer/main/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/Azmekk/gofer/main/install.ps1 | iex
```

### Self-update

```
gofer --update
```

### Build from source

Requires Go 1.20+.

```
git clone https://github.com/Azmekk/gofer.git
cd gofer/src
go build -o bin/gofer
```

Add `bin/gofer` to your `PATH`, or move it somewhere that already is.

## Usage

### Quick start

```
gofer init
gofer hello
```

`gofer init` creates a starter `gofer.json` in the current directory. By default it also writes a local `gofer_schema.json` next to the config and sets `$schema` to point at it, giving editors (e.g. VS Code) autocomplete and validation. Use `--remote-schema` to reference the upstream schema URL instead, or `--no-schema` to omit `$schema` entirely.

### Running tasks

```
gofer <task> [positional-args...]
gofer <task> -p key=value -p key2=value2
```

Positional args fill parameters in the order they are defined. Named `-p` flags override by name. Unset parameters fall back to their JSON `default`. Missing required parameters cause an error.

```
gofer compile main.c myapp
gofer compile -p output=myapp
gofer compile -c other-config.json
```

### Output

When running tasks, gofer prints status indicators for each step:

- `▸ label` — step starting (bold)
- `✓ label` — step succeeded (green)
- `✗ label: error` — step failed (red)

For concurrent steps, output from each sub-step is prefixed with a colored `[label]` tag so you can tell which step produced which output:

```
▸ concurrent (2 steps)
  [lint]   src/main.go:12 warning: unused var
  [test]   PASS (4 tests)
✓ concurrent (2 steps)
```

Step labels are derived from the step's `name` field if set, otherwise from the command (truncated to 40 chars) or ref name. Set `NO_COLOR=1` to disable colors.

### Remote configs

You can point `--config` at a URL to fetch a remote `gofer.json`:

```
gofer --config https://example.com/gofer.json build
```

This works with all commands (`list`, `validate`, task execution). The env file path is resolved relative to CWD regardless of where the config was loaded from.

Try the included examples remotely:

```
gofer -c https://raw.githubusercontent.com/Azmekk/gofer/main/examples/coffee-break.json brew
gofer -c https://raw.githubusercontent.com/Azmekk/gofer/main/examples/hackathon.json setup
```

### Listing tasks

```
gofer list
```

Output:

```
  hello [name=Gofer] - Prints a greeting

build:
  compile [file=main.c, <output>] - Compiles a C file

backend:
  start - Starts the server
```

Ungrouped tasks are listed first, then tasks grouped by their optional `group` field. Parameters with defaults show `name=default`. Required parameters show `<name>`.

### Validating config

```
gofer validate
```

Checks `gofer.json` for structural errors and prints them.

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--config` | `-c` | `gofer.json` | Path or URL to config file |
| `--param` | `-p` | | Task parameter (`key=value`), repeatable |
| `--version` | `-v` | | Print version |
| `--update` | | | Update gofer to the latest version |
| `--no-schema` | | | (`init` only) Omit `$schema` from generated config |
| `--remote-schema` | | | (`init` only) Use remote GitHub URL for `$schema` instead of writing a local schema file |

## Configuration

The config file is `gofer.json` at project root. Full example:

```json
{
  "env_file": ".env.gofer",
  "tasks": {
    "hello": {
      "desc": "Prints a greeting",
      "params": [
        { "name": "name", "default": "Gofer" }
      ],
      "steps": [
        { "cmd": "echo 'Hello from {{.name}}!'" }
      ]
    },
    "compile": {
      "desc": "Compiles a C file",
      "group": "build",
      "params": [
        { "name": "file", "default": "main.c" },
        { "name": "output" }
      ],
      "steps": [
        { "cmd": "gcc {{.file}} -o {{.output}}", "os": "linux" },
        { "cmd": "gcc {{.file}} -o {{.output}}.exe", "os": "windows" }
      ]
    },
    "start": {
      "desc": "Starts the server",
      "group": "backend",
      "steps": [
        { "ref": "compile" },
        {
          "concurrent": [
            { "name": "task-a", "cmd": "echo 'task A'" },
            { "name": "task-b", "cmd": "echo 'task B'" }
          ]
        }
      ]
    }
  }
}
```

### Top-level fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `env_file` | no | `.env.gofer` | Path to env file (KEY=VALUE format, `#` comments) |
| `tasks` | yes | | Map of task name to task object |

### Task

| Field | Required | Description |
|-------|----------|-------------|
| `desc` | yes | Short description |
| `group` | no | Display group name (used only for grouping in `gofer list`) |
| `params` | no | Array of parameter definitions |
| `steps` | yes | Array of steps to execute sequentially |

### Param

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Parameter name, used in templates as `{{.name}}` |
| `default` | no | Default value. If omitted, the parameter is required |

### Step

Each step must have exactly one of `cmd`, `ref`, or `concurrent`.

| Field | Description |
|-------|-------------|
| `cmd` | Shell command (Go template syntax for parameters) |
| `ref` | Reference to another task by name (e.g. `"compile"`) |
| `concurrent` | Array of steps to run in parallel |
| `name` | Optional display label for the step (used in output formatting) |
| `os` | Restrict to an OS: `linux`, `darwin`, `windows`, or `*` (default: run always) |
| `shell` | Shell to use: `sh`, `bash`, `cmd`, `powershell`, or `pwsh` (default: `bash` on Linux/macOS, `powershell` on Windows) |

### Environment file

The env file (`.env.gofer` by default) uses `KEY=VALUE` format, one per line. Lines starting with `#` are comments. Variables are merged on top of the host environment -- env file values take precedence over existing host variables.

## Examples

The `examples/` directory contains sample configs you can run directly:

- **`coffee-break.json`** — Parameterized coffee brewing, concurrent standup simulation, and a Friday deploy. Demonstrates params with defaults, concurrent steps, and groups.
- **`rubber-duck.json`** — Rubber duck debugging suite. Demonstrates ref steps (task composition), OS filtering per platform, and groups.
- **`hackathon.json`** — Hackathon speedrun from ideation to demo. Demonstrates ref chaining, concurrent project setup, and groups.

```
gofer -c examples/coffee-break.json brew
gofer -c examples/rubber-duck.json full-debug
gofer -c examples/hackathon.json setup
```

## Development

```
cd src && go test ./...
```

## License

GNU GPL v3. See [LICENSE](LICENSE).
