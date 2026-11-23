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
package json_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/json"
)

func TestPublicJSONContract(t *testing.T) {
	want := []string{"lune 🌒", "\"\n"}
	raw, err := json.Marshal(want)
	require.NoError(t, err)
	var got []string
	err = json.Unmarshal(raw, &got)
	require.NoError(t, err, "PIKO_CHECK roundtrip: %v", got)
	require.Equal(t, want, got, "PIKO_CHECK roundtrip: %v %v", got, err)
	require.Error(t, json.Unmarshal([]byte(`{"k":]`), &got), "PIKO_CHECK malformed: accepted invalid input")
}
