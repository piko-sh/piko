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
	"testing"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/tlscert"
)

func TestNewTLSValues_Disabled_ReturnsOff(t *testing.T) {
	t.Parallel()

	tlsConfig := &config.TLSConfig{}
	v := NewTLSValues(tlsConfig)

	if v.Mode != tlscert.TLSModeOff {
		t.Errorf("expected TLSModeOff, got %d", v.Mode)
	}
	if v.Enabled() {
		t.Error("expected Enabled() to return false")
	}
}

func TestNewTLSValues_ExplicitlyDisabled_ReturnsOff(t *testing.T) {
	t.Parallel()

	tlsConfig := &config.TLSConfig{Enabled: new(false)}
	v := NewTLSValues(tlsConfig)

	if v.Mode != tlscert.TLSModeOff {
		t.Errorf("expected TLSModeOff, got %d", v.Mode)
	}
}

func TestNewTLSValues_CertFile_Mode(t *testing.T) {
	t.Parallel()

	tlsConfig := &config.TLSConfig{
		Enabled:  new(true),
		CertFile: new("/certs/server.pem"),
		KeyFile:  new("/certs/server-key.pem"),
	}
	v := NewTLSValues(tlsConfig)

	if v.Mode != tlscert.TLSModeCertFile {
		t.Errorf("expected TLSModeCertFile, got %d", v.Mode)
	}
	if !v.Enabled() {
		t.Error("expected Enabled() to return true")
	}
	if v.CertFile != "/certs/server.pem" {
		t.Errorf("CertFile mismatch: got %q", v.CertFile)
	}
	if v.KeyFile != "/certs/server-key.pem" {
		t.Errorf("KeyFile mismatch: got %q", v.KeyFile)
	}
}

func TestNewTLSValues_ClientAuth_RequireAndVerify(t *testing.T) {
	t.Parallel()

	tlsConfig := &config.TLSConfig{
		Enabled:        new(true),
		CertFile:       new("cert.pem"),
		KeyFile:        new("key.pem"),
		ClientCAFile:   new("ca.pem"),
		ClientAuthType: new("require_and_verify"),
	}
	v := NewTLSValues(tlsConfig)

	if v.ClientAuthType != tls.RequireAndVerifyClientCert {
		t.Errorf("expected RequireAndVerifyClientCert, got %d", v.ClientAuthType)
	}
	if v.ClientCAFile != "ca.pem" {
		t.Errorf("ClientCAFile mismatch: got %q", v.ClientCAFile)
	}
}

func TestNewTLSValues_MinVersion_1_3(t *testing.T) {
	t.Parallel()

	tlsConfig := &config.TLSConfig{
		Enabled:    new(true),
		CertFile:   new("cert.pem"),
		KeyFile:    new("key.pem"),
		MinVersion: new("1.3"),
	}
	v := NewTLSValues(tlsConfig)

	if v.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected VersionTLS13 (%d), got %d", tls.VersionTLS13, v.MinVersion)
	}
}

func TestNewTLSValues_MinVersion_Default_1_2(t *testing.T) {
	t.Parallel()

	tlsConfig := &config.TLSConfig{
		Enabled:  new(true),
		CertFile: new("cert.pem"),
		KeyFile:  new("key.pem"),
	}
	v := NewTLSValues(tlsConfig)

	if v.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected VersionTLS12 (%d), got %d", tls.VersionTLS12, v.MinVersion)
	}
}

func TestNewTLSValues_HotReload(t *testing.T) {
	t.Parallel()

	tlsConfig := &config.TLSConfig{
		Enabled:   new(true),
		CertFile:  new("cert.pem"),
		KeyFile:   new("key.pem"),
		HotReload: new(true),
	}
	v := NewTLSValues(tlsConfig)

	if !v.HotReload {
		t.Error("expected HotReload to be true")
	}
}

func TestParseTLSClientAuth_AllModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected tls.ClientAuthType
	}{
		{"none", tls.NoClientCert},
		{"request", tls.RequestClientCert},
		{"require", tls.RequireAnyClientCert},
		{"verify", tls.VerifyClientCertIfGiven},
		{"require_and_verify", tls.RequireAndVerifyClientCert},
		{"", tls.NoClientCert},
		{"invalid", tls.NoClientCert},
	}

	for _, tc := range tests {
		result := parseTLSClientAuth(tc.input)
		if result != tc.expected {
			t.Errorf("parseTLSClientAuth(%q): expected %d, got %d", tc.input, tc.expected, result)
		}
	}
}

func TestParseTLSMinVersion_AllVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected uint16
	}{
		{"1.2", tls.VersionTLS12},
		{"1.3", tls.VersionTLS13},
		{"", tls.VersionTLS12},
		{"1.1", tls.VersionTLS12},
	}

	for _, tc := range tests {
		result := parseTLSMinVersion(tc.input)
		if result != tc.expected {
			t.Errorf("parseTLSMinVersion(%q): expected %d, got %d", tc.input, tc.expected, result)
		}
	}
}

func TestNewHealthTLSValues_Disabled(t *testing.T) {
	t.Parallel()

	tlsConfig := &config.HealthTLSConfig{}
	v := NewHealthTLSValues(tlsConfig)

	if v.Mode != tlscert.TLSModeOff {
		t.Errorf("expected TLSModeOff, got %d", v.Mode)
	}
}

func TestNewHealthTLSValues_Enabled(t *testing.T) {
	t.Parallel()

	tlsConfig := &config.HealthTLSConfig{
		Enabled:  new(true),
		CertFile: new("health-cert.pem"),
		KeyFile:  new("health-key.pem"),
	}
	v := NewHealthTLSValues(tlsConfig)

	if v.Mode != tlscert.TLSModeCertFile {
		t.Errorf("expected TLSModeCertFile, got %d", v.Mode)
	}
	if v.CertFile != "health-cert.pem" {
		t.Errorf("CertFile mismatch: got %q", v.CertFile)
	}
	if v.KeyFile != "health-key.pem" {
		t.Errorf("KeyFile mismatch: got %q", v.KeyFile)
	}
}

func TestNewHealthTLSValues_MinVersion(t *testing.T) {
	t.Parallel()

	tlsConfig := &config.HealthTLSConfig{
		Enabled:    new(true),
		CertFile:   new("cert.pem"),
		KeyFile:    new("key.pem"),
		MinVersion: new("1.3"),
	}
	v := NewHealthTLSValues(tlsConfig)

	if v.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected VersionTLS13, got %d", v.MinVersion)
	}
}
