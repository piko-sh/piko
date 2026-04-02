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
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

func TestFindQuotedValueBounds(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		line      string
		cursor    int
		wantStart int
		wantEnd   int
	}{
		{
			name:      "cursor inside double-quoted value",
			line:      `p-on:submit="action.email.Contact($form)"`,
			cursor:    20,
			wantStart: 13,
			wantEnd:   40,
		},
		{
			name:      "cursor inside single-quoted value",
			line:      `p-on:submit='action.email.Contact($form)'`,
			cursor:    20,
			wantStart: 13,
			wantEnd:   40,
		},
		{
			name:      "cursor outside quotes",
			line:      `p-on:submit="action.email.Contact($form)"`,
			cursor:    5,
			wantStart: -1,
			wantEnd:   -1,
		},
		{
			name:      "cursor at start of value",
			line:      `p-on:submit="action.email.Contact($form)"`,
			cursor:    13,
			wantStart: 13,
			wantEnd:   40,
		},
		{
			name:      "no quotes in line",
			line:      `just some text here`,
			cursor:    5,
			wantStart: -1,
			wantEnd:   -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			start, end := findQuotedValueBounds(tc.line, tc.cursor)
			if start != tc.wantStart || end != tc.wantEnd {
				t.Errorf("findQuotedValueBounds(%q, %d) = (%d, %d), want (%d, %d)",
					tc.line, tc.cursor, start, end, tc.wantStart, tc.wantEnd)
			}
		})
	}
}

func TestFindActionSegmentAtCursor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		line         string
		cursor       int
		wantNil      bool
		wantIndex    int
		wantSegCount int
		wantSegStart int
		wantSegEnd   int
	}{
		{
			name:         "cursor on action keyword",
			line:         `p-on:submit="action.email.Contact($form)"`,
			cursor:       16,
			wantIndex:    0,
			wantSegCount: 3,
			wantSegStart: 13,
			wantSegEnd:   19,
		},
		{
			name:         "cursor on namespace segment",
			line:         `p-on:submit="action.email.Contact($form)"`,
			cursor:       22,
			wantIndex:    1,
			wantSegCount: 3,
			wantSegStart: 20,
			wantSegEnd:   25,
		},
		{
			name:         "cursor on action name segment",
			line:         `p-on:submit="action.email.Contact($form)"`,
			cursor:       28,
			wantIndex:    2,
			wantSegCount: 3,
			wantSegStart: 26,
			wantSegEnd:   33,
		},
		{
			name:    "cursor on $form argument - not an action segment",
			line:    `p-on:submit="action.email.Contact($form)"`,
			cursor:  36,
			wantNil: true,
		},
		{
			name:    "non-action attribute value",
			line:    `p-on:click="handleClick()"`,
			cursor:  15,
			wantNil: true,
		},
		{
			name:    "cursor outside quotes",
			line:    `p-on:submit="action.email.Contact($form)"`,
			cursor:  5,
			wantNil: true,
		},
		{
			name:         "action with prevent modifier",
			line:         `p-on:submit.prevent="action.email.Contact($form)"`,
			cursor:       30,
			wantIndex:    1,
			wantSegCount: 3,
			wantSegStart: 28,
			wantSegEnd:   33,
		},
		{
			name:    "only action. prefix with no segments",
			line:    `p-on:submit="action."`,
			cursor:  15,
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			info := findActionSegmentAtCursor(tc.line, tc.cursor)

			if tc.wantNil {
				if info != nil {
					t.Errorf("expected nil, got segments=%v index=%d", info.segments, info.segmentIndex)
				}
				return
			}

			if info == nil {
				t.Fatal("expected non-nil result")
			}
			if info.segmentIndex != tc.wantIndex {
				t.Errorf("segmentIndex = %d, want %d", info.segmentIndex, tc.wantIndex)
			}
			if len(info.segments) != tc.wantSegCount {
				t.Errorf("segment count = %d, want %d", len(info.segments), tc.wantSegCount)
			}
			if info.segmentStart != tc.wantSegStart {
				t.Errorf("segmentStart = %d, want %d", info.segmentStart, tc.wantSegStart)
			}
			if info.segmentEnd != tc.wantSegEnd {
				t.Errorf("segmentEnd = %d, want %d", info.segmentEnd, tc.wantSegEnd)
			}
		})
	}
}

func TestActionSegmentToKind(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		info     actionSegmentInfo
		wantKind PKDefinitionKind
	}{
		{
			name:     "first segment is action root",
			info:     actionSegmentInfo{segments: []string{"action", "email", "Contact"}, segmentIndex: 0},
			wantKind: PKDefActionRoot,
		},
		{
			name:     "middle segment is namespace",
			info:     actionSegmentInfo{segments: []string{"action", "email", "Contact"}, segmentIndex: 1},
			wantKind: PKDefActionNamespace,
		},
		{
			name:     "last segment with 3+ parts is action name",
			info:     actionSegmentInfo{segments: []string{"action", "email", "Contact"}, segmentIndex: 2},
			wantKind: PKDefActionName,
		},
		{
			name:     "two segments - second is namespace not name",
			info:     actionSegmentInfo{segments: []string{"action", "email"}, segmentIndex: 1},
			wantKind: PKDefActionNamespace,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := actionSegmentToKind(&tc.info)
			if got != tc.wantKind {
				t.Errorf("actionSegmentToKind() = %d, want %d", got, tc.wantKind)
			}
		})
	}
}

func TestBuildActionName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		segments []string
		want     string
	}{
		{
			name:     "three segments",
			segments: []string{"action", "email", "Contact"},
			want:     "email.Contact",
		},
		{
			name:     "two segments",
			segments: []string{"action", "email"},
			want:     "email",
		},
		{
			name:     "single segment",
			segments: []string{"action"},
			want:     "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			info := &actionSegmentInfo{segments: tc.segments}
			got := buildActionName(info)
			if got != tc.want {
				t.Errorf("buildActionName() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtractNamespaceFromActionName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "with dot", input: "email.Contact", want: "email"},
		{name: "without dot", input: "email", want: "email"},
		{name: "empty string", input: "", want: ""},
		{name: "multiple dots", input: "deep.nested.Action", want: "deep"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := extractNamespaceFromActionName(tc.input)
			if got != tc.want {
				t.Errorf("extractNamespaceFromActionName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCheckActionHoverContext(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		line     string
		cursor   int
		wantNil  bool
		wantKind PKDefinitionKind
		wantName string
	}{
		{
			name:     "hover on action keyword",
			line:     `p-on:submit.prevent="action.email.Contact($form)"`,
			cursor:   24,
			wantKind: PKDefActionRoot,
			wantName: "email.Contact",
		},
		{
			name:     "hover on namespace",
			line:     `p-on:submit.prevent="action.email.Contact($form)"`,
			cursor:   30,
			wantKind: PKDefActionNamespace,
			wantName: "email.Contact",
		},
		{
			name:     "hover on action name",
			line:     `p-on:submit.prevent="action.email.Contact($form)"`,
			cursor:   38,
			wantKind: PKDefActionName,
			wantName: "email.Contact",
		},
		{
			name:    "hover on $form",
			line:    `p-on:submit.prevent="action.email.Contact($form)"`,
			cursor:  45,
			wantNil: true,
		},
		{
			name:    "not an action expression",
			line:    `p-on:click="handleClick()"`,
			cursor:  15,
			wantNil: true,
		},
	}

	document := &document{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			position := protocol.Position{Line: 0, Character: uint32(tc.cursor)}
			ctx := document.checkActionHoverContext(tc.line, tc.cursor, position)

			if tc.wantNil {
				if ctx != nil {
					t.Errorf("expected nil, got kind=%d name=%q", ctx.Kind, ctx.Name)
				}
				return
			}

			if ctx == nil {
				t.Fatal("expected non-nil result")
			}
			if ctx.Kind != tc.wantKind {
				t.Errorf("Kind = %d, want %d", ctx.Kind, tc.wantKind)
			}
			if ctx.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", ctx.Name, tc.wantName)
			}
		})
	}
}

func TestCheckActionDefinitionContext(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		line     string
		cursor   int
		wantNil  bool
		wantName string
	}{
		{
			name:     "cursor on action name produces definition",
			line:     `p-on:submit.prevent="action.email.Contact($form)"`,
			cursor:   38,
			wantName: "email.Contact",
		},
		{
			name:    "cursor on action root produces no definition",
			line:    `p-on:submit.prevent="action.email.Contact($form)"`,
			cursor:  24,
			wantNil: true,
		},
		{
			name:    "cursor on namespace produces no definition",
			line:    `p-on:submit.prevent="action.email.Contact($form)"`,
			cursor:  30,
			wantNil: true,
		},
		{
			name:    "non-action expression",
			line:    `p-on:click="handleClick()"`,
			cursor:  15,
			wantNil: true,
		},
	}

	document := &document{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			position := protocol.Position{Line: 0, Character: uint32(tc.cursor)}
			ctx := document.checkActionDefinitionContext(tc.line, tc.cursor, position)

			if tc.wantNil {
				if ctx != nil {
					t.Errorf("expected nil, got name=%q", ctx.Name)
				}
				return
			}

			if ctx == nil {
				t.Fatal("expected non-nil result")
			}
			if ctx.Kind != PKDefActionName {
				t.Errorf("Kind = %d, want PKDefActionName (%d)", ctx.Kind, PKDefActionName)
			}
			if ctx.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", ctx.Name, tc.wantName)
			}
		})
	}
}

func TestBuildActionDisplaySignature(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		action annotator_dto.ActionDefinition
		want   string
	}{
		{
			name: "with input and output types",
			action: annotator_dto.ActionDefinition{
				Name:       "email.Contact",
				CallParams: []annotator_dto.ActionTypeInfo{{Name: "ContactInput"}},
				OutputType: &annotator_dto.ActionTypeInfo{Name: "ContactResponse"},
			},
			want: "email.Contact(ContactInput): ActionBuilder<ContactResponse>",
		},
		{
			name: "with input only",
			action: annotator_dto.ActionDefinition{
				Name:       "email.Contact",
				CallParams: []annotator_dto.ActionTypeInfo{{Name: "ContactInput"}},
			},
			want: "email.Contact(ContactInput): ActionBuilder<void>",
		},
		{
			name: "no input or output",
			action: annotator_dto.ActionDefinition{
				Name: "email.Ping",
			},
			want: "email.Ping(): ActionBuilder<void>",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := buildActionDisplaySignature(&tc.action)
			if got != tc.want {
				t.Errorf("buildActionDisplaySignature() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseActionCompletionPrefix(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		prefix        string
		wantNamespace string
		wantFilter    string
		wantHasNS     bool
	}{
		{
			name:       "empty prefix",
			prefix:     "",
			wantFilter: "",
			wantHasNS:  false,
		},
		{
			name:       "partial namespace",
			prefix:     "em",
			wantFilter: "em",
			wantHasNS:  false,
		},
		{
			name:          "full namespace with dot",
			prefix:        "email.",
			wantNamespace: "email",
			wantFilter:    "",
			wantHasNS:     true,
		},
		{
			name:          "namespace with action filter",
			prefix:        "email.Con",
			wantNamespace: "email",
			wantFilter:    "Con",
			wantHasNS:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ns, filter, hasNS := parseActionCompletionPrefix(tc.prefix)
			if ns != tc.wantNamespace {
				t.Errorf("namespace = %q, want %q", ns, tc.wantNamespace)
			}
			if filter != tc.wantFilter {
				t.Errorf("filter = %q, want %q", filter, tc.wantFilter)
			}
			if hasNS != tc.wantHasNS {
				t.Errorf("hasNamespace = %v, want %v", hasNS, tc.wantHasNS)
			}
		})
	}
}

func TestGetActionNamespaceCompletions_Hierarchical(t *testing.T) {
	t.Parallel()

	manifest := annotator_dto.NewActionManifest()
	manifest.AddAction(annotator_dto.ActionDefinition{
		Name:           "email.Contact",
		TSFunctionName: "emailContact",
	})
	manifest.AddAction(annotator_dto.ActionDefinition{
		Name:           "email.Valuation",
		TSFunctionName: "emailValuation",
	})
	manifest.AddAction(annotator_dto.ActionDefinition{
		Name:           "user.Create",
		TSFunctionName: "userCreate",
	})

	document := &document{
		ProjectResult: &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ActionManifest: manifest,
			},
		},
	}

	t.Run("empty prefix shows namespace groups", func(t *testing.T) {
		t.Parallel()
		result, err := document.getActionNamespaceCompletions("")
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Items) != 2 {
			t.Fatalf("expected 2 namespace items, got %d", len(result.Items))
		}

		labels := make(map[string]bool)
		for _, item := range result.Items {
			labels[item.Label] = true
			if item.Kind != protocol.CompletionItemKindModule {
				t.Errorf("expected Module kind for %q, got %v", item.Label, item.Kind)
			}
		}
		if !labels["email"] || !labels["user"] {
			t.Errorf("expected email and user namespaces, got %v", labels)
		}
	})

	t.Run("partial prefix filters namespaces", func(t *testing.T) {
		t.Parallel()
		result, err := document.getActionNamespaceCompletions("em")
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(result.Items))
		}
		if result.Items[0].Label != "email" {
			t.Errorf("expected email, got %q", result.Items[0].Label)
		}
	})

	t.Run("namespace dot shows actions in namespace", func(t *testing.T) {
		t.Parallel()
		result, err := document.getActionNamespaceCompletions("email.")
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Items) != 2 {
			t.Fatalf("expected 2 action items, got %d", len(result.Items))
		}

		labels := make(map[string]bool)
		for _, item := range result.Items {
			labels[item.Label] = true
			if item.Kind != protocol.CompletionItemKindFunction {
				t.Errorf("expected Function kind for %q, got %v", item.Label, item.Kind)
			}
		}
		if !labels["Contact"] || !labels["Valuation"] {
			t.Errorf("expected Contact and Valuation, got %v", labels)
		}
	})

	t.Run("namespace dot with filter narrows actions", func(t *testing.T) {
		t.Parallel()
		result, err := document.getActionNamespaceCompletions("email.Con")
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(result.Items))
		}
		if result.Items[0].Label != "Contact" {
			t.Errorf("expected Contact, got %q", result.Items[0].Label)
		}
	})

	t.Run("non-matching prefix returns empty", func(t *testing.T) {
		t.Parallel()
		result, err := document.getActionNamespaceCompletions("xyz")
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Items) != 0 {
			t.Errorf("expected 0 items, got %d", len(result.Items))
		}
	})
}

func TestLookupAction(t *testing.T) {
	t.Parallel()

	manifest := annotator_dto.NewActionManifest()
	manifest.AddAction(annotator_dto.ActionDefinition{
		Name:           "email.Contact",
		TSFunctionName: "emailContact",
	})

	testDocument := &document{
		ProjectResult: &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ActionManifest: manifest,
			},
		},
	}

	t.Run("exact match", func(t *testing.T) {
		t.Parallel()
		action := testDocument.lookupAction("email.Contact")
		if action == nil {
			t.Fatal("expected to find action")
		}
		if action.Name != "email.Contact" {
			t.Errorf("Name = %q, want %q", action.Name, "email.Contact")
		}
	})

	t.Run("case insensitive match", func(t *testing.T) {
		t.Parallel()
		action := testDocument.lookupAction("Email.contact")
		if action == nil {
			t.Fatal("expected to find action (case insensitive)")
		}
		if action.Name != "email.Contact" {
			t.Errorf("Name = %q, want %q", action.Name, "email.Contact")
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		action := testDocument.lookupAction("nonexistent.Action")
		if action != nil {
			t.Errorf("expected nil, got %v", action)
		}
	})

	t.Run("nil project result", func(t *testing.T) {
		t.Parallel()
		emptyDoc := &document{}
		action := emptyDoc.lookupAction("email.Contact")
		if action != nil {
			t.Errorf("expected nil for nil project, got %v", action)
		}
	})
}

func TestGetActionHover_ActionName(t *testing.T) {
	t.Parallel()

	manifest := annotator_dto.NewActionManifest()
	manifest.AddAction(annotator_dto.ActionDefinition{
		Name:           "email.Contact",
		TSFunctionName: "emailContact",
		HTTPMethod:     "POST",
		FilePath:       "actions/email/contact.go",
		Description:    "Sends a contact email.",
		StructLine:     15,
		CallParams: []annotator_dto.ActionTypeInfo{
			{
				Name: "ContactInput",
				Fields: []annotator_dto.ActionFieldInfo{
					{Name: "Email", GoType: "string", JSONName: "email"},
					{Name: "Message", GoType: "string", JSONName: "message"},
				},
			},
		},
		OutputType: &annotator_dto.ActionTypeInfo{
			Name: "ContactResponse",
		},
	})

	document := &document{
		ProjectResult: &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ActionManifest: manifest,
			},
		},
		Resolver: &resolver_domain.MockResolver{},
	}

	ctx := &PKHoverContext{
		Kind:     PKDefActionName,
		Name:     "email.Contact",
		Position: protocol.Position{Line: 0, Character: 28},
		Range:    protocol.Range{},
	}

	hover, err := document.getActionHover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if hover == nil {
		t.Fatal("expected hover, got nil")
	}

	content := hover.Contents.Value

	if !strings.Contains(content, "email.Contact") {
		t.Error("expected hover to contain action name")
	}
	if !strings.Contains(content, "POST") {
		t.Error("expected hover to contain HTTP method")
	}
	if !strings.Contains(content, "ContactInput") {
		t.Error("expected hover to contain input type")
	}
	if !strings.Contains(content, "ContactResponse") {
		t.Error("expected hover to contain output type")
	}
	if !strings.Contains(content, "email") {
		t.Error("expected hover to contain field name")
	}
	if !strings.Contains(content, "Sends a contact email.") {
		t.Error("expected hover to contain description")
	}
}

func TestGetActionHover_Root(t *testing.T) {
	t.Parallel()

	document := &document{}
	ctx := &PKHoverContext{
		Kind:     PKDefActionRoot,
		Name:     "email.Contact",
		Position: protocol.Position{},
		Range:    protocol.Range{},
	}

	hover, err := document.getActionHover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if hover == nil {
		t.Fatal("expected hover, got nil")
	}

	content := hover.Contents.Value
	if !strings.Contains(content, "action") {
		t.Error("expected hover to mention action namespace")
	}
}

func TestGetActionHover_Namespace(t *testing.T) {
	t.Parallel()

	manifest := annotator_dto.NewActionManifest()
	manifest.AddAction(annotator_dto.ActionDefinition{
		Name:           "email.Contact",
		TSFunctionName: "emailContact",
	})
	manifest.AddAction(annotator_dto.ActionDefinition{
		Name:           "email.Valuation",
		TSFunctionName: "emailValuation",
	})

	document := &document{
		ProjectResult: &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ActionManifest: manifest,
			},
		},
	}

	ctx := &PKHoverContext{
		Kind:     PKDefActionNamespace,
		Name:     "email.Contact",
		Position: protocol.Position{},
		Range:    protocol.Range{},
	}

	hover, err := document.getActionHover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if hover == nil {
		t.Fatal("expected hover, got nil")
	}

	content := hover.Contents.Value
	if !strings.Contains(content, "action.email") {
		t.Error("expected hover to contain namespace path")
	}
	if !strings.Contains(content, "2 action(s)") {
		t.Errorf("expected hover to mention 2 actions, got: %s", content)
	}
}

func TestFindActionDefinition(t *testing.T) {
	t.Parallel()

	manifest := annotator_dto.NewActionManifest()
	manifest.AddAction(annotator_dto.ActionDefinition{
		Name:       "email.Contact",
		FilePath:   "actions/email/contact.go",
		StructName: "ContactAction",
		StructLine: 15,
	})

	testDocument := &document{
		ProjectResult: &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ActionManifest: manifest,
			},
		},
		Resolver: &resolver_domain.MockResolver{},
	}

	t.Run("finds action definition", func(t *testing.T) {
		t.Parallel()
		locs, err := testDocument.findActionDefinition("email.Contact")
		if err != nil {
			t.Fatal(err)
		}
		if len(locs) != 1 {
			t.Fatalf("expected 1 location, got %d", len(locs))
		}
		if locs[0].Range.Start.Line != 14 {
			t.Errorf("line = %d, want 14 (0-based from StructLine=15)", locs[0].Range.Start.Line)
		}
	})

	t.Run("returns nil for unknown action", func(t *testing.T) {
		t.Parallel()
		locs, err := testDocument.findActionDefinition("nonexistent.Action")
		if err != nil {
			t.Fatal(err)
		}
		if locs != nil {
			t.Errorf("expected nil, got %v", locs)
		}
	})

	t.Run("falls back to line 1 when StructLine is 0", func(t *testing.T) {
		t.Parallel()
		manifest2 := annotator_dto.NewActionManifest()
		manifest2.AddAction(annotator_dto.ActionDefinition{
			Name:       "email.Old",
			FilePath:   "actions/email/old.go",
			StructName: "OldAction",
			StructLine: 0,
		})
		doc2 := &document{
			ProjectResult: &annotator_dto.ProjectAnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ActionManifest: manifest2,
				},
			},
			Resolver: &resolver_domain.MockResolver{},
		}
		locs, err := doc2.findActionDefinition("email.Old")
		if err != nil {
			t.Fatal(err)
		}
		if len(locs) != 1 {
			t.Fatalf("expected 1 location, got %d", len(locs))
		}
		if locs[0].Range.Start.Line != 0 {
			t.Errorf("line = %d, want 0 (fallback line 1, 0-based)", locs[0].Range.Start.Line)
		}
	})
}

func TestFindObjectKeyAtCursor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		line      string
		cursor    int
		wantKey   string
		wantStart int
		wantEnd   int
	}{
		{
			name:      "cursor on snake_case key",
			line:      `{environment_id: state.X, blueprint_id: state.Y}`,
			cursor:    5,
			wantKey:   "environment_id",
			wantStart: 1,
			wantEnd:   15,
		},
		{
			name:      "cursor on second key",
			line:      `{environment_id: state.X, blueprint_id: state.Y}`,
			cursor:    28,
			wantKey:   "blueprint_id",
			wantStart: 26,
			wantEnd:   38,
		},
		{
			name:      "cursor on camelCase key",
			line:      `{userId: state.ID}`,
			cursor:    3,
			wantKey:   "userId",
			wantStart: 1,
			wantEnd:   7,
		},
		{
			name:    "cursor on value (not a key)",
			line:    `{name: state.Name}`,
			cursor:  10,
			wantKey: "",
		},
		{
			name:    "cursor on non-identifier",
			line:    `{name: state.Name}`,
			cursor:  0,
			wantKey: "",
		},
		{
			name:      "cursor at end of key",
			line:      `{name: val}`,
			cursor:    4,
			wantKey:   "name",
			wantStart: 1,
			wantEnd:   5,
		},
		{
			name:      "key with spaces before colon",
			line:      `{name  : val}`,
			cursor:    3,
			wantKey:   "name",
			wantStart: 1,
			wantEnd:   5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			key, start, end := findObjectKeyAtCursor(tc.line, tc.cursor)
			if key != tc.wantKey {
				t.Errorf("key = %q, want %q", key, tc.wantKey)
			}
			if tc.wantKey != "" {
				if start != tc.wantStart {
					t.Errorf("start = %d, want %d", start, tc.wantStart)
				}
				if end != tc.wantEnd {
					t.Errorf("end = %d, want %d", end, tc.wantEnd)
				}
			}
		})
	}
}

func TestCheckActionParamKeyHoverContext(t *testing.T) {
	t.Parallel()

	document := &document{}

	testCases := []struct {
		name     string
		line     string
		cursor   int
		wantNil  bool
		wantKind PKDefinitionKind
		wantName string
	}{
		{
			name: "cursor on key inside action call",

			line:     `p-event:confirm="action.blueprint.FieldDelete({environment_id: state.X})"`,
			cursor:   52,
			wantKind: PKDefActionParamKey,
			wantName: "blueprint.FieldDelete:environment_id",
		},
		{
			name:    "cursor on value inside action call",
			line:    `p-event:confirm="action.blueprint.FieldDelete({environment_id: state.X})"`,
			cursor:  68,
			wantNil: true,
		},
		{
			name:    "cursor outside parens on action name",
			line:    `p-event:confirm="action.blueprint.FieldDelete({environment_id: state.X})"`,
			cursor:  38,
			wantNil: true,
		},
		{
			name:    "not an action expression",
			line:    `p-on:click="someFunc({key: val})"`,
			cursor:  23,
			wantNil: true,
		},
		{
			name:    "cursor outside quotes",
			line:    `p-event:confirm="action.blueprint.FieldDelete({environment_id: state.X})"`,
			cursor:  5,
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			position := protocol.Position{Line: 0, Character: uint32(tc.cursor)}
			ctx := document.checkActionParamKeyHoverContext(tc.line, tc.cursor, position)

			if tc.wantNil {
				if ctx != nil {
					t.Errorf("expected nil, got kind=%d name=%q", ctx.Kind, ctx.Name)
				}
				return
			}

			if ctx == nil {
				t.Fatal("expected non-nil result")
			}
			if ctx.Kind != tc.wantKind {
				t.Errorf("Kind = %d, want %d", ctx.Kind, tc.wantKind)
			}
			if ctx.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", ctx.Name, tc.wantName)
			}
		})
	}
}

func TestFindActionFieldByJSONName(t *testing.T) {
	t.Parallel()

	params := []annotator_dto.ActionTypeInfo{
		{
			Name: "FieldDeleteInput",
			Fields: []annotator_dto.ActionFieldInfo{
				{Name: "EnvironmentID", GoType: "string", JSONName: "environment_id", Validation: "required,uuid"},
				{Name: "BlueprintID", GoType: "string", JSONName: "blueprint_id", Validation: "required,uuid"},
				{Name: "FieldID", GoType: "string", JSONName: "field_id"},
			},
		},
	}

	t.Run("finds matching field", func(t *testing.T) {
		t.Parallel()
		field := findActionFieldByJSONName(params, "environment_id")
		if field == nil {
			t.Fatal("expected field, got nil")
		}
		if field.Name != "EnvironmentID" {
			t.Errorf("Name = %q, want %q", field.Name, "EnvironmentID")
		}
		if field.GoType != "string" {
			t.Errorf("GoType = %q, want %q", field.GoType, "string")
		}
		if field.Validation != "required,uuid" {
			t.Errorf("Validation = %q, want %q", field.Validation, "required,uuid")
		}
	})

	t.Run("returns nil for unknown key", func(t *testing.T) {
		t.Parallel()
		field := findActionFieldByJSONName(params, "unknown_key")
		if field != nil {
			t.Errorf("expected nil, got %v", field)
		}
	})

	t.Run("returns nil for empty params", func(t *testing.T) {
		t.Parallel()
		field := findActionFieldByJSONName(nil, "environment_id")
		if field != nil {
			t.Errorf("expected nil, got %v", field)
		}
	})
}

func TestSplitActionParamKey(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		combined       string
		wantActionName string
		wantKeyName    string
	}{
		{"blueprint.FieldDelete:environment_id", "blueprint.FieldDelete", "environment_id"},
		{"email.Contact:email", "email.Contact", "email"},
		{"nocolon", "nocolon", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.combined, func(t *testing.T) {
			t.Parallel()
			actionName, keyName := splitActionParamKey(tc.combined)
			if actionName != tc.wantActionName {
				t.Errorf("actionName = %q, want %q", actionName, tc.wantActionName)
			}
			if keyName != tc.wantKeyName {
				t.Errorf("keyName = %q, want %q", keyName, tc.wantKeyName)
			}
		})
	}
}

func TestGetActionParamKeyHover(t *testing.T) {
	t.Parallel()

	manifest := annotator_dto.NewActionManifest()
	manifest.AddAction(annotator_dto.ActionDefinition{
		Name: "blueprint.FieldDelete",
		CallParams: []annotator_dto.ActionTypeInfo{
			{
				Name: "FieldDeleteInput",
				Fields: []annotator_dto.ActionFieldInfo{
					{
						Name:        "EnvironmentID",
						GoType:      "string",
						TSType:      "string",
						JSONName:    "environment_id",
						Validation:  "required,uuid",
						Description: "The environment to delete from.",
					},
					{
						Name:     "Count",
						GoType:   "int",
						TSType:   "number",
						JSONName: "count",
						Optional: true,
					},
				},
			},
		},
	})

	document := &document{
		ProjectResult: &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ActionManifest: manifest,
			},
		},
	}

	t.Run("shows field type and validation", func(t *testing.T) {
		t.Parallel()
		ctx := &PKHoverContext{
			Kind: PKDefActionParamKey,
			Name: "blueprint.FieldDelete:environment_id",
		}

		hover, err := document.getActionParamKeyHover(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if hover == nil {
			t.Fatal("expected hover, got nil")
		}

		content := hover.Contents.Value
		if !strings.Contains(content, "environment_id") {
			t.Error("expected hover to contain JSON name")
		}
		if !strings.Contains(content, "string") {
			t.Error("expected hover to contain Go type")
		}
		if !strings.Contains(content, "required,uuid") {
			t.Error("expected hover to contain validation")
		}
		if !strings.Contains(content, "EnvironmentID") {
			t.Error("expected hover to contain Go field name")
		}
		if !strings.Contains(content, "The environment to delete from.") {
			t.Error("expected hover to contain description")
		}
	})

	t.Run("shows optional badge", func(t *testing.T) {
		t.Parallel()
		ctx := &PKHoverContext{
			Kind: PKDefActionParamKey,
			Name: "blueprint.FieldDelete:count",
		}

		hover, err := document.getActionParamKeyHover(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if hover == nil {
			t.Fatal("expected hover, got nil")
		}

		content := hover.Contents.Value
		if !strings.Contains(content, "Optional") {
			t.Error("expected hover to contain Optional badge")
		}
		if !strings.Contains(content, "number") {
			t.Error("expected hover to contain TS type")
		}
	})

	t.Run("shows TS type when different from Go type", func(t *testing.T) {
		t.Parallel()
		ctx := &PKHoverContext{
			Kind: PKDefActionParamKey,
			Name: "blueprint.FieldDelete:count",
		}

		hover, err := document.getActionParamKeyHover(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if hover == nil {
			t.Fatal("expected hover, got nil")
		}

		content := hover.Contents.Value
		if !strings.Contains(content, "**TypeScript:** `number`") {
			t.Errorf("expected TypeScript type annotation, got: %s", content)
		}
	})

	t.Run("returns nil for unknown key", func(t *testing.T) {
		t.Parallel()
		ctx := &PKHoverContext{
			Kind: PKDefActionParamKey,
			Name: "blueprint.FieldDelete:nonexistent",
		}

		hover, err := document.getActionParamKeyHover(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if hover != nil {
			t.Errorf("expected nil, got hover with content: %s", hover.Contents.Value)
		}
	})

	t.Run("returns nil for unknown action", func(t *testing.T) {
		t.Parallel()
		ctx := &PKHoverContext{
			Kind: PKDefActionParamKey,
			Name: "nonexistent.Action:key",
		}

		hover, err := document.getActionParamKeyHover(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if hover != nil {
			t.Errorf("expected nil, got hover with content: %s", hover.Contents.Value)
		}
	})
}
