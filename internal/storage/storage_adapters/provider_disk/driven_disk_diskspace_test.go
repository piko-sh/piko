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

//go:build linux || darwin || freebsd || openbsd || netbsd || windows

package provider_disk

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDiskSpace(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name: "reports space for an existing directory",
			path: t.TempDir(),
		},
		{
			name:    "fails for a path that does not exist",
			path:    filepath.Join(t.TempDir(), "missing", "directory"),
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			availableMB, totalMB, err := getDiskSpace(testCase.path)
			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "querying disk space")
				assert.Zero(t, availableMB)
				assert.Zero(t, totalMB)
				return
			}

			require.NoError(t, err)
			assert.Positive(t, totalMB)
			assert.LessOrEqual(t, availableMB, totalMB)
		})
	}
}
