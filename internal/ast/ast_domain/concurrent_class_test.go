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

package ast_domain

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestConcurrentDirectWriterClassBytes(t *testing.T) {
	t.Parallel()

	const goroutines = 50
	const iterations = 1000

	var errorCount atomic.Int64
	var wg sync.WaitGroup

	classValues := []string{
		"nav-item",
		"text-emphasis",
		"sidebar-link",
		"menu-item",
		"btn-primary",
		"active",
		"highlight",
		"disabled",
	}

	for g := range goroutines {
		id := g
		wg.Go(func() {
			for i := range iterations {

				for index, expected := range classValues {

					bufferPointer := GetByteBuf()
					*bufferPointer = append((*bufferPointer)[:0], expected...)

					dw := GetDirectWriter()
					dw.SetName("class")
					dw.AppendPooledBytes(bufferPointer)

					var output []byte
					output = dw.WriteTo(output)

					got := string(output)
					if got != expected {
						errorCount.Add(1)
						if errorCount.Load() <= 10 {
							t.Errorf("ERROR [g=%d, i=%d, index=%d]: expected %q, got %q",
								id, i, index, expected, got)
						}
					}

					PutDirectWriter(dw)
				}
			}
		})
	}

	wg.Wait()

	total := goroutines * iterations * len(classValues)
	errors := errorCount.Load()
	if errors > 0 {
		t.Errorf("Completed %d operations, %d errors (%.4f%%)",
			total, errors, float64(errors)/float64(total)*100)
	}
}

func TestConcurrentMixedDirectWriterOperations(t *testing.T) {
	t.Parallel()

	const goroutines = 50
	const iterations = 1000

	var errorCount atomic.Int64
	var wg sync.WaitGroup

	type testCase struct {
		name    string
		content string
		isClass bool
	}

	testCases := []testCase{
		{name: "class", content: "nav-item", isClass: true},
		{name: "p-event:click", content: "eyJmIjoiYWN0aW9uIn0", isClass: false},
		{name: "class", content: "text-emphasis btn", isClass: true},
		{name: "p-event:submit", content: "eyJmIjoic3VibWl0In0", isClass: false},
		{name: "class", content: "sidebar active", isClass: true},
	}

	for g := range goroutines {
		id := g
		wg.Go(func() {
			for i := range iterations {
				for index, tc := range testCases {
					bufferPointer := GetByteBuf()
					*bufferPointer = append((*bufferPointer)[:0], tc.content...)

					dw := GetDirectWriter()
					dw.SetName(tc.name)
					dw.AppendPooledBytes(bufferPointer)

					var output []byte
					output = dw.WriteTo(output)

					got := string(output)
					if got != tc.content {
						errorCount.Add(1)
						if errorCount.Load() <= 10 {
							t.Errorf("ERROR [g=%d, i=%d, index=%d, name=%s]: expected %q, got %q",
								id, i, index, tc.name, tc.content, got)
						}
					}

					PutDirectWriter(dw)
				}
			}
		})
	}

	wg.Wait()

	total := goroutines * iterations * len(testCases)
	errors := errorCount.Load()
	if errors > 0 {
		t.Errorf("Completed %d operations, %d errors (%.4f%%)",
			total, errors, float64(errors)/float64(total)*100)
	}
}
