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

// Provides tree traversal utilities for walking AST structures with visitor
// patterns and iterators. Implements depth-first traversal, parallel walking,
// and Go 1.23+ range-over-func iterators for flexible node processing.

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"sync"
)

// maxWalkDepth is the largest depth allowed when walking through a tree.
// This limit stops stack overflow errors in deeply nested trees.
const maxWalkDepth = 10000

// Nodes returns an iterator over all nodes in the AST using Go 1.23+
// range-over-func.
//
// Returns iter.Seq[*TemplateNode] which yields each node in
// depth-first order.
func (ast *TemplateAST) Nodes() iter.Seq[*TemplateNode] {
	return func(yield func(*TemplateNode) bool) {
		if ast == nil {
			return
		}
		for _, root := range ast.RootNodes {
			if !root.yieldRecursive(yield) {
				return
			}
		}
	}
}

// NodesWithParent returns an iterator over all nodes with their parent using Go
// 1.23+ range-over-func.
//
// Returns iter.Seq2[*TemplateNode, *TemplateNode] which yields each
// node paired with its parent in depth-first order.
func (ast *TemplateAST) NodesWithParent() iter.Seq2[*TemplateNode, *TemplateNode] {
	return func(yield func(node, parent *TemplateNode) bool) {
		if ast == nil {
			return
		}
		for _, root := range ast.RootNodes {
			if !root.yieldWithParentRecursive(nil, yield) {
				return
			}
		}
	}
}

// ParallelWalkFunc processes a single node during parallel traversal.
type ParallelWalkFunc func(ctx context.Context, node *TemplateNode) error

// ParallelWalk traverses the AST in parallel using the specified number of
// workers.
//
// Takes numWorkers (int) which specifies the number of concurrent workers.
// Takes f (ParallelWalkFunc) which is called for each node in the AST.
//
// Returns error when a worker fails or the context is cancelled.
func (ast *TemplateAST) ParallelWalk(ctx context.Context, numWorkers int, f ParallelWalkFunc) error {
	if ast == nil {
		return nil
	}
	return parallelWalkImpl(ctx, numWorkers, f, func(yield func(*TemplateNode) bool) {
		ast.Walk(yield)
	})
}

// StreamNodes returns a channel that streams all nodes from the AST.
//
// Returns <-chan *TemplateNode which yields each node in the tree.
//
// Spawns a goroutine that walks the AST and sends nodes until
// the context is cancelled or all nodes have been sent. The channel is closed
// when complete.
func (ast *TemplateAST) StreamNodes(ctx context.Context) <-chan *TemplateNode {
	nodeChannel := make(chan *TemplateNode)
	go func() {
		defer close(nodeChannel)
		ast.Walk(func(node *TemplateNode) bool {
			select {
			case <-ctx.Done():
				return false
			case nodeChannel <- node:
				return true
			}
		})
	}()
	return nodeChannel
}

// NewIterator creates a new pre-order iterator for traversing the AST.
//
// Returns *Iterator which provides pre-order traversal of all AST nodes.
func (ast *TemplateAST) NewIterator() *Iterator {
	if ast == nil || len(ast.RootNodes) == 0 {
		return &Iterator{}
	}
	stackSize := len(ast.RootNodes)
	const initialChildCapacity = 16
	stack := make([]*iteratorFrame, stackSize, stackSize+initialChildCapacity)
	for i := range stackSize {
		frame, ok := iteratorFramePool.Get().(*iteratorFrame)
		if !ok {
			frame = &iteratorFrame{}
		}
		frame.node = ast.RootNodes[stackSize-1-i]
		frame.parent = nil
		stack[i] = frame
	}
	return &Iterator{Node: nil, Parent: nil, stack: stack}
}

// NewPostOrderIterator creates a post-order iterator for traversing the AST.
//
// Returns *PostOrderIterator which visits child nodes before their parents.
func (ast *TemplateAST) NewPostOrderIterator() *PostOrderIterator {
	if ast == nil || len(ast.RootNodes) == 0 {
		return &PostOrderIterator{}
	}
	stackSize := len(ast.RootNodes)
	const initialChildCapacity = 16
	stack := make([]*TemplateNode, stackSize, stackSize+initialChildCapacity)
	for i := range stackSize {
		stack[i] = ast.RootNodes[stackSize-1-i]
	}
	return &PostOrderIterator{Node: nil, lastVisitedNode: nil, stack: stack}
}

// Accept walks the AST using the Visitor pattern.
//
// Takes ctx (context.Context) which carries cancellation and trace data.
// Takes v (Visitor) which receives calls for each node during the walk.
//
// Returns error when the visitor returns an error.
func (ast *TemplateAST) Accept(ctx context.Context, v Visitor) error {
	if ast == nil || v == nil {
		return nil
	}
	for _, root := range ast.RootNodes {
		if err := WalkWithVisitor(ctx, v, root); err != nil {
			return err
		}
	}
	return nil
}

// AcceptWithError traverses the AST using the VisitorWithError pattern.
//
// Takes v (VisitorWithError) which processes each node in the tree.
//
// Returns error when the visitor returns an error during traversal.
func (ast *TemplateAST) AcceptWithError(v VisitorWithError) error {
	if ast == nil || v == nil {
		return nil
	}
	for _, root := range ast.RootNodes {
		if err := WalkWithError(v, root); err != nil {
			return err
		}
	}
	return nil
}

// WalkFunc is a function that processes a node and returns whether to continue
// traversal.
type WalkFunc func(node *TemplateNode) (continueTraversal bool)

// Walk visits all nodes in the AST using depth-first order.
//
// Takes f (WalkFunc) which is called for each node visited.
func (ast *TemplateAST) Walk(f WalkFunc) {
	if ast == nil {
		return
	}
	for _, root := range ast.RootNodes {
		if !root.walkRecursive(f) {
			break
		}
	}
}

// Find returns the first node that matches the predicate.
//
// Takes predicate (func(node *TemplateNode) bool) which tests each node.
//
// Returns *TemplateNode which is the first matching node, or nil if none found.
func (ast *TemplateAST) Find(predicate func(node *TemplateNode) bool) *TemplateNode {
	var found *TemplateNode
	ast.Walk(func(node *TemplateNode) bool {
		if predicate(node) {
			found = node
			return false
		}
		return true
	})
	return found
}

// FindAll returns all nodes that match the given predicate function.
//
// Takes predicate (func(node *TemplateNode) bool) which tests each node and
// returns true if the node should be included.
//
// Returns []*TemplateNode which contains all matching nodes, or an empty slice
// if none match.
func (ast *TemplateAST) FindAll(predicate func(node *TemplateNode) bool) []*TemplateNode {
	found := make([]*TemplateNode, 0)
	for node := range ast.Nodes() {
		if predicate(node) {
			found = append(found, node)
		}
	}
	return found
}

// Nodes returns an iterator over this node and all descendants using Go 1.23+
// range-over-func.
//
// Returns iter.Seq[*TemplateNode] which yields each node in
// depth-first order.
func (n *TemplateNode) Nodes() iter.Seq[*TemplateNode] {
	return func(yield func(*TemplateNode) bool) {
		if n == nil {
			return
		}
		n.yieldRecursive(yield)
	}
}

// NodesWithParent returns an iterator over this node and descendants with their
// parent using Go 1.23+ range-over-func.
//
// Returns iter.Seq2[*TemplateNode, *TemplateNode] which yields each
// node paired with its parent in depth-first order.
func (n *TemplateNode) NodesWithParent() iter.Seq2[*TemplateNode, *TemplateNode] {
	return func(yield func(node, parent *TemplateNode) bool) {
		if n == nil {
			return
		}
		n.yieldWithParentRecursive(nil, yield)
	}
}

// ParallelWalk traverses this node and descendants in parallel.
//
// Takes numWorkers (int) which specifies the number of concurrent workers.
// Takes f (ParallelWalkFunc) which is called for each node visited.
//
// Returns error when walking fails or f returns an error.
func (n *TemplateNode) ParallelWalk(ctx context.Context, numWorkers int, f ParallelWalkFunc) error {
	if n == nil {
		return nil
	}
	return parallelWalkImpl(ctx, numWorkers, f, func(yield func(*TemplateNode) bool) {
		n.Walk(yield)
	})
}

// StreamNodes returns a channel that streams this node and all its children.
//
// Returns <-chan *TemplateNode which yields each node in the tree.
//
// Spawns a goroutine that walks the tree and sends nodes until
// the context is cancelled or all nodes have been sent. The channel is closed
// when done.
func (n *TemplateNode) StreamNodes(ctx context.Context) <-chan *TemplateNode {
	nodeChannel := make(chan *TemplateNode)
	go func() {
		defer close(nodeChannel)
		n.Walk(func(node *TemplateNode) bool {
			select {
			case <-ctx.Done():
				return false
			case nodeChannel <- node:
				return true
			}
		})
	}()
	return nodeChannel
}

// NewIterator creates a pre-order iterator for traversing this node and its
// descendants.
//
// Returns *Iterator which provides access to nodes one at a time in pre-order.
func (n *TemplateNode) NewIterator() *Iterator {
	if n == nil {
		return &Iterator{}
	}
	const initialChildCapacity = 16
	stack := make([]*iteratorFrame, 1, 1+initialChildCapacity)
	frame, ok := iteratorFramePool.Get().(*iteratorFrame)
	if !ok {
		frame = &iteratorFrame{}
	}
	frame.node = n
	frame.parent = nil
	stack[0] = frame
	return &Iterator{Node: nil, Parent: nil, stack: stack}
}

// NewPostOrderIterator creates a post-order iterator for traversing this node
// and its children.
//
// Returns *PostOrderIterator which visits children before their parents.
func (n *TemplateNode) NewPostOrderIterator() *PostOrderIterator {
	if n == nil {
		return &PostOrderIterator{}
	}
	const initialChildCapacity = 16
	stack := make([]*TemplateNode, 1, 1+initialChildCapacity)
	stack[0] = n
	return &PostOrderIterator{Node: nil, lastVisitedNode: nil, stack: stack}
}

// Accept walks this node and its children using the Visitor pattern.
//
// Takes ctx (context.Context) which carries cancellation and trace data.
// Takes v (Visitor) which provides callbacks for each node type.
//
// Returns error when the visitor returns an error during the walk.
func (n *TemplateNode) Accept(ctx context.Context, v Visitor) error {
	return WalkWithVisitor(ctx, v, n)
}

// AcceptWithError traverses this node and descendants using the
// VisitorWithError pattern.
//
// Takes v (VisitorWithError) which processes the node and its children.
//
// Returns error when the visitor returns an error during traversal.
func (n *TemplateNode) AcceptWithError(v VisitorWithError) error {
	return WalkWithError(v, n)
}

// Walk traverses this node and all its children in depth-first order.
//
// Takes f (WalkFunc) which is called for each node visited.
func (n *TemplateNode) Walk(f WalkFunc) {
	if n == nil {
		return
	}
	n.walkRecursive(f)
}

// Find returns the first node in this subtree that matches the predicate.
//
// Takes predicate (func(node *TemplateNode) bool) which tests each node.
//
// Returns *TemplateNode which is the first matching node, or nil if none match.
func (n *TemplateNode) Find(predicate func(node *TemplateNode) bool) *TemplateNode {
	var found *TemplateNode
	n.Walk(func(node *TemplateNode) bool {
		if predicate(node) {
			found = node
			return false
		}
		return true
	})
	return found
}

// FindAll returns all nodes in this subtree that match the predicate.
//
// Takes predicate (func(node *TemplateNode) bool) which tests each node.
//
// Returns []*TemplateNode which contains all matching nodes, or nil if the
// receiver is nil.
func (n *TemplateNode) FindAll(predicate func(node *TemplateNode) bool) []*TemplateNode {
	if n == nil {
		return nil
	}
	found := make([]*TemplateNode, 0)
	for node := range n.Nodes() {
		if predicate(node) {
			found = append(found, node)
		}
	}
	return found
}

// iteratorFrame holds a node and its parent during tree traversal.
type iteratorFrame struct {
	// node is the template node at this position in the tree traversal.
	node *TemplateNode

	// parent is the parent node in the tree; nil for root frames.
	parent *TemplateNode
}

// iteratorFramePool reuses iteratorFrame instances to reduce allocation pressure
// during AST traversal.
var iteratorFramePool = sync.Pool{
	New: func() any {
		return new(iteratorFrame)
	},
}

// Iterator provides depth-first, pre-order traversal of the template tree with
// parent tracking.
type Iterator struct {
	// Node is the current template node in the iteration.
	Node *TemplateNode

	// Parent is the parent node in the template hierarchy; nil for root nodes.
	Parent *TemplateNode

	// stack holds nodes to visit during depth-first traversal.
	stack []*iteratorFrame
}

// Next moves the iterator to the next node in the tree.
//
// Returns bool which is true if a node is available, or false when there
// are no more nodes to visit.
func (it *Iterator) Next() bool {
	if len(it.stack) == 0 {
		it.Node, it.Parent = nil, nil
		return false
	}

	frame := it.stack[len(it.stack)-1]
	it.stack = it.stack[:len(it.stack)-1]
	it.Node, it.Parent = frame.node, frame.parent

	*frame = iteratorFrame{}
	iteratorFramePool.Put(frame)

	children := it.Node.Children
	for i := len(children) - 1; i >= 0; i-- {
		if child := children[i]; child != nil {
			childFrame, ok := iteratorFramePool.Get().(*iteratorFrame)
			if !ok {
				childFrame = &iteratorFrame{}
			}
			childFrame.node = child
			childFrame.parent = it.Node
			it.stack = append(it.stack, childFrame)
		}
	}
	return true
}

// SkipChildren prevents the iterator from descending into the current node's
// children.
func (it *Iterator) SkipChildren() {
	if it.Node == nil || len(it.Node.Children) == 0 {
		return
	}
	childCount := 0
	for _, child := range it.Node.Children {
		if child != nil {
			childCount++
		}
	}
	if childCount > 0 && len(it.stack) >= childCount {
		start := len(it.stack) - childCount
		end := len(it.stack)
		for i := start; i < end; i++ {
			frame := it.stack[i]
			*frame = iteratorFrame{}
			iteratorFramePool.Put(frame)
		}
		it.stack = it.stack[:start]
	}
}

// PostOrderIterator provides depth-first, post-order traversal of the template
// tree.
type PostOrderIterator struct {
	// Node is the current node in the iteration.
	Node *TemplateNode

	// lastVisitedNode holds the most recently visited node during traversal.
	lastVisitedNode *TemplateNode

	// stack holds nodes to visit in LIFO order for depth-first traversal.
	stack []*TemplateNode
}

// Next moves the iterator to the next node in post-order sequence.
//
// Returns bool which is true if a node is available, or false when there are
// no more nodes to visit.
func (it *PostOrderIterator) Next() bool {
	for len(it.stack) > 0 {
		peekNode := it.stack[len(it.stack)-1]
		isLeaf := len(peekNode.Children) == 0

		if isLeaf || it.isLastChildVisited(peekNode) {
			it.stack = it.stack[:len(it.stack)-1]
			it.Node = peekNode
			it.lastVisitedNode = peekNode
			return true
		}

		for i := len(peekNode.Children) - 1; i >= 0; i-- {
			it.stack = append(it.stack, peekNode.Children[i])
		}
	}
	it.Node = nil
	return false
}

// isLastChildVisited checks whether the most recently visited node is the last
// child of the given node.
//
// Takes peekNode (*TemplateNode) which is the node to check against.
//
// Returns bool which is true if the last visited node matches the final child.
func (it *PostOrderIterator) isLastChildVisited(peekNode *TemplateNode) bool {
	if it.lastVisitedNode == nil || len(peekNode.Children) == 0 {
		return false
	}
	return it.lastVisitedNode == peekNode.Children[len(peekNode.Children)-1]
}

// Visitor provides a way to walk AST nodes using a two-phase pattern.
// It implements ast_domain.Visitor with Enter and Exit callbacks.
type Visitor interface {
	// Enter begins visiting a template node in the AST.
	//
	// Takes ctx (context.Context) which carries cancellation and trace data.
	// Takes node (*TemplateNode) which is the node to visit.
	//
	// Returns Visitor which continues the traversal, or nil to stop.
	// Returns error when the node cannot be processed.
	Enter(ctx context.Context, node *TemplateNode) (Visitor, error)

	// Exit is called when leaving a template node during traversal.
	//
	// Takes ctx (context.Context) which carries cancellation and trace data.
	// Takes node (*TemplateNode) which is the node being exited.
	//
	// Returns error when post-processing of the node fails.
	Exit(ctx context.Context, node *TemplateNode) error
}

// VisitorWithError provides a visitor pattern that can return errors.
// It implements ast_domain.VisitorWithError for traversing template nodes.
type VisitorWithError interface {
	// Visit processes a template node and returns a visitor for child nodes.
	//
	// Takes node (*TemplateNode) which is the template node to process.
	//
	// Returns w (VisitorWithError) which visits child nodes, or nil to skip them.
	// Returns err (error) when the visit fails.
	Visit(node *TemplateNode) (w VisitorWithError, err error)
}

// walkRecursive visits this node and all its children using the given function.
//
// Takes f (WalkFunc) which is called for each node visited.
//
// Returns bool which is false if the walk was stopped early.
func (n *TemplateNode) walkRecursive(f WalkFunc) bool {
	return n.walkRecursiveWithDepth(f, 0)
}

// walkRecursiveWithDepth visits each node in the tree using depth-first order.
//
// Takes f (WalkFunc) which is called for each node visited.
// Takes depth (int) which tracks the current level in the tree.
//
// Returns bool which is false if f returned false, stopping the walk early.
func (n *TemplateNode) walkRecursiveWithDepth(f WalkFunc, depth int) bool {
	if depth > maxWalkDepth {
		return true
	}
	if !f(n) {
		return false
	}
	for _, child := range n.Children {
		if !child.walkRecursiveWithDepth(f, depth+1) {
			return false
		}
	}
	return true
}

// yieldRecursive iterates over this node and all descendants.
//
// Takes yield (func(...)) which is called for each node in the tree.
//
// Returns bool which indicates whether iteration should continue.
func (n *TemplateNode) yieldRecursive(yield func(*TemplateNode) bool) bool {
	return n.yieldRecursiveWithDepth(yield, 0)
}

// yieldRecursiveWithDepth traverses the node and its children depth-first.
//
// Takes yield (func(*TemplateNode) bool) which processes each node and returns
// false to stop traversal.
// Takes depth (int) which tracks the current recursion level.
//
// Returns bool which is false if traversal was stopped early, true otherwise.
// Stops and returns true without crashing when depth exceeds maxWalkDepth.
func (n *TemplateNode) yieldRecursiveWithDepth(yield func(*TemplateNode) bool, depth int) bool {
	if depth > maxWalkDepth {
		return true
	}
	if !yield(n) {
		return false
	}
	for _, child := range n.Children {
		if !child.yieldRecursiveWithDepth(yield, depth+1) {
			return false
		}
	}
	return true
}

// yieldWithParentRecursive walks the tree and calls yield for each node with
// its parent.
//
// Takes parent (*TemplateNode) which is the parent of the current node.
// Takes yield (func(...)) which is called for each node with its parent.
//
// Returns bool which is false if yield stopped the walk early.
func (n *TemplateNode) yieldWithParentRecursive(parent *TemplateNode, yield func(node, parent *TemplateNode) bool) bool {
	return n.yieldWithParentRecursiveWithDepth(parent, yield, 0)
}

// yieldWithParentRecursiveWithDepth traverses the node tree depth-first,
// calling yield for each node with its parent.
//
// Takes parent (*TemplateNode) which is the parent of the current node.
// Takes yield (func(...)) which is called for each node and its parent.
// Takes depth (int) which tracks recursion depth to prevent stack overflow.
//
// Returns bool which is false if yield returned false, stopping traversal.
func (n *TemplateNode) yieldWithParentRecursiveWithDepth(parent *TemplateNode, yield func(node, parent *TemplateNode) bool, depth int) bool {
	if depth > maxWalkDepth {
		return true
	}
	if !yield(n, parent) {
		return false
	}
	for _, child := range n.Children {
		if !child.yieldWithParentRecursiveWithDepth(n, yield, depth+1) {
			return false
		}
	}
	return true
}

// RemoveNodes removes the given nodes from the AST.
// It changes the tree in place.
//
// Takes nodesToRemove ([]*TemplateNode) which specifies the nodes to remove.
func (ast *TemplateAST) RemoveNodes(nodesToRemove []*TemplateNode) {
	if ast == nil || len(nodesToRemove) == 0 {
		return
	}

	removalMap := buildRemovalMap(nodesToRemove)
	ast.RootNodes = filterNodes(ast.RootNodes, removalMap)
	ast.Walk(func(parent *TemplateNode) bool {
		filterChildrenInPlace(parent, removalMap)
		return true
	})
}

// WalkWithVisitor traverses the tree using the Visitor pattern.
//
// Takes ctx (context.Context) which carries cancellation and trace data.
// Takes v (Visitor) which receives callbacks for each node visited.
// Takes node (*TemplateNode) which is the root node to start traversal from.
//
// Returns error when the visitor returns an error during traversal.
func WalkWithVisitor(ctx context.Context, v Visitor, node *TemplateNode) error {
	return walkWithVisitorDepth(ctx, v, node, 0)
}

// WalkWithError walks the tree using the given visitor.
//
// Takes v (VisitorWithError) which provides callbacks for each node.
// Takes node (*TemplateNode) which is the root node to start walking from.
//
// Returns error when the visitor returns an error during the walk.
func WalkWithError(v VisitorWithError, node *TemplateNode) error {
	return walkWithErrorDepth(v, node, 0)
}

// WalkNodeExpressions visits all expressions within a single template node.
//
// It walks through directives, attributes, and event handlers, calling the
// visit function for each expression found.
//
// Takes node (*TemplateNode) which is the node to walk.
// Takes visit (func(Expression)) which is called for each expression found.
func WalkNodeExpressions(node *TemplateNode, visit func(Expression)) {
	if node == nil {
		return
	}
	walkDirectiveExpressions(node, visit)
	walkAttributeExpressions(node, visit)
	walkEventExpressions(node, visit)
}

// VisitExpression walks an expression tree and calls the visitor
// function for each node.
//
// If the visitor returns false, the walk stops for that branch.
//
// Takes expression (Expression) which is the root expression
// to walk.
// Takes visitor (func(...)) which is called for each expression
// in the tree.
func VisitExpression(expression Expression, visitor func(Expression) bool) {
	visitExpressionWithDepth(expression, visitor, 0)
}

// parallelWalkImpl is the shared implementation for parallel tree traversal.
//
// Takes numWorkers (int) which sets how many worker goroutines run at once.
// Takes f (ParallelWalkFunc) which handles each node in the tree.
// Takes walkFunc (func(...)) which produces nodes to process.
//
// Returns error when any worker fails during processing.
//
// Spawns numWorkers goroutines plus one producer goroutine. All
// goroutines stop when processing finishes or the context is cancelled.
func parallelWalkImpl(ctx context.Context, numWorkers int, f ParallelWalkFunc, walkFunc func(yield func(*TemplateNode) bool)) error {
	if numWorkers <= 0 {
		numWorkers = 1
	}

	workerCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(errors.New("parallel walk completed"))

	tasks := make(chan *TemplateNode)
	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error

	wg.Add(numWorkers)
	for range numWorkers {
		go func() {
			defer wg.Done()
			processWorkerTasks(workerCtx, tasks, f, &once, &firstErr, cancel)
		}()
	}

	go func() {
		defer close(tasks)
		produceWalkTasks(workerCtx, tasks, walkFunc)
	}()

	wg.Wait()
	return firstErr
}

// processWorkerTasks handles task processing for a single
// worker.
//
// Takes ctx (context.Context) which controls cancellation of
// the worker.
// Takes tasks (<-chan *TemplateNode) which provides nodes to
// process.
// Takes f (ParallelWalkFunc) which is called for each node.
// Takes once (*sync.Once) which ensures only the first error
// is recorded.
// Takes firstErr (*error) which stores the first error
// encountered.
// Takes cancel (context.CancelCauseFunc) which cancels the worker
// context when the first error occurs.
func processWorkerTasks(ctx context.Context, tasks <-chan *TemplateNode, f ParallelWalkFunc, once *sync.Once, firstErr *error, cancel context.CancelCauseFunc) {
	for {
		select {
		case node, ok := <-tasks:
			if !ok {
				return
			}
			if err := f(ctx, node); err != nil {
				once.Do(func() {
					*firstErr = err
					cancel(fmt.Errorf("parallel walk worker failed: %w", err))
				})
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// produceWalkTasks sends nodes from the walk function into the tasks channel.
//
// Takes tasks (chan<- *TemplateNode) which receives nodes to be processed.
// Takes walkFunc (func(...)) which iterates over nodes and yields them.
func produceWalkTasks(ctx context.Context, tasks chan<- *TemplateNode, walkFunc func(yield func(*TemplateNode) bool)) {
	walkFunc(func(node *TemplateNode) bool {
		select {
		case tasks <- node:
			return true
		case <-ctx.Done():
			return false
		}
	})
}

// resetIteratorFramePool clears the iterator frame pool to ensure test
// isolation.
//
// Call via t.Cleanup(resetIteratorFramePool) in tests.
func resetIteratorFramePool() {
	iteratorFramePool = sync.Pool{
		New: func() any {
			return new(iteratorFrame)
		},
	}
}

// walkWithVisitorDepth traverses a template tree using the visitor pattern.
//
// When v or node is nil, returns immediately without error. When depth exceeds
// maxWalkDepth, stops recursion to prevent stack overflow.
//
// Takes ctx (context.Context) which carries cancellation and trace data.
// Takes v (Visitor) which processes each node during traversal.
// Takes node (*TemplateNode) which is the root of the subtree to traverse.
// Takes depth (int) which tracks the current recursion depth.
//
// Returns error when the visitor's Enter or Exit method fails.
func walkWithVisitorDepth(ctx context.Context, v Visitor, node *TemplateNode, depth int) error {
	if v == nil || node == nil {
		return nil
	}
	if depth > maxWalkDepth {
		return nil
	}

	childVisitor, err := v.Enter(ctx, node)
	if err != nil {
		return err
	}

	if childVisitor != nil {
		for _, child := range node.Children {
			if err := walkWithVisitorDepth(ctx, childVisitor, child, depth+1); err != nil {
				return err
			}
		}
	}

	return v.Exit(ctx, node)
}

// walkWithErrorDepth walks a template tree while tracking how deep into the
// tree the walk has gone.
//
// Takes v (VisitorWithError) which processes each node.
// Takes node (*TemplateNode) which is the starting node.
// Takes depth (int) which tracks the current level of nesting.
//
// Returns error when the visitor returns an error.
func walkWithErrorDepth(v VisitorWithError, node *TemplateNode, depth int) error {
	if depth > maxWalkDepth {
		return nil
	}
	var err error
	if v, err = v.Visit(node); v == nil || err != nil {
		if err != nil {
			return err
		}
		return nil
	}
	for _, child := range node.Children {
		if err := walkWithErrorDepth(v, child, depth+1); err != nil {
			return err
		}
	}
	return nil
}

// walkDirectiveExpressions visits all directive expressions in a template node.
//
// Takes node (*TemplateNode) which contains the directives to walk.
// Takes visit (func(...)) which is called for each expression found.
func walkDirectiveExpressions(node *TemplateNode, visit func(Expression)) {
	visitDirectiveExpr(node.DirIf, visit)
	visitDirectiveExpr(node.DirElseIf, visit)
	visitDirectiveExpr(node.DirFor, visit)
	visitDirectiveExpr(node.DirShow, visit)
	visitDirectiveExpr(node.DirModel, visit)
	visitDirectiveExpr(node.DirClass, visit)
	visitDirectiveExpr(node.DirStyle, visit)
	visitDirectiveExpr(node.DirText, visit)
	visitDirectiveExpr(node.DirHTML, visit)
	visitDirectiveExpr(node.DirKey, visit)
	visitDirectiveExpr(node.DirRef, visit)
	visitDirectiveExpr(node.DirSlot, visit)
	visitDirectiveExpr(node.DirContext, visit)
	visitDirectiveExpr(node.DirScaffold, visit)
	if node.Key != nil {
		visit(node.Key)
	}
}

// visitDirectiveExpr calls the visit function on a directive's expression.
//
// Takes directive (*Directive) which contains the expression to visit.
// Takes visit (func(...)) which is called with the expression.
func visitDirectiveExpr(directive *Directive, visit func(Expression)) {
	if directive != nil {
		visit(directive.Expression)
	}
}

// walkAttributeExpressions visits all expressions within a template node.
//
// Takes node (*TemplateNode) which contains the expressions to walk.
// Takes visit (func(Expression)) which is called for each expression found.
func walkAttributeExpressions(node *TemplateNode, visit func(Expression)) {
	for i := range node.DynamicAttributes {
		attr := &node.DynamicAttributes[i]
		visit(attr.Expression)
	}
	for _, part := range node.RichText {
		if !part.IsLiteral {
			visit(part.Expression)
		}
	}
	for _, bind := range node.Binds {
		visit(bind.Expression)
	}
}

// walkEventExpressions visits all event handler expressions in a template
// node.
//
// Takes node (*TemplateNode) which contains the event handlers to walk.
// Takes visit (func(Expression)) which is called for each expression found.
func walkEventExpressions(node *TemplateNode, visit func(Expression)) {
	for _, handlers := range node.OnEvents {
		for i := range handlers {
			visit(handlers[i].Expression)
		}
	}
	for _, handlers := range node.CustomEvents {
		for i := range handlers {
			visit(handlers[i].Expression)
		}
	}
}

// visitExpressionWithDepth walks an expression tree in
// depth-first order.
//
// Takes expression (Expression) which is the root expression
// to visit.
// Takes visitor (func(...)) which returns true to continue
// into children.
// Takes depth (int) which tracks the current level in the tree.
func visitExpressionWithDepth(expression Expression, visitor func(Expression) bool, depth int) {
	if expression == nil || depth > maxWalkDepth {
		return
	}

	if !visitor(expression) {
		return
	}

	if visitCompoundExprChildrenWithDepth(expression, visitor, depth+1) {
		return
	}
	visitLiteralExprChildrenWithDepth(expression, visitor, depth+1)
}

// visitCompoundExprChildrenWithDepth visits the children of
// compound expression types while tracking the current depth.
//
// Takes expression (Expression) which is the compound
// expression to visit.
// Takes visitor (func(...)) which is called for each child
// expression.
// Takes depth (int) which tracks the current depth in the
// tree.
//
// Returns bool which is true if expression was a compound
// type, false otherwise.
func visitCompoundExprChildrenWithDepth(expression Expression, visitor func(Expression) bool, depth int) bool {
	switch n := expression.(type) {
	case *MemberExpression:
		visitExpressionWithDepth(n.Base, visitor, depth)
		visitExpressionWithDepth(n.Property, visitor, depth)
	case *IndexExpression:
		visitExpressionWithDepth(n.Base, visitor, depth)
		visitExpressionWithDepth(n.Index, visitor, depth)
	case *UnaryExpression:
		visitExpressionWithDepth(n.Right, visitor, depth)
	case *BinaryExpression:
		visitExpressionWithDepth(n.Left, visitor, depth)
		visitExpressionWithDepth(n.Right, visitor, depth)
	case *CallExpression:
		visitExpressionWithDepth(n.Callee, visitor, depth)
		for _, argument := range n.Args {
			visitExpressionWithDepth(argument, visitor, depth)
		}
	case *TernaryExpression:
		visitExpressionWithDepth(n.Condition, visitor, depth)
		visitExpressionWithDepth(n.Consequent, visitor, depth)
		visitExpressionWithDepth(n.Alternate, visitor, depth)
	case *ForInExpression:
		visitExpressionWithDepth(n.Collection, visitor, depth)
	case *LinkedMessageExpression:
		visitExpressionWithDepth(n.Path, visitor, depth)
	default:
		return false
	}
	return true
}

// visitLiteralExprChildrenWithDepth visits child expressions
// within literal types such as templates, objects, and arrays.
//
// Takes expression (Expression) which is the literal expression
// to visit.
// Takes visitor (func(...)) which is called for each child
// expression.
// Takes depth (int) which tracks the current depth in the
// tree.
func visitLiteralExprChildrenWithDepth(expression Expression, visitor func(Expression) bool, depth int) {
	switch n := expression.(type) {
	case *TemplateLiteral:
		for _, part := range n.Parts {
			if !part.IsLiteral {
				visitExpressionWithDepth(part.Expression, visitor, depth)
			}
		}
	case *ObjectLiteral:
		for _, value := range n.Pairs {
			visitExpressionWithDepth(value, visitor, depth)
		}
	case *ArrayLiteral:
		for _, element := range n.Elements {
			visitExpressionWithDepth(element, visitor, depth)
		}
	}
}

// buildRemovalMap creates a lookup map for nodes to be removed.
//
// Takes nodesToRemove ([]*TemplateNode) which specifies the nodes to include
// in the lookup map.
//
// Returns map[*TemplateNode]bool which allows fast membership checks.
func buildRemovalMap(nodesToRemove []*TemplateNode) map[*TemplateNode]bool {
	removalMap := make(map[*TemplateNode]bool, len(nodesToRemove))
	for _, node := range nodesToRemove {
		removalMap[node] = true
	}
	return removalMap
}

// filterNodes returns a new slice with only the nodes not marked for removal.
//
// Takes nodes ([]*TemplateNode) which is the slice to filter.
// Takes removalMap (map[*TemplateNode]bool) which marks nodes to remove.
//
// Returns []*TemplateNode which contains only nodes not in the removal map.
func filterNodes(nodes []*TemplateNode, removalMap map[*TemplateNode]bool) []*TemplateNode {
	result := make([]*TemplateNode, 0, len(nodes))
	for _, node := range nodes {
		if !removalMap[node] {
			result = append(result, node)
		}
	}
	return result
}

// filterChildrenInPlace removes marked nodes from a parent's children list.
// It avoids extra memory use when no children need to be removed.
//
// Takes parent (*TemplateNode) which is the node whose children to filter.
// Takes removalMap (map[*TemplateNode]bool) which marks nodes for removal.
func filterChildrenInPlace(parent *TemplateNode, removalMap map[*TemplateNode]bool) {
	if len(parent.Children) == 0 {
		return
	}

	if !anyChildNeedsRemoval(parent.Children, removalMap) {
		return
	}

	parent.Children = filterNodes(parent.Children, removalMap)
}

// anyChildNeedsRemoval checks whether any child node is marked for removal.
//
// Takes children ([]*TemplateNode) which is the list of child nodes to check.
// Takes removalMap (map[*TemplateNode]bool) which tracks nodes marked for
// removal.
//
// Returns bool which is true if any child is in the removal map.
func anyChildNeedsRemoval(children []*TemplateNode, removalMap map[*TemplateNode]bool) bool {
	for _, child := range children {
		if removalMap[child] {
			return true
		}
	}
	return false
}
