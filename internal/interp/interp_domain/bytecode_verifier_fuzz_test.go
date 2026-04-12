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

//go:build fuzz

package interp_domain

import (
	"context"
	"testing"
)

func FuzzVerifier(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0, 0, 0, 0})
	f.Add([]byte{byte(opAddInt), 1, 1, 2})
	f.Add([]byte{
		byte(opLoadIntConst), 0, 0, 0,
		byte(opAddInt), 1, 0, 0,
		byte(opDrillTier1), byte(subOpDrillTier2), byte(subOpTier2Return), 1,
	})
	f.Add(make([]byte, 64))

	f.Fuzz(func(t *testing.T, raw []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("verifier panicked: %v", r)
			}
		}()
		body := decodeFuzzBody(raw)
		cf := &CompiledFunction{
			name: "fuzz",
			body: body,
		}
		report, err := verifyBytecode(context.Background(), cf)
		if report == nil {
			t.Fatalf("verifyBytecode returned nil report (err=%v)", err)
		}
		_ = report.Format()
	})
}
