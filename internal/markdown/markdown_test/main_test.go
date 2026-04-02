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

package markdown_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMarkdown_Integration(t *testing.T) {
	testdataRoot := "./testdata"

	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		t.Fatalf("Critical test setup error: Failed to read testdata directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			testCasePath := filepath.Join(testdataRoot, entry.Name())

			tc := testCase{
				Name:       entry.Name(),
				Path:       testCasePath,
				SourceFile: "source.md",
			}

			t.Run(tc.Name, func(t *testing.T) {
				runTestCase(t, tc)
			})
		}
	}
}
