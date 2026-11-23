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

package casefile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelSourceCoversEveryModelFile(t *testing.T) {
	source, err := ModelSource()
	require.NoError(t, err)
	names, err := filepath.Glob("*.go")
	require.NoError(t, err)
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			assert.NotContains(t, source, name+"\x00", "test file %s contributes to the profile", name)
			continue
		}
		data, err := os.ReadFile(name)
		require.NoError(t, err)
		assert.Contains(t, source, name+"\x00"+string(data), "model file %s missing from the profile", name)
	}
	for _, required := range []string{"checkout.go", "partial_modes.go", "editions.go"} {
		assert.Contains(t, source, required+"\x00")
	}
}
