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

package typegen_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

type MockActionManifestProvider struct {
	GetLastSuccessfulBuildFunc func() (*annotator_dto.ProjectAnnotationResult, bool)
}

func (m *MockActionManifestProvider) GetLastSuccessfulBuild() (*annotator_dto.ProjectAnnotationResult, bool) {
	if m.GetLastSuccessfulBuildFunc != nil {
		return m.GetLastSuccessfulBuildFunc()
	}
	return nil, false
}

func TestNewTypeInfoService(t *testing.T) {
	t.Parallel()

	t.Run("returns non-nil without options", func(t *testing.T) {
		t.Parallel()
		service := NewTypeInfoService()
		require.NotNil(t, service)
		assert.Nil(t, service.actionProvider)
	})

	t.Run("applies WithActionProvider option", func(t *testing.T) {
		t.Parallel()
		mock := &MockActionManifestProvider{}
		service := NewTypeInfoService(WithActionProvider(mock))
		require.NotNil(t, service)
		assert.NotNil(t, service.actionProvider)
	})
}

func TestTypeInfoService_GetPikoCompletions(t *testing.T) {
	t.Parallel()
	service := NewTypeInfoService()

	testCases := []struct {
		name       string
		namespace  string
		checkLabel string
		wantMinLen int
		checkKind  CompletionItemKind
		wantNil    bool
	}{
		{
			name:       "top-level piko namespace",
			namespace:  "",
			wantMinLen: 1,
			checkLabel: "refs",
			checkKind:  CompletionKindProperty,
		},
		{
			name:       "nav sub-namespace",
			namespace:  "nav",
			wantMinLen: 1,
			checkLabel: "navigate",
			checkKind:  CompletionKindFunction,
		},
		{
			name:       "form sub-namespace",
			namespace:  "form",
			wantMinLen: 1,
			checkLabel: "data",
			checkKind:  CompletionKindFunction,
		},
		{
			name:       "ui sub-namespace",
			namespace:  "ui",
			wantMinLen: 1,
			checkLabel: "loading",
			checkKind:  CompletionKindFunction,
		},
		{
			name:       "event sub-namespace",
			namespace:  "event",
			wantMinLen: 1,
			checkLabel: "dispatch",
			checkKind:  CompletionKindFunction,
		},
		{
			name:       "partials sub-namespace",
			namespace:  "partials",
			wantMinLen: 1,
			checkLabel: "reload",
			checkKind:  CompletionKindFunction,
		},
		{
			name:       "sse sub-namespace",
			namespace:  "sse",
			wantMinLen: 1,
			checkLabel: "subscribe",
			checkKind:  CompletionKindFunction,
		},
		{
			name:       "timing sub-namespace",
			namespace:  "timing",
			wantMinLen: 1,
			checkLabel: "debounce",
			checkKind:  CompletionKindFunction,
		},
		{
			name:       "util sub-namespace",
			namespace:  "util",
			wantMinLen: 1,
			checkLabel: "whenVisible",
			checkKind:  CompletionKindFunction,
		},
		{
			name:       "trace sub-namespace",
			namespace:  "trace",
			wantMinLen: 1,
			checkLabel: "enable",
			checkKind:  CompletionKindFunction,
		},
		{
			name:      "unknown namespace returns nil",
			namespace: "nonexistent",
			wantNil:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			items := service.GetPikoCompletions(tc.namespace)

			if tc.wantNil {
				assert.Nil(t, items)
				return
			}

			require.NotNil(t, items)
			assert.GreaterOrEqual(t, len(items), tc.wantMinLen)

			if tc.checkLabel != "" {
				found := false
				for _, item := range items {
					if item.Label == tc.checkLabel {
						found = true
						assert.Equal(t, tc.checkKind, item.Kind, "kind mismatch for %q", tc.checkLabel)
						assert.NotEmpty(t, item.Detail, "detail should not be empty for %q", tc.checkLabel)
						assert.NotEmpty(t, item.Documentation, "documentation should not be empty for %q", tc.checkLabel)
						assert.Equal(t, tc.checkLabel, item.InsertText, "insertText should match label for %q", tc.checkLabel)
						break
					}
				}
				assert.True(t, found, "expected to find completion with label %q", tc.checkLabel)
			}
		})
	}
}

func TestTypeInfoService_GetPikoSubNamespaces(t *testing.T) {
	t.Parallel()
	service := NewTypeInfoService()
	namespaces := service.GetPikoSubNamespaces()

	expected := []string{"nav", "form", "ui", "event", "partials", "sse", "timing", "util", "trace"}
	assert.Equal(t, expected, namespaces)
}

func TestTypeInfoService_GetActionCompletions(t *testing.T) {
	t.Parallel()

	twoActions := []annotator_dto.ActionDefinition{
		{
			Name:           "customer.create",
			TSFunctionName: "customerCreate",
			Description:    "Creates a customer",
			CallParams:     []annotator_dto.ActionTypeInfo{{Name: "CreateInput", TSType: "CreateInput"}},
			OutputType:     &annotator_dto.ActionTypeInfo{Name: "CustomerResponse", TSType: "CustomerResponse"},
		},
		{
			Name:           "order.submit",
			TSFunctionName: "orderSubmit",
			Description:    "Submits an order",
		},
	}

	testCases := []struct {
		name     string
		provider ActionManifestProvider
		prefix   string
		wantNil  bool
		wantLen  int
	}{
		{
			name:    "nil provider returns nil",
			wantNil: true,
		},
		{
			name: "provider returns false",
			provider: &MockActionManifestProvider{
				GetLastSuccessfulBuildFunc: func() (*annotator_dto.ProjectAnnotationResult, bool) {
					return nil, false
				},
			},
			wantNil: true,
		},
		{
			name: "provider returns nil result",
			provider: &MockActionManifestProvider{
				GetLastSuccessfulBuildFunc: func() (*annotator_dto.ProjectAnnotationResult, bool) {
					return nil, true
				},
			},
			wantNil: true,
		},
		{
			name: "nil virtual module",
			provider: &MockActionManifestProvider{
				GetLastSuccessfulBuildFunc: func() (*annotator_dto.ProjectAnnotationResult, bool) {
					return &annotator_dto.ProjectAnnotationResult{}, true
				},
			},
			wantNil: true,
		},
		{
			name: "nil action manifest",
			provider: &MockActionManifestProvider{
				GetLastSuccessfulBuildFunc: func() (*annotator_dto.ProjectAnnotationResult, bool) {
					return &annotator_dto.ProjectAnnotationResult{
						VirtualModule: &annotator_dto.VirtualModule{},
					}, true
				},
			},
			wantNil: true,
		},
		{
			name: "empty actions returns nil",
			provider: &MockActionManifestProvider{
				GetLastSuccessfulBuildFunc: func() (*annotator_dto.ProjectAnnotationResult, bool) {
					return &annotator_dto.ProjectAnnotationResult{
						VirtualModule: &annotator_dto.VirtualModule{
							ActionManifest: &annotator_dto.ActionManifest{},
						},
					}, true
				},
			},
			wantNil: true,
		},
		{
			name: "returns all actions with empty prefix",
			provider: &MockActionManifestProvider{
				GetLastSuccessfulBuildFunc: func() (*annotator_dto.ProjectAnnotationResult, bool) {
					return &annotator_dto.ProjectAnnotationResult{
						VirtualModule: &annotator_dto.VirtualModule{
							ActionManifest: &annotator_dto.ActionManifest{
								Actions: twoActions,
							},
						},
					}, true
				},
			},
			prefix:  "",
			wantLen: 2,
		},
		{
			name: "filters by prefix",
			provider: &MockActionManifestProvider{
				GetLastSuccessfulBuildFunc: func() (*annotator_dto.ProjectAnnotationResult, bool) {
					return &annotator_dto.ProjectAnnotationResult{
						VirtualModule: &annotator_dto.VirtualModule{
							ActionManifest: &annotator_dto.ActionManifest{
								Actions: twoActions,
							},
						},
					}, true
				},
			},
			prefix:  "customer",
			wantLen: 1,
		},
		{
			name: "prefix matches none",
			provider: &MockActionManifestProvider{
				GetLastSuccessfulBuildFunc: func() (*annotator_dto.ProjectAnnotationResult, bool) {
					return &annotator_dto.ProjectAnnotationResult{
						VirtualModule: &annotator_dto.VirtualModule{
							ActionManifest: &annotator_dto.ActionManifest{
								Actions: twoActions,
							},
						},
					}, true
				},
			},
			prefix:  "zzz",
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var opts []TypeInfoServiceOption
			if tc.provider != nil {
				opts = append(opts, WithActionProvider(tc.provider))
			}
			service := NewTypeInfoService(opts...)
			items := service.GetActionCompletions(tc.prefix)

			if tc.wantNil {
				assert.Empty(t, items)
				return
			}

			require.NotNil(t, items)
			assert.Len(t, items, tc.wantLen)

			for _, item := range items {
				assert.NotEmpty(t, item.Label)
				assert.NotEmpty(t, item.Detail)
				assert.Equal(t, CompletionKindFunction, item.Kind)
				assert.Equal(t, item.Label, item.InsertText)
			}
		})
	}
}

func TestBuildActionSignature(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		wantSuffix string
		action     annotator_dto.ActionDefinition
	}{
		{
			name: "no input no output",
			action: annotator_dto.ActionDefinition{
				TSFunctionName: "doThing",
			},
			wantSuffix: "doThing(): ActionBuilder<void>",
		},
		{
			name: "input only with TSType",
			action: annotator_dto.ActionDefinition{
				TSFunctionName: "createUser",
				CallParams:     []annotator_dto.ActionTypeInfo{{TSType: "CreateInput"}},
			},
			wantSuffix: "createUser(CreateInput): ActionBuilder<void>",
		},
		{
			name: "output only with TSType",
			action: annotator_dto.ActionDefinition{
				TSFunctionName: "getUser",
				OutputType:     &annotator_dto.ActionTypeInfo{TSType: "UserResponse"},
			},
			wantSuffix: "getUser(): ActionBuilder<UserResponse>",
		},
		{
			name: "both input and output",
			action: annotator_dto.ActionDefinition{
				TSFunctionName: "updateUser",
				CallParams:     []annotator_dto.ActionTypeInfo{{TSType: "UpdateInput"}},
				OutputType:     &annotator_dto.ActionTypeInfo{TSType: "UserResponse"},
			},
			wantSuffix: "updateUser(UpdateInput): ActionBuilder<UserResponse>",
		},
		{
			name: "falls back to Name when TSType empty",
			action: annotator_dto.ActionDefinition{
				TSFunctionName: "createUser",
				CallParams:     []annotator_dto.ActionTypeInfo{{Name: "FallbackInput"}},
				OutputType:     &annotator_dto.ActionTypeInfo{Name: "FallbackOutput"},
			},
			wantSuffix: "createUser(FallbackInput): ActionBuilder<FallbackOutput>",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := buildActionSignature(&tc.action)
			assert.Equal(t, tc.wantSuffix, result)
		})
	}
}
