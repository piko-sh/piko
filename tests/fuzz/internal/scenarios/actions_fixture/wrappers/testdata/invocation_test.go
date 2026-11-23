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

package actions

import (
	"encoding/hex"
	"os"
	"strconv"
	"testing"

	"piko.sh/piko/tests/fuzz/internal/scenarios/actions_fixture/actionimpl"
)

func TestInvokeGeneratedBinding(t *testing.T) {
	token := os.Getenv("PIKO_FUZZ_CASE_TOKEN")
	seed, err := hex.DecodeString(token)
	if err != nil || len(seed) < 2 {
		t.Fatalf("invalid case token %q", token)
	}
	want := actionimpl.Input{Label: "item-" + token, Quantity: int(seed[0]) - 128, Enabled: seed[1]%2 == 0}
	jsonArgs := map[string]any{"Label": want.Label, "Quantity": float64(want.Quantity), "Enabled": want.Enabled}
	form := map[string]any{"Label": want.Label, "Quantity": strconv.Itoa(want.Quantity), "Enabled": strconv.FormatBool(want.Enabled)}
	for name, args := range map[string]map[string]any{"nested": {"input": jsonArgs}, "flat": form} {
		t.Run(name, func(t *testing.T) {
			action := &actionimpl.CaptureAction{}
			got, err := invokeFuzzCollect(t.Context(), action, args)
			if err != nil || got != want || action.Calls != 1 {
				t.Fatalf("PIKO_CHECK binding: want %#v + one call, got %#v, calls %d, err %v", want, got, action.Calls, err)
			}
		})
	}
	action := &actionimpl.CaptureAction{}
	form["Quantity"] = "invalid-" + token
	if _, err := invokeFuzzCollect(t.Context(), action, form); err == nil || action.Calls != 0 {
		t.Fatalf("PIKO_CHECK failed-bind: expected binding error and zero action calls, got %v and %d", err, action.Calls)
	}
	if !t.Failed() {
		if err := os.WriteFile(os.Getenv("PIKO_FUZZ_WRAPPER_RESULT"), []byte(token), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
