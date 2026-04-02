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

package annotator_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnnotatorPathsConfig_ZeroValue(t *testing.T) {
	t.Parallel()

	t.Run("zero value has all empty strings", func(t *testing.T) {
		t.Parallel()

		config := AnnotatorPathsConfig{
			PagesSourceDir:    "",
			EmailsSourceDir:   "",
			PartialsSourceDir: "",
			E2ESourceDir:      "",
			AssetsSourceDir:   "",
			PartialServePath:  "",
			ArtefactServePath: "",
		}

		assert.Empty(t, config.PagesSourceDir)
		assert.Empty(t, config.EmailsSourceDir)
		assert.Empty(t, config.PartialsSourceDir)
		assert.Empty(t, config.E2ESourceDir)
		assert.Empty(t, config.AssetsSourceDir)
		assert.Empty(t, config.PartialServePath)
		assert.Empty(t, config.ArtefactServePath)
	})

	t.Run("populated config retains values", func(t *testing.T) {
		t.Parallel()

		config := AnnotatorPathsConfig{
			PagesSourceDir:    "/pages",
			EmailsSourceDir:   "/emails",
			PartialsSourceDir: "/partials",
			E2ESourceDir:      "/e2e",
			AssetsSourceDir:   "/assets",
			PartialServePath:  "/partial",
			ArtefactServePath: "/artefact",
		}

		assert.Equal(t, "/pages", config.PagesSourceDir)
		assert.Equal(t, "/emails", config.EmailsSourceDir)
		assert.Equal(t, "/partials", config.PartialsSourceDir)
		assert.Equal(t, "/e2e", config.E2ESourceDir)
		assert.Equal(t, "/assets", config.AssetsSourceDir)
		assert.Equal(t, "/partial", config.PartialServePath)
		assert.Equal(t, "/artefact", config.ArtefactServePath)
	})
}
