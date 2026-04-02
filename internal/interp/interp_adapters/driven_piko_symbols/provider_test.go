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

package driven_piko_symbols

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProviderExportsContainsExpectedPackages(t *testing.T) {
	t.Parallel()

	provider := NewProvider()
	exports := provider.Exports()

	expectedPackages := []string{
		"piko.sh/piko",
		"piko.sh/piko/wdk/binder",
		"piko.sh/piko/wdk/logger",
		"piko.sh/piko/wdk/runtime",
	}

	for _, pkg := range expectedPackages {
		_, ok := exports[pkg]
		require.True(t, ok, "expected package %q in exports", pkg)
	}
}

func TestProviderExportsContainsKeySymbols(t *testing.T) {
	t.Parallel()

	provider := NewProvider()
	exports := provider.Exports()

	tests := []struct {
		pkg  string
		name string
	}{

		{pkg: "piko.sh/piko/wdk/runtime", name: "GetArena"},
		{pkg: "piko.sh/piko/wdk/runtime", name: "PutArena"},
		{pkg: "piko.sh/piko/wdk/runtime", name: "GetDirectWriter"},
		{pkg: "piko.sh/piko/wdk/runtime", name: "EvaluateTruthiness"},
		{pkg: "piko.sh/piko/wdk/runtime", name: "ValueToString"},
		{pkg: "piko.sh/piko/wdk/runtime", name: "AppendDiagnostic"},
		{pkg: "piko.sh/piko/wdk/runtime", name: "BuildClassBytes4Arena"},
		{pkg: "piko.sh/piko/wdk/runtime", name: "EncodeActionPayloadBytes0Arena"},
		{pkg: "piko.sh/piko/wdk/runtime", name: "RegisterASTFunc"},

		{pkg: "piko.sh/piko/wdk/runtime", name: "TemplateAST"},
		{pkg: "piko.sh/piko/wdk/runtime", name: "TemplateNode"},
		{pkg: "piko.sh/piko/wdk/runtime", name: "DirectWriter"},
		{pkg: "piko.sh/piko/wdk/runtime", name: "HTMLAttribute"},
		{pkg: "piko.sh/piko/wdk/runtime", name: "RenderArena"},

		{pkg: "piko.sh/piko/wdk/runtime", name: "NodeElement"},
		{pkg: "piko.sh/piko/wdk/runtime", name: "NodeText"},

		{pkg: "piko.sh/piko/wdk/binder", name: "Bind"},
		{pkg: "piko.sh/piko/wdk/binder", name: "BindMap"},
		{pkg: "piko.sh/piko/wdk/binder", name: "IgnoreUnknownKeys"},

		{pkg: "piko.sh/piko/wdk/logger", name: "GetLogger"},
		{pkg: "piko.sh/piko/wdk/logger", name: "String"},
		{pkg: "piko.sh/piko/wdk/logger", name: "Error"},

		{pkg: "piko.sh/piko", name: "NotFound"},
		{pkg: "piko.sh/piko", name: "RegisterActions"},
	}

	for _, tt := range tests {
		t.Run(tt.pkg+"."+tt.name, func(t *testing.T) {
			t.Parallel()

			packageSymbols, ok := exports[tt.pkg]
			require.True(t, ok, "package %q not found", tt.pkg)

			value, ok := packageSymbols[tt.name]
			require.True(t, ok, "%s.%s not found", tt.pkg, tt.name)
			require.True(t, value.IsValid(), "%s.%s is invalid", tt.pkg, tt.name)
		})
	}
}

func TestProviderFunctionsAreCallable(t *testing.T) {
	t.Parallel()

	provider := NewProvider()
	exports := provider.Exports()

	truthiness := exports["piko.sh/piko/wdk/runtime"]["EvaluateTruthiness"]
	require.Equal(t, reflect.Func, truthiness.Kind())

	result := truthiness.Call([]reflect.Value{reflect.ValueOf("hello")})
	require.Len(t, result, 1)
	require.True(t, result[0].Bool())

	result = truthiness.Call([]reflect.Value{reflect.ValueOf("")})
	require.Len(t, result, 1)
	require.False(t, result[0].Bool())
}

func TestProviderImplementsInterface(t *testing.T) {
	t.Parallel()

	provider := NewProvider()
	require.NotNil(t, provider.Exports())
}
