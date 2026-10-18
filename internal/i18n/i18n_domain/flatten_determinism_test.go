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

package i18n_domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	flattenDeterminismRuns = 32
)

func TestFlattenTranslations_CollidingKeysResolveTheSameWay(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"nav.home":  "Literal home",
		"nav":       map[string]any{"home": "Nested home", "about": "About"},
		"btn.ok":    "Literal ok",
		"btn":       map[string]any{"ok": "Nested ok"},
		"title":     "Title",
		"footer":    map[string]any{"copy": "Copy"},
		"footer.x":  "X",
		"zzz.final": "Final",
	}

	for range flattenDeterminismRuns {
		result := make(map[string]string)

		require.NoError(t, FlattenTranslations(data, "", result))

		require.Equal(t, "Literal home", result["nav.home"], "a dotted key must win over the nested key it collides with on every run")
		require.Equal(t, "Literal ok", result["btn.ok"])
		require.Equal(t, "About", result["nav.about"])
	}
}
