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

package generator_helpers

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestFullBuildRenderPutLifecycle(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const goroutines = 50
	const iterations = 200

	var errorCount atomic.Int64
	var wg sync.WaitGroup

	testCases := []struct {
		staticClass string
		dynamicMap  map[string]bool
		expected    string
	}{
		{staticClass: "nav-item", dynamicMap: map[string]bool{"active": true}, expected: "nav-item active"},
		{staticClass: "text-emphasis", dynamicMap: map[string]bool{"bold": true, "highlight": false}, expected: "text-emphasis bold"},
		{staticClass: "sidebar", dynamicMap: map[string]bool{"collapsed": true, "hidden": false}, expected: "sidebar collapsed"},
		{staticClass: "btn", dynamicMap: map[string]bool{"primary": true, "disabled": false}, expected: "btn primary"},
		{staticClass: "card", dynamicMap: map[string]bool{"selected": true, "expanded": true}, expected: "card expanded selected"},
	}

	for g := range goroutines {
		goroutineID := g
		wg.Go(func() {
			for i := range iterations {
				tc := testCases[i%len(testCases)]

				arena := ast_domain.GetArena()
				ast := ast_domain.GetTemplateAST()
				ast.SetArena(arena)
				ast.RootNodes = arena.GetRootNodesSlice(1)

				node := arena.GetNode()
				node.NodeType = ast_domain.NodeElement
				node.TagName = "div"
				node.IsPooled = true

				bufferPointer := MergeClassesBytes(tc.staticClass, tc.dynamicMap)
				if bufferPointer != nil {
					dw := arena.GetDirectWriter()
					dw.SetName("class")
					dw.AppendPooledBytes(bufferPointer)
					_, node.AttributeWriters = arena.GetAttrWriterSlice(1)
					node.AttributeWriters = append(node.AttributeWriters, dw)
				}

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

				ast_domain.PutTree(ast)
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

func TestMixedHelpersLifecycle(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const goroutines = 30
	const iterations = 100

	var errorCount atomic.Int64
	var wg sync.WaitGroup

	for g := range goroutines {
		goroutineID := g
		wg.Go(func() {
			for i := range iterations {

				arena := ast_domain.GetArena()
				ast := ast_domain.GetTemplateAST()
				ast.SetArena(arena)
				ast.RootNodes = arena.GetRootNodesSlice(1)

				node := arena.GetNode()
				node.NodeType = ast_domain.NodeElement
				node.TagName = "button"
				node.IsPooled = true

				expectedClass := fmt.Sprintf("btn btn-g%d", goroutineID)

				expectedStyle := fmt.Sprintf("color:red;margin:%dpx;", i%10)

				classBufPtr := MergeClassesBytes("btn", map[string]bool{fmt.Sprintf("btn-g%d", goroutineID): true})
				if classBufPtr != nil {
					dw := arena.GetDirectWriter()
					dw.SetName("class")
					dw.AppendPooledBytes(classBufPtr)
					_, node.AttributeWriters = arena.GetAttrWriterSlice(1)
					node.AttributeWriters = append(node.AttributeWriters, dw)
				}

				styleBufPtr := StylesFromStringMapBytes(map[string]string{
					"color":  "red",
					"margin": fmt.Sprintf("%dpx", i%10),
				})
				if styleBufPtr != nil {
					dw := arena.GetDirectWriter()
					dw.SetName("style")
					dw.AppendEscapePooledBytes(styleBufPtr)
					node.AttributeWriters = append(node.AttributeWriters, dw)
				}

				ast.RootNodes = append(ast.RootNodes, node)

				if len(ast.RootNodes) > 0 {
					for _, attributeDirectWriter := range ast.RootNodes[0].AttributeWriters {
						var output []byte
						output = attributeDirectWriter.WriteTo(output)
						got := string(output)

						var expected string
						switch attributeDirectWriter.Name {
						case "class":
							expected = expectedClass
						case "style":
							expected = expectedStyle
						}

						if got != expected {
							errorCount.Add(1)
							if errorCount.Load() <= 10 {
								t.Errorf("MIXED CORRUPTION [g=%d, i=%d, attr=%s]: expected %q, got %q",
									goroutineID, i, attributeDirectWriter.Name, expected, got)
							}
						}
					}
				}

				ast_domain.PutTree(ast)
			}
		})
	}

	wg.Wait()

	total := int64(goroutines * iterations * 2)
	errors := errorCount.Load()
	if errors > 0 {
		t.Errorf("Completed %d attribute renders, %d errors (%.4f%%)",
			total, errors, float64(errors)/float64(total)*100)
	}
}

func TestActionPayloadLifecycle(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const goroutines = 30
	const iterations = 100

	var errorCount atomic.Int64
	var wg sync.WaitGroup

	for g := range goroutines {
		goroutineID := g
		wg.Go(func() {

			for i := range iterations {

				arena := ast_domain.GetArena()
				ast := ast_domain.GetTemplateAST()
				ast.SetArena(arena)
				ast.RootNodes = arena.GetRootNodesSlice(1)

				node := arena.GetNode()
				node.NodeType = ast_domain.NodeElement
				node.TagName = "button"
				node.IsPooled = true

				functionName := fmt.Sprintf("handleClick_g%d_i%d", goroutineID, i)
				payload := templater_dto.ActionPayload{
					Function: functionName,
					Args: []templater_dto.ActionArgument{
						{Type: "s", Value: "arg1"},
						{Type: "s", Value: i},
					},
				}

				actionBufPtr := EncodeActionPayloadBytes(payload)
				if actionBufPtr != nil {
					dw := arena.GetDirectWriter()
					dw.SetName("p-event:click")
					dw.AppendPooledBytes(actionBufPtr)
					_, node.AttributeWriters = arena.GetAttrWriterSlice(1)
					node.AttributeWriters = append(node.AttributeWriters, dw)
				}

				ast.RootNodes = append(ast.RootNodes, node)

				if len(ast.RootNodes) > 0 && len(ast.RootNodes[0].AttributeWriters) > 0 {
					renderDW := ast.RootNodes[0].AttributeWriters[0]
					var output []byte
					output = renderDW.WriteTo(output)
					got := string(output)

					if len(got) == 0 {
						errorCount.Add(1)
						if errorCount.Load() <= 10 {
							t.Errorf("ACTION EMPTY [g=%d, i=%d]: got empty output", goroutineID, i)
						}
					}

				}

				ast_domain.PutTree(ast)
			}
		})
	}

	wg.Wait()

	total := int64(goroutines * iterations)
	errors := errorCount.Load()
	if errors > 0 {
		t.Errorf("Completed %d action operations, %d errors (%.4f%%)",
			total, errors, float64(errors)/float64(total)*100)
	}
}

func TestNestedASTLifecycle(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const goroutines = 30
	const iterations = 100

	var errorCount atomic.Int64
	var wg sync.WaitGroup

	for g := range goroutines {
		goroutineID := g
		wg.Go(func() {

			for i := range iterations {

				outerExpectedClass := fmt.Sprintf("outer-g%d-i%d", goroutineID, i)
				outerArena := ast_domain.GetArena()
				outerAST := ast_domain.GetTemplateAST()
				outerAST.SetArena(outerArena)
				outerAST.RootNodes = outerArena.GetRootNodesSlice(2)

				outerNode := outerArena.GetNode()
				outerNode.NodeType = ast_domain.NodeElement
				outerNode.TagName = "div"
				outerNode.IsPooled = true

				outerBufPtr := ClassesFromStringBytes(outerExpectedClass)
				if outerBufPtr != nil {
					dw := outerArena.GetDirectWriter()
					dw.SetName("class")
					dw.AppendPooledBytes(outerBufPtr)
					_, outerNode.AttributeWriters = outerArena.GetAttrWriterSlice(1)
					outerNode.AttributeWriters = append(outerNode.AttributeWriters, dw)
				}
				outerAST.RootNodes = append(outerAST.RootNodes, outerNode)

				innerExpectedClass := fmt.Sprintf("inner-g%d-i%d", goroutineID, i)
				innerArena := ast_domain.GetArena()
				innerAST := ast_domain.GetTemplateAST()
				innerAST.SetArena(innerArena)
				innerAST.RootNodes = innerArena.GetRootNodesSlice(1)

				innerNode := innerArena.GetNode()
				innerNode.NodeType = ast_domain.NodeElement
				innerNode.TagName = "span"
				innerNode.IsPooled = true

				innerBufPtr := ClassesFromStringBytes(innerExpectedClass)
				if innerBufPtr != nil {
					dw := innerArena.GetDirectWriter()
					dw.SetName("class")
					dw.AppendPooledBytes(innerBufPtr)
					_, innerNode.AttributeWriters = innerArena.GetAttrWriterSlice(1)
					innerNode.AttributeWriters = append(innerNode.AttributeWriters, dw)
				}
				innerAST.RootNodes = append(innerAST.RootNodes, innerNode)

				outerNode.Children = append(outerNode.Children, innerAST.RootNodes...)

				for _, node := range outerAST.RootNodes {
					if len(node.AttributeWriters) > 0 {
						var output []byte
						output = node.AttributeWriters[0].WriteTo(output)
						got := string(output)

						if node.TagName == "div" && got != outerExpectedClass {
							errorCount.Add(1)
							if errorCount.Load() <= 10 {
								t.Errorf("NESTED OUTER [g=%d, i=%d]: expected %q, got %q",
									goroutineID, i, outerExpectedClass, got)
							}
						}
					}

					for _, child := range node.Children {
						if len(child.AttributeWriters) > 0 {
							var output []byte
							output = child.AttributeWriters[0].WriteTo(output)
							got := string(output)

							if got != innerExpectedClass {
								errorCount.Add(1)
								if errorCount.Load() <= 10 {
									t.Errorf("NESTED INNER [g=%d, i=%d]: expected %q, got %q",
										goroutineID, i, innerExpectedClass, got)
								}
							}
						}
					}
				}

				ast_domain.PutTree(innerAST)
				ast_domain.PutTree(outerAST)
			}
		})
	}

	wg.Wait()

	total := int64(goroutines * iterations * 2)
	errors := errorCount.Load()
	if errors > 0 {
		t.Errorf("Completed %d nested operations, %d errors (%.4f%%)",
			total, errors, float64(errors)/float64(total)*100)
	}
}

func TestRapidPoolCycling(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const goroutines = 50
	const iterations = 1000

	var errorCount atomic.Int64
	var wg sync.WaitGroup

	classes := []string{
		"nav-item",
		"text-emphasis",
		"sidebar-link",
		"menu-item active",
		"btn primary disabled",
	}

	for g := range goroutines {
		goroutineID := g
		wg.Go(func() {
			for i := range iterations {
				expectedClass := classes[i%len(classes)]

				bufferPointer := ast_domain.GetByteBuf()
				*bufferPointer = append((*bufferPointer)[:0], expectedClass...)

				got := string(*bufferPointer)
				if got != expectedClass {
					errorCount.Add(1)
					if errorCount.Load() <= 10 {
						t.Errorf("RAPID CYCLE [g=%d, i=%d]: expected %q, got %q",
							goroutineID, i, expectedClass, got)
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
		t.Errorf("Completed %d rapid cycles, %d errors (%.4f%%)",
			total, errors, float64(errors)/float64(total)*100)
	}
}

func TestDirectWriterReusePattern(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const goroutines = 30
	const iterations = 200

	var errorCount atomic.Int64
	var wg sync.WaitGroup

	for g := range goroutines {
		goroutineID := g
		wg.Go(func() {
			for i := range iterations {

				part1 := fmt.Sprintf("part1-g%d", goroutineID)
				part2 := fmt.Sprintf("part2-i%d", i)
				expected := part1 + " " + part2

				arena := ast_domain.GetArena()
				ast := ast_domain.GetTemplateAST()
				ast.SetArena(arena)
				ast.RootNodes = arena.GetRootNodesSlice(1)

				node := arena.GetNode()
				node.NodeType = ast_domain.NodeElement
				node.TagName = "div"
				node.IsPooled = true

				dw := arena.GetDirectWriter()
				dw.SetName("class")

				buf1Ptr := ast_domain.GetByteBuf()
				*buf1Ptr = append((*buf1Ptr)[:0], part1...)
				dw.AppendPooledBytes(buf1Ptr)

				dw.AppendString(" ")

				buf2Ptr := ast_domain.GetByteBuf()
				*buf2Ptr = append((*buf2Ptr)[:0], part2...)
				dw.AppendPooledBytes(buf2Ptr)

				_, node.AttributeWriters = arena.GetAttrWriterSlice(1)
				node.AttributeWriters = append(node.AttributeWriters, dw)
				ast.RootNodes = append(ast.RootNodes, node)

				if len(ast.RootNodes) > 0 && len(ast.RootNodes[0].AttributeWriters) > 0 {
					var output []byte
					output = ast.RootNodes[0].AttributeWriters[0].WriteTo(output)
					got := string(output)

					if got != expected {
						errorCount.Add(1)
						if errorCount.Load() <= 10 {
							t.Errorf("MULTI-PART [g=%d, i=%d]: expected %q, got %q",
								goroutineID, i, expected, got)
						}
					}
				}

				ast_domain.PutTree(ast)
			}
		})
	}

	wg.Wait()

	total := int64(goroutines * iterations)
	errors := errorCount.Load()
	if errors > 0 {
		t.Errorf("Completed %d multi-part operations, %d errors (%.4f%%)",
			total, errors, float64(errors)/float64(total)*100)
	}
}

func TestHighContentionPoolLifecycle(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const goroutines = 100
	const iterations = 500

	var errorCount atomic.Int64
	var wg sync.WaitGroup

	for g := range goroutines {
		goroutineID := g
		wg.Go(func() {
			for i := range iterations {
				expectedClass := fmt.Sprintf("class-g%d-i%d", goroutineID, i)

				arena := ast_domain.GetArena()
				ast := ast_domain.GetTemplateAST()
				ast.SetArena(arena)
				ast.RootNodes = arena.GetRootNodesSlice(1)

				node := arena.GetNode()
				node.NodeType = ast_domain.NodeElement
				node.TagName = "span"
				node.IsPooled = true

				bufferPointer := ClassesFromStringBytes(expectedClass)
				if bufferPointer != nil {
					dw := arena.GetDirectWriter()
					dw.SetName("class")
					dw.AppendPooledBytes(bufferPointer)
					_, node.AttributeWriters = arena.GetAttrWriterSlice(1)
					node.AttributeWriters = append(node.AttributeWriters, dw)
				}

				ast.RootNodes = append(ast.RootNodes, node)

				if len(ast.RootNodes) > 0 && len(ast.RootNodes[0].AttributeWriters) > 0 {
					var output []byte
					output = ast.RootNodes[0].AttributeWriters[0].WriteTo(output)
					got := string(output)

					if got != expectedClass {
						errorCount.Add(1)
						if errorCount.Load() <= 10 {
							t.Errorf("HIGH CONTENTION [g=%d, i=%d]: expected %q, got %q",
								goroutineID, i, expectedClass, got)
						}
					}
				}

				ast_domain.PutTree(ast)
			}
		})
	}

	wg.Wait()

	total := int64(goroutines * iterations)
	errors := errorCount.Load()
	if errors > 0 {
		t.Errorf("Completed %d high-contention operations, %d errors (%.4f%%)",
			total, errors, float64(errors)/float64(total)*100)
	}
}
