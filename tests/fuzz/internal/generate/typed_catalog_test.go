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

package generate

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestTypedCatalogueShapes(t *testing.T) {
	for _, index := range []uint64{0, 1} {
		c := New("typed-catalog", 7, index)
		width := 8
		if index == 1 {
			width = 4
		}
		require.Len(t, strings.Split(strings.TrimSpace(c.Files["pkg/shelf/items.txt"]), "\n"), width)
		regenerated := New("typed-catalog", 7, index)
		require.Equal(t, c.Hash(), regenerated.Hash(), "non-deterministic sources")
		for branch := range width {
			path := fmt.Sprintf("partials/catalog/branch%d/panel.pk", branch)
			require.Contains(t, c.Files[path], catalogueModel)
			require.Contains(t, c.Files[path], "shared")
		}
		photos := 0
		var actors []string
		for _, probe := range c.Probes {
			if probe.Path != "/catalog" {
				continue
			}
			actors = append(actors, probe.Actor)
			if len(actors) != 1 {
				continue
			}
			for _, element := range probe.Semantic.Elements {
				if element.Tag == "img" {
					photos++
					path := strings.TrimPrefix(element.Attributes["src"], "/_piko/assets/"+casefile.Module+"/")
					require.NotEmpty(t, c.Files[path], "linked image %s missing", path)
				}
			}
		}
		assert.Equal(t, "A/B/A", strings.Join(actors, "/"))
		assert.Equal(t, width, photos)
		assert.NotNil(t, c.Expected.Semantic)
		assert.NotEmpty(t, c.Files["pages/!404.pk"])
	}
}

func TestCatalogueElementPairsAttributes(t *testing.T) {
	element := catalogueElement("buy", "a", "Buy", "href", "/item", "data-sku")
	assert.Equal(t, map[string]string{"href": "/item"}, element.Attributes)
	assert.Nil(t, catalogueElement("plain", "p", "Text").Attributes)
}
