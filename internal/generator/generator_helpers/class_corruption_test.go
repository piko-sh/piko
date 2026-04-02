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

package generator_helpers

import (
	"sync"
	"sync/atomic"
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
)

func TestMergeClassesBytesCrossContamination(t *testing.T) {
	t.Parallel()

	const goroutines = 50
	const iterations = 1000

	var errorCount atomic.Int64
	var wg sync.WaitGroup

	classValues := []struct {
		static   string
		dynamic  string
		expected string
	}{
		{static: "nav-item", dynamic: "active", expected: "nav-item active"},
		{static: "text-emphasis", dynamic: "highlight", expected: "text-emphasis highlight"},
		{static: "sidebar", dynamic: "collapsed", expected: "sidebar collapsed"},
		{static: "btn", dynamic: "primary", expected: "btn primary"},
	}

	for g := range goroutines {
		goroutineID := g
		wg.Go(func() {
			for i := range iterations {

				tc := classValues[i%len(classValues)]

				bufferPointer := MergeClassesBytes(tc.static, tc.dynamic)
				if bufferPointer == nil {
					errorCount.Add(1)
					t.Errorf("ERROR [g=%d, i=%d]: got nil buffer", goroutineID, i)
					continue
				}

				got := string(*bufferPointer)
				if got != tc.expected {
					errorCount.Add(1)
					if errorCount.Load() <= 10 {
						t.Errorf("ERROR [g=%d, i=%d]: expected %q, got %q",
							goroutineID, i, tc.expected, got)
					}
				}

				ast_domain.PutByteBuf(bufferPointer)
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

func TestMergeClassesBytesWithDirectWriter(t *testing.T) {
	t.Parallel()

	const goroutines = 50
	const iterations = 500

	var errorCount atomic.Int64
	var wg sync.WaitGroup

	classValues := []struct {
		static   string
		dynamic  string
		expected string
	}{
		{static: "nav-item", dynamic: "active", expected: "nav-item active"},
		{static: "text-emphasis", dynamic: "bold", expected: "text-emphasis bold"},
	}

	for g := range goroutines {
		goroutineID := g
		wg.Go(func() {
			for i := range iterations {
				tc := classValues[i%len(classValues)]

				bufferPointer := MergeClassesBytes(tc.static, tc.dynamic)
				if bufferPointer == nil {
					continue
				}

				dw := ast_domain.GetDirectWriter()
				dw.SetName("class")
				dw.AppendPooledBytes(bufferPointer)

				var output []byte
				output = dw.WriteTo(output)

				got := string(output)
				if got != tc.expected {
					errorCount.Add(1)
					if errorCount.Load() <= 10 {
						t.Errorf("ERROR [g=%d, i=%d]: expected %q, got %q",
							goroutineID, i, tc.expected, got)
					}
				}

				ast_domain.PutDirectWriter(dw)
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
