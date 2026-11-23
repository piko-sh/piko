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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReloadOrderReverses(t *testing.T) {
	assert.Equal(t, []string{"merge", "replace", "children-only", "attrs-only"}, PartialComponentsInput{}.ReloadOrder())
	assert.Equal(t, []string{"attrs-only", "children-only", "replace", "merge"}, PartialComponentsInput{ReverseReload: true}.ReloadOrder())
}

func TestExpectedPartialComponentsAppendsReloadModes(t *testing.T) {
	plain := ExpectedPartialComponents(PartialComponentsInput{Edited: "x"})
	withModes := ExpectedPartialComponents(PartialComponentsInput{Edited: "x", ReloadModes: true})

	require.Len(t, withModes.States, len(plain.States)+4)
	modes := withModes.States[len(plain.States):]
	assert.Equal(t, "reload-merge", modes[0].Step)
	require.NotNil(t, modes[1].Reload)
	assert.Empty(t, modes[1].Reload.LiveOnly, "replace discards client-only state")
	children := modes[2].Reload
	assert.Equal(t, "replace", children.RootStage, "children-only leaves the root stage")
	assert.Equal(t, "children-only", children.ChildStage)
	attributes := modes[3].Reload
	assert.Equal(t, "attrs-only", attributes.RootStage)
	assert.Equal(t, "children-only", attributes.ChildStage, "attrs-only leaves the child stage")
	assert.Equal(t, "abc-x", attributes.Draft)
}
