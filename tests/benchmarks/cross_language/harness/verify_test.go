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

//go:build crosslang

package harness

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormaliseStdoutCollapsesCRLFToLF(t *testing.T) {
	out := NormaliseStdout([]byte("hello\r\nworld\r\n"))
	assert.Equal(t, "hello\nworld", string(out))
}

func TestNormaliseStdoutStripsSingleTrailingNewline(t *testing.T) {
	out := NormaliseStdout([]byte("hello\nworld\n"))
	assert.Equal(t, "hello\nworld", string(out))
}

func TestNormaliseStdoutKeepsBlankInteriorLines(t *testing.T) {
	out := NormaliseStdout([]byte("hello\n\nworld\n"))
	assert.Equal(t, "hello\n\nworld", string(out))
}

func TestNormaliseStdoutTrimsTrailingWhitespacePerLine(t *testing.T) {
	out := NormaliseStdout([]byte("hello   \t\nworld\n"))
	assert.Equal(t, "hello\nworld", string(out))
}

func TestNormaliseStdoutHandlesEmptyInput(t *testing.T) {
	out := NormaliseStdout(nil)
	assert.Equal(t, "", string(out))
}

func TestSHA256HexIsDeterministic(t *testing.T) {
	hash1 := SHA256Hex([]byte("piko"))
	hash2 := SHA256Hex([]byte("piko"))
	assert.Equal(t, hash1, hash2)
	assert.Len(t, hash1, 64)
}

func TestParseInnerElapsedNanosExtractsValueAndKeepsRemainder(t *testing.T) {
	stderr := []byte("INNER_ELAPSED_NS=12345\nsome other warning\n")
	nanos, remainder := ParseInnerElapsedNanos(stderr)
	assert.Equal(t, int64(12345), nanos)
	assert.Equal(t, "some other warning", remainder)
}

func TestParseInnerElapsedNanosLastWinsOnMultipleMarkers(t *testing.T) {
	stderr := []byte("INNER_ELAPSED_NS=1\nINNER_ELAPSED_NS=999\n")
	nanos, _ := ParseInnerElapsedNanos(stderr)
	assert.Equal(t, int64(999), nanos)
}

func TestParseInnerElapsedNanosReturnsZeroWhenAbsent(t *testing.T) {
	nanos, remainder := ParseInnerElapsedNanos([]byte("just a warning\n"))
	assert.Equal(t, int64(0), nanos)
	assert.Equal(t, "just a warning", remainder)
}

func TestParseInnerElapsedNanosIgnoresMalformedValues(t *testing.T) {
	stderr := []byte("INNER_ELAPSED_NS=abc\nINNER_ELAPSED_NS=42\n")
	nanos, _ := ParseInnerElapsedNanos(stderr)
	assert.Equal(t, int64(42), nanos)
}
