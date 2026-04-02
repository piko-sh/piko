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

package bootstrap

import (
	"crypto/tls"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/tlscert"
)

// NewTLSValues converts the pointer-based TLSConfig into a value-type
// TLSValues for the daemon service.
//
// Takes tlsConfig (*config.TLSConfig) which provides the TLS configuration values
// to convert.
//
// Returns tlscert.TLSValues which contains the resolved TLS
// configuration with defaults applied.
func NewTLSValues(tlsConfig *config.TLSConfig) tlscert.TLSValues {
	if !deref(tlsConfig.Enabled, false) {
		return tlscert.TLSValues{Mode: tlscert.TLSModeOff}
	}

	return tlscert.TLSValues{
		Mode:           tlscert.TLSModeCertFile,
		CertFile:       deref(tlsConfig.CertFile, ""),
		KeyFile:        deref(tlsConfig.KeyFile, ""),
		ClientCAFile:   deref(tlsConfig.ClientCAFile, ""),
		ClientAuthType: parseTLSClientAuth(deref(tlsConfig.ClientAuthType, "none")),
		MinVersion:     parseTLSMinVersion(deref(tlsConfig.MinVersion, "1.2")),
		HotReload:      deref(tlsConfig.HotReload, false),
	}
}

// NewHealthTLSValues converts the pointer-based HealthTLSConfig into a
// value-type TLSValues for the health probe server.
//
// Takes tlsConfig (*config.HealthTLSConfig) which provides the health TLS
// configuration values to convert.
//
// Returns tlscert.TLSValues which contains the resolved health
// TLS configuration with defaults applied.
func NewHealthTLSValues(tlsConfig *config.HealthTLSConfig) tlscert.TLSValues {
	if !deref(tlsConfig.Enabled, false) {
		return tlscert.TLSValues{Mode: tlscert.TLSModeOff}
	}

	return tlscert.TLSValues{
		Mode:       tlscert.TLSModeCertFile,
		CertFile:   deref(tlsConfig.CertFile, ""),
		KeyFile:    deref(tlsConfig.KeyFile, ""),
		MinVersion: parseTLSMinVersion(deref(tlsConfig.MinVersion, "1.2")),
	}
}

// parseTLSClientAuth maps a string client auth type to the corresponding
// tls.ClientAuthType constant.
//
// Takes s (string) which is one of "none", "request", "require", "verify",
// or "require_and_verify".
//
// Returns tls.ClientAuthType which is the corresponding Go TLS constant.
func parseTLSClientAuth(s string) tls.ClientAuthType {
	switch s {
	case "request":
		return tls.RequestClientCert
	case "require":
		return tls.RequireAnyClientCert
	case "verify":
		return tls.VerifyClientCertIfGiven
	case "require_and_verify":
		return tls.RequireAndVerifyClientCert
	default:
		return tls.NoClientCert
	}
}

// parseTLSMinVersion maps a version string to the corresponding TLS version
// constant.
//
// Takes s (string) which is either "1.2" or "1.3".
//
// Returns uint16 which is the TLS version constant.
func parseTLSMinVersion(s string) uint16 {
	switch s {
	case "1.3":
		return tls.VersionTLS13
	default:
		return tls.VersionTLS12
	}
}
