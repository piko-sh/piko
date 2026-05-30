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
	"testing"
)

func helperRestrictedRegistry() *SymbolRegistry {
	return NewSymbolRegistry(SymbolExports{
		"reflect": {
			"TypeOf":    reflect.ValueOf(reflect.TypeOf),
			"ValueOf":   reflect.ValueOf(reflect.ValueOf),
			"DeepEqual": reflect.ValueOf(reflect.DeepEqual),
			"MakeMap":   reflect.ValueOf(reflect.MakeMap),
			"StructOf":  reflect.ValueOf(reflect.StructOf),
		},
		"runtime/debug": {
			"PrintStack": reflect.ValueOf(func() {}),
		},
		"strings": {
			"ToUpper":  reflect.ValueOf(func(s string) string { return s }),
			"Contains": reflect.ValueOf(func(string, string) bool { return false }),
		},
	})
}

func TestApplyRestrictedSurfaceFullyDeniedRemovesPackage(t *testing.T) {
	t.Parallel()
	registry := helperRestrictedRegistry()
	policy := RestrictedPackageSet{
		FullyDenied: map[string]struct{}{"runtime/debug": {}},
	}
	removed := ApplyRestrictedSurface(registry, policy)
	if removed != 1 {
		t.Fatalf("expected 1 symbol removed, got %d", removed)
	}
	if _, ok := registry.PackageSymbols("runtime/debug"); ok {
		t.Fatalf("runtime/debug should be fully denied")
	}
}

func TestApplyRestrictedSurfaceAllowsListedSymbols(t *testing.T) {
	t.Parallel()
	registry := helperRestrictedRegistry()
	policy := RestrictedPackageSet{
		Allowed: map[string]map[string]struct{}{
			"reflect": {
				"TypeOf":    {},
				"DeepEqual": {},
			},
		},
	}
	removed := ApplyRestrictedSurface(registry, policy)
	if removed == 0 {
		t.Fatalf("expected some symbols removed; got 0")
	}
	reflectPkg, ok := registry.PackageSymbols("reflect")
	if !ok {
		t.Fatalf("reflect should still be registered")
	}
	if _, ok := reflectPkg["TypeOf"]; !ok {
		t.Errorf("reflect.TypeOf should be allowed")
	}
	if _, ok := reflectPkg["MakeMap"]; ok {
		t.Errorf("reflect.MakeMap should have been removed")
	}
	if _, ok := reflectPkg["StructOf"]; ok {
		t.Errorf("reflect.StructOf should have been removed")
	}
}

func TestApplyRestrictedSurfacePackagesNotInPolicyAreUntouched(t *testing.T) {
	t.Parallel()
	registry := helperRestrictedRegistry()
	policy := RestrictedPackageSet{
		FullyDenied: map[string]struct{}{"runtime/debug": {}},
	}
	ApplyRestrictedSurface(registry, policy)
	stringsPkg, ok := registry.PackageSymbols("strings")
	if !ok {
		t.Fatalf("strings should still be registered")
	}
	if len(stringsPkg) != 2 {
		t.Errorf("strings should retain all entries, got %d", len(stringsPkg))
	}
}

func TestDefaultRestrictedSurfaceShape(t *testing.T) {
	t.Parallel()
	policy := DefaultRestrictedSurface()
	if _, ok := policy.FullyDenied["unsafe"]; !ok {
		t.Errorf("default policy should deny unsafe")
	}
	if _, ok := policy.Allowed["reflect"]; !ok {
		t.Errorf("default policy should allowlist reflect")
	}
	if _, ok := policy.Allowed["reflect"]["MakeFunc"]; ok {
		t.Errorf("default policy should NOT permit reflect.MakeFunc")
	}
	if _, ok := policy.Allowed["reflect"]["TypeOf"]; !ok {
		t.Errorf("default policy should permit reflect.TypeOf")
	}
}

func TestApplyRestrictedSurfaceNilRegistryIsNoop(t *testing.T) {
	t.Parallel()
	removed := ApplyRestrictedSurface(nil, RestrictedPackageSet{})
	if removed != 0 {
		t.Fatalf("nil registry should remove nothing, got %d", removed)
	}
}

func TestDescribeRestrictedSurfaceFormatsSummary(t *testing.T) {
	t.Parallel()
	policy := DefaultRestrictedSurface()
	description := DescribeRestrictedSurface(policy)
	if description == "" {
		t.Fatalf("DescribeRestrictedSurface should not return empty")
	}
}
