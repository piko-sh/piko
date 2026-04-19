# Project Structure

Use this guide when scaffolding a new Piko project or understanding directory conventions. The wizard scaffolds a fixed shape; the framework reads no config files at runtime apart from an optional `config.json` (only when `WithWebsiteConfig` is not set).

## Scaffolding

Run the interactive wizard:

```bash
piko new
```

It prompts for the project name, destination, Go module path, and a few feature toggles (Sonic JSON, validator, interpreted mode, agents).

## Directory layout

```text
my-app/
├── actions/                # Server actions (Go). One subdirectory per package.
│   ├── greeting/
│   │   ├── print.go        # action.greeting.Print
│   │   └── submit.go       # action.greeting.Submit
│   └── README.md
├── cmd/
│   ├── generator/main.go   # Build-time generator. Calls internal.NewGenerator().
│   └── main/main.go        # Runtime server. Calls internal.NewServer(command).
├── components/             # Client-side .pkc Web Components
├── pages/                  # Page routes (.pk). One file per route.
│   ├── index.pk            # /
│   ├── !404.pk             # Custom 404
│   └── index_test.go       # Scaffolded ComponentTester example
├── partials/               # Reusable .pk fragments invoked via <piko:partial is="...">
│   ├── layout.pk
│   └── feature-card.pk
├── pkg/                    # Public Go packages (empty by default; create as needed)
├── lib/icons/              # SVG icons used by the scaffolded layout
├── dist/                   # Generator output. Committed; cmd/main blank-imports it.
│   └── generated.go        # Stub so a fresh checkout builds before running the generator.
├── e2e/                    # End-to-end tests
├── internal/
│   └── piko.go             # SHARED FRAMEWORK CONFIGURATION (single source of truth)
├── .air.toml               # Air live-reload config
├── .gitignore              # Excludes .piko/, .out/, tmp/, bin/, air.log, IDE/OS files
├── .dockerignore
├── Dockerfile
├── go.mod
└── go.work
```

Optional, only when the wizard's "interpreted mode" toggle is on:

```text
├── internal/interpreted/provider.go   # Yaegi symbol provider
└── .air.interpreted.toml
```

Other folders (`emails/`, `pdfs/`, `content/`, `locales/`, `assets/`, `styles/`, `db/`) are not scaffolded - create them when a feature requires them.

## Single source of configuration

`internal/piko.go` (always scaffolded) centralises every `With*` option in a `baseOptions()` slice. Both binaries draw from it:

- `internal.NewServer(mode, extras...)` - used by `cmd/main`. Adds `WithDevWidget()` and `WithDevHotreload()` in dev modes.
- `internal.NewGenerator(extras...)` - used by `cmd/generator`. Build-time only; skips runtime extras.

Both `cmd/main/main.go` and `cmd/generator/main.go` are deliberately thin: they parse `os.Args[1]`, call the appropriate constructor, and run. Do not put options in `cmd/*/main.go`. Edit `internal/piko.go` and every binary picks up the change.

The scaffolded baseOptions includes `WithCSSReset`, `WithAutoMemoryLimit`, `WithImageProvider("imaging", ...)`, `WithAutoNextPort(true)`, and a default `WithWebsiteConfig` (Inter font + grey palette). Optional toggles add `WithJSONProvider(sonicjson.New())` and `WithValidator(playground.NewValidator())`.

## Action layout

Actions live under `actions/<package>/<file>.go`. Each Go struct embedding `piko.ActionMetadata` becomes a typed RPC endpoint. The struct name (minus the `Action` suffix) is the call name. Example:

```go
// actions/greeting/print.go
package greeting

type PrintAction struct{ piko.ActionMetadata }
type PrintInput struct{}
type PrintResponse struct{ Message string `json:"message"` }
func (a PrintAction) Call(_ PrintInput) (PrintResponse, error) { ... }
```

Invoked from a template as `action.greeting.Print()`. Registration is generated into `dist/` - there is no `actions/actions.go` or central registry to edit by hand.

## Generated output

| Path | Status | Notes |
|---|---|---|
| `dist/` | committed | `cmd/main` blank-imports `_ "<module>/dist"` so action `init()` registration runs at startup. |
| `.piko/` | gitignored | Generator working files. |
| `.out/` | gitignored | Build artefact registry. |

Run `go run ./cmd/generator/main.go all` after adding pages, partials, or actions. Modes: `manifest` (default), `assets`, `sql`, `all`.

## Entry points

```bash
air                                       # dev with live reload
go run ./cmd/main                         # dev, no reload
go run ./cmd/generator/main.go all        # regenerate dist/
go build -o app ./cmd/main && ./app prod  # production
```

Run modes accepted by `cmd/main` from `os.Args[1]`: `dev` (default), `dev-i` (interpreted), `prod`. The mode flows into `internal.NewServer(command)`, which decides which dev-only options to append.

## Environment variables

The framework reads no environment variables at framework level apart from `LOG_LEVEL` (raises the bootstrap logger before options apply). Individual config struct fields may expose env overrides via struct tags (`PIKO_PORT`, `PIKO_I18N_SOURCE_DIR`, `PIKO_ACTION_SERVE_PATH`, etc.) - see `references/configuration.md`.

## LLM mistake checklist

- Putting pages outside `pages/` (they will not route).
- Editing files in `dist/` (regenerated on every build).
- Forgetting `go run ./cmd/generator/main.go all` after adding pages, partials, or actions.
- Adding options to `cmd/main/main.go` instead of `internal/piko.go`.
- Looking for `actions/actions.go` (does not exist; registration is in generated `dist/`).
- Writing `actions/<file>.go` at the top level (real layout is `actions/<package>/<file>.go`).
- Looking for `cmd/tui/` (does not exist; the TUI is a CLI subcommand: `piko watch`).
- Looking for a config file (none - everything is `With*` options in `internal/piko.go`).

## Related

- `references/configuration.md` - the full surface of `With*` options.
- `references/cli-reference.md` - `piko new`, `piko watch`, `piko agents install`.
- `references/server-actions.md` - action conventions and testing.
- `references/routing.md` - how `pages/` filenames map to URLs.
