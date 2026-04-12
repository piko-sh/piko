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

package modules_domain

import (
	"errors"
	"maps"
	"testing"
)

func TestDescriptorValidate(t *testing.T) {
	t.Parallel()

	good := &ModuleDescriptor{
		SchemaVersion: DescriptorVersion,
		Ref:           ModuleRef{Path: "example.com/mod", Version: "v1"},
		Capabilities:  CapabilitySet{{Axis: "network"}},
		Entrypoints:   map[string]string{"init": "PkgInit"},
	}
	if err := good.Validate(); err != nil {
		t.Fatalf("well-formed descriptor failed validation: %v", err)
	}

	tests := []struct {
		name    string
		mutate  func(*ModuleDescriptor)
		wantMsg string
	}{
		{"missing schema version", func(d *ModuleDescriptor) { d.SchemaVersion = 0 }, "schema_version"},
		{"future schema version", func(d *ModuleDescriptor) { d.SchemaVersion = DescriptorVersion + 1 }, "newer than supported"},
		{"missing path", func(d *ModuleDescriptor) { d.Ref.Path = "" }, "ref.path"},
		{"empty capability axis", func(d *ModuleDescriptor) {
			d.Capabilities = CapabilitySet{{Axis: ""}}
		}, "empty axis"},
		{"empty entrypoint key", func(d *ModuleDescriptor) {
			d.Entrypoints = map[string]string{"": "x"}
		}, "empty key"},
		{"empty entrypoint value", func(d *ModuleDescriptor) {
			d.Entrypoints = map[string]string{"k": ""}
		}, "empty function name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := *good
			d.Capabilities = append(CapabilitySet(nil), good.Capabilities...)
			d.Entrypoints = mapCopy(good.Entrypoints)
			tt.mutate(&d)
			err := d.Validate()
			if err == nil {
				t.Fatalf("%s: Validate() returned nil, want error containing %q", tt.name, tt.wantMsg)
			}
			if !errorContains(err, tt.wantMsg) {
				t.Fatalf("%s: error %q does not contain %q", tt.name, err.Error(), tt.wantMsg)
			}
		})
	}

	var nilDescriptor *ModuleDescriptor
	if err := nilDescriptor.Validate(); err == nil {
		t.Fatalf("nil descriptor should fail Validate")
	}
}

func TestDescriptorContentDigestIsStable(t *testing.T) {
	t.Parallel()
	a := &ModuleDescriptor{
		SchemaVersion: DescriptorVersion,
		Ref:           ModuleRef{Path: "example.com/mod", Version: "v1"},
		Capabilities: CapabilitySet{
			{Axis: "network"},
			{Axis: "filesystem.read", Scope: "/etc/*"},
		},
		SymbolAllowlist: []string{"Zeta", "Alpha"},
	}
	b := &ModuleDescriptor{
		SchemaVersion: DescriptorVersion,
		Ref:           ModuleRef{Path: "example.com/mod", Version: "v1"},
		Capabilities: CapabilitySet{
			{Axis: "filesystem.read", Scope: "/etc/*"},
			{Axis: "network"},
		},
		SymbolAllowlist: []string{"Alpha", "Zeta"},
	}
	hashA, err := a.ContentDigest()
	if err != nil {
		t.Fatalf("ContentDigest(a) error: %v", err)
	}
	hashB, err := b.ContentDigest()
	if err != nil {
		t.Fatalf("ContentDigest(b) error: %v", err)
	}
	if hashA != hashB {
		t.Fatalf("hashes differ for equivalent descriptors:\n a=%s\n b=%s", hashA, hashB)
	}
}

func TestUnmarshalDescriptorRoundTrip(t *testing.T) {
	t.Parallel()
	original := &ModuleDescriptor{
		SchemaVersion: DescriptorVersion,
		Ref:           ModuleRef{Path: "example.com/mod", Version: "v1.2.3"},
		Capabilities:  CapabilitySet{{Axis: "network"}},
		StdlibVersion: "1.0.0",
		Entrypoints:   map[string]string{"init": "PkgInit"},
	}
	data, err := original.MarshalCanonicalJSON()
	if err != nil {
		t.Fatalf("MarshalCanonicalJSON error: %v", err)
	}
	decoded, err := UnmarshalDescriptor(data)
	if err != nil {
		t.Fatalf("UnmarshalDescriptor error: %v", err)
	}
	if decoded.Ref != original.Ref {
		t.Errorf("Ref mismatch: got %+v, want %+v", decoded.Ref, original.Ref)
	}
	if decoded.StdlibVersion != original.StdlibVersion {
		t.Errorf("StdlibVersion mismatch")
	}
	if len(decoded.Capabilities) != 1 || decoded.Capabilities[0].Axis != "network" {
		t.Errorf("Capabilities mismatch: %+v", decoded.Capabilities)
	}
}

func TestUnmarshalDescriptorRejectsBadSchema(t *testing.T) {
	t.Parallel()
	bad := `{"schema_version": 0, "ref": {"path": "x"}}`
	if _, err := UnmarshalDescriptor([]byte(bad)); err == nil {
		t.Fatalf("expected error for malformed descriptor")
	}
	if _, err := UnmarshalDescriptor([]byte("not json")); err == nil {
		t.Fatalf("expected error for non-JSON input")
	}
}

func errorContains(err error, substring string) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	for i := 0; i+len(substring) <= len(message); i++ {
		if message[i:i+len(substring)] == substring {
			return true
		}
	}
	return false
}

func mapCopy(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	maps.Copy(out, m)
	return out
}

var (
	_ = errors.Is
)
