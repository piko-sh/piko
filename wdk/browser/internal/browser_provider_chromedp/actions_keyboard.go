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
	"context"
	"fmt"
	"strings"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
)

// keyInfo holds the key name, code, and key code for keyboard events.
type keyInfo struct {
	// Key is the key value string for keyboard input events.
	Key string

	// Code is the physical key code for keyboard event dispatch.
	Code string

	// KeyCode is the virtual key code for Windows and native platforms.
	KeyCode int64
}

var (
	// keyMap maps key names to their chromedp key codes.
	keyMap = map[string]keyInfo{
		"Enter":      {Key: "Enter", Code: "Enter", KeyCode: 13},
		"Tab":        {Key: "Tab", Code: "Tab", KeyCode: 9},
		"Escape":     {Key: "Escape", Code: "Escape", KeyCode: 27},
		"Backspace":  {Key: "Backspace", Code: "Backspace", KeyCode: 8},
		"Delete":     {Key: "Delete", Code: "Delete", KeyCode: 46},
		"ArrowUp":    {Key: "ArrowUp", Code: "ArrowUp", KeyCode: 38},
		"ArrowDown":  {Key: "ArrowDown", Code: "ArrowDown", KeyCode: 40},
		"ArrowLeft":  {Key: "ArrowLeft", Code: "ArrowLeft", KeyCode: 37},
		"ArrowRight": {Key: "ArrowRight", Code: "ArrowRight", KeyCode: 39},
		"Home":       {Key: "Home", Code: "Home", KeyCode: 36},
		"End":        {Key: "End", Code: "End", KeyCode: 35},
		"PageUp":     {Key: "PageUp", Code: "PageUp", KeyCode: 33},
		"PageDown":   {Key: "PageDown", Code: "PageDown", KeyCode: 34},
		"Space":      {Key: " ", Code: "Space", KeyCode: 32},
		"Insert":     {Key: "Insert", Code: "Insert", KeyCode: 45},
		"F1":         {Key: "F1", Code: "F1", KeyCode: 112},
		"F2":         {Key: "F2", Code: "F2", KeyCode: 113},
		"F3":         {Key: "F3", Code: "F3", KeyCode: 114},
		"F4":         {Key: "F4", Code: "F4", KeyCode: 115},
		"F5":         {Key: "F5", Code: "F5", KeyCode: 116},
		"F6":         {Key: "F6", Code: "F6", KeyCode: 117},
		"F7":         {Key: "F7", Code: "F7", KeyCode: 118},
		"F8":         {Key: "F8", Code: "F8", KeyCode: 119},
		"F9":         {Key: "F9", Code: "F9", KeyCode: 120},
		"F10":        {Key: "F10", Code: "F10", KeyCode: 121},
		"F11":        {Key: "F11", Code: "F11", KeyCode: 122},
		"F12":        {Key: "F12", Code: "F12", KeyCode: 123},
	}

	// modifierMap links modifier names to their key details.
	modifierMap = map[string]modifierInfo{
		"Shift":   {Key: "Shift", Code: "ShiftLeft", KeyCode: 16, Modifier: input.ModifierShift},
		"Control": {Key: "Control", Code: "ControlLeft", KeyCode: 17, Modifier: input.ModifierCtrl},
		"Ctrl":    {Key: "Control", Code: "ControlLeft", KeyCode: 17, Modifier: input.ModifierCtrl},
		"Alt":     {Key: "Alt", Code: "AltLeft", KeyCode: 18, Modifier: input.ModifierAlt},
		"Meta":    {Key: "Meta", Code: "MetaLeft", KeyCode: 91, Modifier: input.ModifierMeta},
		"Cmd":     {Key: "Meta", Code: "MetaLeft", KeyCode: 91, Modifier: input.ModifierMeta},
	}
)

// modifierInfo holds details about a keyboard modifier key.
type modifierInfo struct {
	// Key is the key property value sent with keyboard events.
	Key string

	// Code is the keyboard code for CDP key events.
	Code string

	// KeyCode is the virtual key code used for Windows and native key events.
	KeyCode int64

	// Modifier is the bitmask value for this keyboard modifier.
	Modifier input.Modifier
}

// Press presses one or more keys in sequence, supporting modifiers.
// Key format: "Enter", "Tab", "Shift+Enter", "Control+b", "Meta+k".
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes keys (...string) which specifies the key codes to press in order.
//
// Returns error when a key press fails.
func Press(ctx *ActionContext, keys ...string) error {
	for _, keySpec := range keys {
		if err := pressKeySpec(ctx.Ctx, keySpec); err != nil {
			return err
		}
	}
	return nil
}

// Type types text character by character, dispatching keyboard events for
// each character.
//
// Takes ctx (*ActionContext) which provides the browser context for keyboard
// events.
// Takes text (string) which is the text to type character by character.
//
// Returns error when a keyboard event fails to dispatch.
func Type(ctx *ActionContext, text string) error {
	for _, char := range text {
		if err := pressCharacterKey(ctx.Ctx, string(char), 0); err != nil {
			return fmt.Errorf("typing character %c: %w", char, err)
		}
	}
	return nil
}

// KeyDown holds a key down for modifier combinations or hold scenarios.
//
// Takes ctx (*ActionContext) which provides the action execution context.
// Takes key (string) which specifies the key to hold down.
//
// Returns error when the key state dispatch fails.
func KeyDown(ctx *ActionContext, key string) error {
	return dispatchKeyState(ctx.Ctx, key, input.KeyDown)
}

// KeyUp releases a key that was held down with KeyDown.
//
// Takes ctx (*ActionContext) which provides the action execution context.
// Takes key (string) which specifies the key to release.
//
// Returns error when the key state cannot be dispatched.
func KeyUp(ctx *ActionContext, key string) error {
	return dispatchKeyState(ctx.Ctx, key, input.KeyUp)
}

// pressKeySpec presses a single key specification (e.g., "Shift+Enter").
//
// Takes keySpec (string) which specifies the key combination to press.
//
// Returns error when the modifier names are invalid or key pressing fails.
func pressKeySpec(ctx context.Context, keySpec string) error {
	parts := strings.Split(keySpec, "+")
	modifierNames := parts[:len(parts)-1]
	mainKeyName := parts[len(parts)-1]

	modifiers, err := calculateModifiers(modifierNames)
	if err != nil {
		return err
	}

	if err := pressModifierKeys(ctx, modifierNames); err != nil {
		return err
	}

	defer releaseModifiers(ctx, modifierNames)

	return pressMainKey(ctx, mainKeyName, modifiers)
}

// calculateModifiers calculates the combined modifier flags from modifier names.
//
// Takes modifierNames ([]string) which contains the names of modifiers to
// combine.
//
// Returns input.Modifier which is the bitwise OR of all specified modifiers.
// Returns error when an unknown modifier name is provided.
func calculateModifiers(modifierNames []string) (input.Modifier, error) {
	var modifiers input.Modifier
	for _, modName := range modifierNames {
		mod, ok := modifierMap[modName]
		if !ok {
			return 0, fmt.Errorf("unknown modifier: %s", modName)
		}
		modifiers |= mod.Modifier
	}
	return modifiers, nil
}

// pressModifierKeys presses down all modifier keys.
//
// Takes modifierNames ([]string) which specifies the modifier keys to press.
//
// Returns error when a modifier key cannot be pressed.
func pressModifierKeys(ctx context.Context, modifierNames []string) error {
	for _, modName := range modifierNames {
		mod := modifierMap[modName]
		err := chromedp.Run(ctx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				return input.DispatchKeyEvent(input.KeyDown).
					WithKey(mod.Key).
					WithCode(mod.Code).
					WithWindowsVirtualKeyCode(mod.KeyCode).
					WithNativeVirtualKeyCode(mod.KeyCode).
					Do(ctx)
			}),
		)
		if err != nil {
			return fmt.Errorf("pressing modifier %s: %w", modName, err)
		}
	}
	return nil
}

// pressMainKey presses and releases the main key with modifiers.
//
// Takes keyName (string) which specifies the key to press.
// Takes modifiers (input.Modifier) which specifies any modifier keys to hold.
//
// Returns error when the key name is unknown.
func pressMainKey(ctx context.Context, keyName string, modifiers input.Modifier) error {
	key, isSpecial, err := resolveMainKey(keyName)
	if err != nil {
		return err
	}
	if isSpecial {
		return pressSpecialKey(ctx, keyName, key, modifiers)
	}
	return pressCharacterKey(ctx, keyName, modifiers)
}

// resolveMainKey resolves a key name to its keyInfo.
//
// Takes keyName (string) which specifies the key to look up.
//
// Returns keyInfo which contains the resolved key data.
// Returns bool which is true if the key is a special key from keyMap.
// Returns error when the key name is unknown (not in keyMap and not a single
// character).
func resolveMainKey(keyName string) (keyInfo, bool, error) {
	if key, ok := keyMap[keyName]; ok {
		return key, true, nil
	}
	if len(keyName) == 1 {
		return keyInfo{}, false, nil
	}
	return keyInfo{}, false, fmt.Errorf("unknown key: %s", keyName)
}

// pressSpecialKey handles pressing a special key from keyMap.
//
// Takes keyName (string) which identifies the key for error messages.
// Takes key (keyInfo) which contains the key code and related metadata.
// Takes modifiers (input.Modifier) which specifies any modifier keys to apply.
//
// Returns error when the key press or release fails.
func pressSpecialKey(ctx context.Context, keyName string, key keyInfo, modifiers input.Modifier) error {
	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			if err := input.DispatchKeyEvent(input.KeyDown).
				WithKey(key.Key).
				WithCode(key.Code).
				WithWindowsVirtualKeyCode(key.KeyCode).
				WithNativeVirtualKeyCode(key.KeyCode).
				WithModifiers(modifiers).
				Do(ctx); err != nil {
				return err
			}
			return input.DispatchKeyEvent(input.KeyUp).
				WithKey(key.Key).
				WithCode(key.Code).
				WithWindowsVirtualKeyCode(key.KeyCode).
				WithNativeVirtualKeyCode(key.KeyCode).
				WithModifiers(modifiers).
				Do(ctx)
		}),
	)
	if err != nil {
		return fmt.Errorf("pressing key %s: %w", keyName, err)
	}
	return nil
}

// pressCharacterKey handles pressing a single character key.
//
// Takes keyName (string) which specifies the character to press.
// Takes modifiers (input.Modifier) which specifies any modifier keys to hold.
//
// Returns error when the key event cannot be dispatched.
func pressCharacterKey(ctx context.Context, keyName string, modifiers input.Modifier) error {
	key, code, keyCode := characterKeyInfo(keyName)

	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			if err := input.DispatchKeyEvent(input.KeyDown).
				WithKey(key).
				WithCode(code).
				WithWindowsVirtualKeyCode(keyCode).
				WithNativeVirtualKeyCode(keyCode).
				WithModifiers(modifiers).
				Do(ctx); err != nil {
				return err
			}
			if err := input.DispatchKeyEvent(input.KeyChar).
				WithText(keyName).
				WithModifiers(modifiers).
				Do(ctx); err != nil {
				return err
			}
			return input.DispatchKeyEvent(input.KeyUp).
				WithKey(key).
				WithCode(code).
				WithWindowsVirtualKeyCode(keyCode).
				WithNativeVirtualKeyCode(keyCode).
				WithModifiers(modifiers).
				Do(ctx)
		}),
	)
	if err != nil {
		return fmt.Errorf("typing character %s: %w", keyName, err)
	}
	return nil
}

// characterKeyInfo computes the key metadata for a single character.
//
// Takes keyName (string) which specifies a single character key name.
//
// Returns key (string) which is the uppercase key name.
// Returns code (string) which is the key code in "KeyX" format.
// Returns keyCode (int64) which is the virtual key code.
func characterKeyInfo(keyName string) (key, code string, keyCode int64) {
	char := keyName[0]
	key = strings.ToUpper(keyName)
	code = "Key" + key
	keyCode = int64(char)
	if char >= 'a' && char <= 'z' {
		keyCode = int64(char - 'a' + 'A')
	}
	return key, code, keyCode
}

// releaseModifiers releases modifier keys in reverse order.
//
// Takes modifierNames ([]string) which lists the modifier keys to release.
func releaseModifiers(ctx context.Context, modifierNames []string) {
	for i := len(modifierNames) - 1; i >= 0; i-- {
		mod := modifierMap[modifierNames[i]]
		_ = chromedp.Run(ctx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				return input.DispatchKeyEvent(input.KeyUp).
					WithKey(mod.Key).
					WithCode(mod.Code).
					WithWindowsVirtualKeyCode(mod.KeyCode).
					WithNativeVirtualKeyCode(mod.KeyCode).
					Do(ctx)
			}),
		)
	}
}

// dispatchKeyState dispatches a keyboard event for a key.
//
// Takes key (string) which specifies the key name to dispatch.
// Takes eventType (input.KeyType) which specifies the event type.
//
// Returns error when the key is not recognised.
func dispatchKeyState(ctx context.Context, key string, eventType input.KeyType) error {
	if mod, ok := modifierMap[key]; ok {
		return chromedp.Run(ctx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				return input.DispatchKeyEvent(eventType).
					WithKey(mod.Key).
					WithCode(mod.Code).
					WithWindowsVirtualKeyCode(mod.KeyCode).
					WithNativeVirtualKeyCode(mod.KeyCode).
					Do(ctx)
			}),
		)
	}

	if k, ok := keyMap[key]; ok {
		return chromedp.Run(ctx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				return input.DispatchKeyEvent(eventType).
					WithKey(k.Key).
					WithCode(k.Code).
					WithWindowsVirtualKeyCode(k.KeyCode).
					WithNativeVirtualKeyCode(k.KeyCode).
					Do(ctx)
			}),
		)
	}

	if len(key) == 1 {
		char := key[0]
		keyName := strings.ToUpper(key)
		code := "Key" + keyName
		keyCode := int64(char)
		if char >= 'a' && char <= 'z' {
			keyCode = int64(char - 'a' + 'A')
		}

		return chromedp.Run(ctx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				return input.DispatchKeyEvent(eventType).
					WithKey(keyName).
					WithCode(code).
					WithWindowsVirtualKeyCode(keyCode).
					WithNativeVirtualKeyCode(keyCode).
					Do(ctx)
			}),
		)
	}

	return fmt.Errorf("unknown key: %s", key)
}
