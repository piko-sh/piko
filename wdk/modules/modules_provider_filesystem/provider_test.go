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

package modules_provider_filesystem

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"piko.sh/piko/wdk/modules"
)

func helperBundle(path, version string) *modules.ModuleBundle {
	return &modules.ModuleBundle{
		Descriptor: &modules.ModuleDescriptor{
			SchemaVersion: modules.DescriptorVersion,
			Ref:           modules.ModuleRef{Path: path, Version: version},
			Capabilities:  modules.CapabilitySet{{Axis: "network"}},
		},
		Bytecode: []byte("piko-bytecode-" + path + "-" + version),
	}
}

func TestMarshalUnmarshalEnvelopeRoundTrip(t *testing.T) {
	t.Parallel()
	original := helperBundle("example.com/mod", "v1.2.3")
	envelope, err := MarshalEnvelope(original)
	if err != nil {
		t.Fatalf("MarshalEnvelope error: %v", err)
	}
	roundTrip, err := UnmarshalEnvelope(envelope)
	if err != nil {
		t.Fatalf("UnmarshalEnvelope error: %v", err)
	}
	if !bytes.Equal(original.Bytecode, roundTrip.Bytecode) {
		t.Fatalf("bytecode mismatch after round trip")
	}
	if original.Descriptor.Ref != roundTrip.Descriptor.Ref {
		t.Fatalf("ref mismatch after round trip")
	}
}

func TestUnmarshalEnvelopeRejectsMalformed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte("")},
		{"short", []byte("PKBND")},
		{"bad magic", []byte("XXXXXX\x00\x00\x00\x05hello-world")},
		{"truncated descriptor length", []byte("PKBND\x01\x00")},
		{"descriptor length overflows file", []byte("PKBND\x01\x00\x00\xff\xffabc")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := UnmarshalEnvelope(tt.data); err == nil {
				t.Fatalf("expected error for malformed envelope %q", tt.name)
			}
		})
	}
}

func TestWriteThenResolveRoundTrip(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	provider, err := New(root)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	bundle := helperBundle("example.com/mod", "v1.2.3")
	if err := provider.Write(bundle); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	got, err := provider.Resolve(context.Background(), bundle.Descriptor.Ref)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if !bytes.Equal(got.Bytecode, bundle.Bytecode) {
		t.Fatalf("bytecode mismatch after Resolve")
	}
}

func TestResolveMissingReturnsErrModuleNotFound(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	provider, err := New(root)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	_, err = provider.Resolve(context.Background(), modules.ModuleRef{Path: "missing", Version: "v1"})
	if !errors.Is(err, modules.ErrModuleNotFound) {
		t.Fatalf("Resolve missing module returned %v, want ErrModuleNotFound", err)
	}
}

func TestEncodeModuleFolder(t *testing.T) {
	t.Parallel()
	got, err := encodeModuleFolder("github.com/foo/bar")
	if err != nil {
		t.Fatalf("encodeModuleFolder error: %v", err)
	}
	want := "github.com__foo__bar"
	if got != want {
		t.Fatalf("encodeModuleFolder = %q, want %q", got, want)
	}
}

func TestEncodeModuleFolderRejectsTraversal(t *testing.T) {
	t.Parallel()
	if _, err := encodeModuleFolder("../escape"); err == nil {
		t.Fatalf("encodeModuleFolder should refuse '..' paths")
	}
}

func TestEmptyVersionStoredAsLatest(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	provider, err := New(root)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	bundle := helperBundle("example.com/mod", "")
	if err := provider.Write(bundle); err != nil {
		t.Fatalf("Write error: %v", err)
	}
	got, err := provider.Resolve(context.Background(), bundle.Descriptor.Ref)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if !bytes.Equal(got.Bytecode, bundle.Bytecode) {
		t.Fatalf("bytecode mismatch for empty-version bundle")
	}
}
