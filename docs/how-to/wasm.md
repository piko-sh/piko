---
title: How to compile Piko to WebAssembly
description: Build a WASM-compatible Piko binary and serve it alongside a web frontend.
nav:
  sidebar:
    section: "how-to"
    subsection: "operations"
    order: 90
---

# How to compile Piko to WebAssembly

Build a WASM-compatible binary that runs the Piko template engine and interpreter in the browser. This guide covers the build, the JavaScript glue, and the server-side MIME setup. For the capability and limitation envelope see [about WebAssembly in Piko](../explanation/about-wasm.md). [Scenario 021: live playground](../showcase/021-live-playground.md) is the canonical runnable example.

## Build the WASM binary

Compile with Go's WASM target:

```bash
cd path/to/your/wasm/entry
GOOS=js GOARCH=wasm go build -o playground.wasm .
```

A typical entry point exposes a small JavaScript-callable surface:

```go
//go:build js

package main

import (
    "syscall/js"

    "piko.sh/piko/internal/interp"
)

func main() {
    js.Global().Set("pikoRenderTemplate", js.FuncOf(renderTemplate))
    select {} // Keep the WASM binary alive.
}

func renderTemplate(this js.Value, args []js.Value) any {
    template := args[0].String()
    dataJSON := args[1].String()

    html, err := interp.RenderTemplate(template, dataJSON)
    if err != nil {
        return map[string]any{"error": err.Error()}
    }
    return map[string]any{"html": html}
}
```

The exact entry point depends on which parts of Piko your WASM build needs. The playground scenario is the reference.

## Serve the WASM binary

Browsers require the `application/wasm` MIME type on the response. Piko's static-asset handler sets this automatically when the file has a `.wasm` extension. For a separate server (useful during development), set the MIME type explicitly.

Copy Go's WASM support glue (`wasm_exec.js`) from the Go distribution into your assets:

```bash
cp $(go env GOROOT)/lib/wasm/wasm_exec.js ./assets/wasm/
```

Reference both from the host page:

```html
<script src="/assets/wasm/wasm_exec.js"></script>
<script>
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("/assets/wasm/playground.wasm"), go.importObject)
        .then(result => go.run(result.instance));
</script>
```

## `Content-Security-Policy` considerations

WASM execution requires explicit CSP permissions. Add `'wasm-unsafe-eval'` to `script-src`:

```go
ssr := piko.New(
    piko.WithCSP(func(b *piko.CSPBuilder) {
        b.ScriptSrc("'self'", "'wasm-unsafe-eval'")
    }),
)
```

Without this, the browser blocks the WebAssembly instantiation.

## Performance

WASM Piko is slower than server-rendered Piko by an order of magnitude, but eliminates the network round-trip. For interactive templates (a playground, a live-preview editor), the net user experience is faster because the feedback loop collapses to local computation.

The WASM binary is 10-15 MB depending on which parts of Piko the build pulls in. Serve it with Brotli or gzip compression. Reverse proxies handle this transparently, but Go's `http.FileServer` does not.

## Debugging

- The browser devtools show WASM instantiation errors. Open the console before loading the page.
- Go's `syscall/js` panics surface as JavaScript exceptions; wrap entry points in recover blocks and return typed error objects for easier handling.
- Use `js.Global().Get("console").Call("log", ...)` to print from Go.

## See also

- [Scenario 021: live playground](../showcase/021-live-playground.md) for a runnable WASM-based playground.
- [Runtime symbols reference](../reference/runtime-symbols.md) for what the interpreter exposes.
- Integration tests: [`tests/integration/wasm`](https://github.com/piko-sh/piko/tree/master/tests/integration/wasm) and [`tests/integration/wasm_runner`](https://github.com/piko-sh/piko/tree/master/tests/integration/wasm_runner).
