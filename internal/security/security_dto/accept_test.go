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

package security_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAcceptsMediaType(t *testing.T) {
	t.Parallel()

	const eventStream = "text/event-stream"

	testCases := []struct {
		name     string
		accept   string
		expected bool
	}{
		{name: "exact match", accept: "text/event-stream", expected: true},
		{name: "ranked list", accept: "text/event-stream, */*;q=0.1", expected: true},
		{name: "second in the list", accept: "application/json, text/event-stream", expected: true},
		{name: "with a q-value", accept: "text/event-stream;q=0.9", expected: true},
		{name: "surrounding spaces", accept: "  text/event-stream  ", expected: true},
		{name: "different case", accept: "Text/Event-Stream", expected: true},
		{name: "wildcard alone never matches", accept: "*/*", expected: false},
		{name: "type wildcard never matches", accept: "text/*", expected: false},
		{name: "browser navigation", accept: "text/html,application/xhtml+xml,*/*;q=0.8", expected: false},
		{name: "a different type", accept: "application/json", expected: false},
		{name: "a longer type that contains it", accept: "text/event-stream-plus", expected: false},
		{name: "empty header", accept: "", expected: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.expected, AcceptsMediaType(testCase.accept, eventStream))
		})
	}
}
