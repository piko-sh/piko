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

package engine_shared_test

import (
	"testing"

	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
)

func TestIsIdentStart(t *testing.T) {
	t.Parallel()

	for _, character := range []rune{'a', 'z', 'A', 'Z', '_', 'é', '名'} {
		if !engine_shared.IsIdentStart(character) {
			t.Errorf("IsIdentStart(%q) = false, want true", character)
		}
	}
	for _, character := range []rune{'0', '9', ' ', '-', '.', '$'} {
		if engine_shared.IsIdentStart(character) {
			t.Errorf("IsIdentStart(%q) = true, want false", character)
		}
	}
}

func TestIsIdentPart(t *testing.T) {
	t.Parallel()

	for _, character := range []rune{'a', '_', '0', '9', 'é', '名'} {
		if !engine_shared.IsIdentPart(character) {
			t.Errorf("IsIdentPart(%q) = false, want true", character)
		}
	}
	for _, character := range []rune{' ', '-', '.', '$'} {
		if engine_shared.IsIdentPart(character) {
			t.Errorf("IsIdentPart(%q) = true, want false", character)
		}
	}
}

func TestDigitClassifiers(t *testing.T) {
	t.Parallel()

	for _, character := range []byte{'0', '5', '9'} {
		if !engine_shared.IsDigit(character) {
			t.Errorf("IsDigit(%q) = false, want true", character)
		}
	}
	for _, character := range []byte{'0', '9', 'a', 'f', 'A', 'F'} {
		if !engine_shared.IsHexDigit(character) {
			t.Errorf("IsHexDigit(%q) = false, want true", character)
		}
	}
	for _, character := range []byte{'g', 'G', ' '} {
		if engine_shared.IsHexDigit(character) {
			t.Errorf("IsHexDigit(%q) = true, want false", character)
		}
	}
	for _, character := range []byte{'0', '7'} {
		if !engine_shared.IsOctalDigit(character) {
			t.Errorf("IsOctalDigit(%q) = false, want true", character)
		}
	}
	if engine_shared.IsOctalDigit('8') {
		t.Error("IsOctalDigit('8') = true, want false")
	}
	for _, character := range []byte{'0', '1'} {
		if !engine_shared.IsBinaryDigit(character) {
			t.Errorf("IsBinaryDigit(%q) = false, want true", character)
		}
	}
	if engine_shared.IsBinaryDigit('2') {
		t.Error("IsBinaryDigit('2') = true, want false")
	}
}
