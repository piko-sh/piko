---
title: How to compile Piko to WebAssembly
description: Build the Piko WASM binary, wire its JavaScript surface, and call generate, render, and analyse from the browser.
nav:
  sidebar:
    section: "how-to"
    subsection: "operations"
    order: 90
---

# How to compile Piko to WebAssembly

Piko ships a complete WASM entry point at [`cmd/wasm/main.go`](https://github.com/piko-sh/piko/blob/master/cmd/wasm/main.go) that runs the generator, interpreter, and renderer in the browser. This guide shows how to compile it, serve it, and call its JavaScript surface, including `piko.init`, `piko.generate`, `piko.render`, `piko.dynamicRender`, `piko.analyse`, and `piko.validate`. For the capability and limitation envelope see [about WebAssembly in Piko](../explanation/about-wasm.md). [Scenario 021: live playground](../../examples/scenarios/021_live_playground/) is the canonical runnable example.

## Build the WASM binary

Compile Piko's WASM entry with Go's WASM target:

```bash
GOOS=js GOARCH=wasm go build -o piko.wasm piko.sh/piko/cmd/wasm
```

The output is a single `piko.wasm` file (typically 10-15 MB). Copy Go's WASM glue alongside it:

```bash
cp $(go env GOROOT)/lib/wasm/wasm_exec.js ./assets/wasm/
mv piko.wasm ./assets/wasm/
```

Serve both files from the same origin as the page that loads them.

## Boot the runtime

Reference the glue and the binary from the host page:

```html
<script src="/assets/wasm/wasm_exec.js"></script>
<script>
    const go = new Go();
    WebAssembly.instantiateStreaming(
        fetch("/assets/wasm/piko.wasm"),
        go.importObject,
    ).then(async (result) => {
        go.run(result.instance);
        await waitForPiko();
        await piko.init();
        // piko.generate(...), piko.render(...), etc. are now callable.
    });

    function waitForPiko(timeoutMs = 10000) {
        return new Promise((resolve, reject) => {
            const start = Date.now();
            const tick = () => {
                if (typeof piko !== "undefined") return resolve();
                if (Date.now() - start > timeoutMs) return reject(new Error("piko timeout"));
                setTimeout(tick, 10);
            };
            tick();
        });
    }
</script>
```

`go.run(...)` starts the Go runtime in the background and returns immediately. The WASM module installs a `piko` global asynchronously, so wait for it before calling `piko.init()`. Every call after `init` returns a Promise.

## Call the JavaScript surface

The `piko` global exposes the methods in the table below. Each takes a single JSON-shaped request object and returns a Promise that resolves to the documented response.

| Method | Purpose | Request shape |
|---|---|---|
| `piko.init()` | Initialise the orchestrator. Call once after the WASM boots. | none |
| `piko.analyse(req)` | Type-check Go source. | `{ sources: { "main.go": "..." }, moduleName? }` |
| `piko.generate(req)` | Compile `.pk` sources to Go. Returns generated artefacts and the manifest. | `{ sources: { "pages/index.pk": "..." }, moduleName, baseDir? }` |
| `piko.render(req)` | Render a page to HTML using static evaluation only. | `{ sources, moduleName?, entryPoint? }` |
| `piko.dynamicRender(req)` | Generate, interpret, then render. Use this when the page depends on Go code. | `{ sources, moduleName, requestURL?, props? }` |
| `piko.validate(req)` | Quick syntax/error check on Go source without full analysis. | `{ source, filePath? }` |
| `piko.getCompletions(req)` | Code completions at a cursor position. | `{ source, line, column, moduleName? }` |
| `piko.getHover(req)` | Hover info at a cursor position. | `{ source, line, column, moduleName? }` |
| `piko.getRuntimeInfo()` | Diagnostics about the WASM orchestrator. | none |

Render an in-memory page:

```js
const result = await piko.render({
    sources: {
        "pages/main.pk": "<template><h1>Hello {{ props.Name }}</h1></template>",
    },
    moduleName: "playground",
    entryPoint: "pages/main.pk",
});

if (result.success) {
    document.getElementById("preview").innerHTML = result.html;
} else {
    console.error(result.error);
}
```

`piko.dynamicRender` follows the same shape and additionally runs any Go script blocks on the templates through the embedded interpreter. The full request and response shapes appear at [`internal/wasm/wasm_dto/`](https://github.com/piko-sh/piko/tree/master/internal/wasm/wasm_dto) for reference. User code interacts with them only through the JSON surface described in the table above.

## Serve the WASM binary

Browsers require the `application/wasm` MIME type. Piko's static-asset handler sets this automatically when the file extension is `.wasm`. For a separate dev server, set the MIME type explicitly.

The binary compresses well. Serve it with Brotli or gzip through your reverse proxy. Go's `http.FileServer` does not compress on the fly.

## Configure CSP

WASM execution requires explicit CSP permissions. Add `'wasm-unsafe-eval'` to `script-src`:

```go
ssr := piko.New(
    piko.WithCSP(func(b *piko.CSPBuilder) {
        b.ScriptSrc(piko.CSPSelf, piko.CSPWasmUnsafeEval)
    }),
)
```

Without this, the browser blocks WebAssembly instantiation.

## Build a custom WASM entry

Most projects use the shipped `cmd/wasm/main.go` unchanged. The orchestrator and adapter packages that wire the binary together live under `internal/wasm/`. They are not currently re-exported on a public package, so a custom entry written outside Piko's own source tree cannot import them. If you need additional JS bindings or different orchestrator components today, fork the repository or contribute upstream and edit `cmd/wasm/main.go` directly. Refer to `cmd/wasm/main.go` for the complete option list.

## Performance

WASM Piko runs noticeably slower than server-rendered Piko because of the JavaScript-to-Go bridge and the lack of native compilation. Benchmark for the specific workload before relying on absolute numbers. The win is eliminating the network round-trip. For live playgrounds and editor previews, the feedback loop collapses to local computation, which feels faster end-to-end despite the per-call cost.

## Debugging

- The browser devtools console shows WASM instantiation errors. Open it before loading the page.
- Go panics inside `syscall/js` callbacks surface as JavaScript exceptions. The shipped `cmd/wasm/main.go` wraps each handler in `recover()` and rejects the Promise with the panic message.
- Use `js.Global().Get("console").Call("log", ...)` to print from Go.
- `piko.getRuntimeInfo()` returns the orchestrator's internal state. Use it when a render call silently returns empty output.

## See also

- [About WebAssembly in Piko](../explanation/about-wasm.md) for what WASM Piko can and cannot do.
- [Scenario 021: live playground](../../examples/scenarios/021_live_playground/) for a runnable WASM-based playground.
- [Runtime symbols reference](../reference/runtime-symbols.md) for what the embedded interpreter exposes.
- Source: [`cmd/wasm/main.go`](https://github.com/piko-sh/piko/blob/master/cmd/wasm/main.go) and [`internal/wasm/`](https://github.com/piko-sh/piko/tree/master/internal/wasm).
- Integration tests: [`tests/integration/wasm`](https://github.com/piko-sh/piko/tree/master/tests/integration/wasm) and [`tests/integration/wasm_runner`](https://github.com/piko-sh/piko/tree/master/tests/integration/wasm_runner).
