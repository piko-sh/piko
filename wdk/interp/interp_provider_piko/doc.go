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

// Package interp_provider_piko provides an interpreter implementation backed by Piko's
// internal bytecode interpreter for the interpreted development mode (dev-i).
//
// Optional module: users who do not use interpreted mode do not need to import it.
// Includes WASM-aware symbol filtering for browser-based interpreted execution.
// Additional symbols can be exposed to interpreted code via [Provider.RegisterSymbols].
//
// [Provider] is not safe for concurrent use after calling RegisterSymbols. The
// interpreter pool returned by [Provider.NewInterpreterPool] is safe for concurrent use.
//
// # Cancellation and blocking host calls
//
// Every Eval / Execute / CompileFileSet entry point takes a context.Context and honours
// its cancellation in interpreted code: the VM polls ctx.Err() between bytecode steps and
// ctx.Done() inside select handling, so a cancelled context aborts interpreted loops
// within a few microseconds. [WithMaxExecutionTime] is implemented as a
// context.WithTimeout layered on the caller's ctx and so observes the same rules.
//
// What ctx cancellation cannot do is preempt a host function call already in progress.
// When interpreted code calls a registered Go function (for example net/http's
// ListenAndServe), the VM goroutine sits inside reflect.Value.Call until that host
// function returns. Go provides no mechanism to interrupt a blocked goroutine running
// arbitrary code, so even a cancelled context will not surface until control returns to
// the bytecode dispatcher. This affects the wall-clock limit too: a script blocked in a
// host call past WithMaxExecutionTime keeps running.
//
// Embedders who care about prompt shutdown of long-running scripts should either (a) only
// expose host functions that themselves observe a context (and pass that context
// through), or (b) treat persistent unresponsiveness as a process-level concern (e.g.
// escalate to os.Exit after a grace window).
package interp_provider_piko
