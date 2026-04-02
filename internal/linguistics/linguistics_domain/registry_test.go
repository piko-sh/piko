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

package linguistics_domain

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_NewIsEmpty(t *testing.T) {
	r := newRegistry[StemmerPort]("test-stemmer")
	names := r.registeredNames()
	assert.Empty(t, names)
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := newRegistry[StemmerPort]("test-stemmer")

	called := false
	factory := func() (StemmerPort, error) {
		called = true
		return NewNoOpStemmer("english"), nil
	}

	r.register("test-lang", factory)

	got, ok := r.get("test-lang")
	require.True(t, ok)
	require.NotNil(t, got)

	stemmer, err := got()
	require.NoError(t, err)
	assert.NotNil(t, stemmer)
	assert.True(t, called)
}

func TestRegistry_GetUnregistered(t *testing.T) {
	r := newRegistry[StemmerPort]("test-stemmer")

	got, ok := r.get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestRegistry_RegisteredNames(t *testing.T) {
	r := newRegistry[StemmerPort]("test-stemmer")

	factory := func() (StemmerPort, error) {
		return NewNoOpStemmer("english"), nil
	}

	r.register("alpha", factory)
	r.register("beta", factory)
	r.register("gamma", factory)

	names := r.registeredNames()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "alpha")
	assert.Contains(t, names, "beta")
	assert.Contains(t, names, "gamma")
}

func TestRegistry_RegisterDuplicatePanics(t *testing.T) {
	r := newRegistry[StemmerPort]("test-stemmer")

	factory := func() (StemmerPort, error) {
		return NewNoOpStemmer("english"), nil
	}

	r.register("duplicate", factory)

	assert.Panics(t, func() {
		r.register("duplicate", factory)
	})
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := newRegistry[StemmerPort]("test-stemmer")

	const numGoroutines = 50
	var wg sync.WaitGroup

	for i := range numGoroutines {
		wg.Go(func() {
			name := "concurrent-" + string(rune('a'+i))
			factory := func() (StemmerPort, error) {
				return NewNoOpStemmer("english"), nil
			}
			r.register(name, factory)
		})
	}

	wg.Wait()

	for i := range numGoroutines {
		wg.Go(func() {
			name := "concurrent-" + string(rune('a'+i))
			got, ok := r.get(name)
			assert.True(t, ok, "should find %s", name)
			assert.NotNil(t, got)
		})
	}

	wg.Wait()

	names := r.registeredNames()
	assert.Len(t, names, numGoroutines)
}
