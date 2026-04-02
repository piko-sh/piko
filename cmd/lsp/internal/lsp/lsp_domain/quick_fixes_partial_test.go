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

package lsp_domain

import (
	"strings"
	"testing"

	"go.lsp.dev/protocol"
)

func TestGenerateUndefinedPartialAliasFixes(t *testing.T) {
	testCases := []struct {
		data    any
		name    string
		wantLen int
	}{
		{
			name:    "nil data returns empty",
			data:    nil,
			wantLen: 0,
		},
		{
			name:    "wrong type returns empty",
			data:    "not a map",
			wantLen: 0,
		},
		{
			name: "suggestion only returns typo correction",
			data: map[string]any{
				"suggestion": "status_badge",
			},
			wantLen: 1,
		},
		{
			name: "alias and potential_path returns import suggestion",
			data: map[string]any{
				"alias":          "status",
				"potential_path": "/components/status.pk",
			},
			wantLen: 1,
		},
		{
			name: "suggestion and import suggestion returns both",
			data: map[string]any{
				"suggestion":     "status_badge",
				"alias":          "status",
				"potential_path": "/components/status.pk",
			},
			wantLen: 2,
		},
		{
			name: "alias without potential_path returns no import suggestion",
			data: map[string]any{
				"alias": "status",
			},
			wantLen: 0,
		},
		{
			name: "potential_path without alias returns no import suggestion",
			data: map[string]any{
				"potential_path": "/components/status.pk",
			},
			wantLen: 0,
		},
		{
			name:    "empty map returns empty",
			data:    map[string]any{},
			wantLen: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := &document{
				URI: "file:///test.pk",
			}
			ws := &workspace{}

			diagnostic := protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 10},
				},
				Data: tc.data,
			}

			actions := generateUndefinedPartialAliasFixes(diagnostic, document, ws)

			if len(actions) != tc.wantLen {
				t.Errorf("len(actions) = %d, want %d", len(actions), tc.wantLen)
			}
		})
	}
}

func TestGenerateUndefinedPartialAliasFixes_ActionDetails(t *testing.T) {
	document := &document{
		URI: "file:///test.pk",
	}
	ws := &workspace{}

	diagnostic := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 2, Character: 5},
			End:   protocol.Position{Line: 2, Character: 12},
		},
		Data: map[string]any{
			"suggestion":     "StatusBadge",
			"alias":          "Status",
			"potential_path": "/components/status_badge.pk",
		},
	}

	actions := generateUndefinedPartialAliasFixes(diagnostic, document, ws)

	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}

	typoAction := actions[0]
	if !strings.Contains(typoAction.Title, "StatusBadge") {
		t.Errorf("typo action title %q should contain suggestion", typoAction.Title)
	}
	if typoAction.Kind != protocol.QuickFix {
		t.Errorf("typo action Kind = %v, want %v", typoAction.Kind, protocol.QuickFix)
	}
	if typoAction.Edit == nil {
		t.Error("typo action should have an Edit")
	}

	importAction := actions[1]
	if !strings.Contains(importAction.Title, "Status") {
		t.Errorf("import action title %q should contain alias", importAction.Title)
	}
	if importAction.Kind != protocol.QuickFix {
		t.Errorf("import action Kind = %v, want %v", importAction.Kind, protocol.QuickFix)
	}
	if importAction.IsPreferred {
		t.Error("import action should not be preferred")
	}
	if importAction.Edit != nil {
		t.Error("import action should not have an Edit (it's a suggestion only)")
	}
}
