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

package pipitpending

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalFollowsTheEnvironment(t *testing.T) {
	t.Setenv(LocalEnv, "1")
	assert.True(t, Local())

	t.Setenv(LocalEnv, "")
	assert.False(t, Local())
}

func TestSkipRunsTheTestAgainstLocalPipit(t *testing.T) {
	t.Setenv(LocalEnv, "1")
	ran := false
	t.Run("pending", func(t *testing.T) {
		Skip(t, "example fix")
		ran = true
	})
	assert.True(t, ran)
}

func TestSkipSkipsAgainstThePinnedRelease(t *testing.T) {
	t.Setenv(LocalEnv, "")
	var skipped bool
	t.Run("pending", func(t *testing.T) {
		t.Cleanup(func() { skipped = t.Skipped() })
		Skip(t, "example fix")
	})
	assert.True(t, skipped)
}
