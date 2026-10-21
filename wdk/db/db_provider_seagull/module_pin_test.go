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

package db_provider_seagull

import (
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModulePinsSeagullWithoutReplace(t *testing.T) {
	t.Parallel()

	contents, err := os.ReadFile("go.mod")
	require.NoError(t, err)

	goMod := string(contents)
	require.Regexp(t, regexp.MustCompile(`(?m)^\s*seagull\.sh/seagull v\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?\s*$`), goMod)
	require.NotRegexp(t, regexp.MustCompile(`(?m)^\s*replace\b`), goMod)
}
