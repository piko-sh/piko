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

package mem

// String converts a byte slice to a string.
//
// This is the safe (allocating) version used when building with -tags safe.
// It copies the byte slice contents into a new string.
//
// Takes b ([]byte) which is the byte slice to convert.
//
// Returns string which contains a copy of the byte slice data.
func String(b []byte) string {
	return string(b)
}

// Bytes converts a string to a byte slice.
//
// This is the safe version that creates a new allocation. It is used when
// building with -tags safe. The function copies the string contents into a
// new byte slice.
//
// Takes s (string) which is the string to convert.
//
// Returns []byte which contains a copy of the string data.
func Bytes(s string) []byte {
	return []byte(s)
}
