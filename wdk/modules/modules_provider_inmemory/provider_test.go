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

package modules_provider_inmemory

import (
	"context"
	"errors"
	"sync"
	"testing"

	"piko.sh/piko/wdk/modules"
)

func helperBundle(path, version string) *modules.ModuleBundle {
	return &modules.ModuleBundle{
		Descriptor: &modules.ModuleDescriptor{
			SchemaVersion: modules.DescriptorVersion,
			Ref:           modules.ModuleRef{Path: path, Version: version},
		},
		Bytecode: []byte("PIKO-bytecode-" + path + "-" + version),
	}
}

func TestInjectAndResolve(t *testing.T) {
	t.Parallel()
	provider := New()
	ref := modules.ModuleRef{Path: "example.com/mod", Version: "v1"}
	bundle := helperBundle(ref.Path, ref.Version)
	provider.Inject(ref, bundle)

	got, err := provider.Resolve(context.Background(), ref)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got != bundle {
		t.Fatalf("Resolve returned different bundle pointer")
	}
}

func TestResolveUnknownReturnsErrModuleNotFound(t *testing.T) {
	t.Parallel()
	provider := New()
	_, err := provider.Resolve(context.Background(), modules.ModuleRef{Path: "missing", Version: "v1"})
	if err == nil {
		t.Fatalf("expected error for missing module")
	}
	if !errors.Is(err, modules.ErrModuleNotFound) {
		t.Fatalf("error %v does not wrap ErrModuleNotFound", err)
	}
}

func TestRemoveDropsBundle(t *testing.T) {
	t.Parallel()
	provider := New()
	ref := modules.ModuleRef{Path: "example.com/mod", Version: "v1"}
	provider.Inject(ref, helperBundle(ref.Path, ref.Version))
	provider.Remove(ref)
	if _, err := provider.Resolve(context.Background(), ref); !errors.Is(err, modules.ErrModuleNotFound) {
		t.Fatalf("after Remove, Resolve should return ErrModuleNotFound, got %v", err)
	}
}

func TestVersionsAreSeparateKeys(t *testing.T) {
	t.Parallel()
	provider := New()
	v1 := modules.ModuleRef{Path: "example.com/mod", Version: "v1"}
	v2 := modules.ModuleRef{Path: "example.com/mod", Version: "v2"}
	provider.Inject(v1, helperBundle(v1.Path, v1.Version))
	provider.Inject(v2, helperBundle(v2.Path, v2.Version))

	if provider.Len() != 2 {
		t.Fatalf("expected Len() == 2, got %d", provider.Len())
	}
	got1, err := provider.Resolve(context.Background(), v1)
	if err != nil {
		t.Fatalf("Resolve(v1) error: %v", err)
	}
	got2, err := provider.Resolve(context.Background(), v2)
	if err != nil {
		t.Fatalf("Resolve(v2) error: %v", err)
	}
	if got1 == got2 {
		t.Fatalf("v1 and v2 resolved to same bundle")
	}
}

func TestResolveRespectsCancelledContext(t *testing.T) {
	t.Parallel()
	provider := New()
	ref := modules.ModuleRef{Path: "example.com/mod", Version: "v1"}
	provider.Inject(ref, helperBundle(ref.Path, ref.Version))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := provider.Resolve(ctx, ref); err == nil {
		t.Fatalf("expected context cancellation error")
	}
}

func TestConcurrentInjectAndResolve(t *testing.T) {
	t.Parallel()
	provider := New()
	var wg sync.WaitGroup
	wg.Add(20)
	for i := range 10 {
		go func(i int) {
			defer wg.Done()
			ref := modules.ModuleRef{Path: "mod", Version: string(rune('0' + i))}
			provider.Inject(ref, helperBundle(ref.Path, ref.Version))
		}(i)
		go func() {
			defer wg.Done()
			_, _ = provider.Resolve(context.Background(), modules.ModuleRef{Path: "mod", Version: "0"})
		}()
	}
	wg.Wait()

}
