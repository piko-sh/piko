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

package generator_helpers

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/templater/templater_dto"
)

const (
	actionEncoderDeterminismRuns = 32
)

func newMapArgumentPayload(value any) templater_dto.ActionPayload {
	return templater_dto.ActionPayload{
		Function: "save",
		Args:     []templater_dto.ActionArgument{{Type: "s", Value: value}},
	}
}

func decodeStaticPayload(t *testing.T, encoded string) string {
	t.Helper()
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	return string(decoded)
}

func TestEncodeStaticActionPayload_MapKeysSorted(t *testing.T) {
	t.Parallel()

	payload := newMapArgumentPayload(map[string]any{
		"zeta":    1,
		"alpha":   "a",
		"epsilon": true,
		"beta":    map[string]any{"y": 2, "x": 1, "z": 3},
		"delta":   nil,
		"gamma":   []any{"g"},
		"eta":     2.5,
		"theta":   "t",
	})
	want := `{"f":"save","a":[{"t":"s","v":{"alpha":"a","beta":{"x":1,"y":2,"z":3},"delta":null,"epsilon":true,"eta":2.5,"gamma":["g"],"theta":"t","zeta":1}}]}`

	for range actionEncoderDeterminismRuns {
		require.Equal(t, want, decodeStaticPayload(t, EncodeStaticActionPayload(payload)))
	}
}

func TestEncodeStaticActionPayload_ReflectMapKeysSorted(t *testing.T) {
	t.Parallel()

	payload := newMapArgumentPayload(map[string]string{
		"zeta":    "z",
		"alpha":   "a",
		"epsilon": "e",
		"beta":    "b",
		"delta":   "d",
		"gamma":   "g",
		"eta":     "h",
		"theta":   "t",
	})
	want := `{"f":"save","a":[{"t":"s","v":{"alpha":"a","beta":"b","delta":"d","epsilon":"e","eta":"h","gamma":"g","theta":"t","zeta":"z"}}]}`

	for range actionEncoderDeterminismRuns {
		require.Equal(t, want, decodeStaticPayload(t, EncodeStaticActionPayload(payload)))
	}
}
