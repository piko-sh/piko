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

package interp_domain

import (
	"reflect"
	"strings"
	"testing"
)

func helperRegistry() *SymbolRegistry {
	return NewSymbolRegistry(SymbolExports{
		"net/http": {
			"Get":  reflect.ValueOf(func(string) any { return nil }),
			"Post": reflect.ValueOf(func(string) any { return nil }),
			"Head": reflect.ValueOf(func(string) any { return nil }),
		},
		"strings": {
			"ToUpper":  reflect.ValueOf(strings.ToUpper),
			"Contains": reflect.ValueOf(strings.Contains),
		},
	})
}

func TestScopedNilAllowlistReturnsParent(t *testing.T) {
	t.Parallel()
	parent := helperRegistry()
	scoped := parent.Scoped(nil)
	if scoped != parent {
		t.Fatalf("nil allowlist should return parent registry")
	}
}

func TestScopedEmptyAllowlistExposesNothing(t *testing.T) {
	t.Parallel()
	parent := helperRegistry()
	scoped := parent.Scoped([]string{})
	if _, ok := scoped.PackageSymbols("net/http"); ok {
		t.Fatalf("empty allowlist should hide net/http")
	}
	if _, ok := scoped.PackageSymbols("strings"); ok {
		t.Fatalf("empty allowlist should hide strings")
	}
}

func TestScopedWildcardExposesPackage(t *testing.T) {
	t.Parallel()
	parent := helperRegistry()
	scoped := parent.Scoped([]string{"net/http/*"})

	httpPkg, ok := scoped.PackageSymbols("net/http")
	if !ok {
		t.Fatalf("net/http should be in scoped registry")
	}
	for _, want := range []string{"Get", "Post", "Head"} {
		if _, ok := httpPkg[want]; !ok {
			t.Errorf("net/http.%s missing under wildcard", want)
		}
	}
	if _, ok := scoped.PackageSymbols("strings"); ok {
		t.Fatalf("strings should NOT be in scoped registry")
	}
}

func TestScopedSpecificSymbols(t *testing.T) {
	t.Parallel()
	parent := helperRegistry()
	scoped := parent.Scoped([]string{
		"net/http.Get",
		"strings.ToUpper",
	})

	httpPkg, ok := scoped.PackageSymbols("net/http")
	if !ok {
		t.Fatalf("net/http should be in scoped registry")
	}
	if _, ok := httpPkg["Get"]; !ok {
		t.Errorf("net/http.Get missing")
	}
	if _, ok := httpPkg["Post"]; ok {
		t.Errorf("net/http.Post leaked via named allowlist")
	}

	stringsPkg, _ := scoped.PackageSymbols("strings")
	if _, ok := stringsPkg["ToUpper"]; !ok {
		t.Errorf("strings.ToUpper missing")
	}
	if _, ok := stringsPkg["Contains"]; ok {
		t.Errorf("strings.Contains leaked via named allowlist")
	}
}

func TestScopedShareTypeCaches(t *testing.T) {
	t.Parallel()
	parent := helperRegistry()
	parent.SynthesiseAll()
	scoped := parent.Scoped([]string{"strings/*"})
	if scoped.reflectToTypes == nil {
		t.Fatalf("scoped registry must share reflectToTypes")
	}
	if scoped.synthesised == nil {
		t.Fatalf("scoped registry must share synthesised cache")
	}
}

func TestParseAllowlistEntry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input      string
		wantPath   string
		wantSymbol string
	}{
		{input: "net/http.Get", wantPath: "net/http", wantSymbol: "Get"},
		{input: "strings.ToUpper", wantPath: "strings", wantSymbol: "ToUpper"},
		{input: "net/http/*", wantPath: "net/http", wantSymbol: "*"},
		{input: "strings", wantPath: "strings", wantSymbol: ""},
		{input: "", wantPath: "", wantSymbol: ""},
		{input: "example.com/foo/v2.Type", wantPath: "example.com/foo/v2", wantSymbol: "Type"},
	}
	for _, tt := range tests {
		gotPath, gotSymbol := parseAllowlistEntry(tt.input)
		if gotPath != tt.wantPath || gotSymbol != tt.wantSymbol {
			t.Errorf("parseAllowlistEntry(%q) = (%q,%q), want (%q,%q)",
				tt.input, gotPath, gotSymbol, tt.wantPath, tt.wantSymbol)
		}
	}
}
