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
	"testing"
)

func fixtureBundle() *ModuleBundle {
	return &ModuleBundle{
		Descriptor: &ModuleDescriptor{
			SchemaVersion: DescriptorVersion,
			Ref:           ModuleRef{Path: "example.com/mod", Version: "v1.2.3"},
			Capabilities:  CapabilitySet{{Axis: "network"}},
		},
		Bytecode: []byte("PIKO\x06\x02ignore"),
	}
}

func TestBundleValidate(t *testing.T) {
	t.Parallel()
	good := fixtureBundle()
	if err := good.Validate(); err != nil {
		t.Fatalf("well-formed bundle failed validation: %v", err)
	}

	var nilBundle *ModuleBundle
	if err := nilBundle.Validate(); err == nil {
		t.Fatalf("nil bundle should fail Validate")
	}

	noDescriptor := *good
	noDescriptor.Descriptor = nil
	if err := noDescriptor.Validate(); err == nil {
		t.Fatalf("bundle without descriptor should fail Validate")
	}

	noBytecode := *good
	noBytecode.Bytecode = nil
	if err := noBytecode.Validate(); err == nil {
		t.Fatalf("bundle without bytecode should fail Validate")
	}
}

func TestBundleFingerprintStability(t *testing.T) {
	t.Parallel()
	a := fixtureBundle()
	b := fixtureBundle()

	fpA, err := a.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint(a) error: %v", err)
	}
	fpB, err := b.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint(b) error: %v", err)
	}
	if fpA != fpB {
		t.Fatalf("equivalent bundles must share fingerprint: a=%s b=%s", fpA, fpB)
	}

	c := fixtureBundle()
	c.Bytecode = append([]byte(nil), c.Bytecode...)
	c.Bytecode[0] ^= 0xff
	fpC, err := c.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint(c) error: %v", err)
	}
	if fpC == fpA {
		t.Fatalf("mutated bytecode should change fingerprint")
	}

	d := fixtureBundle()
	d.Descriptor = new(*d.Descriptor)
	d.Descriptor.Capabilities = CapabilitySet{{Axis: "exec"}}
	fpD, err := d.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint(d) error: %v", err)
	}
	if fpD == fpA {
		t.Fatalf("mutated descriptor should change fingerprint")
	}
}

func TestBundleVerifyAgainstRef(t *testing.T) {
	t.Parallel()
	bundle := fixtureBundle()
	fingerprint, err := bundle.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint error: %v", err)
	}

	if err := bundle.VerifyAgainstRef(ModuleRef{Path: bundle.Descriptor.Ref.Path}); err != nil {
		t.Fatalf("empty-pin ref should pass verification, got %v", err)
	}

	matchingRef := ModuleRef{Path: bundle.Descriptor.Ref.Path, Pin: fingerprint}
	if err := bundle.VerifyAgainstRef(matchingRef); err != nil {
		t.Fatalf("matching pin should pass verification, got %v", err)
	}

	mismatchRef := ModuleRef{Path: bundle.Descriptor.Ref.Path, Pin: "sha256:0000000000000000000000000000000000000000000000000000000000000000"}
	err = bundle.VerifyAgainstRef(mismatchRef)
	if err == nil {
		t.Fatalf("mismatched pin should fail verification")
	}
	if !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf("mismatched-pin error %v does not wrap ErrIntegrityMismatch", err)
	}
}

func TestBundleBytecodeDigestRoundTrip(t *testing.T) {
	t.Parallel()
	a := fixtureBundle()
	b := fixtureBundle()
	if a.BytecodeDigest() != b.BytecodeDigest() {
		t.Fatalf("equal bytecode should produce equal digests")
	}
	c := fixtureBundle()
	c.Bytecode = append(c.Bytecode, 0x01)
	if c.BytecodeDigest() == a.BytecodeDigest() {
		t.Fatalf("modified bytecode should produce different digest")
	}
}
