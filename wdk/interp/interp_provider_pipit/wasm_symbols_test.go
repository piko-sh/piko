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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"pipit.sh/pipit"
)

type staticSymbolProvider struct {
	exports pipit.SymbolExports
}

func (p *staticSymbolProvider) Exports() pipit.SymbolExports {
	return p.exports
}

func TestIsWASMUnsafePackage(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		path string
		want bool
	}{
		{path: "unsafe", want: true},
		{path: "os/exec", want: true},
		{path: "net", want: true},
		{path: "net/http/cgi", want: true},
		{path: "net/netip", want: true},
		{path: "syscall/js", want: true},
		{path: "strings", want: false},
		{path: "network", want: false},
		{path: "os", want: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.path, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, isWASMUnsafePackage(testCase.path))
		})
	}
}

func TestWASMFilteredProviderDropsUnsafePackages(t *testing.T) {
	t.Parallel()
	source := &staticSymbolProvider{exports: pipit.SymbolExports{
		"strings": {"ToUpper": reflect.ValueOf(func(string) string { return "" })},
		"os/exec": {"Command": reflect.ValueOf(func() {})},
		"net/rpc": {"Dial": reflect.ValueOf(func() {})},
	}}
	filtered := (&wasmFilteredProvider{source: source}).Exports()
	assert.Contains(t, filtered, "strings")
	assert.NotContains(t, filtered, "os/exec")
	assert.NotContains(t, filtered, "net/rpc")
}

func TestWASMSymbolProvidersLeaveOutUnsafePackages(t *testing.T) {
	t.Parallel()
	providers := wasmSymbolProviders()
	assert.NotEmpty(t, providers)
	for _, provider := range providers {
		for path := range provider.Exports() {
			assert.False(t, isWASMUnsafePackage(path), "unsafe package %q leaked into the WASM symbol tables", path)
		}
	}
}
