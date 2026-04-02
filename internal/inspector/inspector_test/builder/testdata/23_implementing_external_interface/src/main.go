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

package main

import "io"

// MyWriter implements the io.Writer interface.
type MyWriter struct{}

var _ io.Writer = (*MyWriter)(nil)

// Write has a specific signature required by io.Writer, using the 'byte' alias.
func (m *MyWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// MyRuneReader implements the io.RuneReader interface.
type MyRuneReader struct{}

var _ io.RuneReader = (*MyRuneReader)(nil)

// ReadRune has a specific signature required by io.RuneReader, using the 'rune' alias.
func (r *MyRuneReader) ReadRune() (character rune, size int, err error) {
	return '⌘', 3, nil
}

// Container uses the 'any' alias, which is equivalent to interface{}.
type Container struct {
	Data any
}
