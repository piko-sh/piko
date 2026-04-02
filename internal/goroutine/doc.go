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

// This project stands against fascism, authoritarianism, and all
// forms of oppression. We built this to empower people, not to
// enable those who would strip others of their rights and dignity.

// Package goroutine provides panic recovery and safe-call
// utilities for goroutine lifecycle management.
//
// [RecoverPanic] and [RecoverPanicToChannel] are deferred helpers
// that catch panics, log them with structured logging, and
// increment an OTel counter. The [SafeCall] family of generic
// functions wraps provider method calls, converting panics into
// [PanicError] values and detecting provider-internal timeouts
// as [ProviderTimeoutError]. All functions are safe for concurrent
// use.
//
// # Design rationale
//
// Go's default behaviour terminates the entire process when a
// goroutine panics without recovery. In a long-running server that
// calls into user-supplied or third-party provider code, a single
// panic would bring down every connection. This package centralises
// recovery with observability (OTel counters, structured logging) so
// panics are caught, reported, and the process continues. The SafeCall
// wrappers also distinguish provider-internal timeouts from caller
// timeouts, which is important for accurate error attribution in
// systems with multiple context scopes.
package goroutine
