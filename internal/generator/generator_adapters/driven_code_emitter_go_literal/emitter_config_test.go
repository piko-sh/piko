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

package driven_code_emitter_go_literal

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveBaseDirectory(t *testing.T) {
	t.Parallel()

	t.Run("makes the directory absolute", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "/project", resolveBaseDirectory(t.Context(), "/project", filepath.Abs))
	})

	t.Run("keeps the directory when it cannot be made absolute", func(t *testing.T) {
		t.Parallel()

		failing := func(string) (string, error) { return "", errors.New("no working directory") }
		assert.Equal(t, "project", resolveBaseDirectory(t.Context(), "project", failing))
	})
}
