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

package casefile

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

var (
	// modelFiles embeds every Go source file of the casefile model so the generator profile
	// changes whenever the model changes.
	//
	//go:embed *.go
	modelFiles embed.FS
)

// ModelSource returns the embedded casefile model used in generator identity.
//
// The non-test Go files are joined in lexical order, so editing any model file changes
// the result.
//
// Returns string which holds model source bytes contributing to the generator profile.
// Returns error which reports an unreadable embedded model file, or nil on success.
func ModelSource() (string, error) {
	entries, err := fs.ReadDir(modelFiles, ".")
	if err != nil {
		return "", fmt.Errorf("reading embedded casefile model: %w", err)
	}
	var parts []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := fs.ReadFile(modelFiles, name)
		if err != nil {
			return "", fmt.Errorf("reading embedded casefile model %s: %w", name, err)
		}
		parts = append(parts, name+"\x00"+string(data))
	}
	return strings.Join(parts, "\x00"), nil
}
