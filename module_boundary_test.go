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

package piko

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
)

func TestPikoRootDoesNotRequirePipit(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("go.mod")
	require.NoError(t, err)

	file, err := modfile.Parse("go.mod", data, nil)
	require.NoError(t, err)

	for _, requirement := range file.Require {
		require.Falsef(t, strings.HasPrefix(requirement.Mod.Path, "pipit.sh/"),
			"root go.mod requires %s; pipit must only be reached through wdk/interp/interp_provider_pipit",
			requirement.Mod.Path)
	}
}
