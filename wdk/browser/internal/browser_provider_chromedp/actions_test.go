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

package browser_provider_chromedp

import (
	"testing"

	"github.com/chromedp/cdproto/input"
)

func TestKeyMapContainsExpectedKeys(t *testing.T) {
	expectedKeys := []string{
		"Enter", "Tab", "Escape", "Backspace", "Delete",
		"ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight",
		"Home", "End", "PageUp", "PageDown", "Space", "Insert",
		"F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12",
	}

	for _, key := range expectedKeys {
		t.Run(key, func(t *testing.T) {
			if _, ok := keyMap[key]; !ok {
				t.Errorf("keyMap missing expected key: %q", key)
			}
		})
	}
}

func TestKeyMapValues(t *testing.T) {
	testCases := []struct {
		name            string
		key             string
		expectedKey     string
		expectedCode    string
		expectedKeyCode int64
	}{
		{name: "Enter maps correctly", key: "Enter", expectedKey: "Enter", expectedCode: "Enter", expectedKeyCode: 13},
		{name: "Tab maps correctly", key: "Tab", expectedKey: "Tab", expectedCode: "Tab", expectedKeyCode: 9},
		{name: "Escape maps correctly", key: "Escape", expectedKey: "Escape", expectedCode: "Escape", expectedKeyCode: 27},
		{name: "ArrowUp maps correctly", key: "ArrowUp", expectedKey: "ArrowUp", expectedCode: "ArrowUp", expectedKeyCode: 38},
		{name: "ArrowDown maps correctly", key: "ArrowDown", expectedKey: "ArrowDown", expectedCode: "ArrowDown", expectedKeyCode: 40},
		{name: "Space maps correctly", key: "Space", expectedKey: " ", expectedCode: "Space", expectedKeyCode: 32},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info := keyMap[tc.key]
			if info.Key != tc.expectedKey {
				t.Errorf("keyMap[%q].Key = %q, expected %q", tc.key, info.Key, tc.expectedKey)
			}
			if info.Code != tc.expectedCode {
				t.Errorf("keyMap[%q].Code = %q, expected %q", tc.key, info.Code, tc.expectedCode)
			}
			if info.KeyCode != tc.expectedKeyCode {
				t.Errorf("keyMap[%q].KeyCode = %d, expected %d", tc.key, info.KeyCode, tc.expectedKeyCode)
			}
		})
	}
}

func TestModifierMapContainsExpectedKeys(t *testing.T) {
	expectedKeys := []string{
		"Shift", "Control", "Ctrl", "Alt", "Meta", "Cmd",
	}

	for _, key := range expectedKeys {
		t.Run(key, func(t *testing.T) {
			if _, ok := modifierMap[key]; !ok {
				t.Errorf("modifierMap missing expected key: %q", key)
			}
		})
	}
}

func TestModifierMapValues(t *testing.T) {
	testCases := []struct {
		name             string
		key              string
		expectedKey      string
		expectedCode     string
		expectedModifier input.Modifier
	}{
		{name: "Shift maps correctly", key: "Shift", expectedKey: "Shift", expectedCode: "ShiftLeft", expectedModifier: input.ModifierShift},
		{name: "Control maps correctly", key: "Control", expectedKey: "Control", expectedCode: "ControlLeft", expectedModifier: input.ModifierCtrl},
		{name: "Ctrl alias maps to Control", key: "Ctrl", expectedKey: "Control", expectedCode: "ControlLeft", expectedModifier: input.ModifierCtrl},
		{name: "Alt maps correctly", key: "Alt", expectedKey: "Alt", expectedCode: "AltLeft", expectedModifier: input.ModifierAlt},
		{name: "Meta maps correctly", key: "Meta", expectedKey: "Meta", expectedCode: "MetaLeft", expectedModifier: input.ModifierMeta},
		{name: "Cmd alias maps to Meta", key: "Cmd", expectedKey: "Meta", expectedCode: "MetaLeft", expectedModifier: input.ModifierMeta},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info := modifierMap[tc.key]
			if info.Key != tc.expectedKey {
				t.Errorf("modifierMap[%q].Key = %q, expected %q", tc.key, info.Key, tc.expectedKey)
			}
			if info.Code != tc.expectedCode {
				t.Errorf("modifierMap[%q].Code = %q, expected %q", tc.key, info.Code, tc.expectedCode)
			}
			if info.Modifier != tc.expectedModifier {
				t.Errorf("modifierMap[%q].Modifier = %v, expected %v", tc.key, info.Modifier, tc.expectedModifier)
			}
		})
	}
}

func TestCtrlAndControlAreEquivalent(t *testing.T) {
	ctrl := modifierMap["Ctrl"]
	control := modifierMap["Control"]
	if ctrl.Modifier != control.Modifier || ctrl.Key != control.Key {
		t.Error("Ctrl and Control should map to the same modifier")
	}
}

func TestCmdAndMetaAreEquivalent(t *testing.T) {
	command := modifierMap["Cmd"]
	meta := modifierMap["Meta"]
	if command.Modifier != meta.Modifier || command.Key != meta.Key {
		t.Error("Cmd and Meta should map to the same modifier")
	}
}

func TestResolveMainKey(t *testing.T) {
	testCases := []struct {
		name     string
		keyName  string
		wantKey  string
		wantCode string
		wantSpec bool
		wantErr  bool
	}{
		{
			name:     "special key Enter",
			keyName:  "Enter",
			wantSpec: true,
			wantKey:  "Enter",
			wantCode: "Enter",
		},
		{
			name:     "special key Tab",
			keyName:  "Tab",
			wantSpec: true,
			wantKey:  "Tab",
			wantCode: "Tab",
		},
		{
			name:     "single character a",
			keyName:  "a",
			wantSpec: false,
		},
		{
			name:     "single character Z",
			keyName:  "Z",
			wantSpec: false,
		},
		{
			name:    "unknown multi-char key",
			keyName: "FooBar",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, isSpecial, err := resolveMainKey(tc.keyName)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if isSpecial != tc.wantSpec {
				t.Errorf("isSpecial = %v, expected %v", isSpecial, tc.wantSpec)
			}
			if tc.wantSpec {
				if info.Key != tc.wantKey {
					t.Errorf("Key = %q, expected %q", info.Key, tc.wantKey)
				}
				if info.Code != tc.wantCode {
					t.Errorf("Code = %q, expected %q", info.Code, tc.wantCode)
				}
			}
		})
	}
}

func TestCharacterKeyInfo(t *testing.T) {
	testCases := []struct {
		name        string
		keyName     string
		wantKey     string
		wantCode    string
		wantKeyCode int64
	}{
		{
			name:        "lowercase a",
			keyName:     "a",
			wantKey:     "A",
			wantCode:    "KeyA",
			wantKeyCode: 65,
		},
		{
			name:        "lowercase z",
			keyName:     "z",
			wantKey:     "Z",
			wantCode:    "KeyZ",
			wantKeyCode: 90,
		},
		{
			name:        "uppercase A",
			keyName:     "A",
			wantKey:     "A",
			wantCode:    "KeyA",
			wantKeyCode: 65,
		},
		{
			name:        "digit 1",
			keyName:     "1",
			wantKey:     "1",
			wantCode:    "Key1",
			wantKeyCode: 49,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key, code, keyCode := characterKeyInfo(tc.keyName)
			if key != tc.wantKey {
				t.Errorf("key = %q, expected %q", key, tc.wantKey)
			}
			if code != tc.wantCode {
				t.Errorf("code = %q, expected %q", code, tc.wantCode)
			}
			if keyCode != tc.wantKeyCode {
				t.Errorf("keyCode = %d, expected %d", keyCode, tc.wantKeyCode)
			}
		})
	}
}

func TestCalculateModifiers(t *testing.T) {
	testCases := []struct {
		name     string
		names    []string
		expected input.Modifier
		wantErr  bool
	}{
		{
			name:     "empty modifiers",
			names:    []string{},
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "single Shift",
			names:    []string{"Shift"},
			expected: input.ModifierShift,
			wantErr:  false,
		},
		{
			name:     "Ctrl and Shift combined",
			names:    []string{"Ctrl", "Shift"},
			expected: input.ModifierCtrl | input.ModifierShift,
			wantErr:  false,
		},
		{
			name:     "Cmd alias resolves to Meta",
			names:    []string{"Cmd"},
			expected: input.ModifierMeta,
			wantErr:  false,
		},
		{
			name:     "all four modifiers",
			names:    []string{"Shift", "Control", "Alt", "Meta"},
			expected: input.ModifierShift | input.ModifierCtrl | input.ModifierAlt | input.ModifierMeta,
			wantErr:  false,
		},
		{
			name:    "unknown modifier returns error",
			names:   []string{"Super"},
			wantErr: true,
		},
		{
			name:    "valid then unknown modifier returns error",
			names:   []string{"Shift", "Unknown"},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := calculateModifiers(tc.names)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tc.expected {
				t.Errorf("calculateModifiers(%v) = %v, expected %v", tc.names, result, tc.expected)
			}
		})
	}
}
