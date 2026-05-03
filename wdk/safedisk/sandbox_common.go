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
	"context"
	"fmt"
	"io"
	"io/fs"

	logger "piko.sh/piko/internal/logger/logger_domain"
)

// readFileViaOpen reads a file's entire contents using the provided open
// function, up to DefaultReadFileMaxBytes.
//
// The cap protects callers from unbounded allocation when a file has grown
// unexpectedly. Callers that need a tighter or looser cap should use
// readFileLimitViaOpen.
//
// Takes opener (func(string) (FileHandle, error)) which opens the
// file for reading.
// Takes name (string) which specifies the path to the file to read.
//
// Returns []byte which contains the complete file contents.
// Returns error when the file cannot be opened or read, or wraps
// ErrFileExceedsLimit when the file exceeds DefaultReadFileMaxBytes.
func readFileViaOpen(opener func(string) (FileHandle, error), name string) ([]byte, error) {
	f, err := opener(name)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			_, l := logger.From(context.Background(), log)
			l.Warn("Failed to close file after read", logger.Error(closeErr), logger.String("file", name))
		}
	}()

	data, err := io.ReadAll(io.LimitReader(f, DefaultReadFileMaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("reading file contents %q: %w", name, err)
	}
	if int64(len(data)) > DefaultReadFileMaxBytes {
		return nil, fmt.Errorf("%w: %q exceeds default read cap of %d bytes", ErrFileExceedsLimit, name, DefaultReadFileMaxBytes)
	}
	return data, nil
}

// readFileLimitViaOpen reads up to maxBytes from a file using the provided
// open and stat functions. The size cap is enforced before allocation by
// statting first; the read itself is wrapped in an io.LimitReader as belt
// and braces against a file growing between stat and read.
//
// Takes opener (func(string) (FileHandle, error)) which opens the file for
// reading.
// Takes statter (func(string) (fs.FileInfo, error)) which returns the
// file's metadata.
// Takes name (string) which is the relative path within the sandbox.
// Takes maxBytes (int64) which caps the byte count read into memory; must
// be positive.
//
// Returns []byte containing the file content (up to maxBytes).
// Returns int64 reporting the stat-observed size at the moment of stat.
// Returns error wrapping ErrFileExceedsLimit when the file is larger than
// maxBytes, ErrInvalidLimit when maxBytes is non-positive, or any
// underlying stat / open / read error.
func readFileLimitViaOpen(
	opener func(string) (FileHandle, error),
	statter func(string) (fs.FileInfo, error),
	name string,
	maxBytes int64,
) ([]byte, int64, error) {
	if maxBytes <= 0 {
		return nil, 0, ErrInvalidLimit
	}

	info, err := statter(name)
	if err != nil {
		return nil, 0, err
	}
	size := info.Size()
	if size > maxBytes {
		return nil, size, fmt.Errorf("%w: %q is %d bytes, limit %d", ErrFileExceedsLimit, name, size, maxBytes)
	}

	f, err := opener(name)
	if err != nil {
		return nil, size, err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			_, l := logger.From(context.Background(), log)
			l.Warn("Failed to close file after limited read",
				logger.Error(closeErr),
				logger.String("file", name),
			)
		}
	}()

	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, size, fmt.Errorf("reading file contents %q: %w", name, err)
	}
	if int64(len(data)) > maxBytes {
		return nil, size, fmt.Errorf("%w: %q grew past %d bytes during read", ErrFileExceedsLimit, name, maxBytes)
	}
	return data, size, nil
}
