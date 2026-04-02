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

//go:build safe || (js && wasm)

package interp_domain

// ownsString reports whether s points into the arena's byte slabs.
// The safe build always returns true so callers unconditionally clone
// strings before they escape, preserving correctness without unsafe.
//
// Returns true always in the safe build.
func (*RegisterArena) ownsString(_ string) bool {
	return true
}

// arenaConcatString concatenates a and b. The safe build delegates to
// the standard string concatenation operator.
//
// Takes a (string) which is the left operand.
// Takes b (string) which is the right operand.
//
// Returns the concatenated string.
func arenaConcatString(_ *RegisterArena, a, b string) string {
	return a + b
}

// arenaConcatRuneString appends the rune r to s. The safe build
// delegates to standard string conversion.
//
// Takes s (string) which is the base string.
// Takes r (rune) which is the rune to append.
//
// Returns the resulting string.
func arenaConcatRuneString(_ *RegisterArena, s string, r rune) string {
	return s + string(r)
}

// arenaRuneToString converts rune r to a string. The safe build
// delegates to the standard string conversion.
//
// Takes r (rune) which is the rune to convert.
//
// Returns the single-rune string.
func arenaRuneToString(_ *RegisterArena, r rune) string {
	return string(r)
}

// arenaBytesToString converts byte slice b to a string. The safe
// build delegates to the standard string conversion.
//
// Takes b ([]byte) which is the byte slice to convert.
//
// Returns the resulting string.
func arenaBytesToString(_ *RegisterArena, b []byte) string {
	return string(b)
}
