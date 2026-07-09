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

package daemon_adapters

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/seo/seo_dto"
)

func TestSubstituteRouteParams_EscapesUnsafeCharacters(t *testing.T) {
	t.Parallel()

	const pattern = "/docs/{slug}"
	params := map[string]string{"slug": `a"><script>alert(1)</script>`}

	got := seo_dto.EscapePathSegments(substituteRouteParams(pattern, params))

	assert.NotContains(t, got, "<", "raw '<' must be percent-encoded to %%3C")
	assert.NotContains(t, got, ">", "raw '>' must be percent-encoded to %%3E")
	assert.NotContains(t, got, `"`, "raw '\"' must be percent-encoded to %%22")

	assert.Contains(t, got, "%3C", "'<' should encode to %%3C")
	assert.Contains(t, got, "%3E", "'>' should encode to %%3E")
	assert.Contains(t, got, "%22", "'\"' should encode to %%22")

	assert.True(t, strings.HasPrefix(got, "/docs/"), "static path structure must be preserved")
}
