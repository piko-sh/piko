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

package querier_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGoTypeOverride(t *testing.T) {
	t.Parallel()

	t.Run("built-in type has no package", func(t *testing.T) {
		t.Parallel()
		goType, problem := parseGoTypeOverride("string")
		require.NotNil(t, goType)
		assert.Empty(t, goType.Package)
		assert.Equal(t, "string", goType.Name)
		assert.Empty(t, problem)
	})

	t.Run("qualified type splits at the last dot", func(t *testing.T) {
		t.Parallel()
		goType, problem := parseGoTypeOverride("github.com/google/uuid.UUID")
		require.NotNil(t, goType)
		assert.Equal(t, "github.com/google/uuid", goType.Package)
		assert.Equal(t, "UUID", goType.Name)
		assert.Empty(t, problem, "A fully qualified import path is fine")
	})

	t.Run("standard library type is accepted", func(t *testing.T) {
		t.Parallel()
		goType, problem := parseGoTypeOverride("time.Time")
		require.NotNil(t, goType)
		assert.Equal(t, "time", goType.Package)
		assert.Equal(t, "Time", goType.Name)
		assert.Empty(t, problem, "Legitimate standard library types must not be flagged")
	})

	t.Run("bare uuid is reported because Go 1.27 made it resolve to the standard library", func(t *testing.T) {
		t.Parallel()
		goType, problem := parseGoTypeOverride("uuid.UUID")
		require.NotNil(t, goType)
		assert.Equal(t, "uuid", goType.Package)
		assert.Contains(t, problem, "github.com/google/uuid")
		assert.Contains(t, problem, "sql.Scanner")
	})

	t.Run("malformed values are rejected", func(t *testing.T) {
		t.Parallel()
		for _, raw := range []string{".UUID", "uuid."} {
			goType, problem := parseGoTypeOverride(raw)
			assert.Nil(t, goType, "%q should not parse", raw)
			assert.Empty(t, problem)
		}
	})
}
