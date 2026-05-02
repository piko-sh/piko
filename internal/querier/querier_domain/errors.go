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

package querier_domain

import (
	"errors"
)

var (
	// ErrMissingEnginePort is returned when a querier service is created without an engine
	// adapter.
	ErrMissingEnginePort = errors.New("querier service requires an engine port")

	// ErrMissingEmitterPort is returned when a querier service is created without a code
	// emitter adapter.
	ErrMissingEmitterPort = errors.New("querier service requires a code emitter port")

	// ErrMissingFileReaderPort is returned when a querier service is created without a file
	// reader adapter.
	ErrMissingFileReaderPort = errors.New("querier service requires a file reader port")

	// ErrMissingConfig is returned by GenerateDatabase when it is called with a nil database
	// configuration.
	//
	// The configuration is the sole source of the engine, migration paths, and query paths
	// the generator reads, so a nil value is rejected at the boundary rather than allowed to
	// surface as an opaque nil-pointer dereference further into generation.
	ErrMissingConfig = errors.New("querier service requires a database configuration")

	// ErrAsyncExecNotSupported provides the canonical message text used by the async-exec
	// diagnostic pass when a query declares the asyncexec command but the engine reports
	// SupportsAsyncMutations() == false. The pass emits a SourceError diagnostic
	// (CodeAsyncExecNotSupported, SeverityError) embedding this message so generation fails
	// rather than silently downgrading to a synchronous Exec.
	//
	// NOTE: only the message text is surfaced (via .Error()); the sentinel is not propagated
	// through Go's error chain, so consumers match the diagnostic by its Code
	// (CodeAsyncExecNotSupported), not via errors.Is.
	ErrAsyncExecNotSupported = errors.New("asyncexec is only supported on engines that surface asynchronous mutation semantics")
)
