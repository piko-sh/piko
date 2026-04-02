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

import (
	"crypto/tls"
	"testing"
)

func TestTLSModeConstants(t *testing.T) {
	t.Parallel()

	if TLSModeOff != 0 {
		t.Errorf("expected TLSModeOff == 0, got %d", TLSModeOff)
	}
	if TLSModeCertFile != 1 {
		t.Errorf("expected TLSModeCertFile == 1, got %d", TLSModeCertFile)
	}
}

func TestTLSValues_Enabled_Off(t *testing.T) {
	t.Parallel()

	v := TLSValues{Mode: TLSModeOff}
	if v.Enabled() {
		t.Error("expected Enabled() to return false for TLSModeOff")
	}
}

func TestTLSValues_Enabled_CertFile(t *testing.T) {
	t.Parallel()

	v := TLSValues{Mode: TLSModeCertFile}
	if !v.Enabled() {
		t.Error("expected Enabled() to return true for TLSModeCertFile")
	}
}

func TestTLSValues_ZeroValue_IsDisabled(t *testing.T) {
	t.Parallel()

	var v TLSValues
	if v.Enabled() {
		t.Error("expected zero-value TLSValues to be disabled")
	}
}

func TestTLSValues_FieldsPreserved(t *testing.T) {
	t.Parallel()

	v := TLSValues{
		Mode:           TLSModeCertFile,
		CertFile:       "/certs/server.pem",
		KeyFile:        "/certs/server-key.pem",
		ClientCAFile:   "/certs/ca.pem",
		ClientAuthType: tls.RequireAndVerifyClientCert,
		MinVersion:     tls.VersionTLS13,
		HotReload:      true,
	}

	if v.CertFile != "/certs/server.pem" {
		t.Errorf("CertFile mismatch: got %q", v.CertFile)
	}
	if v.KeyFile != "/certs/server-key.pem" {
		t.Errorf("KeyFile mismatch: got %q", v.KeyFile)
	}
	if v.ClientCAFile != "/certs/ca.pem" {
		t.Errorf("ClientCAFile mismatch: got %q", v.ClientCAFile)
	}
	if v.ClientAuthType != tls.RequireAndVerifyClientCert {
		t.Errorf("ClientAuthType mismatch: got %d", v.ClientAuthType)
	}
	if v.MinVersion != tls.VersionTLS13 {
		t.Errorf("MinVersion mismatch: got %d", v.MinVersion)
	}
	if !v.HotReload {
		t.Error("expected HotReload to be true")
	}
}
