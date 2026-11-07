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

package main

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionFromBuildInfo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		expected string
		info     debug.BuildInfo
	}{
		{
			name:     "tagged module version wins",
			info:     debug.BuildInfo{Main: debug.Module{Version: "v1.4.0"}},
			expected: "v1.4.0",
		},
		{
			name: "devel build falls back to the clean VCS revision",
			info: debug.BuildInfo{
				Main:     debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "0123456789abcdef0123"}, {Key: "vcs.modified", Value: "false"}},
			},
			expected: "dev+0123456789ab",
		},
		{
			name: "modified tree is marked dirty",
			info: debug.BuildInfo{
				Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc123"}, {Key: "vcs.modified", Value: "true"}},
			},
			expected: "dev+abc123-dirty",
		},
		{
			name:     "no version information yields the bare dev version",
			info:     debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			expected: develVersion,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.expected, versionFromBuildInfo(&testCase.info))
		})
	}
}

func TestVersionFromBuildInfo_NilInfo(t *testing.T) {
	t.Parallel()

	assert.Equal(t, develVersion, versionFromBuildInfo(nil))
}

func TestResolveVersion_UnstampedUsesBuildInfo(t *testing.T) {
	original := Version
	t.Cleanup(func() { Version = original })

	Version = develVersion
	assert.NotEmpty(t, resolveVersion())
}

func TestResolveVersionPrefersStampedVersion(t *testing.T) {
	original := Version
	t.Cleanup(func() { Version = original })

	Version = "2.0.0"
	assert.Equal(t, "2.0.0", resolveVersion())
}
