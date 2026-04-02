# Project Structure

Use this guide when scaffolding a new project, understanding directory conventions, or configuring Piko.

## Scaffolding

Run the interactive wizard:

```bash
piko new
```

Or scaffold non-interactively by answering the prompts for project name, destination, and Go module path.

## Directory layout

```text
my-piko-app/
├── actions/              # Server-side action handlers
│   ├── actions.go        # Action definitions (auto-registered by generator)
│   ├── contact_submit.go # Example action
│   └── README.md
├── cmd/
│   ├── generator/        # Asset generator command
│   │   └── main.go
│   ├── main/             # Application entry point
│   │   └── main.go
│   └── tui/              # TUI monitoring client
│       └── main.go
├── components/           # Client-side .pkc Web Components
├── content/              # Markdown content for collections
│   └── blog/             # Collection name → content/blog/*.md
├── dist/                 # Generated output (do not edit)
│   └── generated.go
├── e2e/                  # End-to-end tests
├── internal/
│   └── yaegi/            # Yaegi interpreter symbols
├── pages/                # Page components (.pk) → routes
│   ├── index.pk          # Home page (/)
│   ├── !404.pk           # Custom error page (optional)
│   └── README.md
├── partials/             # Reusable server-side partials (.pk)
│   ├── layout.pk         # Base layout template
│   └── README.md
├── pkg/                  # Application packages
│   ├── domain/           # Business logic
│   └── dal/              # Data access layer
├── .piko/                # Internal cache/state (auto-generated)
├── .out/                 # Build output registry (auto-generated)
├── .air.toml             # Air live-reload config
├── .air.compiled.toml    # Air config for compiled mode
├── config.json           # Theme variables, app config, i18n settings
├── go.mod
├── go.work
├── piko.yaml             # Main Piko config
├── piko-dev.yaml         # Development overrides
├── piko-prod.yaml        # Production overrides
└── piko.local.yaml       # Local overrides (gitignored)
```

## Key directories

| Directory | Purpose |
|-----------|---------|
| `pages/` | File-based routing - each `.pk` file becomes a URL |
| `partials/` | Reusable server-rendered components |
| `components/` | Client-side `.pkc` Web Components |
| `actions/` | Server actions (form handlers, CRUD) |
| `content/` | Markdown files for collections |
| `pkg/` | Application business logic and data access |
| `dist/` | Generated code - never edit manually |
| `cmd/main/` | Application entry point |
| `cmd/generator/` | Asset/code generation |

## Configuration files

### piko.yaml

Main configuration:

```yaml
network:
  port: 8080
  domain: "localhost"
build:
  default_serve_mode: "interpreted"
paths:
  pages: "pages"
  partials: "partials"
  components: "components"
  actions: "actions"
```

### piko-dev.yaml / piko-prod.yaml

Environment-specific overrides. Values merge with `piko.yaml`.

### config.json

Theme variables and application configuration:

```json
{
  "theme": {
    "colour-primary": "#6F47EB",
    "colour-grey-200": "#CAD1D8",
    "spacing-md": "1rem",
    "font-family": "\"Inter\", sans-serif"
  }
}
```

Theme values become CSS custom properties with `--g-` prefix.

## Entry points

### Development server

```bash
# With live reloading (recommended)
air

# Without live reloading
go run ./cmd/main
```

### Asset generation

```bash
# Generate all assets (required after adding pages, partials, actions)
go run ./cmd/generator/main.go all
```

### Production build

```bash
go build -o app ./cmd/main
./app
```

## Go module setup

The `go.mod` file references the Piko SDK:

```text
module myapp

go 1.23

require piko.sh/piko v0.x.x
```

Run `go mod tidy` after changing dependencies.

## LLM mistake checklist

- Putting pages outside `pages/` directory (they won't route)
- Editing files in `dist/` (they get regenerated)
- Forgetting to run `go run ./cmd/generator/main.go all` after adding new pages or partials
- Forgetting to register actions in `actions/actions.go`
- Using wrong config file name (it's `piko.yaml`, not `piko.toml` or `piko.json`)

## Related

- `references/routing.md` - file-based routing details
- `references/cli-reference.md` - CLI commands
- `references/styling.md` - theme variables from config.json
