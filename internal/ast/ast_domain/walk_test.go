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
	"context"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestWalkAST(t *testing.T) *TemplateAST {
	t.Helper()
	source := `
		<div id="1">
			<p id="2">
				<span id="3"></span>
			</p>
			<span id="4"></span>
		</div>
		<Fragment id="5">
			<h1 id="6"></h1>
			<div id="7">
				<strong id="8"></strong>
			</div>
		</Fragment>
	`
	tree, err := ParseAndTransform(context.Background(), source, "test-walk.html")
	require.NoError(t, err)
	require.False(t, HasErrors(tree.Diagnostics))
	return tree
}

func getNodeID(t *testing.T, n *TemplateNode) string {
	t.Helper()
	if n == nil {
		return "<nil>"
	}
	value, ok := n.GetAttribute("id")
	if !ok {
		return ""
	}
	return value
}

func TestWalk_StandardIterators(t *testing.T) {
	testAST := createTestWalkAST(t)
	expectedPreOrderIDs := []string{"1", "2", "3", "4", "5", "6", "7", "8"}

	t.Run("Nodes iterates in pre-order", func(t *testing.T) {
		var visitedIDs []string
		for node := range testAST.Nodes() {
			visitedIDs = append(visitedIDs, getNodeID(t, node))
		}
		assert.Equal(t, expectedPreOrderIDs, visitedIDs)
	})

	t.Run("Nodes iterator can be stopped early with break", func(t *testing.T) {
		var visitedIDs []string
		for node := range testAST.Nodes() {
			id := getNodeID(t, node)
			visitedIDs = append(visitedIDs, id)
			if id == "4" {
				break
			}
		}
		assert.Equal(t, []string{"1", "2", "3", "4"}, visitedIDs)
	})

	t.Run("NodesWithParent provides correct parent-child relationships", func(t *testing.T) {
		parentMap := make(map[string]string)
		for node, parent := range testAST.NodesWithParent() {
			parentMap[getNodeID(t, node)] = getNodeID(t, parent)
		}

		expectedParents := map[string]string{
			"1": "<nil>", "2": "1", "3": "2", "4": "1",
			"5": "<nil>", "6": "5", "7": "5", "8": "7",
		}
		assert.Equal(t, expectedParents, parentMap)
	})

	t.Run("Nodes on a nil AST does not panic", func(t *testing.T) {
		var nilAST *TemplateAST
		count := 0
		for range nilAST.Nodes() {
			count++
		}
		assert.Zero(t, count)
	})
}

func TestWalk_ClassicIterator(t *testing.T) {
	testAST := createTestWalkAST(t)

	t.Run("Next traverses in pre-order and provides correct Parent", func(t *testing.T) {
		var visitedIDs []string
		it := testAST.NewIterator()
		for it.Next() {
			visitedIDs = append(visitedIDs, getNodeID(t, it.Node))
			if getNodeID(t, it.Node) == "3" {
				assert.Equal(t, "2", getNodeID(t, it.Parent), "Parent of node 3 should be node 2")
			}
			if getNodeID(t, it.Node) == "1" {
				assert.Nil(t, it.Parent, "Parent of root node 1 should be nil")
			}
		}
		assert.Equal(t, []string{"1", "2", "3", "4", "5", "6", "7", "8"}, visitedIDs)
	})

	t.Run("SkipChildren prevents traversal of a subtree", func(t *testing.T) {
		var visitedIDs []string
		it := testAST.NewIterator()
		for it.Next() {
			id := getNodeID(t, it.Node)
			visitedIDs = append(visitedIDs, id)
			if id == "2" {
				it.SkipChildren()
			}
		}
		assert.Equal(t, []string{"1", "2", "4", "5", "6", "7", "8"}, visitedIDs)
	})

	t.Run("SkipChildren on a fragment node works correctly", func(t *testing.T) {
		var visitedIDs []string
		it := testAST.NewIterator()
		for it.Next() {
			id := getNodeID(t, it.Node)
			visitedIDs = append(visitedIDs, id)
			if id == "5" {
				it.SkipChildren()
			}
		}
		assert.Equal(t, []string{"1", "2", "3", "4", "5"}, visitedIDs)
	})

	t.Run("NewIterator on nil AST returns empty iterator", func(t *testing.T) {
		var nilAST *TemplateAST
		it := nilAST.NewIterator()
		assert.False(t, it.Next())
	})
}

func TestWalk_PostOrderIterator(t *testing.T) {
	testAST := createTestWalkAST(t)

	t.Run("Next traverses in post-order", func(t *testing.T) {
		expectedPostOrderIDs := []string{"3", "2", "4", "1", "6", "8", "7", "5"}
		var visitedIDs []string
		it := testAST.NewPostOrderIterator()
		for it.Next() {
			visitedIDs = append(visitedIDs, getNodeID(t, it.Node))
		}
		assert.Equal(t, expectedPostOrderIDs, visitedIDs)
	})

	t.Run("NewPostOrderIterator on nil AST returns empty iterator", func(t *testing.T) {
		var nilAST *TemplateAST
		it := nilAST.NewPostOrderIterator()
		assert.False(t, it.Next())
	})
}

type testVisitor struct {
	t              *testing.T
	PruneAtID      string
	ErrorOnEnterID string
	ErrorOnExitID  string
	EnterOrder     []string
	ExitOrder      []string
}

func (v *testVisitor) Enter(_ context.Context, node *TemplateNode) (Visitor, error) {
	id := getNodeID(v.t, node)
	v.EnterOrder = append(v.EnterOrder, id)

	if id == v.ErrorOnEnterID {
		return nil, errors.New("error on enter at " + id)
	}
	if id == v.PruneAtID {
		return nil, nil
	}
	return v, nil
}

func (v *testVisitor) Exit(_ context.Context, node *TemplateNode) error {
	id := getNodeID(v.t, node)
	v.ExitOrder = append(v.ExitOrder, id)

	if id == v.ErrorOnExitID {
		return errors.New("error on exit at " + id)
	}
	return nil
}

type testVisitorWithSharedState struct {
	t            *testing.T
	EnterOrder   *[]string
	ExitOrder    *[]string
	newVisitor   func(string) *testVisitorWithSharedState
	SwitchAtID   string
	visitorState string
}

func (v *testVisitorWithSharedState) Enter(_ context.Context, node *TemplateNode) (Visitor, error) {
	id := getNodeID(v.t, node)
	*v.EnterOrder = append(*v.EnterOrder, id+":"+v.visitorState)
	if id == v.SwitchAtID {
		return v.newVisitor("switched"), nil
	}
	return v, nil
}

func (v *testVisitorWithSharedState) Exit(_ context.Context, node *TemplateNode) error {
	id := getNodeID(v.t, node)
	*v.ExitOrder = append(*v.ExitOrder, id)
	return nil
}

func TestWalk_ScopedVisitor(t *testing.T) {
	testAST := createTestWalkAST(t)

	t.Run("Visitor traverses in correct Enter/Exit order", func(t *testing.T) {
		visitor := &testVisitor{t: t}
		err := testAST.Accept(context.Background(), visitor)
		require.NoError(t, err)

		expectedEnter := []string{"1", "2", "3", "4", "5", "6", "7", "8"}
		expectedExit := []string{"3", "2", "4", "1", "6", "8", "7", "5"}
		assert.Equal(t, expectedEnter, visitor.EnterOrder)
		assert.Equal(t, expectedExit, visitor.ExitOrder)
	})

	t.Run("Visitor can prune a subtree by returning nil from Enter", func(t *testing.T) {
		visitor := &testVisitor{t: t, PruneAtID: "2"}
		err := testAST.Accept(context.Background(), visitor)
		require.NoError(t, err)

		expectedEnter := []string{"1", "2", "4", "5", "6", "7", "8"}
		expectedExit := []string{"2", "4", "1", "6", "8", "7", "5"}
		assert.Equal(t, expectedEnter, visitor.EnterOrder)
		assert.Equal(t, expectedExit, visitor.ExitOrder)
	})

	t.Run("Visitor can halt traversal by returning an error from Enter", func(t *testing.T) {
		visitor := &testVisitor{t: t, ErrorOnEnterID: "4"}
		err := testAST.Accept(context.Background(), visitor)
		require.Error(t, err)
		assert.Equal(t, "error on enter at 4", err.Error())

		expectedEnter := []string{"1", "2", "3", "4"}
		assert.Equal(t, expectedEnter, visitor.EnterOrder)

		expectedExit := []string{"3", "2"}
		assert.Equal(t, expectedExit, visitor.ExitOrder, "Exit should have been called for nodes whose subtrees were fully visited before the error")
	})

	t.Run("Visitor can halt traversal by returning an error from Exit", func(t *testing.T) {
		visitor := &testVisitor{t: t, ErrorOnExitID: "2"}
		err := testAST.Accept(context.Background(), visitor)
		require.Error(t, err)
		assert.Equal(t, "error on exit at 2", err.Error())

		expectedEnterOrder := []string{"1", "2", "3"}
		assert.Equal(t, expectedEnterOrder, visitor.EnterOrder, "Enter should have been called on nodes up to the point of failure")

		expectedExitOrder := []string{"3", "2"}
		assert.Equal(t, expectedExitOrder, visitor.ExitOrder, "Traversal should halt immediately after the erroring Exit call")
	})

	t.Run("Visitor can switch context for a subtree", func(t *testing.T) {
		enterOrder := &[]string{}
		exitOrder := &[]string{}
		var newVisitor func(string) *testVisitorWithSharedState
		newVisitor = func(state string) *testVisitorWithSharedState {
			return &testVisitorWithSharedState{
				t:            t,
				EnterOrder:   enterOrder,
				ExitOrder:    exitOrder,
				SwitchAtID:   "7",
				visitorState: state,
				newVisitor:   newVisitor,
			}
		}

		visitor := newVisitor("initial")
		err := testAST.Accept(context.Background(), visitor)
		require.NoError(t, err)

		expectedEnter := []string{
			"1:initial", "2:initial", "3:initial", "4:initial",
			"5:initial", "6:initial", "7:initial", "8:switched",
		}
		expectedExit := []string{"3", "2", "4", "1", "6", "8", "7", "5"}

		assert.Equal(t, expectedEnter, *visitor.EnterOrder)
		assert.Equal(t, expectedExit, *visitor.ExitOrder)
	})
}

func TestWalk_StreamNodes(t *testing.T) {
	testAST := createTestWalkAST(t)
	expectedPreOrderIDs := []string{"1", "2", "3", "4", "5", "6", "7", "8"}

	t.Run("Streams nodes correctly in pre-order", func(t *testing.T) {
		var visitedIDs []string
		ctx := context.Background()
		for node := range testAST.StreamNodes(ctx) {
			visitedIDs = append(visitedIDs, getNodeID(t, node))
		}
		assert.Equal(t, expectedPreOrderIDs, visitedIDs)
	})

	t.Run("Stops streaming when context is cancelled", func(t *testing.T) {
		var visitedIDs []string
		var wg sync.WaitGroup
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(fmt.Errorf("test: cleanup"))

		nodeCh := testAST.StreamNodes(ctx)
		wg.Go(func() {
			for node := range nodeCh {
				id := getNodeID(t, node)
				visitedIDs = append(visitedIDs, id)
				if id == "4" {
					cancel(fmt.Errorf("test: simulating cancelled context"))
				}
			}
		})

		wg.Wait()
		time.Sleep(10 * time.Millisecond)
		assert.Less(t, len(visitedIDs), len(expectedPreOrderIDs))
		assert.Contains(t, visitedIDs, "4")
		assert.NotContains(t, visitedIDs, "8")
	})
}

type idCollectorVisitor struct {
	Error      error
	t          *testing.T
	SkipID     string
	ErrorOnID  string
	VisitedIDs []string
}

func (v *idCollectorVisitor) Visit(node *TemplateNode) (VisitorWithError, error) {
	id := getNodeID(v.t, node)
	v.VisitedIDs = append(v.VisitedIDs, id)

	if id == v.ErrorOnID {
		return nil, v.Error
	}
	if id == v.SkipID {
		return nil, nil
	}
	return v, nil
}

func TestWalk_VisitorWithError(t *testing.T) {
	testAST := createTestWalkAST(t)

	t.Run("Visitor traverses all nodes", func(t *testing.T) {
		visitor := &idCollectorVisitor{t: t}
		err := testAST.AcceptWithError(visitor)
		require.NoError(t, err)
		assert.Equal(t, []string{"1", "2", "3", "4", "5", "6", "7", "8"}, visitor.VisitedIDs)
	})

	t.Run("Visitor can skip children by returning nil", func(t *testing.T) {
		visitor := &idCollectorVisitor{t: t, SkipID: "2"}
		err := testAST.AcceptWithError(visitor)
		require.NoError(t, err)
		assert.Equal(t, []string{"1", "2", "4", "5", "6", "7", "8"}, visitor.VisitedIDs, "Node 3 should be skipped")
	})

	t.Run("Visitor can halt traversal by returning an error", func(t *testing.T) {
		testErr := errors.New("stop traversal")
		visitor := &idCollectorVisitor{t: t, ErrorOnID: "4", Error: testErr}
		err := testAST.AcceptWithError(visitor)
		require.Error(t, err)
		assert.ErrorIs(t, err, testErr)
		assert.Equal(t, []string{"1", "2", "3", "4"}, visitor.VisitedIDs, "Traversal should stop at node 4")
	})
}

func TestWalk_FunctionalWalk(t *testing.T) {
	testAST := createTestWalkAST(t)

	t.Run("Walk traverses all nodes", func(t *testing.T) {
		var visitedIDs []string
		testAST.Walk(func(node *TemplateNode) bool {
			visitedIDs = append(visitedIDs, getNodeID(t, node))
			return true
		})
		assert.Equal(t, []string{"1", "2", "3", "4", "5", "6", "7", "8"}, visitedIDs)
	})

	t.Run("Walk can be halted by returning false", func(t *testing.T) {
		var visitedIDs []string
		testAST.Walk(func(node *TemplateNode) bool {
			id := getNodeID(t, node)
			visitedIDs = append(visitedIDs, id)
			return id != "2"
		})
		assert.Equal(t, []string{"1", "2"}, visitedIDs)
	})
}

func TestWalk_FindHelpers(t *testing.T) {
	testAST := createTestWalkAST(t)

	t.Run("Find returns the first matching node", func(t *testing.T) {
		foundNode := testAST.Find(func(node *TemplateNode) bool {
			return node.TagName == "span"
		})
		require.NotNil(t, foundNode)
		assert.Equal(t, "3", getNodeID(t, foundNode))
	})

	t.Run("Find returns nil if no node matches", func(t *testing.T) {
		foundNode := testAST.Find(func(node *TemplateNode) bool {
			return node.TagName == "img"
		})
		assert.Nil(t, foundNode)
	})

	t.Run("FindAll returns all matching nodes", func(t *testing.T) {
		foundNodes := testAST.FindAll(func(node *TemplateNode) bool {
			return node.TagName == "div"
		})
		require.Len(t, foundNodes, 2)
		assert.Equal(t, "1", getNodeID(t, foundNodes[0]))
		assert.Equal(t, "7", getNodeID(t, foundNodes[1]))
	})

	t.Run("FindAll returns an empty slice if no nodes match", func(t *testing.T) {
		foundNodes := testAST.FindAll(func(node *TemplateNode) bool {
			return node.TagName == "img"
		})
		require.NotNil(t, foundNodes)
		assert.Empty(t, foundNodes)
	})

	t.Run("FindAll with a complex predicate", func(t *testing.T) {
		foundNodes := testAST.FindAll(func(node *TemplateNode) bool {
			return len(node.Children) == 0
		})
		require.Len(t, foundNodes, 4)
		ids := make([]string, len(foundNodes))
		for i, n := range foundNodes {
			ids[i] = getNodeID(t, n)
		}
		assert.ElementsMatch(t, []string{"3", "4", "6", "8"}, ids)
	})
}

func TestWalk_ParallelWalk(t *testing.T) {
	testAST := createTestWalkAST(t)

	t.Run("all nodes are visited", func(t *testing.T) {
		var visitedCount atomic.Int32
		var mu sync.Mutex
		visitedIDs := make(map[string]bool)

		err := testAST.ParallelWalk(context.Background(), runtime.NumCPU(), func(ctx context.Context, node *TemplateNode) error {
			visitedCount.Add(1)
			mu.Lock()
			visitedIDs[getNodeID(t, node)] = true
			mu.Unlock()
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, int32(8), visitedCount.Load())
		assert.Len(t, visitedIDs, 8)
		for i := 1; i <= 8; i++ {
			assert.True(t, visitedIDs[strconv.Itoa(i)], "Node %d should have been visited", i)
		}
	})

	t.Run("error in one worker cancels the walk and returns the error", func(t *testing.T) {
		var visitedCount atomic.Int32
		var mu sync.Mutex
		visitedIDs := make(map[string]bool)
		testErr := errors.New("critical failure")

		err := testAST.ParallelWalk(context.Background(), 4, func(ctx context.Context, node *TemplateNode) error {
			id := getNodeID(t, node)
			mu.Lock()
			visitedIDs[id] = true
			mu.Unlock()
			visitedCount.Add(1)

			if id == "4" {
				return testErr
			}

			select {
			case <-time.After(20 * time.Millisecond):
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, testErr)
		assert.Less(t, visitedCount.Load(), int32(8), "Not all nodes should be processed after cancellation")
		assert.True(t, visitedIDs["4"], "The failing node must have been visited")
	})

	t.Run("external context cancellation stops the walk", func(t *testing.T) {
		var visitedCount atomic.Int32
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)

		entered := make(chan struct{}, 1)

		go func() {
			<-entered
			cancel(fmt.Errorf("test: simulating external cancellation"))
		}()

		err := testAST.ParallelWalk(ctx, 2, func(ctx context.Context, node *TemplateNode) error {
			visitedCount.Add(1)
			select {
			case entered <- struct{}{}:
			default:
			}
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Less(t, visitedCount.Load(), int32(8))
	})

	t.Run("nil AST is handled gracefully", func(t *testing.T) {
		var nilAST *TemplateAST
		err := nilAST.ParallelWalk(context.Background(), 4, func(ctx context.Context, node *TemplateNode) error {
			t.Fatal("walk function should not be called for nil AST")
			return nil
		})
		assert.NoError(t, err)
	})
}

func TestWalk_TemplateNodeMethods(t *testing.T) {
	testAST := createTestWalkAST(t)
	startNode := testAST.RootNodes[0]
	require.NotNil(t, startNode)

	expectedSubtreePreOrderIDs := []string{"1", "2", "3", "4"}
	expectedSubtreePostOrderIDs := []string{"3", "2", "4", "1"}

	t.Run("Node.Nodes iterates subtree in pre-order", func(t *testing.T) {
		var visitedIDs []string
		for node := range startNode.Nodes() {
			visitedIDs = append(visitedIDs, getNodeID(t, node))
		}
		assert.Equal(t, expectedSubtreePreOrderIDs, visitedIDs)
	})

	t.Run("Node.NodesWithParent provides correct relationships", func(t *testing.T) {
		parentMap := make(map[string]string)
		for node, parent := range startNode.NodesWithParent() {
			parentMap[getNodeID(t, node)] = getNodeID(t, parent)
		}
		expectedParents := map[string]string{
			"1": "<nil>", "2": "1", "3": "2", "4": "1",
		}
		assert.Equal(t, expectedParents, parentMap)
	})

	t.Run("Node.Walk traverses subtree", func(t *testing.T) {
		var visitedIDs []string
		startNode.Walk(func(node *TemplateNode) bool {
			visitedIDs = append(visitedIDs, getNodeID(t, node))
			return true
		})
		assert.Equal(t, expectedSubtreePreOrderIDs, visitedIDs)
	})

	t.Run("Node.NewIterator traverses subtree", func(t *testing.T) {
		var visitedIDs []string
		it := startNode.NewIterator()
		for it.Next() {
			visitedIDs = append(visitedIDs, getNodeID(t, it.Node))
		}
		assert.Equal(t, expectedSubtreePreOrderIDs, visitedIDs)
	})

	t.Run("Node.NewPostOrderIterator traverses subtree", func(t *testing.T) {
		var visitedIDs []string
		it := startNode.NewPostOrderIterator()
		for it.Next() {
			visitedIDs = append(visitedIDs, getNodeID(t, it.Node))
		}
		assert.Equal(t, expectedSubtreePostOrderIDs, visitedIDs)
	})

	t.Run("Node.Find finds first match in subtree", func(t *testing.T) {
		found := startNode.Find(func(node *TemplateNode) bool {
			return node.TagName == "span"
		})
		require.NotNil(t, found)
		assert.Equal(t, "3", getNodeID(t, found))

		found = startNode.Find(func(node *TemplateNode) bool {
			return node.TagName == "h1"
		})
		assert.Nil(t, found)
	})

	t.Run("Node.FindAll finds all matches in subtree", func(t *testing.T) {
		found := startNode.FindAll(func(node *TemplateNode) bool {
			return node.TagName == "span"
		})
		require.Len(t, found, 2)
		assert.Equal(t, "3", getNodeID(t, found[0]))
		assert.Equal(t, "4", getNodeID(t, found[1]))
	})

	t.Run("Node.Accept with Visitor traverses subtree", func(t *testing.T) {
		visitor := &testVisitor{t: t}
		err := startNode.Accept(context.Background(), visitor)
		require.NoError(t, err)
		assert.Equal(t, []string{"1", "2", "3", "4"}, visitor.EnterOrder)
		assert.Equal(t, []string{"3", "2", "4", "1"}, visitor.ExitOrder)
	})

	t.Run("Node.ParallelWalk visits all nodes in subtree", func(t *testing.T) {
		var visitedCount atomic.Int32
		var mu sync.Mutex
		visitedIDs := make(map[string]bool)

		err := startNode.ParallelWalk(context.Background(), runtime.NumCPU(), func(ctx context.Context, node *TemplateNode) error {
			visitedCount.Add(1)
			mu.Lock()
			visitedIDs[getNodeID(t, node)] = true
			mu.Unlock()
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, int32(4), visitedCount.Load())
		assert.Len(t, visitedIDs, 4)
		assert.Contains(t, visitedIDs, "1")
		assert.Contains(t, visitedIDs, "2")
		assert.Contains(t, visitedIDs, "3")
		assert.Contains(t, visitedIDs, "4")
	})

	t.Run("Methods on nil TemplateNode do not panic", func(t *testing.T) {
		var nilNode *TemplateNode
		assert.NotPanics(t, func() {
			count := 0
			for range nilNode.Nodes() {
				count++
			}
			assert.Zero(t, count)

			it := nilNode.NewIterator()
			assert.False(t, it.Next())

			postIt := nilNode.NewPostOrderIterator()
			assert.False(t, postIt.Next())

			nilNode.Walk(func(node *TemplateNode) bool {
				t.Fatal("walk func should not be called")
				return false
			})

			assert.Nil(t, nilNode.Find(func(n *TemplateNode) bool { return true }))
			assert.Nil(t, nilNode.FindAll(func(n *TemplateNode) bool { return true }))

			err := nilNode.ParallelWalk(context.Background(), 2, func(ctx context.Context, n *TemplateNode) error {
				t.Fatal("parallel walk func should not be called")
				return nil
			})
			assert.NoError(t, err)
		})
	})
}

func TestWalkNodeExpressions(t *testing.T) {
	t.Run("nil node is handled gracefully", func(t *testing.T) {
		var visited []Expression
		WalkNodeExpressions(nil, func(expression Expression) {
			visited = append(visited, expression)
		})
		assert.Empty(t, visited)
	})

	t.Run("visits directive expressions", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
			DirIf:    &Directive{Expression: &Identifier{Name: "isVisible"}},
			DirFor:   &Directive{Expression: &ForInExpression{ItemVariable: &Identifier{Name: "item"}, Collection: &Identifier{Name: "items"}}},
			DirShow:  &Directive{Expression: &BooleanLiteral{Value: true}},
		}

		var visited []Expression
		WalkNodeExpressions(node, func(expression Expression) {
			visited = append(visited, expression)
		})

		assert.Len(t, visited, 3)
		assert.IsType(t, &Identifier{}, visited[0])
		assert.IsType(t, &ForInExpression{}, visited[1])
		assert.IsType(t, &BooleanLiteral{}, visited[2])
	})

	t.Run("visits all directive types", func(t *testing.T) {
		node := &TemplateNode{
			NodeType:    NodeElement,
			TagName:     "div",
			DirIf:       &Directive{Expression: &Identifier{Name: "dirIf"}},
			DirElseIf:   &Directive{Expression: &Identifier{Name: "dirElseIf"}},
			DirFor:      &Directive{Expression: &Identifier{Name: "dirFor"}},
			DirShow:     &Directive{Expression: &Identifier{Name: "dirShow"}},
			DirModel:    &Directive{Expression: &Identifier{Name: "dirModel"}},
			DirClass:    &Directive{Expression: &Identifier{Name: "dirClass"}},
			DirStyle:    &Directive{Expression: &Identifier{Name: "dirStyle"}},
			DirText:     &Directive{Expression: &Identifier{Name: "dirText"}},
			DirHTML:     &Directive{Expression: &Identifier{Name: "dirHTML"}},
			DirKey:      &Directive{Expression: &Identifier{Name: "dirKey"}},
			DirRef:      &Directive{Expression: &Identifier{Name: "dirRef"}},
			DirContext:  &Directive{Expression: &Identifier{Name: "dirContext"}},
			DirScaffold: &Directive{Expression: &Identifier{Name: "dirScaffold"}},
			Key:         &Identifier{Name: "key"},
		}

		var names []string
		WalkNodeExpressions(node, func(expression Expression) {
			if identifier, ok := expression.(*Identifier); ok {
				names = append(names, identifier.Name)
			}
		})

		expectedNames := []string{
			"dirIf", "dirElseIf", "dirFor", "dirShow", "dirModel",
			"dirClass", "dirStyle", "dirText", "dirHTML", "dirKey",
			"dirRef", "dirContext", "dirScaffold", "key",
		}
		assert.Equal(t, expectedNames, names)
	})

	t.Run("visits dynamic attribute expressions", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
			DynamicAttributes: []DynamicAttribute{
				{Name: "class", Expression: &Identifier{Name: "className"}},
				{Name: "style", Expression: &Identifier{Name: "styleObj"}},
			},
		}

		var names []string
		WalkNodeExpressions(node, func(expression Expression) {
			if identifier, ok := expression.(*Identifier); ok {
				names = append(names, identifier.Name)
			}
		})

		assert.Equal(t, []string{"className", "styleObj"}, names)
	})

	t.Run("visits RichText expressions", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeText,
			RichText: []TextPart{
				{IsLiteral: true, Literal: "Hello, "},
				{IsLiteral: false, Expression: &Identifier{Name: "userName"}},
				{IsLiteral: true, Literal: "!"},
			},
		}

		var names []string
		WalkNodeExpressions(node, func(expression Expression) {
			if identifier, ok := expression.(*Identifier); ok {
				names = append(names, identifier.Name)
			}
		})

		assert.Equal(t, []string{"userName"}, names)
	})

	t.Run("visits Binds expressions", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "input",
			Binds: map[string]*Directive{
				"value":    {Expression: &Identifier{Name: "inputValue"}},
				"disabled": {Expression: &Identifier{Name: "isDisabled"}},
			},
		}

		var names []string
		WalkNodeExpressions(node, func(expression Expression) {
			if identifier, ok := expression.(*Identifier); ok {
				names = append(names, identifier.Name)
			}
		})

		assert.Len(t, names, 2)
		assert.Contains(t, names, "inputValue")
		assert.Contains(t, names, "isDisabled")
	})

	t.Run("visits OnEvents expressions", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "button",
			OnEvents: map[string][]Directive{
				"click": {
					{Expression: &CallExpression{Callee: &Identifier{Name: "handleClick"}}},
					{Expression: &CallExpression{Callee: &Identifier{Name: "logEvent"}}},
				},
				"mouseover": {
					{Expression: &Identifier{Name: "onHover"}},
				},
			},
		}

		var visited []Expression
		WalkNodeExpressions(node, func(expression Expression) {
			visited = append(visited, expression)
		})

		assert.Len(t, visited, 3)
	})

	t.Run("visits CustomEvents expressions", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "my-component",
			CustomEvents: map[string][]Directive{
				"submit": {
					{Expression: &CallExpression{Callee: &Identifier{Name: "handleSubmit"}}},
				},
				"cancel": {
					{Expression: &Identifier{Name: "onCancel"}},
				},
			},
		}

		var visited []Expression
		WalkNodeExpressions(node, func(expression Expression) {
			visited = append(visited, expression)
		})

		assert.Len(t, visited, 2)
	})

	t.Run("visits nil directive expression without panic", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
			DirIf:    &Directive{Expression: nil},
		}

		var visited []Expression
		WalkNodeExpressions(node, func(expression Expression) {
			visited = append(visited, expression)
		})

		assert.Len(t, visited, 1)
		assert.Nil(t, visited[0])
	})
}

func TestVisitExpression(t *testing.T) {
	t.Run("nil expression is handled gracefully", func(t *testing.T) {
		var visited []string
		VisitExpression(nil, func(expression Expression) bool {
			visited = append(visited, "visited")
			return true
		})
		assert.Empty(t, visited)
	})

	t.Run("visits simple identifier", func(t *testing.T) {
		expression := &Identifier{Name: "foo"}
		var visited []string
		VisitExpression(expression, func(e Expression) bool {
			if identifier, ok := e.(*Identifier); ok {
				visited = append(visited, identifier.Name)
			}
			return true
		})
		assert.Equal(t, []string{"foo"}, visited)
	})

	t.Run("visits MemberExpr and children", func(t *testing.T) {
		expression := &MemberExpression{
			Base:     &Identifier{Name: "obj"},
			Property: &Identifier{Name: "prop"},
		}

		var visited []string
		VisitExpression(expression, func(e Expression) bool {
			switch n := e.(type) {
			case *MemberExpression:
				visited = append(visited, "MemberExpr")
			case *Identifier:
				visited = append(visited, "Identifier:"+n.Name)
			}
			return true
		})

		assert.Equal(t, []string{"MemberExpr", "Identifier:obj", "Identifier:prop"}, visited)
	})

	t.Run("visits IndexExpr and children", func(t *testing.T) {
		expression := &IndexExpression{
			Base:  &Identifier{Name: "arr"},
			Index: &IntegerLiteral{Value: 0},
		}

		var visited []string
		VisitExpression(expression, func(e Expression) bool {
			switch n := e.(type) {
			case *IndexExpression:
				visited = append(visited, "IndexExpr")
			case *Identifier:
				visited = append(visited, "Identifier:"+n.Name)
			case *IntegerLiteral:
				visited = append(visited, "IntegerLiteral")
			}
			return true
		})

		assert.Equal(t, []string{"IndexExpr", "Identifier:arr", "IntegerLiteral"}, visited)
	})

	t.Run("visits UnaryExpr and children", func(t *testing.T) {
		expression := &UnaryExpression{
			Operator: OpNot,
			Right:    &BooleanLiteral{Value: true},
		}

		var visited []string
		VisitExpression(expression, func(e Expression) bool {
			switch e.(type) {
			case *UnaryExpression:
				visited = append(visited, "UnaryExpr")
			case *BooleanLiteral:
				visited = append(visited, "BooleanLiteral")
			}
			return true
		})

		assert.Equal(t, []string{"UnaryExpr", "BooleanLiteral"}, visited)
	})

	t.Run("visits BinaryExpr and children", func(t *testing.T) {
		expression := &BinaryExpression{
			Left:     &IntegerLiteral{Value: 1},
			Operator: OpPlus,
			Right:    &IntegerLiteral{Value: 2},
		}

		var visited []string
		VisitExpression(expression, func(e Expression) bool {
			switch n := e.(type) {
			case *BinaryExpression:
				visited = append(visited, "BinaryExpr")
			case *IntegerLiteral:
				visited = append(visited, "IntegerLiteral:"+strconv.FormatInt(n.Value, 10))
			}
			return true
		})

		assert.Equal(t, []string{"BinaryExpr", "IntegerLiteral:1", "IntegerLiteral:2"}, visited)
	})

	t.Run("visits CallExpr with arguments", func(t *testing.T) {
		expression := &CallExpression{
			Callee: &Identifier{Name: "myFunc"},
			Args: []Expression{
				&StringLiteral{Value: "arg1"},
				&IntegerLiteral{Value: 42},
			},
		}

		var visited []string
		VisitExpression(expression, func(e Expression) bool {
			switch n := e.(type) {
			case *CallExpression:
				visited = append(visited, "CallExpr")
			case *Identifier:
				visited = append(visited, "Identifier:"+n.Name)
			case *StringLiteral:
				visited = append(visited, "StringLiteral:"+n.Value)
			case *IntegerLiteral:
				visited = append(visited, "IntegerLiteral")
			}
			return true
		})

		assert.Equal(t, []string{"CallExpr", "Identifier:myFunc", "StringLiteral:arg1", "IntegerLiteral"}, visited)
	})

	t.Run("visits TernaryExpr and all branches", func(t *testing.T) {
		expression := &TernaryExpression{
			Condition:  &Identifier{Name: "cond"},
			Consequent: &StringLiteral{Value: "yes"},
			Alternate:  &StringLiteral{Value: "no"},
		}

		var visited []string
		VisitExpression(expression, func(e Expression) bool {
			switch n := e.(type) {
			case *TernaryExpression:
				visited = append(visited, "TernaryExpr")
			case *Identifier:
				visited = append(visited, "Identifier:"+n.Name)
			case *StringLiteral:
				visited = append(visited, "StringLiteral:"+n.Value)
			}
			return true
		})

		assert.Equal(t, []string{"TernaryExpr", "Identifier:cond", "StringLiteral:yes", "StringLiteral:no"}, visited)
	})

	t.Run("visits ForInExpr collection", func(t *testing.T) {
		expression := &ForInExpression{
			ItemVariable: &Identifier{Name: "item"},
			Collection:   &Identifier{Name: "items"},
		}

		var visited []string
		VisitExpression(expression, func(e Expression) bool {
			switch n := e.(type) {
			case *ForInExpression:
				visited = append(visited, "ForInExpr")
			case *Identifier:
				visited = append(visited, "Identifier:"+n.Name)
			}
			return true
		})

		assert.Equal(t, []string{"ForInExpr", "Identifier:items"}, visited)
	})

	t.Run("visits TemplateLiteral with interpolations", func(t *testing.T) {
		expression := &TemplateLiteral{
			Parts: []TemplateLiteralPart{
				{IsLiteral: true, Literal: "Hello, "},
				{IsLiteral: false, Expression: &Identifier{Name: "name"}},
				{IsLiteral: true, Literal: "!"},
			},
		}

		var visited []string
		VisitExpression(expression, func(e Expression) bool {
			switch n := e.(type) {
			case *TemplateLiteral:
				visited = append(visited, "TemplateLiteral")
			case *Identifier:
				visited = append(visited, "Identifier:"+n.Name)
			}
			return true
		})

		assert.Equal(t, []string{"TemplateLiteral", "Identifier:name"}, visited)
	})

	t.Run("visits ObjectLiteral values", func(t *testing.T) {
		expression := &ObjectLiteral{
			Pairs: map[string]Expression{
				"a": &IntegerLiteral{Value: 1},
				"b": &IntegerLiteral{Value: 2},
			},
		}

		var visited []string
		VisitExpression(expression, func(e Expression) bool {
			switch e.(type) {
			case *ObjectLiteral:
				visited = append(visited, "ObjectLiteral")
			case *IntegerLiteral:
				visited = append(visited, "IntegerLiteral")
			}
			return true
		})

		assert.Contains(t, visited, "ObjectLiteral")
		assert.Equal(t, 3, len(visited))
	})

	t.Run("visits ArrayLiteral elements", func(t *testing.T) {
		expression := &ArrayLiteral{
			Elements: []Expression{
				&IntegerLiteral{Value: 1},
				&IntegerLiteral{Value: 2},
				&IntegerLiteral{Value: 3},
			},
		}

		var visited []string
		VisitExpression(expression, func(e Expression) bool {
			switch e.(type) {
			case *ArrayLiteral:
				visited = append(visited, "ArrayLiteral")
			case *IntegerLiteral:
				visited = append(visited, "IntegerLiteral")
			}
			return true
		})

		assert.Equal(t, []string{"ArrayLiteral", "IntegerLiteral", "IntegerLiteral", "IntegerLiteral"}, visited)
	})

	t.Run("stops traversal when visitor returns false", func(t *testing.T) {
		expression := &BinaryExpression{
			Left:     &IntegerLiteral{Value: 1},
			Operator: OpPlus,
			Right: &BinaryExpression{
				Left:     &IntegerLiteral{Value: 2},
				Operator: OpMul,
				Right:    &IntegerLiteral{Value: 3},
			},
		}

		var visited []string
		VisitExpression(expression, func(e Expression) bool {
			switch e.(type) {
			case *BinaryExpression:
				visited = append(visited, "BinaryExpr")
				return false
			case *IntegerLiteral:
				visited = append(visited, "IntegerLiteral")
			}
			return true
		})

		assert.Equal(t, []string{"BinaryExpr"}, visited)
	})

	t.Run("visits deeply nested expression tree", func(t *testing.T) {

		expression := &CallExpression{
			Callee: &MemberExpression{
				Base: &IndexExpression{
					Base: &MemberExpression{
						Base:     &Identifier{Name: "obj"},
						Property: &Identifier{Name: "prop"},
					},
					Index: &IntegerLiteral{Value: 0},
				},
				Property: &Identifier{Name: "method"},
			},
			Args: []Expression{
				&Identifier{Name: "argument"},
			},
		}

		var visited []string
		VisitExpression(expression, func(e Expression) bool {
			switch n := e.(type) {
			case *CallExpression:
				visited = append(visited, "CallExpr")
			case *MemberExpression:
				visited = append(visited, "MemberExpr")
			case *IndexExpression:
				visited = append(visited, "IndexExpr")
			case *Identifier:
				visited = append(visited, "Identifier:"+n.Name)
			case *IntegerLiteral:
				visited = append(visited, "IntegerLiteral")
			}
			return true
		})

		expected := []string{
			"CallExpr",
			"MemberExpr",
			"IndexExpr",
			"MemberExpr",
			"Identifier:obj",
			"Identifier:prop",
			"IntegerLiteral",
			"Identifier:method",
			"Identifier:argument",
		}
		assert.Equal(t, expected, visited)
	})

	t.Run("handles unknown literal types gracefully", func(t *testing.T) {

		literals := []Expression{
			&StringLiteral{Value: "test"},
			&IntegerLiteral{Value: 42},
			&FloatLiteral{Value: 3.14},
			&BooleanLiteral{Value: true},
			&NilLiteral{},
			&RuneLiteral{Value: 'a'},
		}

		for _, lit := range literals {
			var visited int
			VisitExpression(lit, func(e Expression) bool {
				visited++
				return true
			})
			assert.Equal(t, 1, visited)
		}
	})
}

func TestRemoveNodes(t *testing.T) {
	t.Run("nil AST is handled gracefully", func(t *testing.T) {
		var nilAST *TemplateAST
		nodesToRemove := []*TemplateNode{{TagName: "div"}}
		assert.NotPanics(t, func() {
			nilAST.RemoveNodes(nodesToRemove)
		})
	})

	t.Run("empty nodesToRemove is handled gracefully", func(t *testing.T) {
		ast := &TemplateAST{
			RootNodes: []*TemplateNode{{TagName: "div"}},
		}
		originalLen := len(ast.RootNodes)
		ast.RemoveNodes(nil)
		assert.Len(t, ast.RootNodes, originalLen)

		ast.RemoveNodes([]*TemplateNode{})
		assert.Len(t, ast.RootNodes, originalLen)
	})

	t.Run("removes a root node", func(t *testing.T) {
		node1 := &TemplateNode{TagName: "div"}
		node2 := &TemplateNode{TagName: "span"}
		ast := &TemplateAST{
			RootNodes: []*TemplateNode{node1, node2},
		}

		ast.RemoveNodes([]*TemplateNode{node1})

		assert.Len(t, ast.RootNodes, 1)
		assert.Equal(t, "span", ast.RootNodes[0].TagName)
	})

	t.Run("removes multiple root nodes", func(t *testing.T) {
		node1 := &TemplateNode{TagName: "div"}
		node2 := &TemplateNode{TagName: "span"}
		node3 := &TemplateNode{TagName: "p"}
		ast := &TemplateAST{
			RootNodes: []*TemplateNode{node1, node2, node3},
		}

		ast.RemoveNodes([]*TemplateNode{node1, node3})

		assert.Len(t, ast.RootNodes, 1)
		assert.Equal(t, "span", ast.RootNodes[0].TagName)
	})

	t.Run("removes a child node", func(t *testing.T) {
		child1 := &TemplateNode{TagName: "span"}
		child2 := &TemplateNode{TagName: "p"}
		parent := &TemplateNode{
			TagName:  "div",
			Children: []*TemplateNode{child1, child2},
		}
		ast := &TemplateAST{
			RootNodes: []*TemplateNode{parent},
		}

		ast.RemoveNodes([]*TemplateNode{child1})

		assert.Len(t, parent.Children, 1)
		assert.Equal(t, "p", parent.Children[0].TagName)
	})

	t.Run("removes deeply nested nodes", func(t *testing.T) {
		grandchild := &TemplateNode{TagName: "strong"}
		child := &TemplateNode{
			TagName:  "span",
			Children: []*TemplateNode{grandchild},
		}
		parent := &TemplateNode{
			TagName:  "div",
			Children: []*TemplateNode{child},
		}
		ast := &TemplateAST{
			RootNodes: []*TemplateNode{parent},
		}

		ast.RemoveNodes([]*TemplateNode{grandchild})

		assert.Len(t, parent.Children, 1)
		assert.Empty(t, child.Children)
	})

	t.Run("removes nodes at multiple levels", func(t *testing.T) {
		grandchild := &TemplateNode{TagName: "strong"}
		child := &TemplateNode{
			TagName:  "span",
			Children: []*TemplateNode{grandchild},
		}
		sibling := &TemplateNode{TagName: "p"}
		parent := &TemplateNode{
			TagName:  "div",
			Children: []*TemplateNode{child, sibling},
		}
		root2 := &TemplateNode{TagName: "section"}
		ast := &TemplateAST{
			RootNodes: []*TemplateNode{parent, root2},
		}

		ast.RemoveNodes([]*TemplateNode{grandchild, sibling, root2})

		assert.Len(t, ast.RootNodes, 1)
		assert.Len(t, parent.Children, 1)
		assert.Equal(t, "span", parent.Children[0].TagName)
		assert.Empty(t, child.Children)
	})

	t.Run("handles removing node not in tree", func(t *testing.T) {
		node := &TemplateNode{TagName: "div"}
		notInTree := &TemplateNode{TagName: "span"}
		ast := &TemplateAST{
			RootNodes: []*TemplateNode{node},
		}

		ast.RemoveNodes([]*TemplateNode{notInTree})

		assert.Len(t, ast.RootNodes, 1)
		assert.Equal(t, "div", ast.RootNodes[0].TagName)
	})

	t.Run("handles parent with no children needing removal", func(t *testing.T) {
		child1 := &TemplateNode{TagName: "span"}
		child2 := &TemplateNode{TagName: "p"}
		parent := &TemplateNode{
			TagName:  "div",
			Children: []*TemplateNode{child1, child2},
		}
		unrelatedNode := &TemplateNode{TagName: "section"}
		ast := &TemplateAST{
			RootNodes: []*TemplateNode{parent, unrelatedNode},
		}

		ast.RemoveNodes([]*TemplateNode{unrelatedNode})

		assert.Len(t, ast.RootNodes, 1)
		assert.Len(t, parent.Children, 2)
	})
}

func TestStreamNodes_CountMatchesWalk(t *testing.T) {
	t.Parallel()

	tree := createTestWalkAST(t)

	walkCount := 0
	tree.Walk(func(_ *TemplateNode) bool {
		walkCount++
		return true
	})

	ctx := context.Background()
	streamCount := 0
	for range tree.StreamNodes(ctx) {
		streamCount++
	}

	assert.Equal(t, walkCount, streamCount,
		"StreamNodes should yield the same number of nodes as Walk")
	assert.Greater(t, streamCount, 0,
		"Both Walk and StreamNodes should find at least one node")
}

func TestStreamNodes_ContextCancellationClosesChannel(t *testing.T) {
	t.Parallel()

	tree := createTestWalkAST(t)

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(fmt.Errorf("test: cleanup"))
	nodeChannel := tree.StreamNodes(ctx)

	receivedCount := 0
	for node := range nodeChannel {
		receivedCount++
		_ = node
		if receivedCount >= 2 {
			cancel(fmt.Errorf("test: simulating cancelled context"))
		}
	}

	assert.GreaterOrEqual(t, receivedCount, 2,
		"Should have received at least 2 nodes before cancellation")

	walkCount := 0
	tree.Walk(func(_ *TemplateNode) bool {
		walkCount++
		return true
	})
	assert.Less(t, receivedCount, walkCount,
		"Cancelled stream should yield fewer nodes than a full Walk")
}

func TestAcceptWithError_AllNodesVisitedAndErrorPropagation(t *testing.T) {
	t.Parallel()

	t.Run("all nodes visited when no error", func(t *testing.T) {
		t.Parallel()

		tree := createTestWalkAST(t)

		visitedCount := 0
		visitor := &countingVisitorWithError{
			visitFunc: func(_ *TemplateNode) (VisitorWithError, error) {
				visitedCount++
				return &countingVisitorWithError{
					visitFunc: func(_ *TemplateNode) (VisitorWithError, error) {
						visitedCount++
						return nil, nil
					},
				}, nil
			},
		}

		err := tree.AcceptWithError(visitor)
		assert.NoError(t, err)
		assert.Greater(t, visitedCount, 0, "Should visit at least one node")
	})

	t.Run("error stops traversal immediately", func(t *testing.T) {
		t.Parallel()

		tree := createTestWalkAST(t)
		sentinel := errors.New("stop walking")

		visitedCount := 0
		visitor := &countingVisitorWithError{
			visitFunc: func(_ *TemplateNode) (VisitorWithError, error) {
				visitedCount++
				if visitedCount >= 2 {
					return nil, sentinel
				}
				return &countingVisitorWithError{
					visitFunc: func(_ *TemplateNode) (VisitorWithError, error) {
						visitedCount++
						return nil, nil
					},
				}, nil
			},
		}

		err := tree.AcceptWithError(visitor)
		assert.ErrorIs(t, err, sentinel, "AcceptWithError should propagate the error")
		assert.GreaterOrEqual(t, visitedCount, 2,
			"Should have visited at least 2 nodes before stopping")
	})
}

type countingVisitorWithError struct {
	visitFunc func(node *TemplateNode) (VisitorWithError, error)
}

func (v *countingVisitorWithError) Visit(node *TemplateNode) (VisitorWithError, error) {
	return v.visitFunc(node)
}
