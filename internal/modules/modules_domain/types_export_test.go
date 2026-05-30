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
	"bytes"
	"testing"
)

func TestEncodeTypesExport_EmptyReturnsNil(t *testing.T) {
	t.Parallel()
	out, err := EncodeTypesExport(nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != nil {
		t.Fatalf("want nil, got %v", out)
	}
}

func TestDecodeTypesExport_EmptyReturnsNil(t *testing.T) {
	t.Parallel()
	entries, err := DecodeTypesExport(nil)
	if err != nil || entries != nil {
		t.Fatalf("err=%v entries=%v", err, entries)
	}
}

func TestTypesExport_RoundTrip(t *testing.T) {
	t.Parallel()
	in := []TypesExportEntry{
		{ImportPath: "github.com/foo/bar", Data: []byte{0x01, 0x02, 0x03}},
		{ImportPath: "github.com/foo/bar/sub", Data: []byte("hello world")},
	}
	encoded, err := EncodeTypesExport(in)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := DecodeTypesExport(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(decoded) != 2 {
		t.Fatalf("want 2 entries, got %d", len(decoded))
	}
	for i, entry := range decoded {
		if entry.ImportPath != in[i].ImportPath {
			t.Fatalf("entry[%d].ImportPath = %q, want %q", i, entry.ImportPath, in[i].ImportPath)
		}
		if !bytes.Equal(entry.Data, in[i].Data) {
			t.Fatalf("entry[%d].Data mismatch", i)
		}
	}
}

func TestEncodeTypesExport_DeterministicOrder(t *testing.T) {
	t.Parallel()
	a := []TypesExportEntry{
		{ImportPath: "z/last", Data: []byte("a")},
		{ImportPath: "a/first", Data: []byte("b")},
		{ImportPath: "m/middle", Data: []byte("c")},
	}
	b := []TypesExportEntry{
		{ImportPath: "m/middle", Data: []byte("c")},
		{ImportPath: "z/last", Data: []byte("a")},
		{ImportPath: "a/first", Data: []byte("b")},
	}
	encA, _ := EncodeTypesExport(a)
	encB, _ := EncodeTypesExport(b)
	if !bytes.Equal(encA, encB) {
		t.Fatalf("encoding not deterministic across input orderings")
	}
}

func TestDecodeTypesExport_TruncatedErrors(t *testing.T) {
	t.Parallel()
	in := []TypesExportEntry{{ImportPath: "p", Data: []byte{0x42}}}
	encoded, _ := EncodeTypesExport(in)

	for n := 1; n < len(encoded); n++ {
		if _, err := DecodeTypesExport(encoded[:n]); err == nil {
			t.Fatalf("decode of truncated %d-byte prefix should error", n)
		}
	}
}

func TestBundleFingerprint_IncludesTypesExport(t *testing.T) {
	t.Parallel()
	desc := &ModuleDescriptor{
		SchemaVersion: 1,
		Ref:           ModuleRef{Path: "example.com/m"},
	}
	a := &ModuleBundle{Descriptor: desc, Bytecode: []byte("bytecode"), TypesExport: []byte("export-A")}
	b := &ModuleBundle{Descriptor: desc, Bytecode: []byte("bytecode"), TypesExport: []byte("export-B")}
	fa, err := a.Fingerprint()
	if err != nil {
		t.Fatalf("fingerprint A: %v", err)
	}
	fb, err := b.Fingerprint()
	if err != nil {
		t.Fatalf("fingerprint B: %v", err)
	}
	if fa == fb {
		t.Fatalf("fingerprints must differ when TypesExport differs: %s", fa)
	}
}
