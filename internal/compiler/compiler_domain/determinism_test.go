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

package compiler_domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/json"
)

const (
	compilerDeterminismRuns = 32
)

var (
	errUnsortedProviderMarshal = errors.New("provider Marshal does not sort map keys")
)

func TestGetLoopVarNames_ReturnsNamesInSortedOrder(t *testing.T) {
	t.Parallel()

	loopVars := map[string]bool{
		"row":      true,
		"rowIndex": true,
		"item":     true,
		"index":    true,
		"cell":     true,
		"cellKey":  true,
		"group":    true,
		"groupIdx": true,
	}
	want := []string{"cell", "cellKey", "group", "groupIdx", "index", "item", "row", "rowIndex"}

	for range compilerDeterminismRuns {
		require.Equal(t, want, getLoopVarNames(loopVars), "loop variable names must come back in the same order on every run")
	}
}

func TestSerialiseActions_SortsParamsWhateverTheJSONProvider(t *testing.T) {
	original := json.Marshal
	json.Marshal = func(any) ([]byte, error) {
		return nil, errUnsortedProviderMarshal
	}
	t.Cleanup(func() {
		json.Marshal = original
	})

	actions := []timelineAction{{
		Action: "custom",
		Ref:    "box",
		Params: map[string]string{
			"zeta":    "z",
			"alpha":   "a",
			"epsilon": "e",
			"beta":    "b",
			"delta":   "d",
			"gamma":   "g",
			"eta":     "h",
			"theta":   "t",
		},
	}}

	for range compilerDeterminismRuns {
		serialised, err := serialiseActions(actions)

		require.NoError(t, err)
		require.Contains(t, serialised, `"params":{"alpha":"a","beta":"b","delta":"d","epsilon":"e","eta":"h","gamma":"g","theta":"t","zeta":"z"}`)
	}
}
