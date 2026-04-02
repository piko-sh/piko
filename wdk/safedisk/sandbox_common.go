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

	logger "piko.sh/piko/internal/logger/logger_domain"
)

// readFileViaOpen reads a file's entire contents using the provided
// open function.
//
// Takes opener (func(string) (FileHandle, error)) which opens the
// file for reading.
// Takes name (string) which specifies the path to the file to read.
//
// Returns []byte which contains the complete file contents.
// Returns error when the file cannot be opened or read.
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

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading file contents %q: %w", name, err)
	}
	return data, nil
}
