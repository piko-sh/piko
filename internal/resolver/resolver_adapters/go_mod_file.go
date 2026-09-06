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

package resolver_adapters

import (
	"context"
	"fmt"
	"path/filepath"

	"golang.org/x/mod/modfile"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// maxGoModFileBytes caps how much of a go.mod file is read; larger files are rejected
	// with safedisk.ErrFileExceedsLimit.
	maxGoModFileBytes = 16 << 20
)

// ReadModuleName reads a go.mod file through a narrow read-only sandbox and returns the
// module path declared by its module directive.
//
// Takes goModPath (string) which is the absolute path to the go.mod file.
// Takes factory (safedisk.Factory) which creates the sandbox when it allows the module
// directory; when nil or when the directory lies outside its allowed paths, a
// kernel-backed sandbox rooted at the module directory is used.
//
// Returns string which is the module path found in the file.
// Returns error when the file cannot be read, exceeds the size cap, or does not declare a
// module path.
func ReadModuleName(ctx context.Context, goModPath string, factory safedisk.Factory) (string, error) {
	data, err := readGoModFile(ctx, goModPath, factory)
	if err != nil {
		return "", err
	}

	moduleName := modfile.ModulePath(data)
	if moduleName == "" {
		return "", fmt.Errorf("no 'module' line found in %s", goModPath)
	}
	return moduleName, nil
}

// readGoModFile reads a go.mod file through a read-only sandbox rooted at its directory,
// rejecting files larger than maxGoModFileBytes.
//
// Takes goModPath (string) which is the absolute path to the go.mod file.
// Takes factory (safedisk.Factory) which creates the sandbox; see
// openModuleDirectorySandbox.
//
// Returns []byte which holds the file contents.
// Returns error when the sandbox cannot be created or the file cannot be read; an
// oversized file yields an error wrapping safedisk.ErrFileExceedsLimit.
func readGoModFile(ctx context.Context, goModPath string, factory safedisk.Factory) ([]byte, error) {
	sandbox, err := openModuleDirectorySandbox(ctx, filepath.Dir(goModPath), factory)
	if err != nil {
		return nil, fmt.Errorf("creating sandbox for go.mod at %q: %w", goModPath, err)
	}
	defer closeModuleSandbox(ctx, sandbox, goModPath)

	data, _, err := sandbox.ReadFileLimit(filepath.Base(goModPath), maxGoModFileBytes)
	if err != nil {
		return nil, fmt.Errorf("reading go.mod file %q: %w", goModPath, err)
	}
	return data, nil
}

// openModuleDirectorySandbox creates a read-only sandbox rooted at the directory that
// holds go.mod.
//
// The factory is used when it allows the module directory. The go.mod search walks
// upwards from the start directory, so in a monorepo the module directory can lie above
// the factory's allowed paths; a kernel-backed read-only sandbox rooted at that directory
// is used instead, keeping the read confined to it.
//
// Takes moduleDirectory (string) which is the directory containing go.mod.
// Takes factory (safedisk.Factory) which creates the sandbox; when nil, a kernel-backed
// sandbox is created directly.
//
// Returns safedisk.Sandbox which gives read-only access to the module directory.
// Returns error when the sandbox cannot be created.
func openModuleDirectorySandbox(ctx context.Context, moduleDirectory string, factory safedisk.Factory) (safedisk.Sandbox, error) {
	if factory == nil {
		return safedisk.NewSandbox(moduleDirectory, safedisk.ModeReadOnly)
	}
	if factory.IsPathAllowed(moduleDirectory) {
		return factory.Create("go-mod", moduleDirectory, safedisk.ModeReadOnly)
	}
	_, l := logger_domain.From(ctx, log)
	l.Internal("Module directory lies outside the sandbox allowed paths; reading go.mod through a dedicated read-only sandbox",
		logger_domain.String("moduleDirectory", moduleDirectory))
	return safedisk.NewSandbox(moduleDirectory, safedisk.ModeReadOnly)
}

// closeModuleSandbox closes the go.mod sandbox and logs a failure to do so.
//
// Takes sandbox (safedisk.Sandbox) which is the sandbox to close.
// Takes goModPath (string) which is the go.mod path, used for logging.
func closeModuleSandbox(ctx context.Context, sandbox safedisk.Sandbox, goModPath string) {
	if err := sandbox.Close(); err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Failed to close go.mod sandbox",
			logger_domain.String("path", goModPath),
			logger_domain.Error(err))
	}
}
