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

package driven_system_symbols

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCoerceDirectMatch(t *testing.T) {
	t.Parallel()

	require.Equal(t, 42, coerce[int](42))
	require.Equal(t, "hello", coerce[string]("hello"))
	require.Equal(t, true, coerce[bool](true))
	require.Equal(t, 3.14, coerce[float64](3.14))
}

func TestCoerceInt64ToInt(t *testing.T) {
	t.Parallel()

	require.Equal(t, 42, coerce[int](int64(42)))
}

func TestCoerceUint64ToByte(t *testing.T) {
	t.Parallel()

	require.Equal(t, byte(255), coerce[byte](uint64(255)))
}

func TestCoerceIntToFloat64(t *testing.T) {
	t.Parallel()

	require.Equal(t, float64(3), coerce[float64](int(3)))
}

func TestCoerceInt64ToFloat64(t *testing.T) {
	t.Parallel()

	require.Equal(t, float64(7), coerce[float64](int64(7)))
}
