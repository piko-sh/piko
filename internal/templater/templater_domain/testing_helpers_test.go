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

package templater_domain_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

type TestFixture struct {
	Registry     templater_domain.FunctionRegistry
	MockRunner   *templater_domain.MockManifestRunnerPort
	MockRenderer *templater_domain.MockRendererPort
	MockI18N     *i18n_domain.MockService
	Service      templater_domain.TemplaterService
}

func NewTestFixture(t *testing.T) *TestFixture {
	t.Helper()

	registry := templater_domain.NewIsolatedRegistry()
	mockRunner := &templater_domain.MockManifestRunnerPort{}
	mockRenderer := &templater_domain.MockRendererPort{}
	mockI18N := &i18n_domain.MockService{}

	service := templater_domain.NewTemplaterService(
		mockRunner,
		mockRenderer,
		mockI18N,
	)

	return &TestFixture{
		Registry:     registry,
		MockRunner:   mockRunner,
		MockRenderer: mockRenderer,
		MockI18N:     mockI18N,
		Service:      service,
	}
}

func (f *TestFixture) RegisterTestComponent(packagePath string, ast *ast_domain.TemplateAST) {
	f.Registry.RegisterASTFunc(packagePath, func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return ast, templater_dto.InternalMetadata{}, nil
	})
}

func NewTestPageDefinition(path string) templater_dto.PageDefinition {
	return templater_dto.PageDefinition{
		OriginalPath:   path,
		NormalisedPath: path,
		TemplateHTML:   "<div>test content</div>",
	}
}

func NewTestPartialDefinition(path string) templater_dto.PageDefinition {
	return templater_dto.PageDefinition{
		OriginalPath:   path,
		NormalisedPath: path,
		TemplateHTML:   "<span>test partial</span>",
	}
}

func NewTestAST() *ast_domain.TemplateAST {
	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:    ast_domain.NodeText,
				TextContent: "test content",
			},
		},
	}
}

func NewTestASTWithContent(content string) *ast_domain.TemplateAST {
	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:    ast_domain.NodeText,
				TextContent: content,
			},
		},
	}
}

func NewTestASTWithElement(tagName string, children ...*ast_domain.TemplateNode) *ast_domain.TemplateAST {
	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  tagName,
				Children: children,
			},
		},
	}
}

func NewTestRequest(method, url string) *http.Request {
	return httptest.NewRequest(method, url, nil)
}

func NewTestRequestWithLocale(method, url, locale string) *http.Request {
	request := httptest.NewRequest(method, url, nil)
	request.Header.Set("Accept-Language", locale)
	return request
}

func NewTestResponseWriter() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

func NewTestConfig() *config.WebsiteConfig {
	return &config.WebsiteConfig{}
}

func NewTestMetadata() templater_dto.InternalMetadata {
	return templater_dto.InternalMetadata{
		AssetRefs:  []templater_dto.AssetRef{},
		CustomTags: []string{},
	}
}

func NewTestMetadataWithAssets(assets []templater_dto.AssetRef) templater_dto.InternalMetadata {
	return templater_dto.InternalMetadata{
		AssetRefs:  assets,
		CustomTags: []string{},
	}
}

func NewTestRequestData() *templater_dto.RequestData {
	return templater_dto.NewRequestDataBuilder().
		WithMethod("GET").
		WithLocale("en_GB").
		Build()
}

func NewTestRequestDataWithMethod(method string) *templater_dto.RequestData {
	return templater_dto.NewRequestDataBuilder().
		WithMethod(method).
		WithLocale("en_GB").
		Build()
}

type TestCase[T any] struct {
	Input         T
	Setup         func(*TestFixture)
	Validate      func(*testing.T, *TestFixture)
	Name          string
	ErrorContains string
	ExpectedError bool
}

func RunTestCases[T any](t *testing.T, testCases []TestCase[T], testFunc func(*testing.T, *TestFixture, T)) {
	t.Helper()

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			fixture := NewTestFixture(t)

			if tc.Setup != nil {
				tc.Setup(fixture)
			}

			testFunc(t, fixture, tc.Input)

			if tc.Validate != nil {
				tc.Validate(t, fixture)
			}
		})
	}
}

func AssertNoError(t *testing.T, err error, msgAndArgs ...any) {
	t.Helper()
	if err == nil {
		return
	}
	if len(msgAndArgs) > 0 {
		t.Fatalf("Expected no error but got: %v. Context: %v", err, msgAndArgs[0])
	}
	t.Fatalf("Expected no error but got: %v", err)
}

func AssertError(t *testing.T, err error, msgAndArgs ...any) {
	t.Helper()
	if err != nil {
		return
	}
	if len(msgAndArgs) > 0 {
		t.Fatalf("Expected an error but got nil. Context: %v", msgAndArgs[0])
	}
	t.Fatal("Expected an error but got nil")
}

func AssertContains(t *testing.T, s, substr string, msgAndArgs ...any) {
	t.Helper()
	if contains(s, substr) {
		return
	}
	if len(msgAndArgs) > 0 {
		t.Fatalf("Expected %q to contain %q. Context: %v", s, substr, msgAndArgs[0])
	}
	t.Fatalf("Expected %q to contain %q", s, substr)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOfString(s, substr) >= 0)
}

func indexOfString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
