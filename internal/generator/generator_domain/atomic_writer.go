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

package generator_domain

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// errMessageFailedToRemoveTempFile is the warning message logged when a temporary
	// file cannot be deleted during cleanup.
	errMessageFailedToRemoveTempFile = "Failed to remove temporary file during cleanup"
)

// AtomicWriteFile writes data to a file atomically by first writing to a
// temporary file in the same directory and then renaming it to the final
// destination. This prevents partial writes and file corruption if the
// process is interrupted.
//
// The filename should be relative to the sandbox root. All operations are
// performed within the sandbox boundary.
//
// Takes ctx (context.Context) which carries logging context.
// Takes sandbox (safedisk.Sandbox) which provides the sandboxed filesystem.
// Takes filename (string) which specifies the target file path.
// Takes data ([]byte) which contains the content to write.
// Takes perm (fs.FileMode) which sets the file permissions.
//
// Returns error when the directory cannot be created, the temporary file
// cannot be written, or the atomic rename fails.
func AtomicWriteFile(ctx context.Context, sandbox safedisk.Sandbox, filename string, data []byte, perm fs.FileMode) error {
	directory := filepath.Dir(filename)
	if err := sandbox.MkdirAll(directory, DirectoryPermission); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", directory, err)
	}

	tempFile, err := sandbox.CreateTemp(directory, filepath.Base(filename)+".tmp*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempFileName := tempFile.Name()

	if _, err = tempFile.Write(data); err != nil {
		cleanupTempFile(ctx, sandbox, tempFile)
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	if err := tempFile.Sync(); err != nil {
		cleanupTempFile(ctx, sandbox, tempFile)
		return fmt.Errorf("failed to sync temporary file to disk: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		removeTempFile(ctx, sandbox, tempFileName)
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	if err := sandbox.Chmod(tempFileName, perm); err != nil {
		removeTempFile(ctx, sandbox, tempFileName)
		return fmt.Errorf("failed to set permissions on temporary file: %w", err)
	}

	if err := sandbox.Rename(tempFileName, filename); err != nil {
		removeTempFile(ctx, sandbox, tempFileName)
		return fmt.Errorf("failed to atomically rename temporary file to final destination: %w", err)
	}

	return nil
}

// cleanupTempFile closes and removes a temporary file, logging any errors.
// This is a best-effort cleanup function used during error handling.
//
// Takes ctx (context.Context) which carries logging context.
// Takes sandbox (safedisk.Sandbox) which provides file system operations.
// Takes tempFile (safedisk.FileHandle) which is the temporary file to clean up.
func cleanupTempFile(ctx context.Context, sandbox safedisk.Sandbox, tempFile safedisk.FileHandle) {
	if tempFile == nil {
		return
	}

	_, l := logger_domain.From(ctx, log)
	if closeErr := tempFile.Close(); closeErr != nil {
		l.Warn("Failed to close temporary file during cleanup", logger_domain.Error(closeErr))
	}
	if removeErr := sandbox.Remove(tempFile.Name()); removeErr != nil {
		l.Warn(errMessageFailedToRemoveTempFile, logger_domain.Error(removeErr))
	}
}

// removeTempFile removes a temporary file and logs a warning on failure.
//
// Takes ctx (context.Context) which carries logging context.
// Takes sandbox (safedisk.Sandbox) which provides safe file system operations.
// Takes tempFileName (string) which is the path of the temporary file to remove.
func removeTempFile(ctx context.Context, sandbox safedisk.Sandbox, tempFileName string) {
	if removeErr := sandbox.Remove(tempFileName); removeErr != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn(errMessageFailedToRemoveTempFile, logger_domain.Error(removeErr))
	}
}
