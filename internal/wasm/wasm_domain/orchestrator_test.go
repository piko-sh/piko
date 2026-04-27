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

package wasm_domain

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/internal/wasm/wasm_dto"
)

type noOpConsole struct{}

func (c *noOpConsole) Debug(message string, arguments ...any) {}
func (c *noOpConsole) Info(message string, arguments ...any)  {}
func (c *noOpConsole) Warn(message string, arguments ...any)  {}
func (c *noOpConsole) Error(message string, arguments ...any) {}

type mockStdlibLoader struct {
	data *inspector_dto.TypeData
}

func newMockStdlibLoader() *mockStdlibLoader {
	return &mockStdlibLoader{
		data: &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"fmt": {
					Path: "fmt",
					Name: "fmt",
					NamedTypes: map[string]*inspector_dto.Type{
						"Stringer": {
							Name:                 "Stringer",
							TypeString:           "Stringer",
							UnderlyingTypeString: "interface{...}",
						},
					},
				},
				"time": {
					Path: "time",
					Name: "time",
					NamedTypes: map[string]*inspector_dto.Type{
						"Time": {
							Name:                 "Time",
							TypeString:           "Time",
							UnderlyingTypeString: "struct{...}",
						},
						"Duration": {
							Name:                 "Duration",
							TypeString:           "Duration",
							UnderlyingTypeString: "int64",
						},
					},
				},
			},
		},
	}
}

func (m *mockStdlibLoader) Load() (*inspector_dto.TypeData, error) {
	return m.data, nil
}

func (m *mockStdlibLoader) GetPackageList() []string {
	result := make([]string, 0, len(m.data.Packages))
	for pkg := range m.data.Packages {
		result = append(result, pkg)
	}
	return result
}

type failingStdlibLoader struct{}

func (*failingStdlibLoader) Load() (*inspector_dto.TypeData, error) {
	return nil, errors.New("load boom")
}

func (*failingStdlibLoader) GetPackageList() []string {
	return nil
}

type stubGeneratorPort struct {
	response *wasm_dto.GenerateFromSourcesResponse
	err      error
}

func (s *stubGeneratorPort) Generate(_ context.Context, _ *wasm_dto.GenerateFromSourcesRequest) (*wasm_dto.GenerateFromSourcesResponse, error) {
	return s.response, s.err
}

type stubInterpreterPort struct {
	response *wasm_dto.InterpretResponse
	err      error
}

func (s *stubInterpreterPort) Interpret(_ context.Context, _ *wasm_dto.InterpretRequest) (*wasm_dto.InterpretResponse, error) {
	return s.response, s.err
}

type stubJSInterop struct{}

func (*stubJSInterop) RegisterFunction(_ string, _ func(arguments []any) (any, error)) {}
func (*stubJSInterop) Log(_ string, _ string, _ ...any)                                {}
func (*stubJSInterop) MarshalToJS(_ any) (any, error)                                  { return nil, nil }
func (*stubJSInterop) UnmarshalFromJS(_ any, _ any) error                              { return nil }

type sfcparserScript struct {
	Content string
}

func extractScriptBlockInfoFromContent(content string) *wasm_dto.ScriptBlockInfo {
	info := &wasm_dto.ScriptBlockInfo{
		PropsType: "",
		Types:     make([]string, 0),
		HasInit:   false,
	}

	file := parseScriptContent(content)
	if file == nil {
		return info
	}

	for _, declaration := range file.Decls {
		extractDeclInfo(declaration, info)
	}

	return info
}

func TestOrchestrator_Initialise(t *testing.T) {
	o := NewOrchestrator(
		WithStdlibLoader(newMockStdlibLoader()),
		WithConsole(&noOpConsole{}),
	)

	ctx := context.Background()
	err := o.Initialise(ctx)
	require.NoError(t, err)

	info, err := o.GetRuntimeInfo(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, info.GoVersion)
	assert.Contains(t, info.Capabilities, "analyse")
}

func TestOrchestrator_Analyse_SimpleStruct(t *testing.T) {
	o := NewOrchestrator(
		WithStdlibLoader(newMockStdlibLoader()),
		WithConsole(&noOpConsole{}),
	)

	ctx := context.Background()
	require.NoError(t, o.Initialise(ctx))

	response, err := o.Analyse(ctx, &wasm_dto.AnalyseRequest{
		Sources: map[string]string{
			"main.go": `package main

type User struct {
	Name string
	Age  int
}
`,
		},
		ModuleName: "test",
	})

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.True(t, response.Success, "Expected success, got error: %s", response.Error)
	assert.Len(t, response.Types, 1)
	assert.Equal(t, "User", response.Types[0].Name)
	assert.Equal(t, "struct", response.Types[0].Kind)
	assert.Len(t, response.Types[0].Fields, 2)
}

func TestOrchestrator_Analyse_WithStdlibRef(t *testing.T) {
	o := NewOrchestrator(
		WithStdlibLoader(newMockStdlibLoader()),
		WithConsole(&noOpConsole{}),
	)

	ctx := context.Background()
	require.NoError(t, o.Initialise(ctx))

	response, err := o.Analyse(ctx, &wasm_dto.AnalyseRequest{
		Sources: map[string]string{
			"main.go": `package main

import "time"

type Event struct {
	Timestamp time.Time
	Name      string
}
`,
		},
		ModuleName: "test",
	})

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.True(t, response.Success, "Expected success, got error: %s", response.Error)
	assert.Len(t, response.Types, 1)
	assert.Equal(t, "Event", response.Types[0].Name)
}

func TestOrchestrator_Validate_ValidCode(t *testing.T) {
	o := NewOrchestrator(
		WithStdlibLoader(newMockStdlibLoader()),
		WithConsole(&noOpConsole{}),
	)

	ctx := context.Background()
	require.NoError(t, o.Initialise(ctx))

	response, err := o.Validate(ctx, &wasm_dto.ValidateRequest{
		Source:   `package main; func main() {}`,
		FilePath: "main.go",
	})

	require.NoError(t, err)
	assert.True(t, response.Valid)
}

func TestOrchestrator_Validate_InvalidCode(t *testing.T) {
	o := NewOrchestrator(
		WithStdlibLoader(newMockStdlibLoader()),
		WithConsole(&noOpConsole{}),
	)

	ctx := context.Background()
	require.NoError(t, o.Initialise(ctx))

	response, err := o.Validate(ctx, &wasm_dto.ValidateRequest{
		Source:   `package main; func main( { }`,
		FilePath: "main.go",
	})

	require.NoError(t, err)
	assert.False(t, response.Valid)
	assert.NotEmpty(t, response.Diagnostics)
}

func TestOrchestrator_GetCompletions(t *testing.T) {
	o := NewOrchestrator(
		WithStdlibLoader(newMockStdlibLoader()),
		WithConsole(&noOpConsole{}),
	)

	ctx := context.Background()
	require.NoError(t, o.Initialise(ctx))

	response, err := o.GetCompletions(ctx, &wasm_dto.CompletionRequest{
		Source: `package main

func main() {
	f
}`,
		Line:   4,
		Column: 3,
	})

	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.NotEmpty(t, response.Items)
}

func TestOrchestrator_NotInitialised(t *testing.T) {
	o := NewOrchestrator(
		WithStdlibLoader(newMockStdlibLoader()),
	)

	ctx := context.Background()

	_, err := o.Analyse(ctx, &wasm_dto.AnalyseRequest{
		Sources: map[string]string{"main.go": "package main"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not initialised")
}

func TestOrchestrator_GetHover_QualifiedName(t *testing.T) {
	loader := newMockStdlibLoader()
	loader.data.Packages["fmt"].Funcs = map[string]*inspector_dto.Function{
		"Println": {
			Name:       "Println",
			TypeString: "(a ...any) (n int, err error)",
		},
	}

	o := NewOrchestrator(
		WithStdlibLoader(loader),
		WithConsole(&noOpConsole{}),
	)

	ctx := context.Background()
	require.NoError(t, o.Initialise(ctx))

	response, err := o.GetHover(ctx, &wasm_dto.HoverRequest{
		Source: `package main

import "fmt"

func main() {
	fmt.Println("hello")
}`,
		Line:   6,
		Column: 10,
	})

	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Contains(t, response.Content, "Println")
	assert.Contains(t, response.Content, "fmt")
}

func TestOrchestrator_GetHover_PackageName(t *testing.T) {
	o := NewOrchestrator(
		WithStdlibLoader(newMockStdlibLoader()),
		WithConsole(&noOpConsole{}),
	)

	ctx := context.Background()
	require.NoError(t, o.Initialise(ctx))

	response, err := o.GetHover(ctx, &wasm_dto.HoverRequest{
		Source: `package main

import "fmt"

func main() {
	fmt.Println("hello")
}`,
		Line:   6,
		Column: 3,
	})

	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Contains(t, response.Content, "package")
	assert.Contains(t, response.Content, "fmt")
}

func TestOrchestrator_GetCompletions_PackageMember(t *testing.T) {
	loader := newMockStdlibLoader()
	loader.data.Packages["time"].Funcs = map[string]*inspector_dto.Function{
		"Now": {
			Name:       "Now",
			TypeString: "() Time",
		},
		"Sleep": {
			Name:       "Sleep",
			TypeString: "(d Duration)",
		},
	}

	o := NewOrchestrator(
		WithStdlibLoader(loader),
		WithConsole(&noOpConsole{}),
	)

	ctx := context.Background()
	require.NoError(t, o.Initialise(ctx))

	response, err := o.GetCompletions(ctx, &wasm_dto.CompletionRequest{
		Source: `package main

import "time"

func main() {
	time.
}`,
		Line:   6,
		Column: 7,
	})

	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.NotEmpty(t, response.Items)

	var foundTime, foundNow bool
	for _, item := range response.Items {
		if item.Label == "Time" {
			foundTime = true
		}
		if item.Label == "Now" {
			foundNow = true
		}
	}
	assert.True(t, foundTime, "Should contain Time type")
	assert.True(t, foundNow, "Should contain Now function")
}

func TestOrchestrator_GetCompletions_PackageMemberWithPrefix(t *testing.T) {
	loader := newMockStdlibLoader()
	loader.data.Packages["time"].Funcs = map[string]*inspector_dto.Function{
		"Now":   {Name: "Now", TypeString: "() Time"},
		"Sleep": {Name: "Sleep", TypeString: "(d Duration)"},
	}

	o := NewOrchestrator(
		WithStdlibLoader(loader),
		WithConsole(&noOpConsole{}),
	)

	ctx := context.Background()
	require.NoError(t, o.Initialise(ctx))

	response, err := o.GetCompletions(ctx, &wasm_dto.CompletionRequest{
		Source: `package main

import "time"

func main() {
	time.Du
}`,
		Line:   6,
		Column: 9,
	})

	require.NoError(t, err)
	assert.True(t, response.Success)

	var foundDuration bool
	for _, item := range response.Items {
		if item.Label == "Duration" {
			foundDuration = true
		}
	}
	assert.True(t, foundDuration, "Should contain Duration type matching prefix")
}

func TestOrchestrator_ParseTemplate_Basic(t *testing.T) {
	o := NewOrchestrator(
		WithStdlibLoader(newMockStdlibLoader()),
		WithConsole(&noOpConsole{}),
	)

	ctx := context.Background()
	require.NoError(t, o.Initialise(ctx))

	response, err := o.ParseTemplate(ctx, &wasm_dto.ParseTemplateRequest{
		Template: `<template>
<div>Hello, {{ name }}!</div>
</template>

<script lang="go">
type Props struct {
	Name string
}
</script>
`,
	})

	require.NoError(t, err)
	assert.True(t, response.Success, "Expected success, got error: %s", response.Error)
	require.NotNil(t, response.AST)
	assert.NotEmpty(t, response.AST.Nodes)
	require.NotNil(t, response.AST.ScriptBlock)
	assert.Contains(t, response.AST.ScriptBlock.Types, "Props")
	assert.Equal(t, "Props", response.AST.ScriptBlock.PropsType)
}

func TestOrchestrator_ParseTemplate_WithInit(t *testing.T) {
	o := NewOrchestrator(
		WithStdlibLoader(newMockStdlibLoader()),
		WithConsole(&noOpConsole{}),
	)

	ctx := context.Background()
	require.NoError(t, o.Initialise(ctx))

	response, err := o.ParseTemplate(ctx, &wasm_dto.ParseTemplateRequest{
		Template: `<template>
<div>Test</div>
</template>

<script lang="go">
func init() {
	// initialisation
}
</script>
`,
	})

	require.NoError(t, err)
	assert.True(t, response.Success)
	require.NotNil(t, response.AST)
	require.NotNil(t, response.AST.ScriptBlock)
	assert.True(t, response.AST.ScriptBlock.HasInit)
}

func TestWithJSInterop(t *testing.T) {
	t.Parallel()

	interop := &stubJSInterop{}
	o := NewOrchestrator(WithJSInterop(interop))
	assert.NotNil(t, o.jsInterop)
}

func TestWithInterpreter(t *testing.T) {
	t.Parallel()

	interp := &stubInterpreterPort{
		response: nil,
		err:      nil,
	}
	o := NewOrchestrator(WithInterpreter(interp))
	assert.NotNil(t, o.interpreter)
}

func TestOrchestrator_Initialise_LoadError(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(
		WithStdlibLoader(&failingStdlibLoader{}),
	)
	err := o.Initialise(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load stdlib data")
}

func TestOrchestrator_Generate_GeneratorError(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(
		WithGenerator(&stubGeneratorPort{
			response: nil,
			err:      errors.New("gen error"),
		}),
	)
	response, err := o.Generate(t.Context(), &wasm_dto.GenerateFromSourcesRequest{
		Sources:    map[string]string{"a.pk": "<template><p>hi</p></template>"},
		ModuleName: "test",
		BaseDir:    "",
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "generation failed")
}

func TestOrchestrator_Generate_NonSuccess(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(
		WithGenerator(&stubGeneratorPort{
			response: &wasm_dto.GenerateFromSourcesResponse{
				Success:     false,
				Error:       "bad input",
				Artefacts:   nil,
				Diagnostics: nil,
				Manifest:    nil,
			},
			err: nil,
		}),
	)
	response, err := o.Generate(t.Context(), &wasm_dto.GenerateFromSourcesRequest{
		Sources:    map[string]string{"a.pk": "<template><p>hi</p></template>"},
		ModuleName: "test",
		BaseDir:    "",
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "bad input", response.Error)
}

func TestOrchestrator_Generate_Success(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(
		WithGenerator(&stubGeneratorPort{
			response: &wasm_dto.GenerateFromSourcesResponse{
				Success:     true,
				Error:       "",
				Artefacts:   nil,
				Diagnostics: nil,
				Manifest:    nil,
			},
			err: nil,
		}),
	)
	response, err := o.Generate(t.Context(), &wasm_dto.GenerateFromSourcesRequest{
		Sources:    map[string]string{"a.pk": "<template><p>hi</p></template>"},
		ModuleName: "test",
		BaseDir:    "",
	})
	require.NoError(t, err)
	assert.True(t, response.Success)
}

func TestOrchestrator_Render_RendererError(t *testing.T) {
	t.Parallel()

	renderer := &stubRenderPort{
		renderFunc: func(_ context.Context, _ *wasm_dto.RenderFromSourcesRequest) (*wasm_dto.RenderFromSourcesResponse, error) {
			return nil, errors.New("render error")
		},
		renderASTFunc: nil,
	}
	o := NewOrchestrator(WithRenderer(renderer))
	response, err := o.Render(t.Context(), &wasm_dto.RenderFromSourcesRequest{
		Sources:    map[string]string{"p.pk": "<template><p>hi</p></template>"},
		ModuleName: "test",
		EntryPoint: "p.pk",
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "rendering failed")
}

func TestOrchestrator_Render_NonSuccess(t *testing.T) {
	t.Parallel()

	renderer := &stubRenderPort{
		renderFunc: func(_ context.Context, _ *wasm_dto.RenderFromSourcesRequest) (*wasm_dto.RenderFromSourcesResponse, error) {
			return &wasm_dto.RenderFromSourcesResponse{
				Success:      false,
				Error:        "template error",
				HTML:         "",
				CSS:          "",
				Diagnostics:  nil,
				IsStaticOnly: false,
			}, nil
		},
		renderASTFunc: nil,
	}
	o := NewOrchestrator(WithRenderer(renderer))
	response, err := o.Render(t.Context(), &wasm_dto.RenderFromSourcesRequest{
		Sources:    map[string]string{"p.pk": "<template><p>hi</p></template>"},
		ModuleName: "test",
		EntryPoint: "p.pk",
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
}

func TestOrchestrator_Render_Success(t *testing.T) {
	t.Parallel()

	renderer := &stubRenderPort{
		renderFunc: func(_ context.Context, _ *wasm_dto.RenderFromSourcesRequest) (*wasm_dto.RenderFromSourcesResponse, error) {
			return &wasm_dto.RenderFromSourcesResponse{
				Success:      true,
				HTML:         "<p>ok</p>",
				CSS:          "",
				Error:        "",
				Diagnostics:  nil,
				IsStaticOnly: false,
			}, nil
		},
		renderASTFunc: nil,
	}
	o := NewOrchestrator(WithRenderer(renderer))
	response, err := o.Render(t.Context(), &wasm_dto.RenderFromSourcesRequest{
		Sources:    map[string]string{"p.pk": "<template><p>hi</p></template>"},
		ModuleName: "test",
		EntryPoint: "p.pk",
	})
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "<p>ok</p>", response.HTML)
}

func TestOrchestrator_DynamicRender_NoGenerator(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(
		WithInterpreter(&stubInterpreterPort{
			response: nil,
			err:      nil,
		}),
	)
	response, err := o.DynamicRender(t.Context(), &wasm_dto.DynamicRenderRequest{
		Sources:    map[string]string{"p.pk": "<template><p>hi</p></template>"},
		ModuleName: "test",
		RequestURL: "/",
		Props:      nil,
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "generator not configured")
}

func TestOrchestrator_DynamicRender_NoInterpreter(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(
		WithGenerator(&stubGeneratorPort{
			response: nil,
			err:      nil,
		}),
	)
	response, err := o.DynamicRender(t.Context(), &wasm_dto.DynamicRenderRequest{
		Sources:    map[string]string{"p.pk": "<template><p>hi</p></template>"},
		ModuleName: "test",
		RequestURL: "/",
		Props:      nil,
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "interpreter not configured")
}

func TestOrchestrator_DynamicRender_GenerateError(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(
		WithGenerator(&stubGeneratorPort{
			response: nil,
			err:      errors.New("gen fail"),
		}),
		WithInterpreter(&stubInterpreterPort{
			response: nil,
			err:      nil,
		}),
	)
	response, err := o.DynamicRender(t.Context(), &wasm_dto.DynamicRenderRequest{
		Sources:    map[string]string{"p.pk": "<template><p>hi</p></template>"},
		ModuleName: "test",
		RequestURL: "/",
		Props:      nil,
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "code generation failed")
}

func TestOrchestrator_DynamicRender_GenerateNonSuccess(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(
		WithGenerator(&stubGeneratorPort{
			response: &wasm_dto.GenerateFromSourcesResponse{
				Success:     false,
				Error:       "bad template",
				Artefacts:   nil,
				Diagnostics: nil,
				Manifest:    nil,
			},
			err: nil,
		}),
		WithInterpreter(&stubInterpreterPort{
			response: nil,
			err:      nil,
		}),
	)
	response, err := o.DynamicRender(t.Context(), &wasm_dto.DynamicRenderRequest{
		Sources:    map[string]string{"p.pk": "<template><p>hi</p></template>"},
		ModuleName: "test",
		RequestURL: "/",
		Props:      nil,
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "bad template", response.Error)
}

func TestOrchestrator_DynamicRender_NoPageFound(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(
		WithGenerator(&stubGeneratorPort{
			response: &wasm_dto.GenerateFromSourcesResponse{
				Success:     true,
				Error:       "",
				Artefacts:   []wasm_dto.GeneratedArtefact{},
				Diagnostics: nil,
				Manifest:    nil,
			},
			err: nil,
		}),
		WithInterpreter(&stubInterpreterPort{
			response: nil,
			err:      nil,
		}),
	)
	response, err := o.DynamicRender(t.Context(), &wasm_dto.DynamicRenderRequest{
		Sources:    map[string]string{"p.pk": "<template><p>hi</p></template>"},
		ModuleName: "test",
		RequestURL: "/nonexistent",
		Props:      nil,
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "no page found for URL")
}

func TestOrchestrator_DynamicRender_InterpretError(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(
		WithGenerator(&stubGeneratorPort{
			response: &wasm_dto.GenerateFromSourcesResponse{
				Success: true,
				Error:   "",
				Artefacts: []wasm_dto.GeneratedArtefact{
					{
						Path:       "dist/page.go",
						Content:    "package main",
						Type:       wasm_dto.ArtefactTypePage,
						SourcePath: "p.pk",
					},
				},
				Diagnostics: nil,
				Manifest: &wasm_dto.GeneratedManifest{
					Pages: map[string]wasm_dto.ManifestPageEntry{
						"p": {
							SourcePath:    "p.pk",
							PackagePath:   "test/pages/p",
							RoutePatterns: map[string]string{"en": "/"},
							CachePolicy:   nil,
							StyleBlock:    "",
							JSArtefactIDs: nil,
							HasGetData:    false,
							HasRender:     false,
						},
					},
					Partials: nil,
				},
			},
			err: nil,
		}),
		WithInterpreter(&stubInterpreterPort{
			response: &wasm_dto.InterpretResponse{
				Success:     false,
				Error:       "",
				AST:         nil,
				Metadata:    nil,
				Diagnostics: []wasm_dto.Diagnostic{{Severity: "error", Message: "interp fail", Location: wasm_dto.Location{FilePath: "", Line: 0, Column: 0}, Code: ""}},
			},
			err: errors.New("interp boom"),
		}),
		WithConsole(&noOpConsole{}),
	)
	response, err := o.DynamicRender(t.Context(), &wasm_dto.DynamicRenderRequest{
		Sources:    map[string]string{"p.pk": "<template><p>hi</p></template>"},
		ModuleName: "test",
		RequestURL: "/",
		Props:      nil,
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "interpretation failed")
}

func TestOrchestrator_DynamicRender_InterpretNonSuccess(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(
		WithGenerator(&stubGeneratorPort{
			response: &wasm_dto.GenerateFromSourcesResponse{
				Success: true,
				Error:   "",
				Artefacts: []wasm_dto.GeneratedArtefact{
					{
						Path:       "dist/page.go",
						Content:    "package main",
						Type:       wasm_dto.ArtefactTypePage,
						SourcePath: "p.pk",
					},
				},
				Diagnostics: nil,
				Manifest: &wasm_dto.GeneratedManifest{
					Pages: map[string]wasm_dto.ManifestPageEntry{
						"p": {
							SourcePath:    "p.pk",
							PackagePath:   "test/pages/p",
							RoutePatterns: map[string]string{"en": "/"},
							CachePolicy:   nil,
							StyleBlock:    "",
							JSArtefactIDs: nil,
							HasGetData:    false,
							HasRender:     false,
						},
					},
					Partials: nil,
				},
			},
			err: nil,
		}),
		WithInterpreter(&stubInterpreterPort{
			response: &wasm_dto.InterpretResponse{
				Success:     false,
				Error:       "runtime fail",
				AST:         nil,
				Metadata:    nil,
				Diagnostics: nil,
			},
			err: nil,
		}),
		WithConsole(&noOpConsole{}),
	)
	response, err := o.DynamicRender(t.Context(), &wasm_dto.DynamicRenderRequest{
		Sources:    map[string]string{"p.pk": "<template><p>hi</p></template>"},
		ModuleName: "test",
		RequestURL: "/",
		Props:      nil,
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "runtime fail", response.Error)
}

func TestOrchestrator_DynamicRender_RenderError(t *testing.T) {
	t.Parallel()

	renderer := &stubRenderPort{
		renderFunc: nil,
		renderASTFunc: func(_ context.Context, _ *wasm_dto.RenderFromASTRequest) (*wasm_dto.RenderFromASTResponse, error) {
			return nil, errors.New("render boom")
		},
	}
	o := NewOrchestrator(
		WithGenerator(&stubGeneratorPort{
			response: &wasm_dto.GenerateFromSourcesResponse{
				Success: true,
				Error:   "",
				Artefacts: []wasm_dto.GeneratedArtefact{
					{
						Path:       "dist/page.go",
						Content:    "package main",
						Type:       wasm_dto.ArtefactTypePage,
						SourcePath: "p.pk",
					},
				},
				Diagnostics: nil,
				Manifest: &wasm_dto.GeneratedManifest{
					Pages: map[string]wasm_dto.ManifestPageEntry{
						"p": {
							SourcePath:    "p.pk",
							PackagePath:   "test/pages/p",
							RoutePatterns: map[string]string{"en": "/"},
							CachePolicy:   nil,
							StyleBlock:    "",
							JSArtefactIDs: nil,
							HasGetData:    false,
							HasRender:     false,
						},
					},
					Partials: nil,
				},
			},
			err: nil,
		}),
		WithInterpreter(&stubInterpreterPort{
			response: &wasm_dto.InterpretResponse{
				Success:     true,
				Error:       "",
				AST:         &ast_domain.TemplateAST{},
				Metadata:    &templater_dto.InternalMetadata{},
				Diagnostics: nil,
			},
			err: nil,
		}),
		WithRenderer(renderer),
		WithConsole(&noOpConsole{}),
	)
	response, err := o.DynamicRender(t.Context(), &wasm_dto.DynamicRenderRequest{
		Sources:    map[string]string{"p.pk": "<template><p>hi</p></template>"},
		ModuleName: "test",
		RequestURL: "/",
		Props:      nil,
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "rendering failed")
}

func TestOrchestrator_DynamicRender_RenderNonSuccess(t *testing.T) {
	t.Parallel()

	renderer := &stubRenderPort{
		renderFunc: nil,
		renderASTFunc: func(_ context.Context, _ *wasm_dto.RenderFromASTRequest) (*wasm_dto.RenderFromASTResponse, error) {
			return &wasm_dto.RenderFromASTResponse{
				Success: false,
				Error:   "bad ast",
				HTML:    "",
				CSS:     "",
			}, nil
		},
	}
	o := NewOrchestrator(
		WithGenerator(&stubGeneratorPort{
			response: &wasm_dto.GenerateFromSourcesResponse{
				Success: true,
				Error:   "",
				Artefacts: []wasm_dto.GeneratedArtefact{
					{
						Path:       "dist/page.go",
						Content:    "package main",
						Type:       wasm_dto.ArtefactTypePage,
						SourcePath: "p.pk",
					},
				},
				Diagnostics: nil,
				Manifest: &wasm_dto.GeneratedManifest{
					Pages: map[string]wasm_dto.ManifestPageEntry{
						"p": {
							SourcePath:    "p.pk",
							PackagePath:   "test/pages/p",
							RoutePatterns: map[string]string{"en": "/"},
							CachePolicy:   nil,
							StyleBlock:    "",
							JSArtefactIDs: nil,
							HasGetData:    false,
							HasRender:     false,
						},
					},
					Partials: nil,
				},
			},
			err: nil,
		}),
		WithInterpreter(&stubInterpreterPort{
			response: &wasm_dto.InterpretResponse{
				Success:     true,
				Error:       "",
				AST:         &ast_domain.TemplateAST{},
				Metadata:    &templater_dto.InternalMetadata{},
				Diagnostics: nil,
			},
			err: nil,
		}),
		WithRenderer(renderer),
		WithConsole(&noOpConsole{}),
	)
	response, err := o.DynamicRender(t.Context(), &wasm_dto.DynamicRenderRequest{
		Sources:    map[string]string{"p.pk": "<template><p>hi</p></template>"},
		ModuleName: "test",
		RequestURL: "/",
		Props:      nil,
	})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "bad ast")
}

func TestOrchestrator_DynamicRender_Success(t *testing.T) {
	t.Parallel()

	renderer := &stubRenderPort{
		renderFunc: nil,
		renderASTFunc: func(_ context.Context, _ *wasm_dto.RenderFromASTRequest) (*wasm_dto.RenderFromASTResponse, error) {
			return &wasm_dto.RenderFromASTResponse{
				Success: true,
				HTML:    "<p>rendered</p>",
				CSS:     "p { color: blue; }",
				Error:   "",
			}, nil
		},
	}
	o := NewOrchestrator(
		WithGenerator(&stubGeneratorPort{
			response: &wasm_dto.GenerateFromSourcesResponse{
				Success: true,
				Error:   "",
				Artefacts: []wasm_dto.GeneratedArtefact{
					{
						Path:       "dist/page.go",
						Content:    "package main",
						Type:       wasm_dto.ArtefactTypePage,
						SourcePath: "p.pk",
					},
				},
				Diagnostics: nil,
				Manifest: &wasm_dto.GeneratedManifest{
					Pages: map[string]wasm_dto.ManifestPageEntry{
						"p": {
							SourcePath:    "p.pk",
							PackagePath:   "test/pages/p",
							RoutePatterns: map[string]string{"en": "/"},
							CachePolicy:   nil,

							StyleBlock:    "p { color: blue; }",
							JSArtefactIDs: nil,
							HasGetData:    false,
							HasRender:     false,
						},
					},
					Partials: nil,
				},
			},
			err: nil,
		}),
		WithInterpreter(&stubInterpreterPort{
			response: &wasm_dto.InterpretResponse{
				Success:     true,
				Error:       "",
				AST:         &ast_domain.TemplateAST{},
				Metadata:    &templater_dto.InternalMetadata{},
				Diagnostics: nil,
			},
			err: nil,
		}),
		WithRenderer(renderer),
		WithConsole(&noOpConsole{}),
	)
	response, err := o.DynamicRender(t.Context(), &wasm_dto.DynamicRenderRequest{
		Sources:    map[string]string{"p.pk": "<template><p>hi</p></template>"},
		ModuleName: "test",
		RequestURL: "/",
		Props:      nil,
	})
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "<p>rendered</p>", response.HTML)
	assert.Equal(t, "p { color: blue; }", response.CSS)
	assert.Equal(t, defaultRuntimeImports, response.RuntimeImports,
		"every successful dynamic render must echo the framework runtime URLs")
}

func TestOrchestrator_DynamicRender_PopulatesScripts(t *testing.T) {
	t.Parallel()

	renderer := &stubRenderPort{
		renderFunc: nil,
		renderASTFunc: func(_ context.Context, _ *wasm_dto.RenderFromASTRequest) (*wasm_dto.RenderFromASTResponse, error) {
			return &wasm_dto.RenderFromASTResponse{Success: true, HTML: "<p>ok</p>"}, nil
		},
	}
	o := NewOrchestrator(
		WithGenerator(&stubGeneratorPort{
			response: &wasm_dto.GenerateFromSourcesResponse{
				Success: true,
				Artefacts: []wasm_dto.GeneratedArtefact{
					{Path: "dist/page.go", Content: "package main", Type: wasm_dto.ArtefactTypePage, SourcePath: "p.pk"},
					{Path: "pk-js/pages/index.js", Content: "console.log('boot');", Type: wasm_dto.ArtefactTypeJS, SourcePath: "p.pk"},
					{Path: "pk-js/components/c.js", Content: "class C extends PPElement {}", Type: wasm_dto.ArtefactTypeJS, SourcePath: "c.pkc"},
				},
				Manifest: &wasm_dto.GeneratedManifest{
					Pages: map[string]wasm_dto.ManifestPageEntry{
						"p": {
							SourcePath:    "p.pk",
							PackagePath:   "test/pages/p",
							RoutePatterns: map[string]string{"en": "/"},
							StyleBlock:    "p { color: red; }",
						},
					},
				},
			},
		}),
		WithInterpreter(&stubInterpreterPort{
			response: &wasm_dto.InterpretResponse{
				Success: true, AST: &ast_domain.TemplateAST{}, Metadata: &templater_dto.InternalMetadata{},
			},
		}),
		WithRenderer(renderer),
		WithConsole(&noOpConsole{}),
	)
	response, err := o.DynamicRender(t.Context(), &wasm_dto.DynamicRenderRequest{
		Sources:    map[string]string{"p.pk": "<template></template>"},
		ModuleName: "test",
		RequestURL: "/",
	})
	require.NoError(t, err)
	require.True(t, response.Success)

	scriptPaths := make(map[string]string, len(response.Scripts))
	for _, script := range response.Scripts {
		scriptPaths[script.Path] = script.Content
	}
	assert.Equal(t, "console.log('boot');", scriptPaths["pk-js/pages/index.js"])
	assert.Equal(t, "class C extends PPElement {}", scriptPaths["pk-js/components/c.js"])
	assert.NotContains(t, scriptPaths, "dist/page.go", "non-JS artefacts must be filtered out")
	assert.Equal(t, "p { color: red; }", response.CSS)
}

func TestOrchestrator_DynamicRender_OmitsScriptsWhenNoneEmitted(t *testing.T) {
	t.Parallel()

	renderer := &stubRenderPort{
		renderASTFunc: func(_ context.Context, _ *wasm_dto.RenderFromASTRequest) (*wasm_dto.RenderFromASTResponse, error) {
			return &wasm_dto.RenderFromASTResponse{Success: true, HTML: "<p/>"}, nil
		},
	}
	o := NewOrchestrator(
		WithGenerator(&stubGeneratorPort{
			response: &wasm_dto.GenerateFromSourcesResponse{
				Success: true,
				Artefacts: []wasm_dto.GeneratedArtefact{
					{Path: "dist/page.go", Content: "package main", Type: wasm_dto.ArtefactTypePage, SourcePath: "p.pk"},
				},
				Manifest: &wasm_dto.GeneratedManifest{Pages: map[string]wasm_dto.ManifestPageEntry{
					"p": {SourcePath: "p.pk", PackagePath: "test/pages/p", RoutePatterns: map[string]string{"en": "/"}},
				}},
			},
		}),
		WithInterpreter(&stubInterpreterPort{
			response: &wasm_dto.InterpretResponse{
				Success: true, AST: &ast_domain.TemplateAST{}, Metadata: &templater_dto.InternalMetadata{},
			},
		}),
		WithRenderer(renderer),
		WithConsole(&noOpConsole{}),
	)
	response, err := o.DynamicRender(t.Context(), &wasm_dto.DynamicRenderRequest{
		Sources: map[string]string{"p.pk": "x"}, ModuleName: "test", RequestURL: "/",
	})
	require.NoError(t, err)
	require.True(t, response.Success)
	assert.Nil(t, response.Scripts, "omitempty must hide an empty script list")
}

func TestCollectScriptArtefacts_NilSafety(t *testing.T) {
	t.Parallel()

	assert.Nil(t, collectScriptArtefacts(nil))
	assert.Nil(t, collectScriptArtefacts(&wasm_dto.GenerateFromSourcesResponse{}))
}

func TestFindPageStyleBlock_FallsBackToEmpty(t *testing.T) {
	t.Parallel()

	assert.Empty(t, findPageStyleBlock(nil, "/"))
	assert.Empty(t, findPageStyleBlock(&wasm_dto.GenerateFromSourcesResponse{}, "/"))
	assert.Empty(t, findPageStyleBlock(&wasm_dto.GenerateFromSourcesResponse{
		Manifest: &wasm_dto.GeneratedManifest{
			Pages: map[string]wasm_dto.ManifestPageEntry{
				"p": {RoutePatterns: map[string]string{"en": "/other"}, StyleBlock: "x"},
			},
		},
	}, "/no-match"), "non-matching URL should yield empty CSS")
}

func TestOrchestrator_RenderASTToHTML_NoRenderer(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator()
	html, err := o.renderASTToHTML(t.Context(), &ast_domain.TemplateAST{}, &templater_dto.InternalMetadata{}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "renderer not configured")
	assert.Empty(t, html)
}

func TestBuildImportMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expected map[string]string
		name     string
		source   string
	}{
		{
			name:   "standard import",
			source: `package main; import "fmt"`,
			expected: map[string]string{
				"fmt": "fmt",
			},
		},
		{
			name:   "aliased import",
			source: `package main; import f "fmt"`,
			expected: map[string]string{
				"f": "fmt",
			},
		},
		{
			name:     "blank import skipped",
			source:   `package main; import _ "fmt"`,
			expected: map[string]string{},
		},
		{
			name:     "dot import skipped",
			source:   `package main; import . "fmt"`,
			expected: map[string]string{},
		},
		{
			name:   "nested package path uses last segment",
			source: `package main; import "encoding/json"`,
			expected: map[string]string{
				"json": "encoding/json",
			},
		},
		{
			name:     "no imports",
			source:   `package main`,
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.source, parser.ImportsOnly)
			require.NoError(t, err)

			result := buildImportMap(file)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetQualifiedHoverContent(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	source := `package main; import "fmt"; import "time"`
	file, err := parser.ParseFile(fset, "test.go", source, parser.ImportsOnly)
	require.NoError(t, err)

	stdlibData := &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"fmt": {
				Path: "fmt",
				Name: "fmt",
				NamedTypes: map[string]*inspector_dto.Type{
					"Stringer": {
						Name:                 "Stringer",
						UnderlyingTypeString: "interface{String() string}",
						Methods:              nil,
						Fields:               nil,
						TypeString:           "",
						IsAlias:              false,
						DefinedInFilePath:    "",
						DefinitionLine:       0,
						DefinitionColumn:     0,
					},
				},
				Funcs: map[string]*inspector_dto.Function{
					"Println": {
						Name:               "Println",
						TypeString:         "(a ...any) (n int, err error)",
						DefinitionFilePath: "",
						DefinitionLine:     0,
						DefinitionColumn:   0,
					},
				},
				FileImports: nil,
			},
		},
	}

	t.Run("nil stdlibData returns empty", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, getQualifiedHoverContent("fmt", "Println", file, nil))
	})

	t.Run("unknown package alias returns empty", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, getQualifiedHoverContent("unknown", "Println", file, stdlibData))
	})

	t.Run("unknown package path returns empty", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, getQualifiedHoverContent("time", "Now", file, stdlibData))
	})

	t.Run("unknown symbol returns empty", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, getQualifiedHoverContent("fmt", "Unknown", file, stdlibData))
	})

	t.Run("finds type", func(t *testing.T) {
		t.Parallel()
		result := getQualifiedHoverContent("fmt", "Stringer", file, stdlibData)
		assert.Contains(t, result, "Stringer")
		assert.Contains(t, result, "fmt")
	})

	t.Run("finds function", func(t *testing.T) {
		t.Parallel()
		result := getQualifiedHoverContent("fmt", "Println", file, stdlibData)
		assert.Contains(t, result, "Println")
		assert.Contains(t, result, "fmt")
	})
}

func TestGetPackageHoverContent(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	source := `package main; import "fmt"; import "os"`
	file, err := parser.ParseFile(fset, "test.go", source, parser.ImportsOnly)
	require.NoError(t, err)

	stdlibData := &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"fmt": {
				Path: "fmt",
				Name: "fmt",
				NamedTypes: map[string]*inspector_dto.Type{
					"Stringer": {
						Name:                 "Stringer",
						UnderlyingTypeString: "interface{...}",
						Methods:              nil,
						Fields:               nil,
						TypeString:           "",
						IsAlias:              false,
						DefinedInFilePath:    "",
						DefinitionLine:       0,
						DefinitionColumn:     0,
					},
				},
				Funcs: map[string]*inspector_dto.Function{
					"Println": {
						Name:               "Println",
						TypeString:         "(a ...any) (n int, err error)",
						DefinitionFilePath: "",
						DefinitionLine:     0,
						DefinitionColumn:   0,
					},
				},
				FileImports: nil,
			},
		},
	}

	t.Run("nil stdlibData returns empty", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, getPackageHoverContent("fmt", file, nil))
	})

	t.Run("unknown alias returns empty", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, getPackageHoverContent("unknown", file, stdlibData))
	})

	t.Run("package not in stdlib returns basic info", func(t *testing.T) {
		t.Parallel()
		result := getPackageHoverContent("os", file, stdlibData)
		assert.Contains(t, result, "package os")
		assert.Contains(t, result, "os")
	})

	t.Run("known package returns full info", func(t *testing.T) {
		t.Parallel()
		result := getPackageHoverContent("fmt", file, stdlibData)
		assert.Contains(t, result, "package fmt")
		assert.Contains(t, result, "1 types")
		assert.Contains(t, result, "1 functions")
	})
}

func TestGetHoverContent(t *testing.T) {
	t.Parallel()

	t.Run("finds local type", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		source := `package main; type Foo struct { X int }`
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		result := getHoverContent("Foo", file, nil)
		assert.Contains(t, result, "type Foo")
	})

	t.Run("finds local function", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		source := `package main; func DoStuff() {}`
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		result := getHoverContent("DoStuff", file, nil)
		assert.Contains(t, result, "func DoStuff")
	})

	t.Run("falls through to stdlib search", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		source := `package main`
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		stdlibData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"time": {
					Path: "time",
					Name: "time",
					NamedTypes: map[string]*inspector_dto.Type{
						"Duration": {
							Name:                 "Duration",
							UnderlyingTypeString: "int64",
							Methods:              nil,
							Fields:               nil,
							TypeString:           "",
							IsAlias:              false,
							DefinedInFilePath:    "",
							DefinitionLine:       0,
							DefinitionColumn:     0,
						},
					},
					Funcs:       nil,
					FileImports: nil,
				},
			},
		}

		result := getHoverContent("Duration", file, stdlibData)
		assert.Contains(t, result, "Duration")
		assert.Contains(t, result, "int64")
	})

	t.Run("returns empty when nothing found", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		source := `package main`
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		result := getHoverContent("NonExistent", file, nil)
		assert.Empty(t, result)
	})
}

func TestFindLocalTypeContent(t *testing.T) {
	t.Parallel()

	t.Run("skips non-type declarations", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		source := `package main; var x = 1; const y = 2`
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		result := findLocalTypeContent("x", file)
		assert.Empty(t, result)
	})

	t.Run("finds type declaration", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		source := `package main; type MyType struct {}`
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		result := findLocalTypeContent("MyType", file)
		assert.Contains(t, result, "type MyType")
	})

	t.Run("returns empty for non-matching name", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		source := `package main; type MyType struct {}`
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		result := findLocalTypeContent("Other", file)
		assert.Empty(t, result)
	})
}

func TestFindLocalFunctionContent(t *testing.T) {
	t.Parallel()

	t.Run("finds function", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		source := `package main; func DoWork() {}`
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		result := findLocalFunctionContent("DoWork", file)
		assert.Contains(t, result, "func DoWork")
	})

	t.Run("skips methods with receiver", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		source := `package main; type S struct{}; func (s S) Method() {}`
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		result := findLocalFunctionContent("Method", file)
		assert.Empty(t, result)
	})

	t.Run("returns empty for non-matching name", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		source := `package main; func DoWork() {}`
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		result := findLocalFunctionContent("Missing", file)
		assert.Empty(t, result)
	})

	t.Run("skips non-func declarations", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		source := `package main; var x = 1`
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		result := findLocalFunctionContent("x", file)
		assert.Empty(t, result)
	})
}

func TestFindHoverTarget(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for no identifier at position", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "test.go", `package main`, parser.ParseComments)
		require.NoError(t, err)

		target := findHoverTarget(file, token.Pos(9999))
		assert.Nil(t, target)
	})

	t.Run("finds simple identifier", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		source := `package main

var myVar = 42`
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		var identPos token.Pos
		ast.Inspect(file, func(n ast.Node) bool {
			if identifier, ok := n.(*ast.Ident); ok && identifier.Name == "myVar" {
				identPos = identifier.Pos()
				return false
			}
			return true
		})
		require.NotEqual(t, token.Pos(0), identPos)

		target := findHoverTarget(file, identPos)
		require.NotNil(t, target)
		assert.Equal(t, "myVar", target.identifier.Name)
		assert.Nil(t, target.selector)
	})

	t.Run("finds selector expression", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		source := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}`
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		var selPos token.Pos
		ast.Inspect(file, func(n ast.Node) bool {
			if selectorExpression, ok := n.(*ast.SelectorExpr); ok {
				if selectorExpression.Sel.Name == "Println" {
					selPos = selectorExpression.Sel.Pos()
					return false
				}
			}
			return true
		})
		require.NotEqual(t, token.Pos(0), selPos)

		target := findHoverTarget(file, selPos)
		require.NotNil(t, target)
		assert.Equal(t, "Println", target.identifier.Name)
		assert.NotNil(t, target.selector)
	})

	t.Run("finds package name in selector expression", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		source := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}`
		file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
		require.NoError(t, err)

		var fmtPos token.Pos
		ast.Inspect(file, func(n ast.Node) bool {
			if selectorExpression, ok := n.(*ast.SelectorExpr); ok {
				if xIdent, ok := selectorExpression.X.(*ast.Ident); ok && xIdent.Name == "fmt" {
					fmtPos = xIdent.Pos()
					return false
				}
			}
			return true
		})
		require.NotEqual(t, token.Pos(0), fmtPos)

		target := findHoverTarget(file, fmtPos)
		require.NotNil(t, target)
		assert.Equal(t, "fmt", target.identifier.Name)
		assert.Nil(t, target.selector)
	})
}

func TestMatchSimpleIdent_ExistingTarget(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	source := `package main; var x = 1`
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	var identifier *ast.Ident
	ast.Inspect(file, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name == "x" {
			identifier = id
			return false
		}
		return true
	})
	require.NotNil(t, identifier)

	existing := &hoverTarget{
		identifier: identifier,
		selector:   nil,
	}

	result := matchSimpleIdent(identifier, identifier.Pos(), existing)
	assert.Nil(t, result)

	result = matchSimpleIdent(identifier, identifier.Pos(), nil)
	require.NotNil(t, result)
	assert.Equal(t, "x", result.identifier.Name)
}

func TestMatchSelectorExpr_OutOfRange(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	source := `package main; import "fmt"; func main() { fmt.Println("hi") }`
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	var selectorExpression *ast.SelectorExpr
	ast.Inspect(file, func(n ast.Node) bool {
		if s, ok := n.(*ast.SelectorExpr); ok {
			selectorExpression = s
			return false
		}
		return true
	})
	require.NotNil(t, selectorExpression)

	result := matchSelectorExpr(selectorExpression, token.Pos(9999))
	assert.Nil(t, result)

	result = matchSelectorExpr(&ast.BasicLit{ValuePos: token.Pos(1), Kind: token.INT, Value: "42"}, token.Pos(1))
	assert.Nil(t, result)
}

func TestIsDynamicSegment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		seg      string
		expected bool
	}{
		{name: "valid dynamic", seg: "{slug}", expected: true},
		{name: "catch-all is not dynamic", seg: "{path*}", expected: false},
		{name: "too short", seg: "{}", expected: false},
		{name: "single char", seg: "{x}", expected: true},
		{name: "no braces", seg: "slug", expected: false},
		{name: "empty", seg: "", expected: false},
		{name: "open only", seg: "{x", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isDynamicSegment(tt.seg))
		})
	}
}

func TestIsCatchAllSegment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		seg      string
		expected bool
	}{
		{name: "valid catch-all", seg: "{path*}", expected: true},
		{name: "dynamic is not catch-all", seg: "{slug}", expected: false},
		{name: "too short", seg: "{*}", expected: false},
		{name: "empty", seg: "", expected: false},
		{name: "no braces", seg: "path*}", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isCatchAllSegment(tt.seg))
		})
	}
}

func TestMatchSegments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		patternSegs []string
		urlSegs     []string
		expected    bool
	}{
		{
			name:        "exact match",
			patternSegs: []string{"about"},
			urlSegs:     []string{"about"},
			expected:    true,
		},
		{
			name:        "length mismatch",
			patternSegs: []string{"about", "us"},
			urlSegs:     []string{"about"},
			expected:    false,
		},
		{
			name:        "dynamic segment",
			patternSegs: []string{"blog", "{slug}"},
			urlSegs:     []string{"blog", "hello"},
			expected:    true,
		},
		{
			name:        "dynamic segment empty value",
			patternSegs: []string{"blog", "{slug}"},
			urlSegs:     []string{"blog", ""},
			expected:    false,
		},
		{
			name:        "catch-all segment",
			patternSegs: []string{"docs", "{path*}"},
			urlSegs:     []string{"docs", "a", "b", "c"},
			expected:    true,
		},
		{
			name:        "static mismatch",
			patternSegs: []string{"about"},
			urlSegs:     []string{"contact"},
			expected:    false,
		},
		{
			name:        "pattern longer than URL",
			patternSegs: []string{"a", "b"},
			urlSegs:     []string{"a"},
			expected:    false,
		},
		{
			name:        "URL longer than pattern",
			patternSegs: []string{"a"},
			urlSegs:     []string{"a", "b"},
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, matchSegments(tt.patternSegs, tt.urlSegs))
		})
	}
}

func TestAssemblePreviewTemplate(t *testing.T) {
	t.Parallel()

	result := assemblePreviewTemplate("<p>hello</p>", "fmt.Println()")
	assert.Equal(t, "<script>\nfmt.Println()\n</script>\n\n<p>hello</p>", result)
}

func TestFindCompletionsAtPosition_NilFile(t *testing.T) {
	t.Parallel()

	stdlibData := &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"fmt": {
				Path: "fmt",
				Name: "fmt",
				NamedTypes: map[string]*inspector_dto.Type{
					"Stringer": {
						Name:                 "Stringer",
						UnderlyingTypeString: "interface{...}",
						Methods:              nil,
						Fields:               nil,
						TypeString:           "",
						IsAlias:              false,
						DefinedInFilePath:    "",
						DefinitionLine:       0,
						DefinitionColumn:     0,
					},
				},
				Funcs: map[string]*inspector_dto.Function{
					"Println": {
						Name:               "Println",
						TypeString:         "(a ...any) (n int, err error)",
						DefinitionFilePath: "",
						DefinitionLine:     0,
						DefinitionColumn:   0,
					},
				},
				FileImports: nil,
			},
		},
	}

	t.Run("scope completions with nil file", func(t *testing.T) {
		t.Parallel()
		source := `package main
func main() {
	x
}`
		items := findCompletionsAtPosition(nil, source, 3, 3, stdlibData)
		assert.NotEmpty(t, items)
	})

	t.Run("package member completions with nil file", func(t *testing.T) {
		t.Parallel()
		source := `package main
import "fmt"
func main() {
	fmt.Pr
}`
		items := findCompletionsAtPosition(nil, source, 4, 8, stdlibData)
		require.NotEmpty(t, items)

		var found bool
		for _, item := range items {
			if item.Label == "Println" {
				found = true
				break
			}
		}
		assert.True(t, found, "should find Println via regex-extracted imports")
	})

	t.Run("field method context returns nil", func(t *testing.T) {
		t.Parallel()
		source := `package main
func main() {
	myVar.Fie
}`
		items := findCompletionsAtPosition(nil, source, 3, 11, nil)
		assert.Nil(t, items)
	})
}

func TestExtractTypesFromGenDecl_NonTypeToken(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	source := `package main; var x = 1; const y = 2`
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	info := &wasm_dto.ScriptBlockInfo{
		PropsType: "",
		Types:     make([]string, 0),
		HasInit:   false,
	}

	for _, declaration := range file.Decls {
		genDecl, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		extractTypesFromGenDecl(genDecl, info)
	}

	assert.Empty(t, info.Types)
}

func TestExtractInitFromFuncDecl_MethodReceiver(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	source := `package main; type S struct{}; func (s S) init() {}`
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	info := &wasm_dto.ScriptBlockInfo{
		PropsType: "",
		Types:     make([]string, 0),
		HasInit:   false,
	}

	for _, declaration := range file.Decls {
		funcDecl, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		extractInitFromFuncDecl(funcDecl, info)
	}

	assert.False(t, info.HasInit, "method with receiver should not set HasInit")
}

func TestExtractInitFromFuncDecl_RegularFunc(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	source := `package main; func notInit() {}`
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	info := &wasm_dto.ScriptBlockInfo{
		PropsType: "",
		Types:     make([]string, 0),
		HasInit:   false,
	}

	for _, declaration := range file.Decls {
		funcDecl, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		extractInitFromFuncDecl(funcDecl, info)
	}

	assert.False(t, info.HasInit, "non-init func should not set HasInit")
}

func TestExtractDeclInfo_VarDecl(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	source := `package main; var x = 1`
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	info := &wasm_dto.ScriptBlockInfo{
		PropsType: "",
		Types:     make([]string, 0),
		HasInit:   false,
	}

	for _, declaration := range file.Decls {
		extractDeclInfo(declaration, info)
	}

	assert.Empty(t, info.Types)
	assert.False(t, info.HasInit)
}

func TestExtractScriptBlockInfo_InvalidSource(t *testing.T) {
	t.Parallel()

	script := &sfcparserScript{
		Content: "this is not valid Go {{{",
	}

	info := extractScriptBlockInfoFromContent(script.Content)
	require.NotNil(t, info)
	assert.Empty(t, info.Types)
	assert.False(t, info.HasInit)
	assert.Empty(t, info.PropsType)
}

func TestConvertSFCResultToAST_NoScript(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator()
	response, err := o.ParseTemplate(t.Context(), &wasm_dto.ParseTemplateRequest{
		Template:   "<template><p>Hello</p></template>",
		Script:     "",
		ModuleName: "",
	})
	require.NoError(t, err)
	assert.True(t, response.Success)
	require.NotNil(t, response.AST)
	assert.NotEmpty(t, response.AST.Nodes)
	assert.Nil(t, response.AST.ScriptBlock)
}

func TestConvertSFCResultToAST_EmptyTemplate(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator()
	response, err := o.ParseTemplate(t.Context(), &wasm_dto.ParseTemplateRequest{
		Template:   "",
		Script:     "",
		ModuleName: "",
	})
	require.NoError(t, err)

	require.NotNil(t, response)
}

func TestOrchestrator_ParseTemplate_ParseError(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator()
	response, err := o.ParseTemplate(t.Context(), &wasm_dto.ParseTemplateRequest{
		Template:   "<template><p>hello</template>",
		Script:     "",
		ModuleName: "",
	})
	require.NoError(t, err)

	require.NotNil(t, response)
}

func TestFindMatchingPageFromManifest(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when no route matches", func(t *testing.T) {
		t.Parallel()
		artefacts := []wasm_dto.GeneratedArtefact{
			{Type: wasm_dto.ArtefactTypePage, SourcePath: "a.pk", Path: "a.go", Content: "pkg a"},
		}
		manifest := &wasm_dto.GeneratedManifest{
			Pages: map[string]wasm_dto.ManifestPageEntry{
				"a": {
					SourcePath:    "a.pk",
					PackagePath:   "m/pages/a",
					RoutePatterns: map[string]string{"en": "/about"},
					CachePolicy:   nil,
					StyleBlock:    "",
					JSArtefactIDs: nil,
					HasGetData:    false,
					HasRender:     false,
				},
			},
			Partials: nil,
		}
		art, pkg := findMatchingPageFromManifest(artefacts, manifest, "/contact")
		assert.Nil(t, art)
		assert.Empty(t, pkg)
	})

	t.Run("returns artefact when route matches", func(t *testing.T) {
		t.Parallel()
		artefacts := []wasm_dto.GeneratedArtefact{
			{Type: wasm_dto.ArtefactTypePage, SourcePath: "a.pk", Path: "a.go", Content: "pkg a"},
		}
		manifest := &wasm_dto.GeneratedManifest{
			Pages: map[string]wasm_dto.ManifestPageEntry{
				"a": {
					SourcePath:    "a.pk",
					PackagePath:   "m/pages/a",
					RoutePatterns: map[string]string{"en": "/about"},
					CachePolicy:   nil,
					StyleBlock:    "",
					JSArtefactIDs: nil,
					HasGetData:    false,
					HasRender:     false,
				},
			},
			Partials: nil,
		}
		art, pkg := findMatchingPageFromManifest(artefacts, manifest, "/about")
		require.NotNil(t, art)
		assert.Equal(t, "m/pages/a", pkg)
	})

	t.Run("source path not found in artefacts", func(t *testing.T) {
		t.Parallel()
		artefacts := []wasm_dto.GeneratedArtefact{
			{Type: wasm_dto.ArtefactTypePage, SourcePath: "other.pk", Path: "other.go", Content: "pkg other"},
		}
		manifest := &wasm_dto.GeneratedManifest{
			Pages: map[string]wasm_dto.ManifestPageEntry{
				"a": {
					SourcePath:    "a.pk",
					PackagePath:   "m/pages/a",
					RoutePatterns: map[string]string{"en": "/about"},
					CachePolicy:   nil,
					StyleBlock:    "",
					JSArtefactIDs: nil,
					HasGetData:    false,
					HasRender:     false,
				},
			},
			Partials: nil,
		}
		art, pkg := findMatchingPageFromManifest(artefacts, manifest, "/about")
		assert.Nil(t, art)
		assert.Empty(t, pkg)
	})
}

func TestValidateDynamicRenderAdapters(t *testing.T) {
	t.Parallel()

	t.Run("no generator", func(t *testing.T) {
		t.Parallel()
		o := NewOrchestrator()
		response := o.validateDynamicRenderAdapters(t.Context())
		require.NotNil(t, response)
		assert.Contains(t, response.Error, "generator not configured")
	})

	t.Run("no interpreter", func(t *testing.T) {
		t.Parallel()
		o := NewOrchestrator(WithGenerator(&stubGeneratorPort{
			response: nil,
			err:      nil,
		}))
		response := o.validateDynamicRenderAdapters(t.Context())
		require.NotNil(t, response)
		assert.Contains(t, response.Error, "interpreter not configured")
	})

	t.Run("both configured returns nil", func(t *testing.T) {
		t.Parallel()
		o := NewOrchestrator(
			WithGenerator(&stubGeneratorPort{
				response: nil,
				err:      nil,
			}),
			WithInterpreter(&stubInterpreterPort{
				response: nil,
				err:      nil,
			}),
		)
		response := o.validateDynamicRenderAdapters(t.Context())
		assert.Nil(t, response)
	})
}

func TestLineColumnToOffset_ValidFile(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	source := "package main\n\nfunc main() {}\n"
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	tokFile := fset.File(file.Pos())
	require.NotNil(t, tokFile)

	tests := []struct {
		name   string
		line   int
		column int
	}{
		{name: "first line first column", line: 1, column: 1},
		{name: "second line", line: 2, column: 1},
		{name: "third line", line: 3, column: 5},
		{name: "line below min clamped", line: 0, column: 1},
		{name: "line above max clamped", line: 999, column: 1},
		{name: "column zero", line: 1, column: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			offset := lineColumnToOffset(tokFile, tt.line, tt.column)
			assert.GreaterOrEqual(t, offset, 0)
			assert.LessOrEqual(t, offset, tokFile.Size())
		})
	}
}

func TestExtractImportsFromSource_BlockWithAliases(t *testing.T) {
	t.Parallel()

	stdlibData := &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"fmt":  {Path: "fmt", Name: "fmt", NamedTypes: nil, Funcs: nil, FileImports: nil},
			"time": {Path: "time", Name: "time", NamedTypes: nil, Funcs: nil, FileImports: nil},
		},
	}

	source := "import (\n\tf \"fmt\"\n\t\"time\"\n)"
	imports := extractImportsFromSource(source, stdlibData)
	assert.Equal(t, "fmt", imports["f"])
	assert.Equal(t, "time", imports["time"])
}

func TestOrchestrator_RenderPreview_DiagnosticsPropagated(t *testing.T) {
	t.Parallel()

	renderer := &stubRenderPort{
		renderFunc: func(_ context.Context, _ *wasm_dto.RenderFromSourcesRequest) (*wasm_dto.RenderFromSourcesResponse, error) {
			return &wasm_dto.RenderFromSourcesResponse{
				Success: true,
				HTML:    "<p>test</p>",
				CSS:     "",
				Error:   "",
				Diagnostics: []wasm_dto.Diagnostic{
					{Severity: "warning", Message: "unused var", Location: wasm_dto.Location{FilePath: "test.go", Line: 1, Column: 1}, Code: ""},
				},
				IsStaticOnly: false,
			}, nil
		},
		renderASTFunc: nil,
	}

	o := NewOrchestrator(WithRenderer(renderer))
	response, err := o.RenderPreview(t.Context(), &wasm_dto.RenderPreviewRequest{
		Template:   "<p>test</p>",
		Script:     "",
		PropsJSON:  "",
		ModuleName: "",
	})
	require.NoError(t, err)
	assert.True(t, response.Success)
	require.Len(t, response.Diagnostics, 1)
	assert.Equal(t, "unused var", response.Diagnostics[0].Message)
}

func TestOrchestrator_Analyse_NotInitialised(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(WithStdlibLoader(newMockStdlibLoader()))
	_, err := o.Analyse(t.Context(), &wasm_dto.AnalyseRequest{
		Sources:    map[string]string{"main.go": "package main"},
		ModuleName: "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not initialised")
}

func TestOrchestrator_Validate_MultipleErrors(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator()
	response, err := o.Validate(t.Context(), &wasm_dto.ValidateRequest{
		Source:   `package main; func main( { }; func other( { }`,
		FilePath: "multi.go",
	})
	require.NoError(t, err)
	assert.False(t, response.Valid)
	assert.GreaterOrEqual(t, len(response.Diagnostics), 1)

	for _, diagnostic := range response.Diagnostics {
		assert.Equal(t, "error", diagnostic.Severity)
		assert.Equal(t, "multi.go", diagnostic.Location.FilePath)
	}
}

func TestHeadlessRenderOptions_Construction(t *testing.T) {
	t.Parallel()

	opts := HeadlessRenderOptions{
		Template:               &ast_domain.TemplateAST{},
		Metadata:               &templater_dto.InternalMetadata{},
		Styling:                "body { margin: 0; }",
		IncludeDocumentWrapper: true,
	}
	assert.NotNil(t, opts.Template)
	assert.NotNil(t, opts.Metadata)
	assert.Equal(t, "body { margin: 0; }", opts.Styling)
	assert.True(t, opts.IncludeDocumentWrapper)
}

func TestConfig_Construction(t *testing.T) {
	t.Parallel()

	config := Config{
		DefaultModuleName: "mymod",
		StdlibPackages:    []string{"fmt", "time"},
		MaxSourceSize:     2048,
		EnableMetrics:     true,
	}
	assert.Equal(t, "mymod", config.DefaultModuleName)
	assert.Equal(t, []string{"fmt", "time"}, config.StdlibPackages)
	assert.Equal(t, 2048, config.MaxSourceSize)
	assert.True(t, config.EnableMetrics)
}

func TestExtractFunctionsFromPackage(t *testing.T) {
	t.Parallel()

	pkg := &inspector_dto.Package{
		Path: "test/pkg",
		Name: "pkg",
		Funcs: map[string]*inspector_dto.Function{
			"Exported": {
				Name:               "Exported",
				TypeString:         "(x int) error",
				DefinitionFilePath: "pkg.go",
				DefinitionLine:     10,
				DefinitionColumn:   1,
			},
			"unexported": {
				Name:               "unexported",
				TypeString:         "()",
				DefinitionFilePath: "pkg.go",
				DefinitionLine:     20,
				DefinitionColumn:   1,
			},
		},
		NamedTypes:  nil,
		FileImports: nil,
	}

	response := &wasm_dto.AnalyseResponse{
		Success:     false,
		Types:       make([]wasm_dto.TypeInfo, 0),
		Functions:   make([]wasm_dto.FunctionInfo, 0),
		Imports:     make([]wasm_dto.ImportInfo, 0),
		Diagnostics: nil,
		Error:       "",
	}

	extractFunctionsFromPackage(pkg, response)
	assert.Len(t, response.Functions, 2)
}

func TestExtractImportsFromPackage(t *testing.T) {
	t.Parallel()

	pkg := &inspector_dto.Package{
		Path:       "test/pkg",
		Name:       "pkg",
		NamedTypes: nil,
		Funcs:      nil,
		FileImports: map[string]map[string]string{
			"file1.go": {"fmt": "fmt", "os": "os"},
			"file2.go": {"time": "time"},
		},
	}

	response := &wasm_dto.AnalyseResponse{
		Success:     false,
		Types:       make([]wasm_dto.TypeInfo, 0),
		Functions:   make([]wasm_dto.FunctionInfo, 0),
		Imports:     make([]wasm_dto.ImportInfo, 0),
		Diagnostics: nil,
		Error:       "",
	}

	extractImportsFromPackage(pkg, response)
	assert.Len(t, response.Imports, 3)
}

func TestOrchestrator_Log_UnknownLevel(t *testing.T) {
	t.Parallel()

	calls := make(map[string]int)
	console := &trackingConsole{calls: calls}
	o := NewOrchestrator(WithConsole(console))

	assert.NotPanics(t, func() {
		o.log("trace", "should do nothing")
	})
	assert.Empty(t, calls)
}
