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

package db_engine_postgres

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthDiagnosticBuilders(t *testing.T) {
	t.Parallel()

	t.Run("failed probe is unhealthy and names the error", func(t *testing.T) {
		t.Parallel()

		diagnostics := failedProbeDiagnostic("database_size", errors.New("connection refused"))

		require.Len(t, diagnostics, 1)
		assert.Equal(t, "database_size", diagnostics[0].Name)
		assert.Equal(t, healthStateUnhealthy, diagnostics[0].State)
		assert.Equal(t, "query failed: connection refused", diagnostics[0].Message)
		assert.Empty(t, diagnostics[0].Value)
	})

	t.Run("probe value carries no state", func(t *testing.T) {
		t.Parallel()

		diagnostics := probeValueDiagnostic("active_connections", "3")

		require.Len(t, diagnostics, 1)
		assert.Equal(t, "active_connections", diagnostics[0].Name)
		assert.Equal(t, "3", diagnostics[0].Value)
		assert.Empty(t, diagnostics[0].State)
		assert.Empty(t, diagnostics[0].Message)
	})
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		want  string
		bytes int64
	}{
		{name: "bytes", bytes: 512, want: "512 B"},
		{name: "kibibytes", bytes: 1536, want: "1.5 KiB"},
		{name: "mebibytes", bytes: 5 * 1024 * 1024, want: "5.0 MiB"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, formatBytes(testCase.bytes))
		})
	}
}
