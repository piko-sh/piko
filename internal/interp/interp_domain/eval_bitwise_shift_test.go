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

func TestEvalBitwiseShiftRuntime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect int64
	}{
		{
			name:   "shift left via variables",
			code:   "x := 1\nn := 4\nx << n",
			expect: int64(16),
		},
		{
			name:   "shift right via variables",
			code:   "x := 256\nn := 4\nx >> n",
			expect: int64(16),
		},
		{
			name:   "bitwise AND via variables",
			code:   "x := 0xFF\ny := 0x0F\nx & y",
			expect: int64(15),
		},
		{
			name:   "bitwise OR via variables",
			code:   "x := 0xF0\ny := 0x0F\nx | y",
			expect: int64(255),
		},
		{
			name:   "bitwise XOR via variables",
			code:   "x := 0xFF\ny := 0x0F\nx ^ y",
			expect: int64(240),
		},
		{
			name:   "bitwise AND NOT via variables",
			code:   "x := 0xFF\ny := 0x0F\nx &^ y",
			expect: int64(240),
		},
		{
			name:   "combined shift and OR via variables",
			code:   "x := 3\ny := 2\n(x << y) | (x >> y)",
			expect: int64(12),
		},
		{
			name:   "large shift via variables",
			code:   "x := 1\nn := 32\nx << n",
			expect: int64(4294967296),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}

func TestEvalBitwiseCompound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect int64
	}{
		{
			name:   "AND assign",
			code:   "x := 0xFF\nx &= 0x0F\nx",
			expect: int64(15),
		},
		{
			name:   "OR assign",
			code:   "x := 0\nx |= 0xFF\nx",
			expect: int64(255),
		},
		{
			name:   "XOR assign",
			code:   "x := 0xFF\nx ^= 0x0F\nx",
			expect: int64(240),
		},
		{
			name:   "shift left assign",
			code:   "x := 1\nx <<= 8\nx",
			expect: int64(256),
		},
		{
			name:   "shift right assign",
			code:   "x := 256\nx >>= 4\nx",
			expect: int64(16),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}
