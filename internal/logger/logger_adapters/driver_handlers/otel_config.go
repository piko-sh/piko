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

package driver_handlers

// OtelSetupConfig holds the resolved OTEL configuration values. All fields are
// value types; pointer-to-value conversion is performed in the bootstrap layer.
type OtelSetupConfig struct {
	// Headers contains key-value pairs sent as HTTP headers with OTLP requests.
	Headers map[string]string

	// Protocol specifies the transport protocol: "grpc", "http", or "https".
	Protocol string

	// Endpoint is the target address for the OTLP collector.
	Endpoint string

	// TraceSampleRate is the fraction of traces to sample (0.0 to 1.0).
	TraceSampleRate float64

	// Enabled controls whether OTLP exporting is active.
	Enabled bool

	// TLSInsecure disables TLS certificate verification when true.
	TLSInsecure bool
}
