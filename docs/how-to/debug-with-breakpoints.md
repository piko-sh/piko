---
title: How to debug your Piko app with breakpoints
description: Run your Piko app under Delve to set breakpoints in Go code and step through page renders, actions, and partials.
nav:
  sidebar:
    section: "how-to"
    subsection: "tooling"
    order: 11
---

# How to debug your Piko app with breakpoints

The scaffolded `.air.toml` ships ready for breakpoint debugging with [Delve](https://github.com/go-delve/delve). Two lines of config swap Air's plain `./bin/main` exec for `dlv exec ./bin/main`. Any IDE that speaks the Delve protocol (VS Code, GoLand, Neovim) can then attach on port `2345`. From there it pauses on breakpoints in your Go code, your generated `dist/` page renderers, and your interpreted-mode action handlers.

## Install Delve

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

Confirm `dlv` is on `PATH`:

```bash
dlv version
```

## Switch Air to launch the binary under Delve

Open `.air.toml` at the project root. The scaffolded file already contains both forms - comment the plain one and uncomment the Delve line:

```toml
[build]
cmd = "go build -o ./bin/generator ./cmd/generator/main.go && ./bin/generator all && go build -gcflags \"all=-N -l\" -o ./bin/main ./cmd/main/main.go"
bin = "./bin/main"
# full_bin = "./bin/main"
full_bin = "dlv --continue --accept-multiclient --listen=:2345 --headless=true --api-version=2 exec ./bin/main --"
```

The `-gcflags "all=-N -l"` flag in `cmd` is already there. It disables compiler optimisations and inlining so line numbers and variable values match your source. Keep it whenever you debug. Remove it for performance work.

Restart Air. The runner now wraps every rebuild in `dlv exec`, leaves the process paused waiting for a client, and continues automatically once one attaches.

## Attach your IDE

### VS Code

Add to `.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Attach to Piko (Delve)",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "host": "127.0.0.1",
            "port": 2345
        }
    ]
}
```

Press F5 to attach. Set breakpoints by clicking left of any line in a `.go` file, including files under `dist/`.

### GoLand / IntelliJ

Run -> Edit Configurations -> + -> Go Remote -> set host `localhost`, port `2345`. Click the bug icon to attach.

### Neovim (nvim-dap)

```lua
local dap = require("dap")
dap.adapters.delve = {
    type = "server",
    host = "127.0.0.1",
    port = 2345,
}
dap.configurations.go = {
    {
        type = "delve",
        name = "Attach to Piko",
        request = "attach",
        mode = "remote",
    },
}
```

## What you can break in

| Target | Status |
|---|---|
| Your action handlers, services, lifecycle hooks (any `.go` file) | Supported |
| Generated page renderers under `dist/` | Supported |
| Generator output for partials, components, collections | Supported |
| Go code inside `<script lang="go">` blocks running under `dev-i` (interpreted mode) | Supported |
| `.pk` source lines mapped back from the generated renderer | Work in progress |

To step through what happens when a request hits a page, set a breakpoint in `dist/pages/<route>/<hash>/page.go` inside the rendered function and reload the page in the browser.

## Performance caveat

A binary built with `-N -l` runs slower and allocates differently. Treat any benchmark, watchdog reading, or load-test number captured in this configuration as untrustworthy. Switch back to the plain `full_bin = "./bin/main"` line and rebuild before running performance work.

## Multi-client attachment

`--accept-multiclient` keeps the Delve session alive across IDE disconnects and across Air rebuilds. You can detach, edit code, let Air rebuild, and re-attach without restarting the runner. Without this flag Delve exits the moment the first client disconnects.

## Troubleshooting

**Air rebuilds but the IDE never hits the breakpoint.** Confirm `cmd` still contains `-gcflags "all=-N -l"`. Without it the compiler inlines and reorders code, so line numbers in the binary do not match your source.

**`dlv` exits immediately on rebuild.** Check `air.log` for the build output. A failed `go build` leaves `dlv` with no binary to launch. The `stop_on_error = true` line in `.air.toml` keeps Air paused on a failed build so you can see the error.

**Breakpoint set in `dist/` does not bind.** Every `.pk` edit regenerates the `dist/` tree, which invalidates the source path Delve resolved. Re-set the breakpoint after the generator runs, or set it on a stable function name and let your IDE re-bind on attach.

**`address already in use` on port 2345.** A previous Delve process is still holding the port. Find it and stop it:

```bash
lsof -i :2345
kill <pid>
```

**Cannot step into interpreted `.pk` lines.** Source-mapping from generated Go back to `.pk` source is not yet implemented. Set the breakpoint inside the generated `dist/.../page.go` file instead - the rendered function preserves the order and shape of your template.

## See also

- [CLI reference](../reference/cli.md) for `piko` build modes.
- [How to profile a Piko app](profiling.md) for non-breakpoint performance work.
- [About interpreted mode](../explanation/about-interpreted-mode.md) for how `dev-i` runs your code without a recompile.
- [Delve documentation](https://github.com/go-delve/delve/tree/master/Documentation) for the full client protocol.
