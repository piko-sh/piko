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

package daemon_adapters

import (
	"crypto/tls"
	"crypto/x509"
)

// TLSAdapterConfig holds the TLS settings needed by the HTTP server adapter
// to wrap a plain TCP listener in TLS. When nil on the adapter, the server
// uses plain HTTP.
type TLSAdapterConfig struct {
	// GetCertificate returns the server certificate dynamically, supporting
	// hot-reload and ACME. It receives the ClientHelloInfo containing the
	// SNI server name.
	GetCertificate func(*tls.ClientHelloInfo) (*tls.Certificate, error)

	// ClientCAs is the pool of certificate authorities for verifying client
	// certificates in mTLS mode. Nil when client auth is not required.
	ClientCAs *x509.CertPool

	// NextProtos lists ALPN protocol identifiers in preference order.
	// Typically ["h2", "http/1.1"] for HTTP/2 support.
	NextProtos []string

	// ClientAuth is the client certificate verification mode for mTLS.
	ClientAuth tls.ClientAuthType

	// MinVersion is the minimum TLS version to accept (e.g. tls.VersionTLS12).
	MinVersion uint16
}
