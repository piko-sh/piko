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

package components_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/components"
)

func TestPiko(t *testing.T) {
	definitions := components.Piko()
	require.Len(t, definitions, 2, "expected piko-card and piko-counter")

	tagNames := make(map[string]bool, len(definitions))
	for _, definition := range definitions {
		tagNames[definition.TagName] = true

		assert.True(t, definition.IsExternal, "IsExternal should be true for %s", definition.TagName)
		assert.NotEmpty(t, definition.SourcePath, "SourcePath should be set for %s", definition.TagName)
		assert.Equal(t, "piko.sh/piko/components", definition.ModulePath)
	}

	assert.True(t, tagNames["piko-card"], "should contain piko-card")
	assert.True(t, tagNames["piko-counter"], "should contain piko-counter")
}

func TestExample(t *testing.T) {
	definitions := components.Example()
	require.Len(t, definitions, 1, "expected example-greeting")

	definition := definitions[0]
	assert.Equal(t, "example-greeting", definition.TagName)
	assert.True(t, definition.IsExternal)
	assert.Contains(t, definition.SourcePath, "example/")
	assert.Equal(t, "piko.sh/piko/components", definition.ModulePath)
}

func TestM3E(t *testing.T) {
	definitions := components.M3E()
	require.Len(t, definitions, 46, "expected 46 M3E components")

	tagNames := make(map[string]bool, len(definitions))
	for _, definition := range definitions {
		tagNames[definition.TagName] = true

		assert.Truef(t, definition.IsExternal, "IsExternal should be true for %s", definition.TagName)
		assert.NotEmptyf(t, definition.SourcePath, "SourcePath should be set for %s", definition.TagName)
		assert.Containsf(t, definition.SourcePath, "m3e/", "SourcePath should be under m3e/ for %s", definition.TagName)
		assert.Equalf(t, "piko.sh/piko/components", definition.ModulePath, "ModulePath wrong for %s", definition.TagName)
		assert.Truef(t, len(definition.TagName) > 4 && definition.TagName[:4] == "m3e-",
			"tag should start with m3e- but got %s", definition.TagName)
	}

	assert.True(t, tagNames["m3e-button"], "should contain m3e-button")
	assert.True(t, tagNames["m3e-checkbox"], "should contain m3e-checkbox")
	assert.True(t, tagNames["m3e-dialog"], "should contain m3e-dialog")
	assert.True(t, tagNames["m3e-navigation-drawer"], "should contain m3e-navigation-drawer")
	assert.True(t, tagNames["m3e-text-field"], "should contain m3e-text-field")
}

func TestM2(t *testing.T) {
	definitions := components.M2()
	require.Len(t, definitions, 5, "expected 5 M2 components")

	tagNames := make(map[string]bool, len(definitions))
	for _, definition := range definitions {
		tagNames[definition.TagName] = true

		assert.Truef(t, definition.IsExternal, "IsExternal should be true for %s", definition.TagName)
		assert.NotEmptyf(t, definition.SourcePath, "SourcePath should be set for %s", definition.TagName)
		assert.Containsf(t, definition.SourcePath, "m2/", "SourcePath should be under m2/ for %s", definition.TagName)
		assert.Equalf(t, "piko.sh/piko/components", definition.ModulePath, "ModulePath wrong for %s", definition.TagName)
		assert.Truef(t, len(definition.TagName) > 3 && definition.TagName[:3] == "m2-",
			"tag should start with m2- but got %s", definition.TagName)
	}

	assert.True(t, tagNames["m2-data-table"], "should contain m2-data-table")
	assert.True(t, tagNames["m2-data-table-cell"], "should contain m2-data-table-cell")
	assert.True(t, tagNames["m2-data-table-header"], "should contain m2-data-table-header")
	assert.True(t, tagNames["m2-data-table-pagination"], "should contain m2-data-table-pagination")
	assert.True(t, tagNames["m2-data-table-row"], "should contain m2-data-table-row")
}

func TestAll(t *testing.T) {
	all := components.All()
	piko := components.Piko()
	example := components.Example()
	m2 := components.M2()
	m3e := components.M3E()

	assert.Len(t, all, len(piko)+len(example)+len(m2)+len(m3e), "All() should combine Piko(), Example(), M2(), and M3E()")

	tagNames := make(map[string]bool, len(all))
	for _, definition := range all {
		tagNames[definition.TagName] = true
	}

	for _, definition := range piko {
		assert.True(t, tagNames[definition.TagName], "All() should include piko component %s", definition.TagName)
	}
	for _, definition := range example {
		assert.True(t, tagNames[definition.TagName], "All() should include example component %s", definition.TagName)
	}
	for _, definition := range m2 {
		assert.True(t, tagNames[definition.TagName], "All() should include m2 component %s", definition.TagName)
	}
	for _, definition := range m3e {
		assert.True(t, tagNames[definition.TagName], "All() should include m3e component %s", definition.TagName)
	}
}
