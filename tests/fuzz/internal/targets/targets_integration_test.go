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

//go:build integration

package targets_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/tests/fuzz/internal/generate"
	"piko.sh/piko/tests/fuzz/internal/targets"
)

func TestGeneratedSubsystemSuites(t *testing.T) {
	if testing.Short() {
		t.Skip("multi-process integration fixtures")
	}
	repository, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	for _, name := range []string{"database", "release", "cache", "vector-search", "assets", "incremental", "wasm", "lsp", "collections", "orchestrator", "persistence", "actions", "documents"} {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, targets.Run(t.Context(), repository, t.TempDir(), new(generate.New(name, 1, 0))))
		})
	}
}
