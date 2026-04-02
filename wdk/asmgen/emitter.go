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

package asmgen

import "bytes"

// Emitter writes Plan 9 assembly text to an internal buffer. It
// provides convenience methods for common assembly formatting patterns
// but performs no validation or transformation; callers are responsible
// for producing correctly formatted text.
//
// The emitter intentionally has no formatting intelligence. It writes
// exactly what it is told. This is critical for character-by-character
// matching against existing hand-written assembly files.
type Emitter struct {
	// buf accumulates the emitted assembly text.
	buf bytes.Buffer
}

// NewEmitter creates a new assembly text emitter with an empty buffer.
//
// Returns *Emitter ready for use.
func NewEmitter() *Emitter {
	return &Emitter{}
}

// Instruction emits a tab-indented instruction line. The parts are joined
// with no separator and terminated with a newline.
//
// Example: e.Instruction("MOVQ    DX, AX") produces "\tMOVQ    DX, AX\n".
//
// Takes parts (...string) which are concatenated to form the
// instruction text.
func (e *Emitter) Instruction(parts ...string) {
	_ = e.buf.WriteByte('\t')
	for _, part := range parts {
		_, _ = e.buf.WriteString(part)
	}
	_ = e.buf.WriteByte('\n')
}

// Label emits a label on its own line, followed by a newline.
//
// Example: e.Label("dn") produces "dn:\n".
//
// Takes name (string) which is the label identifier.
func (e *Emitter) Label(name string) {
	_, _ = e.buf.WriteString(name)
	_, _ = e.buf.WriteString(":\n")
}

// Comment emits a non-indented comment line.
//
// Example: e.Comment("Data movement handlers.") produces
// "// Data movement handlers.\n".
//
// Takes text (string) which is the comment body.
func (e *Emitter) Comment(text string) {
	_, _ = e.buf.WriteString("// ")
	_, _ = e.buf.WriteString(text)
	_ = e.buf.WriteByte('\n')
}

// IndentedComment emits a tab-indented comment line.
//
// Example: e.IndentedComment("A = dest") produces "\t// A = dest\n".
//
// Takes text (string) which is the comment body.
func (e *Emitter) IndentedComment(text string) {
	_, _ = e.buf.WriteString("\t// ")
	_, _ = e.buf.WriteString(text)
	_ = e.buf.WriteByte('\n')
}

// Blank emits a single blank line.
func (e *Emitter) Blank() {
	_ = e.buf.WriteByte('\n')
}

// Line emits a raw line of text followed by a newline.
//
// Takes text (string) which is written verbatim before the newline.
func (e *Emitter) Line(text string) {
	_, _ = e.buf.WriteString(text)
	_ = e.buf.WriteByte('\n')
}

// Raw writes text to the buffer with no additional formatting.
//
// Takes s (string) which is written verbatim.
func (e *Emitter) Raw(s string) {
	_, _ = e.buf.WriteString(s)
}

// Bytes returns the accumulated assembly text as a byte slice.
//
// Returns []byte containing all emitted text.
func (e *Emitter) Bytes() []byte {
	return e.buf.Bytes()
}

// String returns the accumulated assembly text as a string.
//
// Returns string containing all emitted text.
func (e *Emitter) String() string {
	return e.buf.String()
}

// Len returns the number of bytes written so far.
//
// Returns int which is the current buffer length.
func (e *Emitter) Len() int {
	return e.buf.Len()
}

// Reset clears the buffer for reuse.
func (e *Emitter) Reset() {
	e.buf.Reset()
}
