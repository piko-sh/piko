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
package system_memlimit_automemlimit_test

import (
	"os"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"
	limit "piko.sh/piko/wdk/system/system_memlimit_automemlimit"
)

func TestPublicMemlimitContract(t *testing.T) {
	prev := debug.SetMemoryLimit(-1)
	defer debug.SetMemoryLimit(prev)
	t.Setenv("GOMEMLIMIT", "")
	require.NoError(t, os.Unsetenv("GOMEMLIMIT"))
	_, err := limit.Provider(limit.WithRatio(-1))()
	require.Error(t, err, "PIKO_CHECK invalid-ratio: accepted negative ratio")
	n, err := limit.Provider(limit.WithRatio(0.5))()
	require.True(t, err == nil && n > 0 && n == debug.SetMemoryLimit(-1), "PIKO_CHECK applied-limit: %d %v", n, err)
}
