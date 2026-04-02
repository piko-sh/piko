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

package typegen_domain

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// filePermissions is the permission mode for created type definition files.
	// Owner can read and write, group and others can only read.
	filePermissions = 0o640

	// dirPermissions is the permission mode for created directories.
	// Owner full access, group/other read and execute.
	dirPermissions = 0o750
)

// TypeDefinitionService manages TypeScript type definitions for the Piko
// frontend framework.
//
// The service copies embedded type definitions to a project's dist/ts/
// directory for IDE integration during development.
type TypeDefinitionService struct {
	// pikoTypes holds the embedded Piko framework type definitions.
	pikoTypes embed.FS

	// actionStubTypes holds the embedded stub action type definitions.
	actionStubTypes string
}

// EnsureOptions controls how type definitions are written.
type EnsureOptions struct {
	// OnlyIfNotExists skips writing files that already exist. The LSP uses this
	// to avoid overwriting files written by the dev server.
	OnlyIfNotExists bool
}

// NewTypeDefinitionService creates a new TypeDefinitionService with the
// provided type definitions.
//
// The pikoTypes parameter should be the embedded filesystem containing
// the built type definitions from typegen_frontend.
// The actionStubTypes parameter should contain the embedded content of the
// piko-actions-stub.d.ts file.
//
// Takes pikoTypes (embed.FS) which provides the embedded filesystem containing
// the built type definitions from typegen_frontend.
// Takes actionStubTypes (string) which provides the embedded content of the
// piko-actions-stub.d.ts file.
//
// Returns *TypeDefinitionService which is ready to provide type definitions.
func NewTypeDefinitionService(pikoTypes embed.FS, actionStubTypes string) *TypeDefinitionService {
	return &TypeDefinitionService{
		pikoTypes:       pikoTypes,
		actionStubTypes: actionStubTypes,
	}
}

// EnsureTypeDefinitions copies all embedded type definitions to the specified
// directory.
//
// This method:
//   - Creates the directory structure if it doesn't exist
//   - Copies all Piko framework types (piko-ide.d.ts and subdirectories)
//   - Writes the action stub types to piko-actions.d.ts
//
// This is called during daemon startup in development mode to provide
// IDE integration for .pk file script blocks.
//
// Takes destDir (string) which is the target directory for type definitions.
//
// Returns error when the directory cannot be created or files cannot be
// written.
func (s *TypeDefinitionService) EnsureTypeDefinitions(ctx context.Context, destDir string) error {
	return s.EnsureTypeDefinitionsWithOptions(ctx, destDir, EnsureOptions{})
}

// EnsureTypeDefinitionsWithOptions copies all embedded type definitions to the
// specified directory with the given options.
//
// When OnlyIfNotExists is true, existing files are preserved. This is used by
// the LSP to provide a fallback when the dev server hasn't run yet.
//
// All file writes are atomic using safedisk to prevent IDEs from reading
// partially written files.
//
// Takes destDir (string) which is the target directory for type definitions.
// Takes opts (EnsureOptions) which controls the write behaviour.
//
// Returns error when the directory cannot be created or files cannot be
// written.
func (s *TypeDefinitionService) EnsureTypeDefinitionsWithOptions(ctx context.Context, destDir string, opts EnsureOptions) error {
	factory, err := safedisk.NewFactory(safedisk.FactoryConfig{
		CWD:          destDir,
		AllowedPaths: []string{destDir},
		Enabled:      true,
	})
	if err != nil {
		typeDefsWriteErrors.Add(ctx, 1)
		return fmt.Errorf("creating safedisk factory: %w", err)
	}

	sandbox, err := factory.Create("typegen", destDir, safedisk.ModeReadWrite)
	if err != nil {
		typeDefsWriteErrors.Add(ctx, 1)
		return fmt.Errorf("creating safedisk sandbox: %w", err)
	}
	defer func() { _ = sandbox.Close() }()

	if err := s.copyEmbeddedTypes(ctx, sandbox, opts); err != nil {
		return fmt.Errorf("copying embedded type definitions: %w", err)
	}

	if err := s.writeActionStub(ctx, sandbox, opts); err != nil {
		return fmt.Errorf("writing action stub types: %w", err)
	}

	return nil
}

// copyEmbeddedTypes walks the embedded filesystem and copies all type
// definition files to the sandbox directory.
//
// Takes sandbox (safedisk.Sandbox) which is the target directory for the files.
// Takes opts (EnsureOptions) which controls the copy behaviour.
//
// Returns error when walking the filesystem fails or a file cannot be copied.
func (s *TypeDefinitionService) copyEmbeddedTypes(ctx context.Context, sandbox safedisk.Sandbox, opts EnsureOptions) error {
	return fs.WalkDir(s.pikoTypes, "built", func(srcPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel("built", srcPath)
		if err != nil {
			return fmt.Errorf("computing relative path for %q: %w", srcPath, err)
		}

		if d.IsDir() {
			return sandbox.MkdirAll(relPath, dirPermissions)
		}

		return s.copyEmbeddedFile(ctx, sandbox, srcPath, relPath, opts)
	})
}

// copyEmbeddedFile copies a single embedded file to the sandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides safe file system access.
// Takes srcPath (string) which is the path within the embedded file system.
// Takes relPath (string) which is the destination path relative to sandbox.
// Takes opts (EnsureOptions) which controls write behaviour.
//
// Returns error when reading the embedded file fails or writing to sandbox
// fails.
func (s *TypeDefinitionService) copyEmbeddedFile(ctx context.Context, sandbox safedisk.Sandbox, srcPath, relPath string, opts EnsureOptions) error {
	if opts.OnlyIfNotExists {
		exists, err := fileExists(sandbox, relPath)
		if err != nil {
			return fmt.Errorf("checking file %q: %w", relPath, err)
		}
		if exists {
			return nil
		}
	}

	content, err := s.pikoTypes.ReadFile(srcPath)
	if err != nil {
		typeDefsWriteErrors.Add(ctx, 1)
		return fmt.Errorf("reading embedded file %q: %w", srcPath, err)
	}

	if err := sandbox.WriteFileAtomic(relPath, content, filePermissions); err != nil {
		typeDefsWriteErrors.Add(ctx, 1)
		return fmt.Errorf("writing file %q: %w", relPath, err)
	}

	return nil
}

// writeActionStub writes the action stub type definitions to the sandbox
// directory.
//
// Takes sandbox (safedisk.Sandbox) which provides safe file system access.
// Takes opts (EnsureOptions) which controls write behaviour.
//
// Returns error when the file cannot be written.
func (s *TypeDefinitionService) writeActionStub(ctx context.Context, sandbox safedisk.Sandbox, opts EnsureOptions) error {
	ctx, l := logger_domain.From(ctx, log)
	const fileName = "piko-actions.d.ts"

	if opts.OnlyIfNotExists {
		exists, err := fileExists(sandbox, fileName)
		if err != nil {
			return fmt.Errorf("checking file %q: %w", fileName, err)
		}
		if exists {
			return nil
		}
	}

	if err := sandbox.WriteFileAtomic(fileName, []byte(s.actionStubTypes), filePermissions); err != nil {
		typeDefsWriteErrors.Add(ctx, 1)
		return fmt.Errorf("writing action stub types: %w", err)
	}

	typeDefsWritten.Add(ctx, 1)
	l.Internal("Wrote TypeScript type definitions to project",
		logger_domain.String("dest_dir", sandbox.Root()),
	)

	return nil
}

// fileExists checks if a file exists in the sandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides safe file system access.
// Takes path (string) which specifies the file path to check.
//
// Returns bool which is true if the file exists, false otherwise.
// Returns error when the file status cannot be determined.
func fileExists(sandbox safedisk.Sandbox, path string) (bool, error) {
	_, err := sandbox.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("checking file existence %q: %w", path, err)
}
