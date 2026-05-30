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

package modules_provider_chain

import (
	"context"
	"errors"
	"testing"

	"piko.sh/piko/wdk/modules"
)

func helperBundle(path string) *modules.ModuleBundle {
	return &modules.ModuleBundle{
		Descriptor: &modules.ModuleDescriptor{
			SchemaVersion: modules.DescriptorVersion,
			Ref:           modules.ModuleRef{Path: path, Version: "v1"},
		},
		Bytecode: []byte("bytes for " + path),
	}
}

type stubProvider struct {
	bundle *modules.ModuleBundle
	err    error
	calls  int
}

func (s *stubProvider) Resolve(_ context.Context, _ modules.ModuleRef) (*modules.ModuleBundle, error) {
	s.calls++
	return s.bundle, s.err
}

func TestChainReturnsFirstHit(t *testing.T) {
	t.Parallel()
	first := &stubProvider{bundle: helperBundle("first")}
	second := &stubProvider{bundle: helperBundle("second")}
	chain := New(first, second)

	got, err := chain.Resolve(context.Background(), modules.ModuleRef{Path: "x"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got != first.bundle {
		t.Fatalf("Resolve returned %v, want first.bundle", got)
	}
	if second.calls != 0 {
		t.Fatalf("second provider should not be consulted after first hit")
	}
}

func TestChainFallsThroughOnNotFound(t *testing.T) {
	t.Parallel()
	first := &stubProvider{err: modules.ErrModuleNotFound}
	second := &stubProvider{bundle: helperBundle("second")}
	chain := New(first, second)

	got, err := chain.Resolve(context.Background(), modules.ModuleRef{Path: "x"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got != second.bundle {
		t.Fatalf("Resolve returned wrong bundle")
	}
	if first.calls != 1 || second.calls != 1 {
		t.Fatalf("call counts unexpected: first=%d second=%d", first.calls, second.calls)
	}
}

func TestChainOtherErrorsShortCircuit(t *testing.T) {
	t.Parallel()
	denial := errors.New("policy denial")
	first := &stubProvider{err: denial}
	second := &stubProvider{bundle: helperBundle("second")}
	chain := New(first, second)

	_, err := chain.Resolve(context.Background(), modules.ModuleRef{Path: "x"})
	if !errors.Is(err, denial) {
		t.Fatalf("Resolve error %v, want denial", err)
	}
	if second.calls != 0 {
		t.Fatalf("second provider must not be consulted after non-not-found error")
	}
}

func TestChainAllNotFoundReturnsErrModuleNotFound(t *testing.T) {
	t.Parallel()
	chain := New(
		&stubProvider{err: modules.ErrModuleNotFound},
		&stubProvider{err: modules.ErrModuleNotFound},
	)
	_, err := chain.Resolve(context.Background(), modules.ModuleRef{Path: "x"})
	if !errors.Is(err, modules.ErrModuleNotFound) {
		t.Fatalf("Resolve error %v, want ErrModuleNotFound", err)
	}
}

func TestChainSkipsNilLinks(t *testing.T) {
	t.Parallel()
	hit := &stubProvider{bundle: helperBundle("hit")}
	chain := New(nil, hit, nil)
	if chain.Len() != 1 {
		t.Fatalf("nil links should be skipped; Len() = %d", chain.Len())
	}
	got, err := chain.Resolve(context.Background(), modules.ModuleRef{Path: "x"})
	if err != nil || got != hit.bundle {
		t.Fatalf("nil-skipping chain failed: got %v err %v", got, err)
	}
}

func TestChainCancelledContextShortCircuits(t *testing.T) {
	t.Parallel()
	hit := &stubProvider{bundle: helperBundle("hit")}
	chain := New(hit)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := chain.Resolve(ctx, modules.ModuleRef{Path: "x"})
	if err == nil {
		t.Fatalf("expected context error")
	}
	if hit.calls != 0 {
		t.Fatalf("cancelled context should not consult chain links")
	}
}
