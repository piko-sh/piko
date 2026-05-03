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

// Provides streaming iteration over AST nodes with event-based traversal for
// memory-efficient processing. Implements iterator patterns with pooled event
// objects for element open/close, text nodes, and raw HTML content.

import (
	"sync"
	"sync/atomic"
)

// astEventType represents the kind of event sent during AST streaming.
type astEventType int

const (
	// elementOpen marks the start of an element tag.
	elementOpen astEventType = iota

	// elementClose indicates a closing element tag event.
	elementClose

	// textNode indicates a text node event.
	textNode

	// rawHTMLNode indicates a raw HTML content event.
	rawHTMLNode

	// commentNode indicates an HTML comment event.
	commentNode
)

// String returns the human-readable name of the AST event type.
//
// Returns string which is the event type name such as "ElementOpen" or
// "Unknown".
func (t astEventType) String() string {
	switch t {
	case elementOpen:
		return "ElementOpen"
	case elementClose:
		return "ElementClose"
	case textNode:
		return "TextNode"
	case rawHTMLNode:
		return "RawHTMLNode"
	case commentNode:
		return "CommentNode"
	default:
		return "Unknown"
	}
}

// astEvent represents a single event during AST streaming, used for incremental
// processing.
type astEvent struct {
	// node is the parsed AST node for this event; nil when the event has no node.
	node *TemplateNode

	// eventType specifies the kind of AST event; 0 means unset.
	eventType astEventType

	// isVoid indicates whether the AST event represents a void return.
	isVoid bool
}

// astEventPool reuses astEvent instances to reduce allocation pressure during
// AST streaming. Wrapped in atomic.Pointer so resetASTEventPool can swap the
// underlying pool without racing concurrent Get/Put callers.
var astEventPool atomic.Pointer[sync.Pool]

func init() {
	astEventPool.Store(newASTEventPool())
}

// newASTEventPool builds a fresh sync.Pool whose New func returns a
// zero-valued astEvent. Used by init and resetASTEventPool.
//
// Returns *sync.Pool which is the freshly constructed pool.
func newASTEventPool() *sync.Pool {
	return &sync.Pool{
		New: func() any {
			return new(astEvent)
		},
	}
}

// getASTEvent gets an astEvent from the sync.Pool.
//
// Returns *astEvent which is a reused event from the pool, or a new one if the
// pool is empty.
func getASTEvent() *astEvent {
	event, ok := astEventPool.Load().Get().(*astEvent)
	if !ok {
		event = new(astEvent)
	}
	return event
}

// putASTEvent returns an astEvent to the pool after resetting it.
//
// When event is nil, returns at once without action.
//
// Takes event (*astEvent) which is the event to reset and return to the pool.
func putASTEvent(event *astEvent) {
	if event == nil {
		return
	}

	event.eventType = 0
	event.node = nil
	event.isVoid = false

	astEventPool.Load().Put(event)
}

// resetASTEventPool atomically swaps in a fresh AST event pool for test
// isolation. Safe to call concurrently with Get/Put.
func resetASTEventPool() {
	astEventPool.Store(newASTEventPool())
}
