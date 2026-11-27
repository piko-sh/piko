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

package generator_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

func TestClientScriptTarget_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		wantErr string
		target  ClientScriptTarget
	}{
		{name: "page", target: PageScriptTarget(), wantErr: ""},
		{name: "partial with a name", target: PartialScriptTarget("widgets/counter"), wantErr: ""},
		{name: "module", target: ModuleScriptTarget(), wantErr: ""},
		{name: "zero value", target: ClientScriptTarget{}, wantErr: "unknown kind ClientScriptKind(0)"},
		{name: "unknown kind", target: ClientScriptTarget{Kind: ClientScriptModule + 1}, wantErr: "unknown kind ClientScriptKind(4)"},
		{name: "partial without a name", target: PartialScriptTarget(""), wantErr: "partial script without a partial name"},
		{
			name:    "page with a partial name",
			target:  ClientScriptTarget{Kind: ClientScriptPage, PartialName: "layout"},
			wantErr: `page script with partial name "layout"`,
		},
		{
			name:    "module with a partial name",
			target:  ClientScriptTarget{Kind: ClientScriptModule, PartialName: "layout"},
			wantErr: `module script with partial name "layout"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.target.Validate()

			if tt.wantErr != "" {
				require.ErrorIs(t, err, ErrInvalidClientScriptTarget)
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestClientScriptKind_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
		kind ClientScriptKind
	}{
		{name: "page", kind: ClientScriptPage, want: "page"},
		{name: "partial", kind: ClientScriptPartial, want: "partial"},
		{name: "module", kind: ClientScriptModule, want: "module"},
		{name: "zero value", kind: 0, want: "ClientScriptKind(0)"},
		{name: "unknown kind", kind: ClientScriptModule + 1, want: "ClientScriptKind(4)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.kind.String())
		})
	}
}

func TestClientScriptTargetFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		component *annotator_dto.VirtualComponent
		want      ClientScriptTarget
		name      string
		wantErr   bool
	}{
		{
			name:      "page",
			component: &annotator_dto.VirtualComponent{IsPage: true},
			want:      PageScriptTarget(),
		},
		{
			name:      "error page",
			component: &annotator_dto.VirtualComponent{IsErrorPage: true},
			want:      PageScriptTarget(),
		},
		{
			name:      "error page that is also a page",
			component: &annotator_dto.VirtualComponent{IsPage: true, IsErrorPage: true},
			want:      PageScriptTarget(),
		},
		{
			name:      "email",
			component: &annotator_dto.VirtualComponent{IsEmail: true},
			want:      PageScriptTarget(),
		},
		{
			name:      "pdf",
			component: &annotator_dto.VirtualComponent{IsPdf: true},
			want:      PageScriptTarget(),
		},
		{
			name:      "partial under the default partials directory",
			component: &annotator_dto.VirtualComponent{PartialName: "widgets/counter"},
			want:      PartialScriptTarget("widgets/counter"),
		},
		{
			name:      "partial under a custom partials directory",
			component: &annotator_dto.VirtualComponent{PartialName: "layout"},
			want:      PartialScriptTarget("layout"),
		},
		{
			name:      "partial from another module",
			component: &annotator_dto.VirtualComponent{PartialName: "card"},
			want:      PartialScriptTarget("card"),
		},
		{
			name:    "nil component is rejected",
			wantErr: true,
		},
		{
			name:      "component with no kind and no partial name is rejected",
			component: &annotator_dto.VirtualComponent{},
			wantErr:   true,
		},
		{
			name:      "page with a partial name is rejected",
			component: &annotator_dto.VirtualComponent{IsPage: true, PartialName: "layout"},
			wantErr:   true,
		},
		{
			name:      "page that is also an email is rejected",
			component: &annotator_dto.VirtualComponent{IsPage: true, IsEmail: true},
			wantErr:   true,
		},
		{
			name:      "email that is also a pdf is rejected",
			component: &annotator_dto.VirtualComponent{IsEmail: true, IsPdf: true},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ClientScriptTargetFor(tt.component)

			if tt.wantErr {
				require.ErrorIs(t, err, ErrInvalidClientScriptTarget)
				assert.Equal(t, ClientScriptTarget{}, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
