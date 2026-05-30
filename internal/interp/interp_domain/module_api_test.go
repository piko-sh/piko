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

package interp_domain

import (
	"context"
	"errors"
	"testing"

	"piko.sh/piko/internal/modules/modules_domain"
)

func stubPacker(_ *CompiledFileSet) []byte {
	return []byte("stub-packed-bytecode")
}

func stubUnpacker(_ []byte, _ *SymbolRegistry) (*CompiledFileSet, error) {
	return &CompiledFileSet{}, nil
}

func TestPackageModuleRequiresPacker(t *testing.T) {
	t.Parallel()
	service := NewService()
	_, err := service.PackageModule(context.Background(),
		modules_domain.ModuleDescriptor{},
		"example.com/mod",
		nil,
		nil,
	)
	if err == nil {
		t.Fatalf("expected error when bytecodePacker is nil")
	}
}

func TestPackageModuleRejectsInvalidDescriptor(t *testing.T) {
	t.Parallel()
	service := NewService()
	_, err := service.PackageModule(context.Background(),
		modules_domain.ModuleDescriptor{},
		"example.com/mod",
		nil,
		stubPacker,
	)
	if err == nil {
		t.Fatalf("expected validation error for empty descriptor")
	}
}

func TestPackageModuleFingerprintIsStable(t *testing.T) {
	t.Parallel()
	t.Skip("CompileProgram requires a populated symbol registry; covered by integration tests in pinkas plane work")
}

func TestLoadModuleVerifiesPin(t *testing.T) {
	t.Parallel()
	service := NewService()
	bundle := &modules_domain.ModuleBundle{
		Descriptor: &modules_domain.ModuleDescriptor{
			SchemaVersion: modules_domain.DescriptorVersion,
			Ref:           modules_domain.ModuleRef{Path: "example.com/mod", Version: "v1"},
		},
		Bytecode: []byte("payload"),
	}
	wrongRef := modules_domain.ModuleRef{
		Path:    bundle.Descriptor.Ref.Path,
		Version: bundle.Descriptor.Ref.Version,
		Pin:     "sha256:0000000000000000000000000000000000000000000000000000000000000000",
	}
	_, err := service.LoadModule(context.Background(), bundle, wrongRef, nil, stubUnpacker)
	if !errors.Is(err, modules_domain.ErrIntegrityMismatch) {
		t.Fatalf("expected ErrIntegrityMismatch, got %v", err)
	}
}

func TestLoadModuleAcceptsMatchingPin(t *testing.T) {
	t.Parallel()
	service := NewService()
	bundle := &modules_domain.ModuleBundle{
		Descriptor: &modules_domain.ModuleDescriptor{
			SchemaVersion: modules_domain.DescriptorVersion,
			Ref:           modules_domain.ModuleRef{Path: "example.com/mod", Version: "v1"},
		},
		Bytecode: []byte("payload"),
	}
	fingerprint, err := bundle.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint error: %v", err)
	}
	ref := modules_domain.ModuleRef{
		Path:    bundle.Descriptor.Ref.Path,
		Version: bundle.Descriptor.Ref.Version,
		Pin:     fingerprint,
	}
	loaded, err := service.LoadModule(context.Background(), bundle, ref, nil, stubUnpacker)
	if err != nil {
		t.Fatalf("LoadModule error: %v", err)
	}
	if loaded.Fingerprint != fingerprint {
		t.Fatalf("LoadedModule.Fingerprint mismatch: got %s want %s", loaded.Fingerprint, fingerprint)
	}
}

func TestLoadModuleEmptyPinSkipsVerification(t *testing.T) {
	t.Parallel()
	service := NewService()
	bundle := &modules_domain.ModuleBundle{
		Descriptor: &modules_domain.ModuleDescriptor{
			SchemaVersion: modules_domain.DescriptorVersion,
			Ref:           modules_domain.ModuleRef{Path: "example.com/mod", Version: "v1"},
		},
		Bytecode: []byte("payload"),
	}
	ref := modules_domain.ModuleRef{Path: bundle.Descriptor.Ref.Path, Version: bundle.Descriptor.Ref.Version}
	if _, err := service.LoadModule(context.Background(), bundle, ref, nil, stubUnpacker); err != nil {
		t.Fatalf("LoadModule with empty pin should succeed, got %v", err)
	}
}

func TestLoadModuleConsultsCapabilityHook(t *testing.T) {
	t.Parallel()
	hook := &recordingHook{}
	service := NewService(WithCapabilityHook(hook))
	bundle := &modules_domain.ModuleBundle{
		Descriptor: &modules_domain.ModuleDescriptor{
			SchemaVersion: modules_domain.DescriptorVersion,
			Ref:           modules_domain.ModuleRef{Path: "example.com/mod", Version: "v1"},
			Capabilities: modules_domain.CapabilitySet{
				{Axis: "network"},
				{Axis: "filesystem.read", Scope: "/etc/*"},
			},
		},
		Bytecode: []byte("payload"),
	}
	ref := modules_domain.ModuleRef{Path: bundle.Descriptor.Ref.Path, Version: bundle.Descriptor.Ref.Version}
	if _, err := service.LoadModule(context.Background(), bundle, ref, nil, stubUnpacker); err != nil {
		t.Fatalf("LoadModule error: %v", err)
	}
	calls := hook.snapshot()
	if len(calls) < 2 {
		t.Fatalf("expected hook to be consulted at least twice (once per capability), got %d calls", len(calls))
	}
	for _, call := range calls {
		if call.ModulePath != "example.com/mod" {
			t.Errorf("hook saw modulePath=%q, want example.com/mod", call.ModulePath)
		}
	}
}

func TestLoadModuleHookDenialBlocksLoad(t *testing.T) {
	t.Parallel()
	hook := &recordingHook{
		denyFn: func(call recordedHookCall) error {
			return errors.New("module load denied")
		},
	}
	service := NewService(WithCapabilityHook(hook))
	bundle := &modules_domain.ModuleBundle{
		Descriptor: &modules_domain.ModuleDescriptor{
			SchemaVersion: modules_domain.DescriptorVersion,
			Ref:           modules_domain.ModuleRef{Path: "example.com/mod", Version: "v1"},
			Capabilities:  modules_domain.CapabilitySet{{Axis: "network"}},
		},
		Bytecode: []byte("payload"),
	}
	ref := modules_domain.ModuleRef{Path: bundle.Descriptor.Ref.Path, Version: bundle.Descriptor.Ref.Version}
	_, err := service.LoadModule(context.Background(), bundle, ref, nil, stubUnpacker)
	if !errors.Is(err, modules_domain.ErrCapabilityDenied) {
		t.Fatalf("expected ErrCapabilityDenied, got %v", err)
	}
}

func TestLoadModuleRequiresUnpacker(t *testing.T) {
	t.Parallel()
	service := NewService()
	_, err := service.LoadModule(context.Background(),
		&modules_domain.ModuleBundle{
			Descriptor: &modules_domain.ModuleDescriptor{
				SchemaVersion: modules_domain.DescriptorVersion,
				Ref:           modules_domain.ModuleRef{Path: "x"},
			},
			Bytecode: []byte("data"),
		},
		modules_domain.ModuleRef{Path: "x"},
		nil,
		nil,
	)
	if err == nil {
		t.Fatalf("expected error when unpacker is nil")
	}
}
