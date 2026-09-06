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

//go:build js && wasm

package wasm_adapters

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/internal/wasm/wasm_domain"
	"piko.sh/piko/internal/wasm/wasm_dto"
)

type fakeProgramInterpreter struct {
	compile func(packagePath string) error
}

func (f *fakeProgramInterpreter) CompileAndExecute(_ context.Context, _ string, packagePath string, _ map[string]string) error {
	return f.compile(packagePath)
}

type fakeInterpreterFactory struct {
	interpreter wasm_domain.ProgramInterpreterPort
}

func (f *fakeInterpreterFactory) NewInterpreter() wasm_domain.ProgramInterpreterPort {
	return f.interpreter
}

func registerBuilder(packagePath string, diagnostics []*generator_dto.RuntimeDiagnostic, ast *ast_domain.TemplateAST) {
	templater_domain.RegisterASTFunc(packagePath, func(*templater_dto.RequestData, any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return ast, templater_dto.InternalMetadata{}, diagnostics
	})
}

func TestInterpreterAdapterInterpret(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		compile     func(packagePath string) error
		name        string
		packagePath string
		wantError   string
		staleFirst  bool
		wantSuccess bool
	}{
		{
			name:        "program registers its builder",
			packagePath: "playground/pages/registers",
			compile: func(packagePath string) error {
				registerBuilder(packagePath, nil, &ast_domain.TemplateAST{})
				return nil
			},
			wantSuccess: true,
		},
		{
			name:        "stale builder from an earlier request is never served",
			packagePath: "playground/pages/stale",
			staleFirst:  true,
			compile:     func(string) error { return nil },
			wantError:   "BuildAST not registered",
		},
		{
			name:        "compile failure",
			packagePath: "playground/pages/broken",
			compile:     func(string) error { return errors.New("does not compile") },
			wantError:   "batch compilation failed",
		},
		{
			name:        "interpreter panic",
			packagePath: "playground/pages/panics",
			compile:     func(string) error { panic("interpreter exploded") },
			wantError:   "interpreter panicked",
		},
		{
			name:        "nil AST with diagnostics",
			packagePath: "playground/pages/diagnostics",
			compile: func(packagePath string) error {
				registerBuilder(packagePath, []*generator_dto.RuntimeDiagnostic{{Message: "bad", Severity: generator_dto.Error}}, nil)
				return nil
			},
			wantError: "BuildAST returned nil AST",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if testCase.staleFirst {
				registerBuilder(testCase.packagePath, nil, &ast_domain.TemplateAST{})
			}
			adapter := NewInterpreterAdapter(WithInterpreterFactory(&fakeInterpreterFactory{
				interpreter: &fakeProgramInterpreter{compile: testCase.compile},
			}))
			response, err := adapter.Interpret(context.Background(), &wasm_dto.InterpretRequest{
				PackagePath: testCase.packagePath,
				RequestURL:  "https://example.com/page?tab=one",
			})
			require.NoError(t, err)
			require.NotNil(t, response)
			assert.Equal(t, testCase.wantSuccess, response.Success)
			if testCase.wantError != "" {
				assert.Contains(t, response.Error, testCase.wantError)
				assert.NotContains(t, response.Error, "goroutine")
			}
		})
	}
}

func TestInterpreterAdapterWithoutFactory(t *testing.T) {
	t.Parallel()
	response, err := NewInterpreterAdapter().Interpret(context.Background(), &wasm_dto.InterpretRequest{})
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "not configured")
}

func TestConvertRuntimeDiagnostics(t *testing.T) {
	t.Parallel()
	assert.Nil(t, convertRuntimeDiagnostics(nil))
	converted := convertRuntimeDiagnostics([]*generator_dto.RuntimeDiagnostic{
		nil,
		{Message: "debug", Severity: generator_dto.Debug},
		{Message: "info", Severity: generator_dto.Info},
		{Message: "warning", Severity: generator_dto.Warning},
		{Message: "error", Severity: generator_dto.Error},
		{Message: "unknown", Severity: generator_dto.Severity(99)},
	})
	require.Len(t, converted, 5)
	for _, diagnostic := range converted {
		if diagnostic.Message == "unknown" {
			assert.Equal(t, "unknown", diagnostic.Severity)
			continue
		}
		assert.Equal(t, diagnostic.Message, diagnostic.Severity)
	}
}
