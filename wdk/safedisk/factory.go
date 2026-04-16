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

package safedisk

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// Factory creates sandboxes with validated paths.
// It restricts sandbox creation to paths within the allowed list,
// preventing malicious or accidental access to sensitive directories.
type Factory interface {
	// Create creates a new sandbox for the given path. The path must be within
	// the allowed paths list if configured; CWD is always implicitly allowed.
	//
	// Takes purpose (string) which is a descriptive name for logging.
	// Takes path (string) which is the directory to sandbox.
	// Takes mode (Mode) which specifies whether to allow write operations.
	//
	// Returns Sandbox which is the sandboxed filesystem.
	// Returns error when the path is not allowed or sandbox creation fails.
	Create(purpose string, path string, mode Mode) (Sandbox, error)

	// MustCreate works like Create but panics on error.
	// Use only during application startup when failure cannot be recovered.
	//
	// Takes purpose (string) which describes why the sandbox is needed.
	// Takes path (string) which is the root directory for the sandbox.
	// Takes mode (Mode) which sets the sandbox restrictions.
	//
	// Returns Sandbox which is the created sandbox ready for use.
	MustCreate(purpose string, path string, mode Mode) Sandbox

	// IsPathAllowed checks if a path would be allowed for sandbox creation.
	//
	// Takes path (string) the path to check.
	//
	// Returns bool true if the path is allowed.
	IsPathAllowed(path string) bool

	// AllowedPaths returns the list of paths that are allowed.
	//
	// Returns []string which is empty if all paths are allowed.
	AllowedPaths() []string
}

// FactoryConfig configures the sandbox factory.
type FactoryConfig struct {
	// CWD is the current working directory, which is always allowed.
	// If empty, it is detected automatically.
	CWD string

	// AllowedPaths is a list of absolute paths that can be
	// sandboxed, normalised during factory creation, where an
	// empty list allows all paths (backwards compatible).
	AllowedPaths []string

	// Enabled controls whether sandboxing uses os.Root (true)
	// or NoOpSandbox (false), where false provides no actual
	// security and only basic path validation (default: true).
	Enabled bool
}

// factoryImpl is the default implementation of the Factory interface.
type factoryImpl struct {
	// allowedPaths holds absolute paths that limit where file operations can occur.
	allowedPaths []string

	// config holds the sandbox factory settings.
	config FactoryConfig

	// mu guards factory fields for safe concurrent access.
	mu sync.RWMutex
}

// Create creates a new sandbox for the given path.
//
// Takes purpose (string) which describes the sandbox's intended use.
// Takes path (string) which specifies the filesystem path to sandbox.
// Takes mode (Mode) which sets the access mode for the sandbox.
//
// Returns Sandbox which is either a real sandbox or a no-op implementation
// depending on the factory configuration.
// Returns error when path is empty, invalid, or not within allowed paths.
func (f *factoryImpl) Create(purpose string, path string, mode Mode) (Sandbox, error) {
	if path == "" {
		return nil, fmt.Errorf("safedisk: %s: %w", purpose, errEmptyPath)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("safedisk: %s: invalid path %q: %w", purpose, path, err)
	}

	if !f.checkPathAllowed(absPath) {
		return nil, fmt.Errorf("safedisk: %s: %w: %q is not within allowed paths %v", purpose, errPathNotAllowed, absPath, f.allowedPaths)
	}

	if f.config.Enabled {
		return NewSandbox(absPath, mode)
	}

	return NewNoOpSandbox(absPath, mode)
}

// MustCreate is like Create but panics on error.
//
// Takes purpose (string) which describes the sandbox's intended use.
// Takes path (string) which specifies the directory path for the sandbox.
// Takes mode (Mode) which defines the sandbox behaviour mode.
//
// Returns Sandbox which is the created sandbox instance.
//
// Panics when sandbox creation fails.
func (f *factoryImpl) MustCreate(purpose string, path string, mode Mode) Sandbox {
	sandbox, err := f.Create(purpose, path, mode)
	if err != nil {
		panic(fmt.Sprintf("safedisk: failed to create sandbox for %s at %q: %v", purpose, path, err))
	}
	return sandbox
}

// IsPathAllowed checks if a path would be allowed for sandbox creation.
//
// Takes path (string) which specifies the file path to check.
//
// Returns bool which is true if the path is allowed, false otherwise.
func (f *factoryImpl) IsPathAllowed(path string) bool {
	if path == "" {
		return false
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	return f.checkPathAllowed(absPath)
}

// AllowedPaths returns the list of configured allowed paths.
//
// Returns []string which is a copy of the allowed paths to prevent mutation.
//
// Safe for concurrent use.
func (f *factoryImpl) AllowedPaths() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	paths := make([]string, len(f.allowedPaths))
	copy(paths, f.allowedPaths)
	return paths
}

// checkPathAllowed checks if an absolute path is within the allowed paths.
//
// Takes absPath (string) which is the path to check.
//
// Returns bool which is true if the path is allowed, false otherwise.
//
// Safe for concurrent use; protected by a read lock on the factory.
func (f *factoryImpl) checkPathAllowed(absPath string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if len(f.allowedPaths) == 0 {
		return true
	}

	for _, allowed := range f.allowedPaths {
		if isWithinOrEqual(allowed, absPath) {
			return true
		}
	}

	return false
}

var (
	// globalFactory is a shared factory instance for use across the package.
	// It must be set up before use by calling initialiseGlobalFactory.
	globalFactory Factory

	// globalFactoryInit indicates whether the global factory has been initialised.
	globalFactoryInit bool

	// globalFactoryMu guards concurrent access to globalFactory and globalFactoryInit.
	globalFactoryMu sync.RWMutex
)

// NewFactory creates a new sandbox factory with the given settings.
//
// Takes config (FactoryConfig) which specifies the factory settings.
//
// Returns Factory which is the configured factory ready for use.
// Returns error when the configuration contains an invalid path or when the
// current working directory cannot be determined.
func NewFactory(config FactoryConfig) (Factory, error) {
	cwd := config.CWD
	if cwd == "" {
		var err error
		cwd, err = filepath.Abs(".")
		if err != nil {
			return nil, fmt.Errorf("safedisk: failed to get current working directory: %w", err)
		}
	} else {
		var err error
		cwd, err = filepath.Abs(cwd)
		if err != nil {
			return nil, fmt.Errorf("safedisk: invalid CWD path: %w", err)
		}
	}

	var allowedPaths []string
	seen := make(map[string]bool)

	seen[cwd] = true
	allowedPaths = append(allowedPaths, cwd)

	for _, p := range config.AllowedPaths {
		if p == "" {
			continue
		}

		absPath, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("safedisk: invalid allowed path %q: %w", p, err)
		}

		if !seen[absPath] {
			seen[absPath] = true
			allowedPaths = append(allowedPaths, absPath)
		}
	}

	return &factoryImpl{
		allowedPaths: allowedPaths,
		config:       config,
		mu:           sync.RWMutex{},
	}, nil
}

// NewCLIFactory creates a factory suitable for CLI tools and standalone
// commands. It enables kernel-level sandboxing, uses the given working
// directory, and places no restrictions on allowed paths.
//
// Takes cwd (string) which is the working directory for the factory. When
// empty, the current working directory is detected automatically.
//
// Returns Factory which creates sandboxes with no path restrictions.
// Returns error when factory creation fails.
func NewCLIFactory(cwd string) (Factory, error) {
	return NewFactory(FactoryConfig{
		CWD:          cwd,
		AllowedPaths: nil,
		Enabled:      true,
	})
}

// Create uses the global factory to create a new sandbox.
//
// Takes purpose (string) which describes why the sandbox is being created.
// Takes path (string) which specifies the filesystem path for the sandbox.
// Takes mode (Mode) which controls the sandbox behaviour.
//
// Returns Sandbox which is the created sandbox instance.
// Returns error when the global factory has not been initialised or creation
// fails.
func Create(purpose string, path string, mode Mode) (Sandbox, error) {
	return getGlobalFactory().Create(purpose, path, mode)
}

// newDefaultFactory creates a factory with default settings.
// Sandboxing is enabled and the current working directory is detected
// automatically.
//
// Returns Factory which is configured with default sandboxing settings.
// Returns error when the working directory cannot be detected.
func newDefaultFactory() (Factory, error) {
	return NewFactory(FactoryConfig{
		CWD:          "",
		AllowedPaths: nil,
		Enabled:      true,
	})
}

// isWithinOrEqual checks if a path is equal to or inside a parent directory.
//
// Takes parent (string) which is the directory path to check against.
// Takes path (string) which is the path to test.
//
// Returns bool which is true if path equals parent or is inside it.
func isWithinOrEqual(parent, path string) bool {
	parent = filepath.Clean(parent)
	path = filepath.Clean(path)

	if path == parent {
		return true
	}

	if strings.HasPrefix(path, parent+string(filepath.Separator)) {
		return true
	}

	return false
}

// initialiseGlobalFactory initialises the global factory with the
// given configuration.
// Call this once during application startup.
//
// When called after the factory is already initialised, returns nil immediately.
//
// Takes config (FactoryConfig) which specifies the factory settings.
//
// Returns error when NewFactory fails to create the factory.
//
// Safe for concurrent use. Uses a mutex to protect the global factory state.
func initialiseGlobalFactory(config FactoryConfig) error {
	globalFactoryMu.Lock()
	defer globalFactoryMu.Unlock()

	if globalFactoryInit {
		return nil
	}

	factory, err := NewFactory(config)
	if err != nil {
		return err
	}

	globalFactory = factory
	globalFactoryInit = true
	return nil
}

// getGlobalFactory returns the global factory instance.
//
// Returns Factory which is the initialised global factory.
//
// Panics if initialiseGlobalFactory has not been called.
//
// Safe for concurrent use by multiple goroutines.
func getGlobalFactory() Factory {
	globalFactoryMu.RLock()
	defer globalFactoryMu.RUnlock()

	if globalFactory == nil {
		panic("safedisk: global factory not initialised - call initialiseGlobalFactory first")
	}
	return globalFactory
}

// resetGlobalFactory resets the global factory for testing.
// This must only be called from tests.
//
// Safe for concurrent use by multiple goroutines.
func resetGlobalFactory() {
	globalFactoryMu.Lock()
	globalFactory = nil
	globalFactoryInit = false
	globalFactoryMu.Unlock()
}

// mustCreate creates a sandbox using the global factory.
//
// Takes purpose (string) which describes the sandbox's intended use.
// Takes path (string) which specifies the root directory for the sandbox.
// Takes mode (Mode) which determines the sandbox's permission settings.
//
// Returns Sandbox which is the created sandbox ready for use.
func mustCreate(purpose string, path string, mode Mode) Sandbox {
	return getGlobalFactory().MustCreate(purpose, path, mode)
}
