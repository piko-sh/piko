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
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	lruSourceForInliner = `type lruNode struct {
	key      int64
	value    int64
	previous *lruNode
	next     *lruNode
}
type lruCache struct {
	capacity int
	size     int
	lookup   map[int64]*lruNode
	head     *lruNode
	tail     *lruNode
}
func newCache(capacity int) *lruCache {
	return &lruCache{capacity: capacity, lookup: map[int64]*lruNode{}}
}
func (c *lruCache) detach(node *lruNode) {
	if node.previous != nil {
		node.previous.next = node.next
	} else {
		c.head = node.next
	}
	if node.next != nil {
		node.next.previous = node.previous
	} else {
		c.tail = node.previous
	}
}
func (c *lruCache) attachAtFront(node *lruNode) {
	node.previous = nil
	node.next = c.head
	if c.head != nil {
		c.head.previous = node
	}
	c.head = node
	if c.tail == nil {
		c.tail = node
	}
}
func (c *lruCache) moveToFront(node *lruNode) {
	if node == c.head {
		return
	}
	c.detach(node)
	c.attachAtFront(node)
}
func (c *lruCache) put(key, value int64) {
	if existing, found := c.lookup[key]; found {
		existing.value = value
		c.moveToFront(existing)
		return
	}
	node := &lruNode{key: key, value: value}
	c.lookup[key] = node
	c.attachAtFront(node)
	c.size++
}
c := newCache(8)
c.put(1, 100)
c.put(2, 200)
c.put(3, 300)
c.size`
)

func TestInlining_LRUDiagnostic(t *testing.T) {
	withInlineEnabled(t, func() {
		service := NewService()
		result, err := service.Eval(context.Background(), lruSourceForInliner)
		require.NoError(t, err)
		require.Equal(t, int64(3), result, "result should be size after 3 puts")
	})
}
