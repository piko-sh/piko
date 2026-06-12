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

package interp_provider_piko_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/wdk/interp/interp_provider_piko"
)

func compileSource(t *testing.T, restricted bool, extras templater_domain.SymbolExports, source string) error {
	t.Helper()
	var opts []interp_provider_piko.ProviderOption
	if restricted {
		opts = append(opts, interp_provider_piko.WithRestrictedSymbolSurface())
	}
	provider := interp_provider_piko.NewProvider(opts...)
	if len(extras) > 0 {
		provider.RegisterSymbols(extras)
	}
	pool := provider.NewInterpreterPool(provider.NewSymbolProvider())
	interpreter, err := pool.Get()
	if err != nil {
		t.Fatalf("pool.Get: %v", err)
	}
	defer pool.Put(interpreter)

	batch, ok := interpreter.(templater_domain.BatchInterpreterPort)
	if !ok {
		t.Fatalf("interpreter %T is not a BatchInterpreterPort", interpreter)
	}
	return batch.CompileAndExecute(context.Background(), "main", map[string]map[string]string{
		"main": {"main.pi": source},
	})
}

func compileImport(t *testing.T, restricted bool, importPath string) error {
	t.Helper()
	return compileSource(t, restricted, nil, "package main\nimport _ \""+importPath+"\"\n")
}

func TestRestrictedSurfaceDeniesStdlib(t *testing.T) {
	t.Parallel()
	for _, importPath := range []string{"os", "os/exec", "net", "syscall", "unsafe", "reflect", "runtime"} {
		err := compileImport(t, true, importPath)
		if err == nil {
			t.Errorf("restricted surface must deny import %q, but it compiled", importPath)
			continue
		}
		if !strings.Contains(err.Error(), "not permitted") {
			t.Errorf("import %q denial should report the boundary, got: %v", importPath, err)
		}
	}
}

func TestRestrictedSurfaceDeniesRawStringUnsafe(t *testing.T) {
	t.Parallel()
	err := compileSource(t, true, nil, "package main\nimport _ "+"`unsafe`"+"\n")
	if err == nil {
		t.Fatal("restricted surface must deny a raw-string `unsafe` import, but it compiled")
	}
	if !strings.Contains(err.Error(), "not permitted") {
		t.Errorf("raw-string unsafe denial should report the boundary, got: %v", err)
	}
}

func TestRestrictedSurfacePermitsHostNamespace(t *testing.T) {
	t.Parallel()
	extras := templater_domain.SymbolExports{
		"host/widgets": {"Name": reflect.ValueOf("widgets")},
	}
	if err := compileSource(t, true, extras, "package main\nimport _ \"host/widgets\"\n"); err != nil {
		t.Errorf("restricted surface must permit a registered host namespace, got: %v", err)
	}
}

func TestFullSurfaceStillLoadsStdlib(t *testing.T) {
	t.Parallel()
	if err := compileImport(t, false, "os"); err != nil {
		t.Errorf("default surface should still load os for trusted mode: %v", err)
	}
}
