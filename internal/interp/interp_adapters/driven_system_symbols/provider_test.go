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

package driven_system_symbols

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
		"bytes",
		"context",
		"encoding/base64",
		"encoding/json",
		"errors",
		"fmt",
		"io",
		"math",
		"os",
		"path",
		"regexp",
		"sort",
		"strconv",
		"strings",
		"sync",
	}

	for _, pkg := range expectedPackages {
		_, ok := exports[pkg]
		require.True(t, ok, "expected package %q in exports", pkg)
	}
}

func TestProviderExportsContainsKeyFunctions(t *testing.T) {
	t.Parallel()

	provider := NewProvider()
	exports := provider.Exports()

	tests := []struct {
		pkg  string
		name string
	}{
		{pkg: "fmt", name: "Sprintf"},
		{pkg: "fmt", name: "Println"},
		{pkg: "fmt", name: "Errorf"},
		{pkg: "strings", name: "ToUpper"},
		{pkg: "strings", name: "Contains"},
		{pkg: "strings", name: "Join"},
		{pkg: "strings", name: "Split"},
		{pkg: "strconv", name: "Itoa"},
		{pkg: "strconv", name: "Atoi"},
		{pkg: "math", name: "Sqrt"},
		{pkg: "sort", name: "Ints"},
		{pkg: "errors", name: "New"},
		{pkg: "encoding/json", name: "Marshal"},
		{pkg: "encoding/json", name: "Unmarshal"},
		{pkg: "regexp", name: "Compile"},
		{pkg: "path", name: "Join"},
		{pkg: "io", name: "Copy"},
		{pkg: "bytes", name: "NewBuffer"},
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

	toUpper := exports["strings"]["ToUpper"]
	require.Equal(t, reflect.Func, toUpper.Kind())

	result := toUpper.Call([]reflect.Value{reflect.ValueOf("hello")})
	require.Len(t, result, 1)
	require.Equal(t, "HELLO", result[0].String())

	sprintf := exports["fmt"]["Sprintf"]
	require.Equal(t, reflect.Func, sprintf.Kind())

	result = sprintf.Call([]reflect.Value{
		reflect.ValueOf("value: %d"),
		reflect.ValueOf(42),
	})
	require.Len(t, result, 1)
	require.Equal(t, "value: 42", result[0].String())
}

func TestProviderImplementsInterface(t *testing.T) {
	t.Parallel()

	provider := NewProvider()
	require.NotNil(t, provider.Exports())
}
