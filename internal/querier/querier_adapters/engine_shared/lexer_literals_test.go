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

package engine_shared_test

import (
	"testing"

	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
)

func TestScanDoubledDelimiter(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		input     string
		delimiter byte
		wantValue string
		wantEnd   int
		wantOK    bool
	}{
		{name: "simple string", input: "'hello'", delimiter: '\'', wantValue: "hello", wantEnd: 7, wantOK: true},
		{name: "doubled quote", input: "'it''s'", delimiter: '\'', wantValue: "it's", wantEnd: 7, wantOK: true},
		{name: "empty literal", input: "''", delimiter: '\'', wantValue: "", wantEnd: 2, wantOK: true},
		{name: "double quoted identifier", input: "\"col\"", delimiter: '"', wantValue: "col", wantEnd: 5, wantOK: true},
		{name: "doubled in identifier", input: "\"a\"\"b\"", delimiter: '"', wantValue: "a\"b", wantEnd: 6, wantOK: true},
		{name: "trailing text after close", input: "'x' y", delimiter: '\'', wantValue: "x", wantEnd: 3, wantOK: true},
		{name: "unterminated", input: "'oops", delimiter: '\'', wantValue: "", wantEnd: 5, wantOK: false},
	}

	for index := range cases {
		testCase := cases[index]
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			value, end, ok := engine_shared.ScanDoubledDelimiter(testCase.input, 0, testCase.delimiter)
			if ok != testCase.wantOK {
				t.Fatalf("ok = %v, want %v", ok, testCase.wantOK)
			}
			if !testCase.wantOK {
				return
			}
			if value != testCase.wantValue {
				t.Fatalf("value = %q, want %q", value, testCase.wantValue)
			}
			if end != testCase.wantEnd {
				t.Fatalf("end = %d, want %d", end, testCase.wantEnd)
			}
		})
	}
}
