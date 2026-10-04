---
title: Interpreter API
description: Pipit provider registration, resource limits, restricted imports, and bytecode emission.
nav:
  sidebar:
    section: "reference"
    subsection: "bootstrap"
    order: 25
---

# Interpreter API

The `piko.sh/piko/wdk/interp/interp_provider_pipit` package supplies Piko's Pipit interpreter provider. For setup instructions, see [how to run interpreted mode](../how-to/interpreted-mode.md). For the design and trade-offs, see [about interpreted mode](../explanation/about-interpreted-mode.md).

## Registration

| Function or method | Purpose |
|---|---|
| `interp_provider_pipit.NewProvider(opts ...ProviderOption)` | Creates the provider with the specified options. |
| `(*SSRServer).WithInterpreterProvider(provider)` | Registers the provider before `Run`. Required for `dev-i`. |
| `(*SSRServer).WithSymbols(symbols)` | Registers additional native Go symbols. Optional. See [runtime symbols](runtime-symbols.md#register-custom-symbols). |

The registration methods belong to the server returned by `piko.New(...)`. Compiled `dev` and `prod` modes resolve Go symbols at build time.

## Resource limits

The following provider options apply to each interpreter the provider creates. Without explicit limits, development mode leaves these resources uncapped except for Pipit's built-in call-depth limit.

| Option | Bounds | Default with `WithRestrictedSymbolSurface()` |
|---|---|---|
| `WithMaxExecutionTime(time.Duration)` | Wall-clock time of one evaluation, including package initialisation during compilation. | 30 seconds |
| `WithCostBudget(int64)` | Metered computation cost of one execution. | No cap |
| `WithMaxCallDepth(int)` | Interpreted call-stack depth. | Pipit's built-in limit |
| `WithMaxAllocSize(int)` | Elements in one slice or channel allocation. | 16,777,216 elements |
| `WithMaxOutputSize(int)` | Bytes written by `print` and `println`. | 16 MiB |
| `WithMaxSourceSize(int)` | Total source bytes in one compilation. | 16 MiB |
| `WithMaxStringSize(int)` | Bytes produced by one string concatenation. | 64 MiB |
| `WithMaxGoroutines(int32)` | Concurrent interpreted goroutines. | 1,024 |

An explicit option overrides the restricted-mode default. Zero disables that cap, except for `WithMaxCallDepth(0)`, which retains Pipit's built-in limit. Negative values behave as zero.

### Restricted imports

`WithRestrictedSymbolSurface()` limits imports to registered host namespaces, the script's local packages, and queued modules. Language built-ins remain available. Other imports fail at compile time, including standard-library and Piko packages unless the host registers them. `unsafe` imports always fail in this mode.

The option also applies the resource defaults in the table above. Registered host functions determine what capabilities a script can access.

## Bytecode emission

`WithBytecodeEmission(directory string)` enables experimental diagnostic output. By default the provider keeps bytecode in memory.

For each compilation, emission records source files and a manifest of package paths under the chosen directory. Successful compilation also writes `compiled/bytecode-*.bin` and a matching `.pkasm` disassembly. Failed compilation writes a matching `.error.txt` file instead of bytecode.

`piko inspect bytecode <file>` decodes an emitted binary. The [setup guide](../how-to/interpreted-mode.md#inspect-generated-bytecode) shows how to enable emission and inspect a batch.

## See also

- [Bootstrap options](bootstrap-options.md#server-instance-methods-not-options) for server registration methods.
- [Runtime symbols](runtime-symbols.md) for the native packages available in interpreted mode.
- Provider source in [`wdk/interp/interp_provider_pipit`](https://github.com/piko-sh/piko/tree/master/wdk/interp/interp_provider_pipit).
