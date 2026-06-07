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

package interp_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithSafeMode_PlumbedToConfigAndLimits(t *testing.T) {
	t.Parallel()
	service := NewService(WithSafeMode())

	require.NotNil(t, service.config)
	require.True(t, service.config.safeMode, "WithSafeMode must set config.safeMode")
	require.True(t, service.limits.safeMode, "safeMode must propagate to vmLimits")
	require.True(t, service.limits.forceGoDispatch,
		"safe mode must force Go dispatch so guards run off the ASM fast path")
}

func TestSafeModeParityWithDefault(t *testing.T) {
	t.Parallel()

	programs := []struct {
		want any
		name string
		code string
	}{
		{name: "arithmetic", code: `x := 0; for i := 0; i < 10; i++ { x += i }; x`, want: int64(45)},
		{name: "string build", code: `s := ""; for i := 0; i < 3; i++ { s += "a" }; s`, want: "aaa"},
		{name: "slice sum", code: `xs := []int{1, 2, 3, 4}; t := 0; for _, v := range xs { t += v }; t`, want: int64(10)},
		{name: "map lookup", code: `m := map[string]int{"a": 1, "b": 2}; m["a"] + m["b"]`, want: int64(3)},
		{name: "closure", code: `add := func(a, b int) int { return a + b }; add(3, 4)`, want: int64(7)},
	}

	for _, program := range programs {
		t.Run(program.name, func(t *testing.T) {
			t.Parallel()
			fast, fastErr := NewService().Eval(context.Background(), program.code)
			require.NoError(t, fastErr)

			safe, safeErr := NewService(WithSafeMode()).Eval(context.Background(), program.code)
			require.NoError(t, safeErr)

			require.Equal(t, program.want, fast)
			require.Equal(t, fast, safe, "safe mode must produce identical results to fast mode")
		})
	}
}
