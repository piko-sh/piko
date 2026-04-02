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

package generator_domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_prepareSourceWithImports(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		source              string
		wantFrameworkImport []string
		wantNoSource        bool
	}{
		{
			name:         "empty source returns empty",
			source:       "",
			wantNoSource: true,
		},
		{
			name:   "no pk identifiers still adds action import",
			source: "function hello() { console.log('hello'); }",
		},
		{
			name:   "piko namespace refs not detected as bare identifier",
			source: "const el = piko.refs.myButton;",
		},
		{
			name:   "piko namespace navigate not detected as bare identifier",
			source: "piko.nav.navigate('/home'); const route = piko.nav.current();",
		},
		{
			name:   "word boundary prevents false positives - preferences",
			source: "const preferences = {}; const myRefs = [];",
		},
		{
			name:   "action identifier detected for import",
			source: "action('submit').post();",
		},
		{
			name:   "piko namespace form helpers not detected as bare identifiers",
			source: "const data = piko.form.data(piko.refs.form); piko.form.validate(data, rules);",
		},
		{
			name:   "piko namespace event helpers not detected as bare identifiers",
			source: "piko.event.dispatch('my-event'); piko.event.listen('click', handler);",
		},
		{
			name:   "piko namespace advanced helpers not detected as bare identifiers",
			source: "piko.util.whenVisible(piko.refs.lazy, () => load()); piko.timing.poll(() => fetch(), 5000);",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := prepareSourceWithImports(tt.source)

			if tt.wantNoSource {
				assert.Equal(t, tt.source, result, "empty source should be unchanged")
				return
			}

			assert.Contains(t, result, pkActionsGenPath, "result should contain action import from generated file")
			assert.Contains(t, result, "import { action }", "result should import action")

			if len(tt.wantFrameworkImport) > 0 {
				assert.Contains(t, result, pkFrameworkPath, "result should contain framework import")
				for _, id := range tt.wantFrameworkImport {
					assert.Contains(t, result, id, "import should contain %s", id)
				}
			} else {
				assert.NotContains(t, result, pkFrameworkPath, "result should not contain framework import when no bare identifiers used")
			}

			assert.True(t, strings.HasSuffix(result, tt.source), "original source should be preserved")
		})
	}
}

func TestDetectUsedIdentifiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		want   []string
	}{
		{
			name:   "empty source",
			source: "",
			want:   nil,
		},
		{
			name:   "action detected",
			source: "action('submit').post();",
			want:   []string{"action"},
		},
		{
			name:   "piko namespace identifiers not detected as bare names",
			source: "piko.refs.y; piko.bus.emit('z');",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := detectUsedIdentifiers(tt.source)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestBuildImportStatement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		wantOthers  string
		identifiers []string
		wantAction  bool
	}{
		{
			name:        "empty identifiers still imports action",
			identifiers: nil,
			wantAction:  true,
		},
		{
			name:        "action in identifiers is filtered to generated file",
			identifiers: []string{"action"},
			wantAction:  true,
		},
		{
			name:        "framework identifiers generate framework import",
			identifiers: []string{"_createRefs", "getGlobalPageContext"},
			wantAction:  true,
			wantOthers:  "import { _createRefs, getGlobalPageContext } from \"/_piko/dist/ppframework.core.es.js\";\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := buildImportStatement(tt.identifiers)

			if tt.wantAction {
				assert.Contains(t, result, "import { action } from \"/_piko/assets/pk-js/pk/actions.gen.js\";\n",
					"should import action from generated file")
			}

			if tt.wantOthers != "" {
				assert.Contains(t, result, tt.wantOthers, "should contain framework imports")
			}
		})
	}
}
