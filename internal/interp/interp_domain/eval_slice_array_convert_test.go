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

func TestEvalSliceToArrayPointerBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		want    any
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "pointer to array exact length",
			code: `s := []int{10, 20, 30}; q := (*[3]int)(s); (*q)[1]`,
			want: int64(20),
		},
		{
			name: "pointer to shorter array from longer slice",
			code: `s := []int{10, 20, 30, 40}; q := (*[2]int)(s); (*q)[0] + (*q)[1]`,
			want: int64(30),
		},
		{
			name:    "pointer to longer array from shorter slice panics",
			code:    `s := []int{10, 20}; q := (*[8]int)(s); len(q)`,
			wantErr: true,
		},
		{
			name:    "value array longer than slice panics",
			code:    `s := []int{10, 20}; a := [8]int(s); a[0]`,
			wantErr: true,
		},
	}

	modes := []struct {
		name string
		opts []Option
	}{
		{name: "asm"},
		{name: "go", opts: []Option{WithForceGoDispatch()}},
		{name: "safe", opts: []Option{WithSafeMode()}},
	}

	for _, mode := range modes {
		for _, tt := range tests {
			t.Run(mode.name+"/"+tt.name, func(t *testing.T) {
				t.Parallel()
				service := NewService(mode.opts...)
				result, err := service.Eval(context.Background(), tt.code)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.Equal(t, tt.want, result)
			})
		}
	}
}
