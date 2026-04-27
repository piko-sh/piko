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

package wasm_adapters

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryPKJSEmitter_EmitJS_StripsTypeAnnotations(t *testing.T) {
	t.Parallel()

	emitter := NewInMemoryPKJSEmitter()
	source := "const greet = (name: string): string => `hello ${name}`;\n"

	artefactID, err := emitter.EmitJS(context.Background(), source, "pages/index", "example.com/site", "", false)
	require.NoError(t, err)
	assert.Equal(t, "pk-js/pages/index.js", artefactID)

	out := emitter.GetArtefacts()[artefactID]
	require.NotEmpty(t, out)
	assert.NotContains(t, out, ": string", "type annotations stripped by esbuild")
	assert.Contains(t, out, "greet")
}

func TestInMemoryPKJSEmitter_EmitJS_PartialDerivesComponentName(t *testing.T) {
	t.Parallel()

	emitter := NewInMemoryPKJSEmitter()
	source := "export function increment() { return 1; }\n"

	artefactID, err := emitter.EmitJS(context.Background(), source, "partials/cart", "example.com/site", "", true)
	require.NoError(t, err)
	assert.Equal(t, "pk-js/partials/cart.js", artefactID)

	out := emitter.GetArtefacts()[artefactID]
	require.NotEmpty(t, out)
	assert.Contains(t, out, "_createPKContext", "partial sources must get the factory wrapper")
}

func TestInMemoryPKJSEmitter_EmitJS_BareSourceWithNoFunctionsPassesThrough(t *testing.T) {
	t.Parallel()

	emitter := NewInMemoryPKJSEmitter()
	source := "console.log('boot');\n"

	artefactID, err := emitter.EmitJS(context.Background(), source, "pages/about", "example.com/site", "", false)
	require.NoError(t, err)

	out := emitter.GetArtefacts()[artefactID]
	require.NotEmpty(t, out)
	assert.NotContains(t, out, "_createPKContext", "bare source should not get factory wrapper")
	assert.Contains(t, out, "console.log")
}

func TestInMemoryPKJSEmitter_EmitJS_EmptySourceReturnsEmpty(t *testing.T) {
	t.Parallel()

	emitter := NewInMemoryPKJSEmitter()

	artefactID, err := emitter.EmitJS(context.Background(), "", "pages/index", "example.com/site", "", false)
	require.NoError(t, err)
	assert.Empty(t, artefactID)
	assert.Empty(t, emitter.GetArtefacts())

	artefactID, err = emitter.EmitJS(context.Background(), "  \n\t  ", "pages/index", "example.com/site", "", false)
	require.NoError(t, err)
	assert.Empty(t, artefactID)
	assert.Empty(t, emitter.GetArtefacts())
}

func TestInMemoryPKJSEmitter_EmitJS_ParseErrorPropagates(t *testing.T) {
	t.Parallel()

	emitter := NewInMemoryPKJSEmitter()
	source := "const x: number = ;"

	artefactID, err := emitter.EmitJS(context.Background(), source, "pages/index", "example.com/site", "", false)
	require.Error(t, err)
	assert.Empty(t, artefactID)
	assert.Empty(t, emitter.GetArtefacts(), "no artefact stored on parse failure")
	assert.Contains(t, err.Error(), "pages/index", "error wraps the page path")
}

func TestInMemoryPKJSEmitter_EmitJS_StripsPkSuffix(t *testing.T) {
	t.Parallel()

	emitter := NewInMemoryPKJSEmitter()
	source := "const x = 1;\n"

	artefactID1, err := emitter.EmitJS(context.Background(), source, "pages/index.pk", "m", "", false)
	require.NoError(t, err)
	assert.Equal(t, "pk-js/pages/index.js", artefactID1)

	artefactID2, err := emitter.EmitJS(context.Background(), source, "pages/index", "m", "", false)
	require.NoError(t, err)
	assert.Equal(t, "pk-js/pages/index.js", artefactID2)
}

func TestInMemoryPKJSEmitter_EmitJS_RewritesAtAliasImports(t *testing.T) {
	t.Parallel()

	emitter := NewInMemoryPKJSEmitter()
	source := "import { helper } from '@/lib/utils';\nhelper();\n"

	artefactID, err := emitter.EmitJS(context.Background(), source, "pages/index", "example.com/site", "", false)
	require.NoError(t, err)

	out := emitter.GetArtefacts()[artefactID]
	require.NotEmpty(t, out)
	assert.True(t,
		strings.Contains(out, "/_piko/assets/example.com/site/lib/utils.js"),
		"expected @/ alias rewritten to served path; got: %s", out)
}

func TestInMemoryPKJSEmitter_EmitJS_RejectsTraversalPaths(t *testing.T) {
	t.Parallel()

	emitter := NewInMemoryPKJSEmitter()
	source := "const x = 1;\n"

	for _, badPath := range []string{
		"../etc/passwd",
		"../../etc/passwd.pk",
		"/etc/passwd",
		"./",
		"",
	} {
		_, err := emitter.EmitJS(context.Background(), source, badPath, "m", "", false)
		require.Error(t, err, "path %q must be rejected", badPath)
		require.True(t, errors.Is(err, errPKJSPathInvalid), "errPKJSPathInvalid expected, got %v", err)
		assert.Empty(t, emitter.GetArtefacts(), "rejected path must not store anything")
	}
}

func TestInMemoryPKJSEmitter_TranspileCacheRoundTrip(t *testing.T) {
	t.Parallel()

	emitter := NewInMemoryPKJSEmitter()
	source := "const greet = (name: string): string => 'hi ' + name;\n"

	emitter.Reset()
	artefactID1, err := emitter.EmitJS(context.Background(), source, "pages/index", "example.com/site", "", false)
	require.NoError(t, err)
	originalOutput := emitter.GetArtefacts()[artefactID1]
	require.NotEmpty(t, originalOutput)

	emitter.Sweep()
	emitter.mu.Lock()
	require.Len(t, emitter.transpileCache, 1, "cache entry survives when produced this run")
	emitter.mu.Unlock()

	emitter.Reset()
	require.Empty(t, emitter.GetArtefacts())
	emitter.mu.Lock()
	require.Len(t, emitter.transpileCache, 1, "Reset must NOT touch the cache")
	emitter.mu.Unlock()

	artefactID2, err := emitter.EmitJS(context.Background(), "const x = 1;\n", "pages/other", "example.com/site", "", false)
	require.NoError(t, err)
	require.NotEqual(t, artefactID1, artefactID2)

	emitter.Sweep()
	emitter.mu.Lock()
	require.Len(t, emitter.transpileCache, 1, "Sweep evicts entries not produced this run")
	emitter.mu.Unlock()
}

func TestInMemoryPKJSEmitter_Put_BypassesTranspile(t *testing.T) {
	t.Parallel()

	emitter := NewInMemoryPKJSEmitter()
	rawJS := "class PPCounter extends PPElement {}\ncustomElements.define('pp-counter', PPCounter);\n"

	require.NoError(t, emitter.Put("pk-js/components/pp-counter.js", rawJS))

	got := emitter.GetArtefacts()["pk-js/components/pp-counter.js"]
	assert.Equal(t, rawJS, got, "Put stores content verbatim, no transform")

	emitter.mu.Lock()
	assert.Empty(t, emitter.transpileCache, "Put does not interact with the transpile cache")
	emitter.mu.Unlock()
}

func TestInMemoryPKJSEmitter_Put_RejectsInvalidArtefactID(t *testing.T) {
	t.Parallel()

	emitter := NewInMemoryPKJSEmitter()

	for _, bad := range []string{"", "/abs/path.js", "../escape.js", "./", "."} {
		err := emitter.Put(bad, "content")
		require.Error(t, err, "Put(%q) must error", bad)
		require.True(t, errors.Is(err, errPKJSPathInvalid), "wrong error class for %q: %v", bad, err)
	}
	assert.Empty(t, emitter.GetArtefacts())
}

func TestInMemoryPKJSEmitter_Put_EmptyContentIsNoOp(t *testing.T) {
	t.Parallel()

	emitter := NewInMemoryPKJSEmitter()
	require.NoError(t, emitter.Put("pk-js/components/empty.js", ""))
	assert.Empty(t, emitter.GetArtefacts())
}

func TestInMemoryPKJSEmitter_Sweep_ClearsProducedThisRun(t *testing.T) {
	t.Parallel()

	emitter := NewInMemoryPKJSEmitter()
	source := "const x = 1;\n"

	_, err := emitter.EmitJS(context.Background(), source, "pages/index", "m", "", false)
	require.NoError(t, err)

	emitter.mu.Lock()
	require.Len(t, emitter.producedThisRun, 1)
	emitter.mu.Unlock()

	emitter.Sweep()

	emitter.mu.Lock()
	assert.Empty(t, emitter.producedThisRun, "Sweep must clear producedThisRun so a missing Reset cannot leak")
	emitter.mu.Unlock()
}

func TestInMemoryPKJSEmitter_Sweep_EnforcesCap(t *testing.T) {
	t.Parallel()

	emitter := NewInMemoryPKJSEmitter()

	emitter.mu.Lock()
	for index := range pkJSTranspileCacheCap + 50 {
		key := pkJSCacheKey("source"+stringFromInt(index), "m", "f.ts")
		emitter.transpileCache[key] = "code"
		emitter.producedThisRun[key] = struct{}{}
	}
	emitter.mu.Unlock()

	emitter.Sweep()

	emitter.mu.Lock()
	got := len(emitter.transpileCache)
	emitter.mu.Unlock()
	assert.LessOrEqual(t, got, pkJSTranspileCacheCap, "Sweep must cap the cache regardless of producedThisRun")
}

func TestInMemoryPKJSEmitter_GetArtefacts_ReturnsCopy(t *testing.T) {
	t.Parallel()

	emitter := NewInMemoryPKJSEmitter()
	_, err := emitter.EmitJS(context.Background(), "const x = 1;", "pages/index", "m", "", false)
	require.NoError(t, err)

	artefacts := emitter.GetArtefacts()
	artefacts["injected"] = "injected"

	again := emitter.GetArtefacts()
	assert.Len(t, again, 1, "internal map must not reflect mutations to the returned copy")
}

func TestInMemoryPKJSEmitter_ConcurrentEmitJSIsSafe(t *testing.T) {
	t.Parallel()

	emitter := NewInMemoryPKJSEmitter()
	source := "const x = 1;\n"

	const goroutines = 8
	const perGoroutine = 16

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for goroutineIndex := range goroutines {
		go func(goroutineIndex int) {
			defer wg.Done()
			for iteration := range perGoroutine {
				_, err := emitter.EmitJS(context.Background(), source,
					"pages/p"+stringFromInt(goroutineIndex)+"_"+stringFromInt(iteration),
					"m", "", false)
				require.NoError(t, err)
			}
		}(goroutineIndex)
	}
	wg.Wait()

	assert.Len(t, emitter.GetArtefacts(), goroutines*perGoroutine)
}

func stringFromInt(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	const decBase = 10
	var digits [20]byte
	pos := len(digits)
	for value > 0 {
		pos--
		digits[pos] = byte('0' + value%decBase)
		value /= decBase
	}
	if negative {
		pos--
		digits[pos] = '-'
	}
	return string(digits[pos:])
}
