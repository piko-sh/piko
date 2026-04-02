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

package pikotest_domain_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pikotest/pikotest_domain"
	"piko.sh/piko/internal/pikotest/pikotest_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestNewComponentTester_DefaultConfig(t *testing.T) {
	buildAST := func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*pikotest_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}

	tester := pikotest_domain.NewComponentTester(t, buildAST)
	require.NotNil(t, tester)
}

func TestNewComponentTester_WithPageID(t *testing.T) {
	buildAST := func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*pikotest_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}

	tester := pikotest_domain.NewComponentTester(t, buildAST,
		pikotest_domain.WithPageID("my-custom-page"),
	)
	require.NotNil(t, tester)
}

func TestRender_ReturnsTestView(t *testing.T) {
	h1 := makeElementWithChildren("h1", makeTextNode("Test Page"))

	buildAST := func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*pikotest_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{h1},
			}, templater_dto.InternalMetadata{
				Metadata: templater_dto.Metadata{Title: "Test Page"},
			}, nil
	}

	tester := pikotest_domain.NewComponentTester(t, buildAST)
	request := pikotest_domain.NewRequest("GET", "/").Build(context.Background())
	view := tester.Render(request, nil)

	require.NotNil(t, view)
	view.AssertTitle("Test Page")
}

func TestRender_EmptyAST(t *testing.T) {
	buildAST := func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*pikotest_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}

	tester := pikotest_domain.NewComponentTester(t, buildAST)
	request := pikotest_domain.NewRequest("GET", "/").Build(context.Background())
	view := tester.Render(request, nil)

	require.NotNil(t, view)
	assert.NotNil(t, view.AST())
}

func TestRender_PropsPassedThrough(t *testing.T) {
	type TestProps struct {
		Title string
	}

	var receivedProps any

	buildAST := func(_ *templater_dto.RequestData, props any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*pikotest_dto.RuntimeDiagnostic) {
		receivedProps = props
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}

	tester := pikotest_domain.NewComponentTester(t, buildAST)
	request := pikotest_domain.NewRequest("GET", "/").Build(context.Background())
	tester.Render(request, TestProps{Title: "Hello"})

	require.NotNil(t, receivedProps)
	assert.Equal(t, "Hello", receivedProps.(TestProps).Title)
}

func TestRender_RequestDataPassedThrough(t *testing.T) {
	var receivedReq *templater_dto.RequestData

	buildAST := func(r *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*pikotest_dto.RuntimeDiagnostic) {
		receivedReq = r
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}

	tester := pikotest_domain.NewComponentTester(t, buildAST)
	request := pikotest_domain.NewRequest("GET", "/customers").
		WithQueryParam("sort", "desc").
		Build(context.Background())
	tester.Render(request, nil)

	require.NotNil(t, receivedReq)
	assert.Equal(t, "desc", receivedReq.QueryParam("sort"))
}
