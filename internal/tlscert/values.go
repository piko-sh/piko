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

package tlscert

import "crypto/tls"

// TLSMode describes the TLS operating mode for a server.
type TLSMode uint8

const (
	// TLSModeOff disables TLS; the server uses plain HTTP or gRPC.
	TLSModeOff TLSMode = iota

	// TLSModeCertFile enables TLS using user-provided certificate files.
	TLSModeCertFile
)

// TLSValues holds the resolved, value-type TLS configuration for a server.
// All pointer-to-value conversion happens in the bootstrap layer.
type TLSValues struct {
	// CertFile is the path to the PEM-encoded certificate.
	CertFile string

	// KeyFile is the path to the PEM-encoded private key.
	KeyFile string

	// ClientCAFile is the path to the PEM-encoded client CA bundle for mTLS.
	ClientCAFile string

	// ClientAuthType is the resolved tls.ClientAuthType value.
	ClientAuthType tls.ClientAuthType

	// MinVersion is the resolved minimum TLS version constant
	// (e.g. tls.VersionTLS12).
	MinVersion uint16

	// Mode indicates how TLS certificates are obtained.
	Mode TLSMode

	// HotReload enables automatic certificate reloading when files change.
	HotReload bool
}

// Enabled returns true when TLS is configured in any mode.
//
// Returns bool which is true when Mode is not TLSModeOff.
func (t TLSValues) Enabled() bool {
	return t.Mode != TLSModeOff
}
