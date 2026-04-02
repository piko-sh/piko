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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestASTEventType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		expectedString string
		eventType      astEventType
	}{
		{name: "ElementOpen", expectedString: "ElementOpen", eventType: elementOpen},
		{name: "ElementClose", expectedString: "ElementClose", eventType: elementClose},
		{name: "TextNode", expectedString: "TextNode", eventType: textNode},
		{name: "RawHTMLNode", expectedString: "RawHTMLNode", eventType: rawHTMLNode},
		{name: "CommentNode", expectedString: "CommentNode", eventType: commentNode},
		{name: "UnknownType", expectedString: "Unknown", eventType: astEventType(99)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedString, tt.eventType.String())
		})
	}
}

func TestASTEventPool_GetAndPut(t *testing.T) {
	t.Parallel()

	initialEvent := getASTEvent()
	require.NotNil(t, initialEvent, "getASTEvent should never return nil")

	assert.Equal(t, astEventType(0), initialEvent.eventType, "Fresh event should have zeroed Type")
	assert.Nil(t, initialEvent.node, "Fresh event should have a nil Node")
	assert.False(t, initialEvent.isVoid, "Fresh event should have IsVoid set to false")

	nodeForEvent := &TemplateNode{
		TagName:     "div",
		TextContent: "some content",
	}

	initialEvent.eventType = elementOpen
	initialEvent.isVoid = true
	initialEvent.node = nodeForEvent

	putASTEvent(initialEvent)

	recycledEvent := getASTEvent()
	require.NotNil(t, recycledEvent, "getASTEvent after Put should still return a valid event")

	assert.Equal(t, astEventType(0), recycledEvent.eventType, "Recycled event should have its Type reset")
	assert.Nil(t, recycledEvent.node, "Recycled event should have its Node reset to nil")
	assert.False(t, recycledEvent.isVoid, "Recycled event should have its IsVoid flag reset")
}

func TestASTEventPool_EventWithoutNode(t *testing.T) {
	t.Parallel()

	event := getASTEvent()
	event.eventType = elementClose
	event.isVoid = false

	assert.NotPanics(t, func() {
		putASTEvent(event)
	}, "Putting an event with a nil node should not panic")

	recycledEvent := getASTEvent()
	assert.Equal(t, astEventType(0), recycledEvent.eventType)
	assert.Nil(t, recycledEvent.node)
}

func TestASTEventPool_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	numGoroutines := 100
	numCyclesPerGoroutine := 1000
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(goroutineID int) {
			defer wg.Done()
			for range numCyclesPerGoroutine {
				event := getASTEvent()
				node := &TemplateNode{}

				require.Equal(t, astEventType(0), event.eventType)
				require.Nil(t, event.node)
				require.Equal(t, "", node.TagName)

				event.eventType = textNode
				event.node = node
				event.node.TextContent = "test"

				putASTEvent(event)
			}
		}(i)
	}

	wg.Wait()
}

func TestPutASTEvent_NilSafety(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		putASTEvent(nil)
	}, "putASTEvent(nil) should not cause a panic")
}
