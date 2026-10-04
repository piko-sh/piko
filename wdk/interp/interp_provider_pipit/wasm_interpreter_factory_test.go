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

package interp_provider_pipit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWASMInterpreterFactoryClonesOneGoldenInterpreter(t *testing.T) {
	t.Parallel()
	factory := NewWASMInterpreterFactory()
	first, ok := factory.NewInterpreter().(*wasmProgramInterpreter)
	require.True(t, ok)
	second, ok := factory.NewInterpreter().(*wasmProgramInterpreter)
	require.True(t, ok)
	assert.NotSame(t, first.interpreter, second.interpreter)
	assert.Same(t, factory.golden(), factory.golden())
}

func TestWASMProgramInterpreterCompileAndExecute(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		dependencies map[string]string
		name         string
		mainCode     string
		wantErr      string
	}{
		{
			name:         "program with a dependency",
			mainCode:     "package home\n\nimport \"playground/pages/shared\"\n\nvar Value = shared.Value\n",
			dependencies: map[string]string{"playground/pages/shared": "package shared\n\nvar Value = 1\n"},
		},
		{
			name:     "compile failure",
			mainCode: "package home\n\nvar Value int = \"text\"\n",
			wantErr:  "interpreting module",
		},
		{
			name:     "init failure",
			mainCode: "package home\n\nfunc init() {\n\tpanic(\"broken\")\n}\n",
			wantErr:  "initialising module",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			interpreter := NewWASMInterpreterFactory().NewInterpreter()
			err := interpreter.CompileAndExecute(context.Background(), testCase.mainCode, "playground/pages/home", testCase.dependencies)
			if testCase.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, testCase.wantErr)
		})
	}
}

func TestWASMProgramInterpreterRecoversPanics(t *testing.T) {
	t.Parallel()
	interpreter := &wasmProgramInterpreter{interpreter: nil}
	var err error
	require.NotPanics(t, func() {
		err = interpreter.CompileAndExecute(context.Background(), "package home\n", "playground/pages/home", nil)
	})
	require.ErrorIs(t, err, errInterpreterPanic)
}

func TestExtractModulePath(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		packagePath string
		want        string
	}{
		{packagePath: "playground/internal/pages/home", want: "playground"},
		{packagePath: "playground", want: "playground"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.packagePath, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, extractModulePath(testCase.packagePath))
		})
	}
}
