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

package db_engine_timescaledb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIdentifierNeedsQuoting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "plain lowercase", value: "metric", want: false},
		{name: "with underscore", value: "metric_value", want: false},
		{name: "with trailing digit", value: "col1", want: false},
		{name: "mixed case left bare", value: "MetricValue", want: false},
		{name: "unicode letters left bare", value: "naïve", want: false},
		{name: "unicode greek left bare", value: "температура", want: false},
		{name: "empty needs quoting", value: "", want: true},
		{name: "leading digit needs quoting", value: "1col", want: true},
		{name: "whitespace needs quoting", value: "two words", want: true},
		{name: "punctuation needs quoting", value: "a-b", want: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, identifierNeedsQuoting(testCase.value))
		})
	}
}

func TestRequoteIdentifierLeavesUnicodeBare(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "naïve", requoteIdentifier("naïve"))
	assert.Equal(t, `"two words"`, requoteIdentifier("two words"))
	assert.Equal(t, `"a""b"`, requoteIdentifier(`a"b`))
}
