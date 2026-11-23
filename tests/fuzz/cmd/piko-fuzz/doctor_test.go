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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDoctorToolsForNativeFuzzingNeedOnlyGo(t *testing.T) {
	options := flags{target: "binder", engine: engineGo, browser: browserSample, mode: modeBoth}

	tools := doctorTools(t.TempDir(), &options)

	assert.Equal(t, []string{"go"}, mapKeys(tools))
}

func TestDoctorToolsFollowSelectedTargets(t *testing.T) {
	options := flags{target: "lsp,database", engine: engineSeeded, browser: browserOff, mode: modeCompiled, n: 1}

	tools := doctorTools(t.TempDir(), &options)

	assert.Contains(t, tools, toolPikoplsSource)
	assert.Contains(t, tools, "sqlite_engine_source")
	assert.NotContains(t, tools, "browser")
	assert.NotContains(t, tools, "interpreter_provider_source")
	assert.Nil(t, tools[toolPikoplsSource], "a temporary repository has no pikopls module")
}

func TestBrowserSelectedHonoursPolicyAndBrowserTargets(t *testing.T) {
	every := flags{browser: browserEvery}
	assert.True(t, browserSelected([]string{"pages"}, false, &every))

	off := flags{browser: browserOff}
	assert.False(t, browserSelected([]string{"components"}, true, &off))

	sample := flags{browser: browserSample, n: 0}
	assert.True(t, browserSelected([]string{"components"}, false, &sample))
	assert.False(t, browserSelected([]string{"pages"}, false, &sample), "no cases means no sampled browser")

	sample.n = 20
	assert.True(t, browserSelected([]string{"pages"}, false, &sample), "twenty cases cover a sampled browser decade")
	assert.False(t, browserSelected([]string{"binder"}, false, &sample), "package targets never launch a browser")
}

func TestMissingToolsListsAbsentChecksInOrder(t *testing.T) {
	found := "/usr/bin/go"

	missing := missingTools(map[string]*string{"go": &found, "node": nil, "firefox": nil})

	assert.Equal(t, []string{"firefox", "node"}, missing)
}

func mapKeys(tools map[string]*string) []string {
	keys := make([]string, 0, len(tools))
	for key := range tools {
		keys = append(keys, key)
	}
	return keys
}
