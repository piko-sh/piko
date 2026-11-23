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

package interp_provider_pipit_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/testutil/pipitpending"
	"piko.sh/piko/wdk/interp/interp_provider_pipit"
)

const (
	registeredLibraryModule = "example.org/jit"
	registeredLibrary       = `package library
import "example.org/bridge"
var prefix = "record"
var revision = 10
func init() { bridge.Initialised() }
func Prefix() string { return prefix }
func Advance(delta int) int { revision += delta; return revision }
type Ledger int
func (units Ledger) Advance(delta int) int {
	revision += int(units) * delta
	return revision
}
`
)

type registeredPage struct {
	label   func() string
	advance func(int) int
}

func TestRegisteredPackageGlobalAfterJIT(t *testing.T) {
	for _, target := range []struct {
		name, expression string
		units            int
	}{
		{name: "function", expression: "library.Advance(delta)", units: 1},
		{name: "named-method", expression: "library.Ledger(2).Advance(delta)", units: 2},
	} {
		for _, pooled := range []bool{false, true} {
			mode := "same-interpreter"
			if pooled {
				mode = "pooled-clones"
			}
			t.Run(target.name+"/"+mode, func(t *testing.T) {
				checkRegisteredCallbacks(t, target.expression, target.units, pooled, 2, 3)
			})
		}
	}
}

func checkRegisteredCallbacks(t *testing.T, expression string, units int, pooled bool, before, after int) {
	t.Helper()
	if pooled {
		pipitpending.Skip(t, "pooled interpreter clones share registered package globals")
	}
	pages := map[string]registeredPage{}
	registrations := map[string]int{}
	var initLabels []string
	var initialisations int
	provider := interp_provider_pipit.NewProvider()
	provider.RegisterSymbols(templater_domain.SymbolExports{
		"example.org/bridge": {
			"Initialised": reflect.ValueOf(func() { initialisations++ }),
			"Record":      reflect.ValueOf(func(label string) { initLabels = append(initLabels, label) }),
			"Register": reflect.ValueOf(func(version string, label func() string, advance func(int) int) {
				registrations[version]++
				pages[version] = registeredPage{label: label, advance: advance}
			}),
		},
	})
	load := func(interpreter templater_domain.InterpreterPort, version string, first bool) registeredPage {
		t.Helper()
		require.Equal(t, !first, interpreter.HasRegisteredPackage(registeredLibraryModule+"/library"), "registered library before %s", version)
		sources := map[string]map[string]string{
			"page": {"page.go": fmt.Sprintf(`package page
import ("example.org/bridge"; "example.org/jit/library")
var version = %q
func Label() string { return library.Prefix() + "-" + version }
func Advance(delta int) int { return %s }
func init() { bridge.Record(Label()); bridge.Register(version, Label, Advance) }
`, version, expression)},
		}
		if first {
			sources["library"] = map[string]string{"ledger.go": registeredLibrary}
		}
		require.NoError(t, interpreter.CompileAndExecute(t.Context(), registeredLibraryModule, sources), "compiling %s", version)
		require.Contains(t, pages, version)
		return pages[version]
	}
	pool := provider.NewInterpreterPool()
	worker, err := pool.Get()
	require.NoError(t, err)
	old := load(worker, "before", true)
	require.Equal(t, 1, initialisations)
	require.Equal(t, "record-before", old.label())
	require.Equal(t, 10+before*units, old.advance(before), "the initial callback mutates its own library")

	if pooled {
		worker, err = pool.Get()
		require.NoError(t, err)
	}
	current := load(worker, "after", false)
	assert.Equal(t, 1, initialisations, "a new consumer must reuse the initialised library")
	assert.Equal(t, []string{"record-before", "record-after"}, initLabels, "each version reads the library during init")

	assert.NotPanics(t, func() {
		assert.Equal(t, "record-after", current.label())
		assert.Equal(t, "record-before", old.label())
		assert.Equal(t, 10+(before+after)*units, current.advance(after), "new consumer sees old write")
		assert.Equal(t, 10+(before+after-1)*units, old.advance(-1), "old consumer sees new write")
		assert.Equal(t, 10+(before+after-1)*units, current.advance(0), "new consumer sees the second old write")
	})

	originalRevision := old.advance(0)
	fresh, err := provider.NewInterpreterPool().Get()
	require.NoError(t, err)
	independent := load(fresh, "fresh", true)
	assert.Equal(t, 2, initialisations)
	assert.Equal(t, map[string]int{"before": 1, "after": 1, "fresh": 1}, registrations)
	assert.NotPanics(t, func() {
		assert.Equal(t, "record-fresh", independent.label())
		assert.Equal(t, 10+units, independent.advance(1), "fresh pool starts at its own initialised value")
		assert.Equal(t, originalRevision+2*units, old.advance(2), "fresh pool leaves original library untouched")
		assert.Equal(t, 10+units, independent.advance(0), "original pool leaves fresh library untouched")
	})
}
