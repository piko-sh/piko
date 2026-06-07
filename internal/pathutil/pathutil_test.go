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

package pathutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRelWithin(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		dir        string
		target     string
		wantRel    string
		wantWithin bool
	}{
		{
			name:       "file directly inside dir",
			dir:        "/work/polite",
			target:     "/work/polite/partials/card.pk",
			wantRel:    "partials/card.pk",
			wantWithin: true,
		},
		{
			name:       "dir itself",
			dir:        "/work/polite",
			target:     "/work/polite",
			wantRel:    ".",
			wantWithin: true,
		},
		{

			name:       "sibling sharing a name prefix is external",
			dir:        "/work/polite",
			target:     "/work/politeperch/partials/seo/seo.pk",
			wantRel:    "",
			wantWithin: false,
		},
		{
			name:       "unrelated sibling is external",
			dir:        "/work/polite",
			target:     "/work/other/card.pk",
			wantRel:    "",
			wantWithin: false,
		},
		{

			name:       "exact parent is external",
			dir:        "/work/polite/partials",
			target:     "/work/polite",
			wantRel:    "",
			wantWithin: false,
		},
		{
			name:       "trailing separator on dir does not misclassify",
			dir:        "/work/polite/",
			target:     "/work/polite/partials/card.pk",
			wantRel:    "partials/card.pk",
			wantWithin: true,
		},
		{
			name:       "empty dir is never a container",
			dir:        "",
			target:     "/work/polite/card.pk",
			wantRel:    "",
			wantWithin: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rel, within := RelWithin(tc.dir, tc.target)
			assert.Equal(t, tc.wantWithin, within)
			assert.Equal(t, tc.wantRel, rel)
		})
	}
}

func TestContains(t *testing.T) {
	t.Parallel()

	assert.True(t, Contains("/work/polite", "/work/polite/x.pk"))
	assert.False(t, Contains("/work/polite", "/work/politeperch/x.pk"))
	assert.False(t, Contains("", "/work/polite/x.pk"))
}
