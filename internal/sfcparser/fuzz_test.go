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

//go:build fuzz

package sfcparser_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/sfcparser"
)

func FuzzParse(f *testing.F) {
	fixtures, err := filepath.Glob(filepath.Join("testdata", "*.pk"))
	require.NoError(f, err)
	for _, fixture := range fixtures {
		data, err := os.ReadFile(fixture)
		require.NoError(f, err)
		f.Add(data)
	}
	f.Add([]byte(`<template><div>{{ a }}</div></template><script type="application/x-go">package main</script>`))
	f.Add([]byte(`<template><template><p>nested</p></template></template><style>a{}</style>`))
	f.Add([]byte(`<script>let s = "<script>"; let e = "</script";</script><i18n>{"a":1}</i18n>`))
	f.Add([]byte(`<piko:timeline media="(min-width: 1px)">x</piko:timeline><template/>`))
	f.Add([]byte("<template>\n\xff\x80</template><script"))
	f.Add([]byte("<script>"))
	f.Add([]byte(""))

	f.Fuzz(func(t *testing.T, input []byte) {
		result, err := sfcparser.Parse(input)
		if err != nil {
			return
		}
		require.NotNil(t, result)

		assert.True(t, bytes.Contains(input, []byte(result.Template)), "template content must come from the input")
		for _, script := range result.Scripts {
			assert.True(t, bytes.Contains(input, []byte(script.Content)), "script content must come from the input")
			assertLocationInInput(t, input, script.Location)
		}
		for _, style := range result.Styles {
			assert.True(t, bytes.Contains(input, []byte(style.Content)), "style content must come from the input")
			assertLocationInInput(t, input, style.Location)
		}
		for _, block := range result.I18nBlocks {
			assertLocationInInput(t, input, block.Location)
		}
		for _, timeline := range result.Timelines {
			assertLocationInInput(t, input, timeline.Location)
		}
	})
}

func assertLocationInInput(t *testing.T, input []byte, location sfcparser.Location) {
	t.Helper()
	assert.GreaterOrEqual(t, location.Line, 1, "line must be 1-based")
	assert.GreaterOrEqual(t, location.Column, 1, "column must be 1-based")
	assert.LessOrEqual(t, location.Line, bytes.Count(input, []byte{'\n'})+1, "line must lie within the input")
	assert.LessOrEqual(t, location.Column, len(input)+1, "column must lie within the input")
}
