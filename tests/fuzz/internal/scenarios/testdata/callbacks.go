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

package interp_provider_pipit_test

import (
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratedPikoScenario(t *testing.T) {
	draws, err := hex.DecodeString(os.Getenv("PIKO_FUZZ_CASE_TOKEN"))
	require.NoError(t, err)
	require.Len(t, draws, 8)
	const invocation = "library.Ledger(1).Advance(delta)"
	units := 1 + int(draws[0]%6)
	for _, pooled := range []bool{false, true} {
//line piko_fuzz_scenario_test.go:21
		t.Run(fmt.Sprintf("pooled-%t", pooled), func(t *testing.T) {
			checkRegisteredCallbacks(t, invocation, units, pooled, int(draws[1]%9)-4, int(draws[2]%9)-4)
		})
	}
}
