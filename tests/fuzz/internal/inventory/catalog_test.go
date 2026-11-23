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

package inventory

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryCapabilitiesHaveOwners(t *testing.T) {
	got, err := Discover(os.DirFS("../../../.."))
	require.NoError(t, err)
	assert.Empty(t, got.Unassigned, "unassigned capabilities")
	assert.Empty(t, got.Removed, "stale assignments")
	require.NotEmpty(t, got.Capabilities, "no repository capabilities discovered")
}

func TestClassifyRejectsUnreviewedAndRemovedCapabilities(t *testing.T) {
	got, err := classify(map[string]Capability{"cli\tadded": {Kind: "cli", Name: "added"}}, "cli\told\toperator contract\n")
	require.NoError(t, err)
	assert.Equal(t, []string{"cli\tadded"}, got.Unassigned)
	assert.Equal(t, []string{"cli\told"}, got.Removed)
}

func TestClassifyAssignsOwners(t *testing.T) {
	got, err := classify(map[string]Capability{"cli\tbuild": {Kind: "cli", Name: "build"}}, "# comment\n\ncli\tbuild\toperator contract\n")
	require.NoError(t, err)
	require.Len(t, got.Capabilities, 1)
	assert.Equal(t, "operator contract", got.Capabilities[0].Owner)
	assert.Empty(t, got.Unassigned)
	assert.Empty(t, got.Removed)
}

func TestClassifyRejectsInvalidAssignments(t *testing.T) {
	testCases := []struct {
		name        string
		assignments string
		message     string
	}{
		{name: "missing owner", assignments: "cli\tbuild\t\n", message: "invalid capability assignment"},
		{name: "too few columns", assignments: "cli\tbuild\n", message: "invalid capability assignment"},
		{name: "duplicate", assignments: "cli\tbuild\tone\ncli\tbuild\ttwo\n", message: "duplicate capability"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := classify(map[string]Capability{}, testCase.assignments)
			require.ErrorContains(t, err, testCase.message)
		})
	}
}
