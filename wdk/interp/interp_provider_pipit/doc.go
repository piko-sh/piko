// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

// Package interp_provider_pipit provides Piko's interpreter provider for the interpreted
// development mode (dev-i), backed by the pipit bytecode interpreter
// (https://github.com/piko-sh/pipit).
//
// Users who do not use interpreted mode do not need to import it, and the root
// piko.sh/piko module never depends on pipit. This module is the only place that knows
// both piko's and pipit's vocabularies.
//
// Interpreters are built once with pipit's standard library, the Piko runtime symbols
// (wdk/interp/interp_piko_symbols) and any symbols added through
// [Provider.RegisterSymbols], in increasing order of precedence. The package also
// provides the WASM interpreter factory used by the browser playground, which leaves out
// the packages that cannot work in a browser.
//
// [Provider] is not safe for concurrent use after calling RegisterSymbols. The
// interpreter pool returned by [Provider.NewInterpreterPool] is safe for concurrent use.
// Every interpreter a pool hands out shares the pool's symbol registry; a package
// compiled by one is visible to the others, so a fresh registry needs a new pool.
//
// # Modules
//
// Bundles queued with [Provider.LoadModule] are loaded, and checked against their pins,
// when the pool's LoadModules method is called with the application context; a pool with
// queued modules refuses Get until then. Piko calls LoadModules once at start-up.
//
// # Resource limits
//
// The WithMax* and WithCostBudget options pass the interpreter's resource limits through.
// Trusted (development) use is unlimited unless configured. [WithRestrictedSymbolSurface]
// applies generous caps on execution time, allocation, output, source, string and
// goroutine use for untrusted scripts, each of which an explicit option overrides.
//
// # Panics
//
// A panic escaping the interpreter is recovered at the provider boundary. Its stack is
// logged once at warning level and the caller receives an error without it.
//
// # Cancellation and blocking host calls
//
// Every compile and execute entry point takes a context.Context and honours its
// cancellation in interpreted code. The VM polls ctx.Err() between bytecode steps and
// ctx.Done() inside select handling, so a cancelled context aborts interpreted loops
// within a few microseconds. pipit's execution time limit is implemented as a
// context.WithTimeout layered on the caller's ctx and so observes the same rules.
//
// What ctx cancellation cannot do is pre-empt a host function call already in progress.
// When interpreted code calls a registered Go function (for example net/http's
// ListenAndServe), the VM goroutine sits inside reflect.Value.Call until that host
// function returns. Go provides no mechanism to interrupt a blocked goroutine running
// arbitrary code, so even a cancelled context will not surface until control returns to
// the bytecode dispatcher. This affects the wall-clock limit too: a script blocked in a
// host call past its execution time limit keeps running.
//
// Embedders who care about prompt shutdown of long-running scripts should either (a) only
// expose host functions that themselves observe a context (and pass that context
// through), or (b) treat persistent unresponsiveness as a process-level concern (e.g.
// escalate to os.Exit after a grace window).
package interp_provider_pipit
