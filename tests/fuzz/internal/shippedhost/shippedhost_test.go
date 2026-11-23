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

package shippedhost

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko"
)

func TestOptionsRegistersEverySelectedComponent(t *testing.T) {
	options, err := Options()
	require.NoError(t, err)
	assert.Len(t, options, len(tags))
}

func TestOptionsForReportsMissingTags(t *testing.T) {
	catalogue := []piko.ComponentDefinition{{TagName: "present-tag"}}
	options, err := optionsFor(catalogue, []string{"present-tag"})
	require.NoError(t, err)
	assert.Len(t, options, 1)
	_, err = optionsFor(catalogue, []string{"present-tag", "absent-one", "absent-two"})
	require.ErrorIs(t, err, ErrMissingComponent)
	assert.Contains(t, err.Error(), "absent-one")
	assert.Contains(t, err.Error(), "absent-two")
}

func TestTagsReturnsCopy(t *testing.T) {
	first := Tags()
	require.NotEmpty(t, first)
	first[0] = "changed"
	assert.NotEqual(t, "changed", Tags()[0])
}
