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

package inspector_adapters

import (
	"context"
	"errors"
	"testing"

	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestInMemoryProvider_New(t *testing.T) {
	t.Run("nil initial data creates empty map", func(t *testing.T) {
		p := NewInMemoryProvider(nil)
		if p.data == nil {
			t.Fatal("expected non-nil data map")
		}
		if len(p.data) != 0 {
			t.Fatalf("expected empty data map, got %d entries", len(p.data))
		}
	})

	t.Run("non-nil initial data is used", func(t *testing.T) {
		initial := map[string]*inspector_dto.TypeData{
			"key1": {Packages: map[string]*inspector_dto.Package{"pkg": {Name: "test"}}},
		}
		p := NewInMemoryProvider(initial)
		if len(p.data) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(p.data))
		}
	})
}

func TestInMemoryProvider_GetTypeData(t *testing.T) {
	ctx := context.Background()
	td := &inspector_dto.TypeData{Packages: map[string]*inspector_dto.Package{"pkg": {Name: "test"}}}

	t.Run("cache hit", func(t *testing.T) {
		p := NewInMemoryProvider(map[string]*inspector_dto.TypeData{"key": td})
		got, err := p.GetTypeData(ctx, "key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != td {
			t.Error("expected same TypeData pointer")
		}
	})

	t.Run("cache miss", func(t *testing.T) {
		p := NewInMemoryProvider(nil)
		_, err := p.GetTypeData(ctx, "missing")
		if err == nil {
			t.Fatal("expected error for cache miss")
		}
	})

	t.Run("returns error when err is set", func(t *testing.T) {
		p := NewInMemoryProvider(map[string]*inspector_dto.TypeData{"key": td})
		testErr := errors.New("forced error")
		p.err = testErr

		_, err := p.GetTypeData(ctx, "key")
		if !errors.Is(err, testErr) {
			t.Fatalf("expected forced error, got: %v", err)
		}
	})
}

func TestInMemoryProvider_SaveTypeData(t *testing.T) {
	ctx := context.Background()
	td := &inspector_dto.TypeData{Packages: map[string]*inspector_dto.Package{"pkg": {Name: "saved"}}}

	t.Run("success", func(t *testing.T) {
		p := NewInMemoryProvider(nil)
		if err := p.SaveTypeData(ctx, "key", td); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := p.GetTypeData(ctx, "key")
		if err != nil {
			t.Fatalf("unexpected error on get: %v", err)
		}
		if got != td {
			t.Error("expected same TypeData pointer")
		}
	})

	t.Run("returns error when err is set", func(t *testing.T) {
		p := NewInMemoryProvider(nil)
		testErr := errors.New("forced save error")
		p.err = testErr

		err := p.SaveTypeData(ctx, "key", td)
		if !errors.Is(err, testErr) {
			t.Fatalf("expected forced error, got: %v", err)
		}
	})
}

func TestInMemoryProvider_InvalidateCache(t *testing.T) {
	ctx := context.Background()
	td := &inspector_dto.TypeData{Packages: map[string]*inspector_dto.Package{"pkg": {Name: "test"}}}

	t.Run("removes existing key", func(t *testing.T) {
		p := NewInMemoryProvider(map[string]*inspector_dto.TypeData{"key": td})
		if err := p.InvalidateCache(ctx, "key"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_, err := p.GetTypeData(ctx, "key")
		if err == nil {
			t.Fatal("expected cache miss after invalidation")
		}
	})

	t.Run("non-existent key is no-op", func(t *testing.T) {
		p := NewInMemoryProvider(nil)
		if err := p.InvalidateCache(ctx, "missing"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error when err is set", func(t *testing.T) {
		p := NewInMemoryProvider(nil)
		testErr := errors.New("forced invalidate error")
		p.err = testErr

		err := p.InvalidateCache(ctx, "key")
		if !errors.Is(err, testErr) {
			t.Fatalf("expected forced error, got: %v", err)
		}
	})
}

func TestInMemoryProvider_ClearCache(t *testing.T) {
	ctx := context.Background()

	t.Run("clears all entries", func(t *testing.T) {
		td := &inspector_dto.TypeData{Packages: map[string]*inspector_dto.Package{"pkg": {Name: "test"}}}
		p := NewInMemoryProvider(map[string]*inspector_dto.TypeData{
			"key1": td,
			"key2": td,
		})
		if err := p.ClearCache(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(p.data) != 0 {
			t.Fatalf("expected empty map after clear, got %d entries", len(p.data))
		}
	})

	t.Run("returns error when err is set", func(t *testing.T) {
		p := NewInMemoryProvider(nil)
		testErr := errors.New("forced clear error")
		p.err = testErr

		err := p.ClearCache(ctx)
		if !errors.Is(err, testErr) {
			t.Fatalf("expected forced error, got: %v", err)
		}
	})
}
