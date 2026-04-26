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

package tui_domain

import "errors"

const (
	// fmtDecimal is the printf verb used to render a single integer
	// value as a decimal number.
	fmtDecimal = "%d"

	// titleBuild is the panel title used by BuildPanel and its
	// section headings.
	titleBuild = "Build"

	// titleProcess is the panel title used by ProcessPanel and its
	// section headings.
	titleProcess = "Process"

	// titleProfiling is the panel title used by ProfilingPanel and
	// its section headings.
	titleProfiling = "Profiling"
)

// errNoProfilingInspector is the sentinel returned when the profiling
// inspector port is not configured. Used as the value embedded in the
// per-action message err fields so callers can errors.Is against it
// rather than string-matching the message.
var errNoProfilingInspector = errors.New("no profiling inspector")

// errNoTracesProvider is the sentinel returned by panels when no traces
// provider has been configured. Embedded in the err field of refresh
// messages so callers can errors.Is against it.
var errNoTracesProvider = errors.New("no traces provider")

// errNoMetricsProvider is the sentinel returned by panels when no metrics
// provider has been configured.
var errNoMetricsProvider = errors.New("no metrics provider")

// errNoResourceProvider is the sentinel returned by panels when no
// resource provider (content/orchestrator) has been configured. Distinct
// from errNoResourcesProvider, which covers file-descriptor inventories.
var errNoResourceProvider = errors.New("no resource provider")

// errNoHealthProvider is the sentinel returned by panels when no health
// provider has been configured.
var errNoHealthProvider = errors.New("no health provider")

// errNoSystemProvider is the sentinel returned by panels when no system
// provider has been configured.
var errNoSystemProvider = errors.New("no system provider")

// errNoResourcesProvider is the sentinel returned by panels when no
// resources provider has been configured.
var errNoResourcesProvider = errors.New("no resources provider")

// errNoDLQInspector is the sentinel returned by panels when no DLQ
// inspector has been configured.
var errNoDLQInspector = errors.New("no DLQ inspector")

// errNoProvidersInspector is the sentinel returned by panels when no
// providers inspector has been configured.
var errNoProvidersInspector = errors.New("no providers inspector")

// errNoRateLimiterInspector is the sentinel returned by panels when no
// rate-limiter inspector has been configured.
var errNoRateLimiterInspector = errors.New("no rate-limiter inspector")
