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

package piko

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestEnsurePikoInternalDir(t *testing.T) {
	t.Parallel()

	t.Run("creates internal directory", func(t *testing.T) {
		t.Parallel()
		baseDir := t.TempDir()

		err := ensurePikoInternalDir(baseDir, ".piko")

		require.NoError(t, err)
	})

	t.Run("returns error for invalid base directory", func(t *testing.T) {
		t.Parallel()

		err := ensurePikoInternalDir("/nonexistent/path/that/cannot/exist", ".piko")

		assert.Error(t, err)
	})
}

func TestGetErrorContext(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for nil request", func(t *testing.T) {
		t.Parallel()

		result := GetErrorContext(nil)
		assert.Nil(t, result)
	})

	t.Run("returns nil when no error context in request", func(t *testing.T) {
		t.Parallel()

		rd := templater_dto.NewRequestDataBuilder().Build()
		defer rd.Release()

		result := GetErrorContext(rd)
		assert.Nil(t, result)
	})

	t.Run("returns error context when present", func(t *testing.T) {
		t.Parallel()

		epc := daemon_dto.ErrorPageContext{
			StatusCode:   404,
			Message:      "page not found",
			OriginalPath: "/missing",
		}
		ctx := daemon_dto.WithErrorPageContext(t.Context(), epc)
		rd := templater_dto.NewRequestDataBuilder().
			WithContext(ctx).
			Build()
		defer rd.Release()

		result := GetErrorContext(rd)
		require.NotNil(t, result)
		assert.Equal(t, 404, result.StatusCode)
		assert.Equal(t, "page not found", result.Message)
		assert.Equal(t, "/missing", result.OriginalPath)
	})
}
