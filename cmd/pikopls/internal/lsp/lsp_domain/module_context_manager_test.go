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

package lsp_domain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"piko.sh/piko/internal/config"
)

func TestFindGoModRoot(t *testing.T) {
	t.Run("finds go.mod in current directory", func(t *testing.T) {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}

		root, err := FindGoModRoot(context.Background(), cwd, nil)
		if err != nil {
			t.Fatalf("FindGoModRoot failed: %v", err)
		}

		goModPath := filepath.Join(root, "go.mod")
		if _, err := os.Stat(goModPath); os.IsNotExist(err) {
			t.Errorf("expected go.mod at %s but it doesn't exist", goModPath)
		}
	})

	t.Run("returns ErrNoModuleFound for root directory", func(t *testing.T) {
		_, err := FindGoModRoot(context.Background(), "/", nil)
		if !errors.Is(err, ErrNoModuleFound) {
			t.Errorf("expected ErrNoModuleFound, got %v", err)
		}
	})
}

func TestModuleContextManager(t *testing.T) {
	baseConfig := &config.PathsConfig{
		PagesSourceDir:    new("pages"),
		PartialsSourceDir: new("partials"),
		EmailsSourceDir:   new("emails"),
		AssetsSourceDir:   new("lib"),
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	t.Run("caches module contexts", func(t *testing.T) {
		manager := NewModuleContextManager(baseConfig, cwd, nil)

		ctx := context.Background()
		mc1, err := manager.GetContextForFile(ctx, cwd)
		if err != nil {
			t.Fatalf("GetContextForFile failed: %v", err)
		}

		mc2, err := manager.GetContextForFile(ctx, cwd)
		if err != nil {
			t.Fatalf("GetContextForFile failed second time: %v", err)
		}

		if mc1 != mc2 {
			t.Error("expected same ModuleContext instance from cache")
		}

		if manager.GetCachedContextCount() != 1 {
			t.Errorf("expected 1 cached context, got %d", manager.GetCachedContextCount())
		}
	})

	t.Run("invalidate clears cache", func(t *testing.T) {
		manager := NewModuleContextManager(baseConfig, cwd, nil)

		ctx := context.Background()
		_, err := manager.GetContextForFile(ctx, cwd)
		if err != nil {
			t.Fatalf("GetContextForFile failed: %v", err)
		}

		if manager.GetCachedContextCount() != 1 {
			t.Errorf("expected 1 cached context, got %d", manager.GetCachedContextCount())
		}

		manager.InvalidateAll(context.Background())

		if manager.GetCachedContextCount() != 0 {
			t.Errorf("expected 0 cached contexts after invalidation, got %d", manager.GetCachedContextCount())
		}
	})

	t.Run("uses fallback when no go.mod found", func(t *testing.T) {
		manager := NewModuleContextManager(baseConfig, cwd, nil)

		ctx := context.Background()
		mc, err := manager.GetContextForFile(ctx, "/tmp")
		if err != nil {
			t.Fatalf("GetContextForFile failed: %v", err)
		}

		if mc.ModuleRoot != cwd {
			t.Errorf("expected fallback to %s, got %s", cwd, mc.ModuleRoot)
		}
	})
}

func TestModuleContext(t *testing.T) {
	baseConfig := &config.PathsConfig{
		PagesSourceDir:    new("pages"),
		PartialsSourceDir: new("partials"),
		EmailsSourceDir:   new("emails"),
		AssetsSourceDir:   new("lib"),
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	moduleRoot, err := FindGoModRoot(context.Background(), cwd, nil)
	if err != nil {
		t.Fatalf("FindGoModRoot failed: %v", err)
	}

	t.Run("creates module context with resolver", func(t *testing.T) {
		ctx := context.Background()
		mc, err := NewModuleContext(ctx, moduleRoot, baseConfig)
		if err != nil {
			t.Fatalf("NewModuleContext failed: %v", err)
		}

		if mc.ModuleRoot != moduleRoot {
			t.Errorf("expected ModuleRoot %s, got %s", moduleRoot, mc.ModuleRoot)
		}

		if mc.Resolver == nil {
			t.Error("expected Resolver to be set")
		}

		if mc.ModuleName == "" {
			t.Error("expected ModuleName to be set")
		}
	})

	t.Run("invalidates entry points", func(t *testing.T) {
		ctx := context.Background()
		mc, err := NewModuleContext(ctx, moduleRoot, baseConfig)
		if err != nil {
			t.Fatalf("NewModuleContext failed: %v", err)
		}

		eps1, err := mc.GetEntryPoints(ctx)
		if err != nil {
			t.Fatalf("GetEntryPoints failed: %v", err)
		}

		mc.InvalidateEntryPoints()

		eps2, err := mc.GetEntryPoints(ctx)
		if err != nil {
			t.Fatalf("GetEntryPoints failed after invalidation: %v", err)
		}

		if len(eps1) != len(eps2) {
			t.Errorf("entry point count changed after invalidation: %d vs %d", len(eps1), len(eps2))
		}
	})
}
