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

package markdown_domain

import (
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/markdown/markdown_dto"
)

const (
	frontmatterDeterminismRuns = 32
)

func TestParseFrontmatter_DuplicateKnownKeysResolveTheSameWay(t *testing.T) {
	t.Parallel()

	rawData := map[string]any{
		"Title":       "Capitalised title",
		"title":       "Lower-case title",
		"TITLE":       "Shouted title",
		"Description": "Capitalised description",
		"description": "Lower-case description",
		"nav":         &markdown_dto.NavigationMetadata{},
		"navigation":  &markdown_dto.NavigationMetadata{},
		"draft":       true,
	}

	for range frontmatterDeterminismRuns {
		fm, err := ParseFrontmatter(rawData)

		require.NoError(t, err)
		require.Equal(t, "Lower-case title", fm.Title, "the same spelling of a duplicated key must win on every run")
		require.Equal(t, "Lower-case description", fm.Description)
		require.Same(t, rawData["navigation"], fm.Navigation)
	}
}
