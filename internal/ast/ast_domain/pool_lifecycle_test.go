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

package ast_domain

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArenaLifecycle(t *testing.T) {
	t.Parallel()

	const goroutines = 50
	const iterations = 200

	var errorCount atomic.Int64
	var wg sync.WaitGroup

	testCases := []struct {
		staticClass  string
		dynamicClass string
		expected     string
	}{
		{staticClass: "nav-item", dynamicClass: "active", expected: "nav-item active"},
		{staticClass: "text-emphasis", dynamicClass: "bold", expected: "text-emphasis bold"},
		{staticClass: "sidebar", dynamicClass: "collapsed", expected: "sidebar collapsed"},
		{staticClass: "btn", dynamicClass: "primary", expected: "btn primary"},
		{staticClass: "card", dynamicClass: "highlighted", expected: "card highlighted"},
	}

	for g := range goroutines {
		goroutineID := g
		wg.Go(func() {
			for i := range iterations {
				tc := testCases[i%len(testCases)]

				arena := GetArena()
				ast := GetTemplateAST()
				ast.SetArena(arena)
				ast.RootNodes = arena.GetRootNodesSlice(1)

				node := arena.GetNode()
				node.NodeType = NodeElement
				node.TagName = "div"
				node.IsPooled = true

				bufferPointer := GetByteBuf()
				*bufferPointer = append((*bufferPointer)[:0], tc.staticClass...)
				*bufferPointer = append(*bufferPointer, ' ')
				*bufferPointer = append(*bufferPointer, tc.dynamicClass...)

				dw := arena.GetDirectWriter()
				dw.SetName("class")
				dw.AppendPooledBytes(bufferPointer)
				_, node.AttributeWriters = arena.GetAttrWriterSlice(1)
				node.AttributeWriters = append(node.AttributeWriters, dw)

				ast.RootNodes = append(ast.RootNodes, node)

				if len(ast.RootNodes) > 0 && len(ast.RootNodes[0].AttributeWriters) > 0 {
					renderDW := ast.RootNodes[0].AttributeWriters[0]
					var output []byte
					output = renderDW.WriteTo(output)
					got := string(output)

					if got != tc.expected {
						errorCount.Add(1)
						if errorCount.Load() <= 10 {
							t.Errorf("CORRUPTION [g=%d, i=%d]: expected %q, got %q",
								goroutineID, i, tc.expected, got)
						}
					}
				}

				PutTree(ast)
			}
		})
	}

	wg.Wait()

	total := int64(goroutines * iterations)
	errors := errorCount.Load()
	if errors > 0 {
		t.Errorf("Completed %d operations, %d errors (%.4f%%)",
			total, errors, float64(errors)/float64(total)*100)
	}
}

func TestArenaLifecycleWithMultipleElements(t *testing.T) {
	t.Parallel()

	const goroutines = 30
	const iterations = 100

	var errorCount atomic.Int64
	var wg sync.WaitGroup

	elements := []struct {
		tag   string
		class string
	}{
		{tag: "div", class: "container"},
		{tag: "nav", class: "nav-item active"},
		{tag: "span", class: "text-emphasis highlight"},
		{tag: "button", class: "btn btn-primary"},
		{tag: "section", class: "sidebar collapsed"},
	}

	for g := range goroutines {
		goroutineID := g
		wg.Go(func() {
			for i := range iterations {
				arena := GetArena()
				ast := GetTemplateAST()
				ast.SetArena(arena)
				ast.RootNodes = arena.GetRootNodesSlice(len(elements))

				for _, element := range elements {
					node := arena.GetNode()
					node.NodeType = NodeElement
					node.TagName = element.tag
					node.IsPooled = true

					bufferPointer := GetByteBuf()
					*bufferPointer = append((*bufferPointer)[:0], element.class...)

					dw := arena.GetDirectWriter()
					dw.SetName("class")
					dw.AppendPooledBytes(bufferPointer)
					_, node.AttributeWriters = arena.GetAttrWriterSlice(1)
					node.AttributeWriters = append(node.AttributeWriters, dw)

					ast.RootNodes = append(ast.RootNodes, node)
				}

				for index, node := range ast.RootNodes {
					if len(node.AttributeWriters) == 0 {
						continue
					}
					renderDW := node.AttributeWriters[0]
					var output []byte
					output = renderDW.WriteTo(output)
					got := string(output)
					expected := elements[index].class

					if got != expected {
						errorCount.Add(1)
						if errorCount.Load() <= 10 {
							t.Errorf("CORRUPTION [g=%d, i=%d, element=%d]: expected %q, got %q",
								goroutineID, i, index, expected, got)
						}
					}
				}

				PutTree(ast)
			}
		})
	}

	wg.Wait()

	total := int64(goroutines * iterations * len(elements))
	errors := errorCount.Load()
	if errors > 0 {
		t.Errorf("Completed %d element renders, %d errors (%.4f%%)",
			total, errors, float64(errors)/float64(total)*100)
	}
}

func TestArenaLifecycleWithDeepClone(t *testing.T) {
	t.Parallel()

	const goroutines = 30
	const iterations = 100

	var errorCount atomic.Int64
	var wg sync.WaitGroup

	for g := range goroutines {
		goroutineID := g
		wg.Go(func() {
			for i := range iterations {
				expectedClass := fmt.Sprintf("class-g%d-i%d", goroutineID, i)

				arena := GetArena()
				originalAST := GetTemplateAST()
				originalAST.SetArena(arena)
				originalAST.RootNodes = arena.GetRootNodesSlice(1)

				node := arena.GetNode()
				node.NodeType = NodeElement
				node.TagName = "div"
				node.IsPooled = true

				bufferPointer := GetByteBuf()
				*bufferPointer = append((*bufferPointer)[:0], expectedClass...)

				dw := arena.GetDirectWriter()
				dw.SetName("class")
				dw.AppendPooledBytes(bufferPointer)
				_, node.AttributeWriters = arena.GetAttrWriterSlice(1)
				node.AttributeWriters = append(node.AttributeWriters, dw)

				originalAST.RootNodes = append(originalAST.RootNodes, node)

				clonedAST := originalAST.DeepClone()

				PutTree(originalAST)

				if len(clonedAST.RootNodes) > 0 && len(clonedAST.RootNodes[0].AttributeWriters) > 0 {
					renderDW := clonedAST.RootNodes[0].AttributeWriters[0]
					var output []byte
					output = renderDW.WriteTo(output)
					got := string(output)

					if got != expectedClass {
						errorCount.Add(1)
						if errorCount.Load() <= 10 {
							t.Errorf("CLONE CORRUPTION [g=%d, i=%d]: expected %q, got %q",
								goroutineID, i, expectedClass, got)
						}
					}
				}

				PutTree(clonedAST)
			}
		})
	}

	wg.Wait()

	total := int64(goroutines * iterations)
	errors := errorCount.Load()
	if errors > 0 {
		t.Errorf("Completed %d clone operations, %d errors (%.4f%%)",
			total, errors, float64(errors)/float64(total)*100)
	}
}

func TestArenaLifecycleInterleavedOperations(t *testing.T) {
	t.Parallel()

	const goroutines = 20
	const iterations = 50

	var errorCount atomic.Int64
	var wg sync.WaitGroup

	type astWithExpected struct {
		ast      *TemplateAST
		expected string
	}

	for g := range goroutines {
		goroutineID := g
		wg.Go(func() {
			inFlight := make([]astWithExpected, 0, 5)

			for i := range iterations {
				expectedClass := fmt.Sprintf("interleaved-g%d-i%d", goroutineID, i)

				arena := GetArena()
				ast := GetTemplateAST()
				ast.SetArena(arena)
				ast.RootNodes = arena.GetRootNodesSlice(1)

				node := arena.GetNode()
				node.NodeType = NodeElement
				node.TagName = "div"
				node.IsPooled = true

				bufferPointer := GetByteBuf()
				*bufferPointer = append((*bufferPointer)[:0], expectedClass...)

				dw := arena.GetDirectWriter()
				dw.SetName("class")
				dw.AppendPooledBytes(bufferPointer)
				_, node.AttributeWriters = arena.GetAttrWriterSlice(1)
				node.AttributeWriters = append(node.AttributeWriters, dw)

				ast.RootNodes = append(ast.RootNodes, node)

				inFlight = append(inFlight, astWithExpected{ast: ast, expected: expectedClass})

				if len(inFlight) >= 3 || i == iterations-1 {
					for _, item := range inFlight {
						if len(item.ast.RootNodes) > 0 && len(item.ast.RootNodes[0].AttributeWriters) > 0 {
							renderDW := item.ast.RootNodes[0].AttributeWriters[0]
							var output []byte
							output = renderDW.WriteTo(output)
							got := string(output)

							if got != item.expected {
								errorCount.Add(1)
								if errorCount.Load() <= 10 {
									t.Errorf("INTERLEAVED CORRUPTION [g=%d]: expected %q, got %q",
										goroutineID, item.expected, got)
								}
							}
						}

						PutTree(item.ast)
					}
					inFlight = inFlight[:0]
				}
			}
		})
	}

	wg.Wait()

	total := int64(goroutines * iterations)
	errors := errorCount.Load()
	if errors > 0 {
		t.Errorf("Completed %d interleaved operations, %d errors (%.4f%%)",
			total, errors, float64(errors)/float64(total)*100)
	}
}

func TestArenaHighWaterMarkProtection(t *testing.T) {
	ResetArenaPool()

	t.Run("nodes slab shrinks when exceeding limit", func(t *testing.T) {
		arena := GetArena()
		initialCap := len(arena.nodes)

		for range maxNodeCount + 100 {
			arena.GetNode()
		}

		bloatedCap := len(arena.nodes)
		if bloatedCap <= maxNodeCount {
			t.Fatalf("Expected nodes to grow beyond %d, got %d", maxNodeCount, bloatedCap)
		}

		arena.Reset()

		if len(arena.nodes) > maxNodeCount {
			t.Errorf("Expected nodes to shrink below %d after Reset(), got %d", maxNodeCount, len(arena.nodes))
		}
		if len(arena.nodes) != initialCap {
			t.Logf("Nodes shrunk to %d (initial was %d)", len(arena.nodes), initialCap)
		}

		PutArena(arena)
	})

	t.Run("DirectWriters slab shrinks when exceeding limit", func(t *testing.T) {
		arena := GetArena()
		initialCap := len(arena.directWriters)

		for range maxDirectWriters + 100 {
			arena.GetDirectWriter()
		}

		bloatedCap := len(arena.directWriters)
		if bloatedCap <= maxDirectWriters {
			t.Fatalf("Expected directWriters to grow beyond %d, got %d", maxDirectWriters, bloatedCap)
		}

		arena.Reset()

		if len(arena.directWriters) > maxDirectWriters {
			t.Errorf("Expected directWriters to shrink below %d after Reset(), got %d", maxDirectWriters, len(arena.directWriters))
		}
		if len(arena.directWriters) != initialCap {
			t.Logf("DirectWriters shrunk to %d (initial was %d)", len(arena.directWriters), initialCap)
		}

		PutArena(arena)
	})

	t.Run("byteBufs slab shrinks when exceeding limit", func(t *testing.T) {
		arena := GetArena()
		initialCap := len(arena.byteBufs)

		for range maxByteBufs + 100 {
			arena.GetByteBuf()
		}

		bloatedCap := len(arena.byteBufs)
		if bloatedCap <= maxByteBufs {
			t.Fatalf("Expected byteBufs to grow beyond %d, got %d", maxByteBufs, bloatedCap)
		}

		arena.Reset()

		if len(arena.byteBufs) > maxByteBufs {
			t.Errorf("Expected byteBufs to shrink below %d after Reset(), got %d", maxByteBufs, len(arena.byteBufs))
		}
		if len(arena.byteBufs) != initialCap {
			t.Logf("ByteBufs shrunk to %d (initial was %d)", len(arena.byteBufs), initialCap)
		}

		PutArena(arena)
	})

	t.Run("individual byteBuf shrinks when exceeding capacity limit", func(t *testing.T) {
		arena := GetArena()

		buffer := arena.GetByteBuf()

		largeData := make([]byte, maxByteBufCapacity+1000)
		*buffer = append(*buffer, largeData...)

		if cap(*buffer) <= maxByteBufCapacity {
			t.Fatalf("Expected buffer capacity to exceed %d, got %d", maxByteBufCapacity, cap(*buffer))
		}

		arena.Reset()

		if cap(arena.byteBufs[0]) > maxByteBufCapacity {
			t.Errorf("Expected individual byteBuf to shrink below %d after Reset(), got %d",
				maxByteBufCapacity, cap(arena.byteBufs[0]))
		}

		PutArena(arena)
	})

	t.Run("normal-sized arena remains unchanged after Reset", func(t *testing.T) {
		arena := GetArena()
		initialNodesCap := len(arena.nodes)
		initialDWCap := len(arena.directWriters)
		initialByteBufsCap := len(arena.byteBufs)

		for range 100 {
			arena.GetNode()
			arena.GetDirectWriter()
			arena.GetByteBuf()
		}

		arena.Reset()

		if len(arena.nodes) != initialNodesCap {
			t.Errorf("Expected nodes capacity %d to remain unchanged, got %d", initialNodesCap, len(arena.nodes))
		}
		if len(arena.directWriters) != initialDWCap {
			t.Errorf("Expected directWriters capacity %d to remain unchanged, got %d", initialDWCap, len(arena.directWriters))
		}
		if len(arena.byteBufs) != initialByteBufsCap {
			t.Errorf("Expected byteBufs capacity %d to remain unchanged, got %d", initialByteBufsCap, len(arena.byteBufs))
		}

		PutArena(arena)
	})
}

func TestArena_GetChildSlice(t *testing.T) {
	arena := GetArena()
	defer PutArena(arena)

	tests := []struct {
		name        string
		capacity    int
		wantMinCap  int
		wantNilPtr  bool
		wantNilSlc  bool
		wantBacking bool
	}{
		{
			name:       "zero capacity returns nil",
			capacity:   0,
			wantNilPtr: true,
			wantNilSlc: true,
		},
		{
			name:        "capacity 1 uses bucket 0 (cap 2)",
			capacity:    1,
			wantBacking: true,
			wantMinCap:  2,
		},
		{
			name:        "capacity 5 uses bucket 2 (cap 6)",
			capacity:    5,
			wantBacking: true,
			wantMinCap:  6,
		},
		{
			name:        "capacity 128 uses last bucket (cap 128)",
			capacity:    128,
			wantBacking: true,
			wantMinCap:  128,
		},
		{
			name:       "capacity 200 falls back to heap allocation",
			capacity:   200,
			wantNilPtr: true,
			wantMinCap: 200,
		},
		{
			name:       "negative capacity returns nil",
			capacity:   -5,
			wantNilPtr: true,
			wantNilSlc: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptr, slc := arena.GetChildSlice(tt.capacity)

			if tt.wantNilSlc {
				assert.Nil(t, slc, "expected nil slice")
				assert.Nil(t, ptr, "expected nil pointer")
				return
			}

			assert.NotNil(t, slc, "expected non-nil slice")
			assert.Equal(t, 0, len(slc), "slice should have length 0")
			assert.GreaterOrEqual(t, cap(slc), tt.wantMinCap,
				"slice cap should be at least %d", tt.wantMinCap)

			if tt.wantNilPtr {
				assert.Nil(t, ptr, "expected nil backing pointer for fallback allocation")
			}
			if tt.wantBacking {
				assert.NotNil(t, ptr, "expected non-nil backing pointer for pooled allocation")
			}
		})
	}
}

func TestArena_GetChildSlice_Growth(t *testing.T) {
	arena := GetArena()
	defer PutArena(arena)

	initialCount := initialChildCounts[0]
	for range initialCount {
		ptr, slc := arena.GetChildSlice(1)
		assert.NotNil(t, ptr, "pooled allocation should return backing pointer")
		assert.NotNil(t, slc, "pooled allocation should return slice")
	}

	ptr, slc := arena.GetChildSlice(1)
	assert.NotNil(t, ptr, "allocation after growth should return backing pointer")
	assert.NotNil(t, slc, "allocation after growth should return slice")
	assert.Equal(t, 0, len(slc), "slice should have length 0 after growth")
	assert.GreaterOrEqual(t, cap(slc), 2, "slice cap should be at least 2 after growth")
}

func TestArena_GetAttrSlice(t *testing.T) {
	arena := GetArena()
	defer PutArena(arena)

	tests := []struct {
		name        string
		capacity    int
		wantMinCap  int
		wantNilPtr  bool
		wantNilSlc  bool
		wantBacking bool
	}{
		{
			name:       "zero capacity returns nil",
			capacity:   0,
			wantNilPtr: true,
			wantNilSlc: true,
		},
		{
			name:        "capacity 2 uses bucket 0",
			capacity:    2,
			wantBacking: true,
			wantMinCap:  2,
		},
		{
			name:        "capacity 4 uses bucket 1",
			capacity:    4,
			wantBacking: true,
			wantMinCap:  4,
		},
		{
			name:        "capacity 6 uses bucket 2",
			capacity:    6,
			wantBacking: true,
			wantMinCap:  6,
		},
		{
			name:        "capacity 8 uses bucket 3",
			capacity:    8,
			wantBacking: true,
			wantMinCap:  8,
		},
		{
			name:        "capacity 10 uses bucket 4",
			capacity:    10,
			wantBacking: true,
			wantMinCap:  10,
		},
		{
			name:        "capacity 12 uses bucket 5",
			capacity:    12,
			wantBacking: true,
			wantMinCap:  12,
		},
		{
			name:        "capacity 16 uses bucket 6",
			capacity:    16,
			wantBacking: true,
			wantMinCap:  16,
		},
		{
			name:       "capacity 20 falls back to heap allocation",
			capacity:   20,
			wantNilPtr: true,
			wantMinCap: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptr, slc := arena.GetAttrSlice(tt.capacity)

			if tt.wantNilSlc {
				assert.Nil(t, slc, "expected nil slice")
				assert.Nil(t, ptr, "expected nil pointer")
				return
			}

			assert.NotNil(t, slc, "expected non-nil slice")
			assert.Equal(t, 0, len(slc), "slice should have length 0")
			assert.GreaterOrEqual(t, cap(slc), tt.wantMinCap,
				"slice cap should be at least %d", tt.wantMinCap)

			if tt.wantNilPtr {
				assert.Nil(t, ptr, "expected nil backing pointer for fallback allocation")
			}
			if tt.wantBacking {
				assert.NotNil(t, ptr, "expected non-nil backing pointer for pooled allocation")
			}
		})
	}
}

func TestArena_GetAttrSlice_Growth(t *testing.T) {
	arena := GetArena()
	defer PutArena(arena)

	initialCount := initialAttrCounts[0]
	for range initialCount {
		ptr, slc := arena.GetAttrSlice(1)
		assert.NotNil(t, ptr, "pooled allocation should return backing pointer")
		assert.NotNil(t, slc, "pooled allocation should return slice")
	}

	ptr, slc := arena.GetAttrSlice(1)
	assert.NotNil(t, ptr, "allocation after growth should return backing pointer")
	assert.NotNil(t, slc, "allocation after growth should return slice")
	assert.Equal(t, 0, len(slc), "slice should have length 0 after growth")
	assert.GreaterOrEqual(t, cap(slc), 2, "slice cap should be at least 2 after growth")
}

func TestArena_GetAttrWriterSlice_Growth(t *testing.T) {
	arena := GetArena()
	defer PutArena(arena)

	initialCount := initialWriterCounts[0]
	for range initialCount {
		ptr, slc := arena.GetAttrWriterSlice(1)
		assert.NotNil(t, ptr, "pooled allocation should return backing pointer")
		assert.NotNil(t, slc, "pooled allocation should return slice")
	}

	ptr, slc := arena.GetAttrWriterSlice(1)
	assert.NotNil(t, ptr, "allocation after growth should return backing pointer")
	assert.NotNil(t, slc, "allocation after growth should return slice")
	assert.Equal(t, 0, len(slc), "slice should have length 0 after growth")
	assert.GreaterOrEqual(t, cap(slc), 2, "slice cap should be at least 2 after growth")
}

func TestArena_GetRuntimeAnnotation(t *testing.T) {
	arena := GetArena()
	defer PutArena(arena)

	annotations := make([]*RuntimeAnnotation, 0, initialAnnotations+1)
	for range initialAnnotations + 1 {
		ra := arena.GetRuntimeAnnotation()
		require.NotNil(t, ra, "GetRuntimeAnnotation should never return nil")
		annotations = append(annotations, ra)
	}

	for i, ra := range annotations {
		ra.NeedsCSRF = true
		assert.True(t, annotations[i].NeedsCSRF,
			"annotation %d should be usable after allocation", i)
	}
}

func TestArena_GetTemplateAST(t *testing.T) {
	arena := GetArena()
	defer PutArena(arena)

	first := arena.GetTemplateAST()
	require.NotNil(t, first, "first GetTemplateAST should return non-nil")
	assert.True(t, first.isPooled, "arena AST should be marked as pooled")

	second := arena.GetTemplateAST()
	require.NotNil(t, second, "second GetTemplateAST should return non-nil")
	assert.True(t, second.isPooled, "fallback AST should be marked as pooled")

	assert.NotSame(t, first, second,
		"first and second AST should be different pointers")

	PutTemplateAST(second)
}

func TestArena_GetRootNodesSlice(t *testing.T) {
	arena := GetArena()
	defer PutArena(arena)

	tests := []struct {
		name       string
		capacity   int
		wantNil    bool
		wantMinCap int
	}{
		{
			name:     "zero capacity returns nil",
			capacity: 0,
			wantNil:  true,
		},
		{
			name:       "capacity 1 uses bucket 0 (cap 1)",
			capacity:   1,
			wantMinCap: 1,
		},
		{
			name:       "capacity 2 uses bucket 1 (cap 2)",
			capacity:   2,
			wantMinCap: 2,
		},
		{
			name:       "capacity 4 uses bucket 2 (cap 4)",
			capacity:   4,
			wantMinCap: 4,
		},
		{
			name:       "capacity 64 uses bucket 11 (cap 64)",
			capacity:   64,
			wantMinCap: 64,
		},
		{
			name:       "capacity 128 uses last bucket (cap 128)",
			capacity:   128,
			wantMinCap: 128,
		},
		{
			name:       "capacity 200 falls back to heap allocation",
			capacity:   200,
			wantMinCap: 200,
		},
		{
			name:     "negative capacity returns nil",
			capacity: -1,
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slc := arena.GetRootNodesSlice(tt.capacity)

			if tt.wantNil {
				assert.Nil(t, slc, "expected nil slice")
				return
			}

			assert.NotNil(t, slc, "expected non-nil slice")
			assert.Equal(t, 0, len(slc), "slice should have length 0")
			assert.GreaterOrEqual(t, cap(slc), tt.wantMinCap,
				"slice cap should be at least %d", tt.wantMinCap)
		})
	}

	growArena := GetArena()
	defer PutArena(growArena)

	initialCount := initialRootNodesCounts[0]
	for range initialCount {
		slc := growArena.GetRootNodesSlice(1)
		assert.NotNil(t, slc, "pooled allocation should return a slice")
	}

	slc := growArena.GetRootNodesSlice(1)
	assert.NotNil(t, slc, "allocation after growth should return a slice")
	assert.Equal(t, 0, len(slc), "slice should have length 0 after growth")
	assert.GreaterOrEqual(t, cap(slc), 1, "slice cap should be at least 1 after growth")
}

func TestArena_childBucketIndex(t *testing.T) {
	t.Parallel()

	arena := &RenderArena{}

	tests := []struct {
		name     string
		capacity int
		want     int
	}{
		{name: "capacity 1 maps to bucket 0", capacity: 1, want: 0},
		{name: "capacity 2 maps to bucket 0", capacity: 2, want: 0},
		{name: "capacity 3 maps to bucket 1", capacity: 3, want: 1},
		{name: "capacity 4 maps to bucket 1", capacity: 4, want: 1},
		{name: "capacity 5 maps to bucket 2", capacity: 5, want: 2},
		{name: "capacity 6 maps to bucket 2", capacity: 6, want: 2},
		{name: "capacity 7 maps to bucket 3", capacity: 7, want: 3},
		{name: "capacity 8 maps to bucket 3", capacity: 8, want: 3},
		{name: "capacity 9 maps to bucket 4", capacity: 9, want: 4},
		{name: "capacity 10 maps to bucket 4", capacity: 10, want: 4},
		{name: "capacity 11 maps to bucket 5", capacity: 11, want: 5},
		{name: "capacity 12 maps to bucket 5", capacity: 12, want: 5},
		{name: "capacity 13 maps to bucket 6", capacity: 13, want: 6},
		{name: "capacity 16 maps to bucket 6", capacity: 16, want: 6},
		{name: "capacity 17 maps to bucket 7", capacity: 17, want: 7},
		{name: "capacity 24 maps to bucket 7", capacity: 24, want: 7},
		{name: "capacity 25 maps to bucket 8", capacity: 25, want: 8},
		{name: "capacity 32 maps to bucket 8", capacity: 32, want: 8},
		{name: "capacity 33 maps to bucket 9", capacity: 33, want: 9},
		{name: "capacity 48 maps to bucket 9", capacity: 48, want: 9},
		{name: "capacity 49 maps to bucket 10", capacity: 49, want: 10},
		{name: "capacity 64 maps to bucket 10", capacity: 64, want: 10},
		{name: "capacity 65 maps to bucket 11", capacity: 65, want: 11},
		{name: "capacity 96 maps to bucket 11", capacity: 96, want: 11},
		{name: "capacity 97 maps to bucket 12", capacity: 97, want: 12},
		{name: "capacity 128 maps to bucket 12", capacity: 128, want: 12},
		{name: "capacity 129 overflows to -1", capacity: 129, want: -1},
		{name: "capacity 1000 overflows to -1", capacity: 1000, want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := arena.childBucketIndex(tt.capacity)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestArena_attrBucketIndex(t *testing.T) {
	t.Parallel()

	arena := &RenderArena{}

	tests := []struct {
		name     string
		capacity int
		want     int
	}{
		{name: "capacity 1 maps to bucket 0", capacity: 1, want: 0},
		{name: "capacity 2 maps to bucket 0", capacity: 2, want: 0},
		{name: "capacity 3 maps to bucket 1", capacity: 3, want: 1},
		{name: "capacity 4 maps to bucket 1", capacity: 4, want: 1},
		{name: "capacity 5 maps to bucket 2", capacity: 5, want: 2},
		{name: "capacity 6 maps to bucket 2", capacity: 6, want: 2},
		{name: "capacity 7 maps to bucket 3", capacity: 7, want: 3},
		{name: "capacity 8 maps to bucket 3", capacity: 8, want: 3},
		{name: "capacity 9 maps to bucket 4", capacity: 9, want: 4},
		{name: "capacity 10 maps to bucket 4", capacity: 10, want: 4},
		{name: "capacity 11 maps to bucket 5", capacity: 11, want: 5},
		{name: "capacity 12 maps to bucket 5", capacity: 12, want: 5},
		{name: "capacity 13 maps to bucket 6", capacity: 13, want: 6},
		{name: "capacity 16 maps to bucket 6", capacity: 16, want: 6},
		{name: "capacity 17 overflows to -1", capacity: 17, want: -1},
		{name: "capacity 100 overflows to -1", capacity: 100, want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := arena.attributeBucketIndex(tt.capacity)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestArena_rootNodesBucketIndex(t *testing.T) {
	t.Parallel()

	arena := &RenderArena{}

	tests := []struct {
		name     string
		capacity int
		want     int
	}{
		{name: "capacity 1 maps to bucket 0", capacity: 1, want: 0},
		{name: "capacity 2 maps to bucket 1", capacity: 2, want: 1},
		{name: "capacity 3 maps to bucket 2", capacity: 3, want: 2},
		{name: "capacity 4 maps to bucket 2", capacity: 4, want: 2},
		{name: "capacity 5 maps to bucket 3", capacity: 5, want: 3},
		{name: "capacity 6 maps to bucket 3", capacity: 6, want: 3},
		{name: "capacity 7 maps to bucket 4", capacity: 7, want: 4},
		{name: "capacity 8 maps to bucket 4", capacity: 8, want: 4},
		{name: "capacity 9 maps to bucket 5", capacity: 9, want: 5},
		{name: "capacity 10 maps to bucket 5", capacity: 10, want: 5},
		{name: "capacity 11 maps to bucket 6", capacity: 11, want: 6},
		{name: "capacity 12 maps to bucket 6", capacity: 12, want: 6},
		{name: "capacity 13 maps to bucket 7", capacity: 13, want: 7},
		{name: "capacity 16 maps to bucket 7", capacity: 16, want: 7},
		{name: "capacity 17 maps to bucket 8", capacity: 17, want: 8},
		{name: "capacity 24 maps to bucket 8", capacity: 24, want: 8},
		{name: "capacity 25 maps to bucket 9", capacity: 25, want: 9},
		{name: "capacity 32 maps to bucket 9", capacity: 32, want: 9},
		{name: "capacity 33 maps to bucket 10", capacity: 33, want: 10},
		{name: "capacity 48 maps to bucket 10", capacity: 48, want: 10},
		{name: "capacity 49 maps to bucket 11", capacity: 49, want: 11},
		{name: "capacity 64 maps to bucket 11", capacity: 64, want: 11},
		{name: "capacity 65 maps to bucket 12", capacity: 65, want: 12},
		{name: "capacity 96 maps to bucket 12", capacity: 96, want: 12},
		{name: "capacity 97 maps to bucket 13", capacity: 97, want: 13},
		{name: "capacity 128 maps to bucket 13", capacity: 128, want: 13},
		{name: "capacity 129 overflows to -1", capacity: 129, want: -1},
		{name: "capacity 500 overflows to -1", capacity: 500, want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := arena.rootNodesBucketIndex(tt.capacity)
			assert.Equal(t, tt.want, got)
		})
	}
}
