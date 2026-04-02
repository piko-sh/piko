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
	"fmt"
	"sync"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

// ModuleContextManager lazily creates and caches ModuleContext instances for
// each Go module in the workspace. When analysing a file, it walks up to find
// the nearest go.mod and returns the corresponding cached context.
//
// This enables correct import resolution for nested modules like testdata
// directories that have their own go.mod files.
type ModuleContextManager struct {
	// sandboxFactory creates sandboxes for filesystem access. When nil,
	// no-op sandboxes are used as a fallback.
	sandboxFactory safedisk.Factory

	// contexts maps module root paths to their cached contexts.
	contexts map[string]*ModuleContext

	// basePathsConfig holds the base path settings to clone for each module.
	basePathsConfig *config.PathsConfig

	// fallbackModuleRoot is used when no go.mod is found for a file.
	fallbackModuleRoot string

	// mu guards access to the contexts map.
	mu sync.RWMutex
}

// NewModuleContextManager creates a new manager with the given base
// configuration.
//
// Takes basePathsConfig (*config.PathsConfig) which provides the base path
// settings that will be cloned for each module context.
// Takes fallbackModuleRoot (string) which is used when a file is not inside
// any go.mod.
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access. When nil, no-op sandboxes are used as a fallback.
//
// Returns *ModuleContextManager which is ready to provide module contexts.
func NewModuleContextManager(basePathsConfig *config.PathsConfig, fallbackModuleRoot string, factory safedisk.Factory) *ModuleContextManager {
	return &ModuleContextManager{
		sandboxFactory:     factory,
		fallbackModuleRoot: fallbackModuleRoot,
		basePathsConfig:    basePathsConfig,
		contexts:           make(map[string]*ModuleContext),
	}
}

// GetContextForFile returns the ModuleContext for the Go module containing the
// given file. It caches contexts by module root to avoid repeated go.mod
// lookups and resolver initialisation.
//
// Takes filePath (string) which is the absolute path to the file being
// analysed.
//
// Returns *ModuleContext which contains the resolver and entry points for the
// file's module.
// Returns error when the module context cannot be created.
//
// Safe for concurrent use.
func (m *ModuleContextManager) GetContextForFile(ctx context.Context, filePath string) (*ModuleContext, error) {
	if m == nil {
		return nil, errors.New("module context manager is nil")
	}

	ctx, l := logger_domain.From(ctx, log)

	moduleRoot, err := FindGoModRoot(ctx, filePath, m.sandboxFactory)
	if err != nil {
		if !errors.Is(err, ErrNoModuleFound) {
			return nil, fmt.Errorf("finding go.mod root for %s: %w", filePath, err)
		}
		l.Debug("No go.mod found, using fallback module root",
			logger_domain.String("filePath", filePath),
			logger_domain.String("fallback", m.fallbackModuleRoot))
		moduleRoot = m.fallbackModuleRoot
	}

	m.mu.RLock()
	mc, exists := m.contexts[moduleRoot]
	m.mu.RUnlock()

	if exists {
		l.Debug("Using cached module context",
			logger_domain.String("moduleRoot", moduleRoot),
			logger_domain.String("filePath", filePath))
		return mc, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if mc, exists = m.contexts[moduleRoot]; exists {
		return mc, nil
	}

	mc, err = NewModuleContext(ctx, moduleRoot, m.basePathsConfig, WithModuleSandboxFactory(m.sandboxFactory))
	if err != nil {
		return nil, fmt.Errorf("creating module context for %s: %w", moduleRoot, err)
	}

	m.contexts[moduleRoot] = mc
	l.Debug("Created and cached new module context",
		logger_domain.String("moduleRoot", moduleRoot),
		logger_domain.String("moduleName", mc.ModuleName))

	return mc, nil
}

// InvalidateAll clears all cached module contexts. Call this when workspace
// folders change or when a go.mod file is modified.
//
// Safe for concurrent use.
func (m *ModuleContextManager) InvalidateAll(ctx context.Context) {
	_, l := logger_domain.From(ctx, log)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.contexts = make(map[string]*ModuleContext)
	l.Debug("Invalidated all module contexts")
}

// InvalidateModule removes a specific module context from the cache.
//
// Takes moduleRoot (string) which is the path to the module to invalidate.
//
// Safe for concurrent use.
func (m *ModuleContextManager) InvalidateModule(ctx context.Context, moduleRoot string) {
	_, l := logger_domain.From(ctx, log)

	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.contexts, moduleRoot)
	l.Debug("Invalidated module context", logger_domain.String("moduleRoot", moduleRoot))
}

// InvalidateAllEntryPoints marks the cached entry points as stale on all
// cached module contexts, forcing rediscovery on the next analysis. Called
// when files are created, deleted, or renamed and the set of entry points
// may have changed.
//
// Safe for concurrent use.
func (m *ModuleContextManager) InvalidateAllEntryPoints() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, mc := range m.contexts {
		mc.InvalidateEntryPoints()
	}
}

// GetCachedContextCount returns the number of cached module contexts. This is
// primarily useful for testing and debugging.
//
// Returns int which is the number of cached contexts.
//
// Safe for concurrent use.
func (m *ModuleContextManager) GetCachedContextCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.contexts)
}
