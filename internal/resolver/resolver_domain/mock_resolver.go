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

package resolver_domain

import (
	"context"
	"sync/atomic"
)

// MockResolver is a test double for ResolverPort that returns zero
// values from nil function fields and tracks call counts atomically.
type MockResolver struct {
	// DetectLocalModuleFunc is the function called by
	// DetectLocalModule.
	DetectLocalModuleFunc func(ctx context.Context) error

	// GetModuleNameFunc is the function called by
	// GetModuleName.
	GetModuleNameFunc func() string

	// GetBaseDirFunc is the function called by
	// GetBaseDir.
	GetBaseDirFunc func() string

	// ResolvePKPathFunc is the function called by
	// ResolvePKPath.
	ResolvePKPathFunc func(ctx context.Context, importPath string, containingFilePath string) (string, error)

	// ResolveCSSPathFunc is the function called by
	// ResolveCSSPath.
	ResolveCSSPathFunc func(ctx context.Context, importPath string, containingDir string) (string, error)

	// ResolveAssetPathFunc is the function called by
	// ResolveAssetPath.
	ResolveAssetPathFunc func(ctx context.Context, importPath string, containingFilePath string) (string, error)

	// ConvertEntryPointPathToManifestKeyFunc is the
	// function called by
	// ConvertEntryPointPathToManifestKey.
	ConvertEntryPointPathToManifestKeyFunc func(entryPointPath string) string

	// GetModuleDirFunc is the function called by
	// GetModuleDir.
	GetModuleDirFunc func(ctx context.Context, modulePath string) (string, error)

	// FindModuleBoundaryFunc is the function called by
	// FindModuleBoundary.
	FindModuleBoundaryFunc func(ctx context.Context, importPath string) (string, string, error)

	// DetectLocalModuleCallCount tracks how many times
	// DetectLocalModule was called.
	DetectLocalModuleCallCount int64

	// GetModuleNameCallCount tracks how many times
	// GetModuleName was called.
	GetModuleNameCallCount int64

	// GetBaseDirCallCount tracks how many times
	// GetBaseDir was called.
	GetBaseDirCallCount int64

	// ResolvePKPathCallCount tracks how many times
	// ResolvePKPath was called.
	ResolvePKPathCallCount int64

	// ResolveCSSPathCallCount tracks how many times
	// ResolveCSSPath was called.
	ResolveCSSPathCallCount int64

	// ResolveAssetPathCallCount tracks how many times
	// ResolveAssetPath was called.
	ResolveAssetPathCallCount int64

	// ConvertEntryPointPathToManifestKeyCallCount
	// tracks how many times
	// ConvertEntryPointPathToManifestKey was called.
	ConvertEntryPointPathToManifestKeyCallCount int64

	// GetModuleDirCallCount tracks how many times
	// GetModuleDir was called.
	GetModuleDirCallCount int64

	// FindModuleBoundaryCallCount tracks how many times
	// FindModuleBoundary was called.
	FindModuleBoundaryCallCount int64
}

var _ ResolverPort = (*MockResolver)(nil)

// DetectLocalModule delegates to DetectLocalModuleFunc if set.
//
// Returns nil if DetectLocalModuleFunc is nil.
func (m *MockResolver) DetectLocalModule(ctx context.Context) error {
	atomic.AddInt64(&m.DetectLocalModuleCallCount, 1)
	if m.DetectLocalModuleFunc != nil {
		return m.DetectLocalModuleFunc(ctx)
	}
	return nil
}

// GetModuleName delegates to GetModuleNameFunc if set.
//
// Returns "" if GetModuleNameFunc is nil.
func (m *MockResolver) GetModuleName() string {
	atomic.AddInt64(&m.GetModuleNameCallCount, 1)
	if m.GetModuleNameFunc != nil {
		return m.GetModuleNameFunc()
	}
	return ""
}

// GetBaseDir delegates to GetBaseDirFunc if set.
//
// Returns "" if GetBaseDirFunc is nil.
func (m *MockResolver) GetBaseDir() string {
	atomic.AddInt64(&m.GetBaseDirCallCount, 1)
	if m.GetBaseDirFunc != nil {
		return m.GetBaseDirFunc()
	}
	return ""
}

// ResolvePKPath delegates to ResolvePKPathFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes importPath (string) which is the import path to resolve.
// Takes containingFilePath (string) which is the path
// of the file containing the import.
//
// Returns ("", nil) if ResolvePKPathFunc is nil.
func (m *MockResolver) ResolvePKPath(ctx context.Context, importPath string, containingFilePath string) (string, error) {
	atomic.AddInt64(&m.ResolvePKPathCallCount, 1)
	if m.ResolvePKPathFunc != nil {
		return m.ResolvePKPathFunc(ctx, importPath, containingFilePath)
	}
	return "", nil
}

// ResolveCSSPath delegates to ResolveCSSPathFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes importPath (string) which is the CSS import path to resolve.
// Takes containingDir (string) which is the directory
// containing the importing file.
//
// Returns ("", nil) if ResolveCSSPathFunc is nil.
func (m *MockResolver) ResolveCSSPath(ctx context.Context, importPath string, containingDir string) (string, error) {
	atomic.AddInt64(&m.ResolveCSSPathCallCount, 1)
	if m.ResolveCSSPathFunc != nil {
		return m.ResolveCSSPathFunc(ctx, importPath, containingDir)
	}
	return "", nil
}

// ResolveAssetPath delegates to ResolveAssetPathFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes importPath (string) which is the asset import path to resolve.
// Takes containingFilePath (string) which is the path
// of the file containing the import.
//
// Returns ("", nil) if ResolveAssetPathFunc is nil.
func (m *MockResolver) ResolveAssetPath(ctx context.Context, importPath string, containingFilePath string) (string, error) {
	atomic.AddInt64(&m.ResolveAssetPathCallCount, 1)
	if m.ResolveAssetPathFunc != nil {
		return m.ResolveAssetPathFunc(ctx, importPath, containingFilePath)
	}
	return "", nil
}

// ConvertEntryPointPathToManifestKey delegates to
// ConvertEntryPointPathToManifestKeyFunc if set.
//
// Takes entryPointPath (string) which is the entry point path to convert.
//
// Returns "" if ConvertEntryPointPathToManifestKeyFunc is nil.
func (m *MockResolver) ConvertEntryPointPathToManifestKey(entryPointPath string) string {
	atomic.AddInt64(&m.ConvertEntryPointPathToManifestKeyCallCount, 1)
	if m.ConvertEntryPointPathToManifestKeyFunc != nil {
		return m.ConvertEntryPointPathToManifestKeyFunc(entryPointPath)
	}
	return ""
}

// GetModuleDir delegates to GetModuleDirFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes modulePath (string) which is the module path to resolve to a directory.
//
// Returns ("", nil) if GetModuleDirFunc is nil.
func (m *MockResolver) GetModuleDir(ctx context.Context, modulePath string) (string, error) {
	atomic.AddInt64(&m.GetModuleDirCallCount, 1)
	if m.GetModuleDirFunc != nil {
		return m.GetModuleDirFunc(ctx, modulePath)
	}
	return "", nil
}

// FindModuleBoundary delegates to FindModuleBoundaryFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes importPath (string) which is the import path to
// find the module boundary for.
//
// Returns ("", "", nil) if FindModuleBoundaryFunc is nil.
func (m *MockResolver) FindModuleBoundary(ctx context.Context, importPath string) (modulePath string, moduleDir string, err error) {
	atomic.AddInt64(&m.FindModuleBoundaryCallCount, 1)
	if m.FindModuleBoundaryFunc != nil {
		return m.FindModuleBoundaryFunc(ctx, importPath)
	}
	return "", "", nil
}
