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
	"sync"
)

// globalStore holds package-level variables. It is shared across all
// function invocations within the same package and across goroutines,
// so access is synchronised.
type globalStore struct {
	// ints holds all package-level int64 global variables.
	ints []int64

	// floats holds all package-level float64 global variables.
	floats []float64

	// strings holds all package-level string global variables.
	strings []string

	// general holds all package-level reflect.Value global variables.
	general []reflect.Value

	// bools holds all package-level bool global variables.
	bools []bool

	// uints holds all package-level uint64 global variables.
	uints []uint64

	// complexes holds all package-level complex128 global variables.
	complexes []complex128

	// mu guards concurrent access to all fields.
	mu sync.RWMutex
}

// allocInt allocates a new int64 global variable and returns its index.
//
// Takes initial (int64) which is the starting value for the variable.
//
// Returns the index of the newly allocated variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) allocInt(initial int64) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	index := len(g.ints)
	g.ints = append(g.ints, initial)
	return index
}

// allocFloat allocates a new float64 global variable and returns its index.
//
// Takes initial (float64) which is the starting value for the variable.
//
// Returns the index of the newly allocated variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) allocFloat(initial float64) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	index := len(g.floats)
	g.floats = append(g.floats, initial)
	return index
}

// allocString allocates a new string global variable and returns its index.
//
// Takes initial (string) which is the starting value for the variable.
//
// Returns the index of the newly allocated variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) allocString(initial string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	index := len(g.strings)
	g.strings = append(g.strings, initial)
	return index
}

// allocGeneral allocates a new reflect.Value global variable
// and returns its index.
//
// Takes initial (reflect.Value) which is the starting value for the
// variable.
//
// Returns the index of the newly allocated variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) allocGeneral(initial reflect.Value) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	index := len(g.general)
	g.general = append(g.general, initial)
	return index
}

// getInt reads an int64 global variable.
//
// Takes index (int) which is the variable to read.
//
// Returns the current value of the variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) getInt(index int) int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.ints[index]
}

// setInt writes an int64 global variable.
//
// Takes index (int) which is the variable to write.
// Takes v (int64) which is the new value.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) setInt(index int, v int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.ints[index] = v
}

// getFloat reads a float64 global variable.
//
// Takes index (int) which is the variable to read.
//
// Returns the current value of the variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) getFloat(index int) float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.floats[index]
}

// setFloat writes a float64 global variable.
//
// Takes index (int) which is the variable to write.
// Takes v (float64) which is the new value.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) setFloat(index int, v float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.floats[index] = v
}

// getString reads a string global variable.
//
// Takes index (int) which is the variable to read.
//
// Returns the current value of the variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) getString(index int) string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.strings[index]
}

// setString writes a string global variable.
//
// Takes index (int) which is the variable to write.
// Takes v (string) which is the new value.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) setString(index int, v string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.strings[index] = v
}

// getGeneral reads a reflect.Value global variable.
//
// Takes index (int) which is the variable to read.
//
// Returns the current value of the variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) getGeneral(index int) reflect.Value {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.general[index]
}

// setGeneral writes a reflect.Value global variable.
//
// Takes index (int) which is the variable to write.
// Takes v (reflect.Value) which is the new value.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) setGeneral(index int, v reflect.Value) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.general[index] = v
}

// allocBool allocates a new bool global variable and returns its index.
//
// Takes initial (bool) which is the starting value for the variable.
//
// Returns the index of the newly allocated variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) allocBool(initial bool) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	index := len(g.bools)
	g.bools = append(g.bools, initial)
	return index
}

// getBool reads a bool global variable.
//
// Takes index (int) which is the variable to read.
//
// Returns the current value of the variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) getBool(index int) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.bools[index]
}

// setBool writes a bool global variable.
//
// Takes index (int) which is the variable to write.
// Takes v (bool) which is the new value.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) setBool(index int, v bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.bools[index] = v
}

// allocUint allocates a new uint64 global variable and returns its index.
//
// Takes initial (uint64) which is the starting value for the variable.
//
// Returns the index of the newly allocated variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) allocUint(initial uint64) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	index := len(g.uints)
	g.uints = append(g.uints, initial)
	return index
}

// getUint reads a uint64 global variable.
//
// Takes index (int) which is the variable to read.
//
// Returns the current value of the variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) getUint(index int) uint64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.uints[index]
}

// setUint writes a uint64 global variable.
//
// Takes index (int) which is the variable to write.
// Takes v (uint64) which is the new value.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) setUint(index int, v uint64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.uints[index] = v
}

// allocComplex allocates a new complex128 global variable and returns
// its index.
//
// Takes initial (complex128) which is the starting value for the
// variable.
//
// Returns the index of the newly allocated variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) allocComplex(initial complex128) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	index := len(g.complexes)
	g.complexes = append(g.complexes, initial)
	return index
}

// getComplex reads a complex128 global variable.
//
// Takes index (int) which is the variable to read.
//
// Returns the current value of the variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) getComplex(index int) complex128 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.complexes[index]
}

// setComplex writes a complex128 global variable.
//
// Takes index (int) which is the variable to write.
// Takes v (complex128) which is the new value.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) setComplex(index int, v complex128) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.complexes[index] = v
}

// materialiseStrings replaces any arena-backed string globals with
// heap-backed copies so that globals do not hold dangling pointers.
//
// Takes arena (*RegisterArena) which is the arena whose byte slabs are
// checked for ownership.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) materialiseStrings(arena *RegisterArena) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, s := range g.strings {
		if arena.ownsString(s) {
			g.strings[i] = strings.Clone(s)
		}
	}
}

// reset clears all global variables.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.ints = g.ints[:0]
	g.floats = g.floats[:0]
	g.strings = g.strings[:0]
	g.general = g.general[:0]
	g.bools = g.bools[:0]
	g.uints = g.uints[:0]
	g.complexes = g.complexes[:0]
}

// newGlobalStore creates an empty global variable store.
//
// Returns a newly allocated globalStore with no variables.
func newGlobalStore() *globalStore {
	return &globalStore{}
}
