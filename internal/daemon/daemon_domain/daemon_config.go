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

package daemon_domain

import (
	"time"

	"piko.sh/piko/internal/tlscert"
)

// DaemonConfig holds the resolved configuration values needed by the daemon
// service for server and health probe startup. All fields are value types;
// pointer-to-value conversion is performed in the bootstrap layer.
type DaemonConfig struct {
	// HealthLivePath is the URL path for the liveness probe endpoint.
	HealthLivePath string

	// HealthReadyPath is the URL path for the readiness probe endpoint.
	HealthReadyPath string

	// NetworkPort is the TCP port for the main HTTP server.
	NetworkPort string

	// HealthPort is the TCP port for the health probe server.
	HealthPort string

	// HealthBindAddress is the address the health server binds to.
	HealthBindAddress string

	// TLSRedirectHTTPPort, when non-empty, starts a plain HTTP listener on
	// this port that 301-redirects all requests to the HTTPS server.
	TLSRedirectHTTPPort string

	// TLS holds the resolved TLS settings for the main server.
	TLS tlscert.TLSValues

	// HealthTLS holds the resolved TLS settings for the health probe server.
	HealthTLS tlscert.TLSValues

	// ShutdownDrainDelay is the duration to wait after signalling drain
	// (readiness returning 503) before shutting down the main HTTP server.
	// This gives load balancers time to deregister the instance.
	ShutdownDrainDelay time.Duration

	// MaxConcurrentSEOJobs caps the number of SEO artefact regenerations that
	// may run at once.
	//
	// Rapid build notifications (such as dev-mode hot reloads) otherwise stack
	// goroutines without bound. Zero or negative values resolve to a sensible
	// default in NewService.
	MaxConcurrentSEOJobs int

	// NetworkAutoNextPort enables automatic port selection when the default
	// port is already in use.
	NetworkAutoNextPort bool

	// HealthEnabled controls whether the health probe server starts.
	HealthEnabled bool

	// HealthAutoNextPort enables automatic port selection for the health server.
	HealthAutoNextPort bool

	// IAmACatPerson swaps the large pixel-art mascot for the small ASCII art.
	IAmACatPerson bool

	// DevelopmentMode indicates whether the daemon is running in development
	// mode (dev or dev-i). When true, internal error details are shown to
	// users instead of safe messages.
	DevelopmentMode bool
}
