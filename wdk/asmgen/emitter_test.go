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

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstruction(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Instruction("MOVQ DX, AX")

	assert.Equal(t, "\tMOVQ DX, AX\n", e.String())
}

func TestInstruction_MultiplePartsJoined(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Instruction("MOVQ", "    DX, AX")

	assert.Equal(t, "\tMOVQ    DX, AX\n", e.String())
}

func TestLabel(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Label("done")

	assert.Equal(t, "done:\n", e.String())
}

func TestComment(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Comment("text")

	assert.Equal(t, "// text\n", e.String())
}

func TestIndentedComment(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.IndentedComment("text")

	assert.Equal(t, "\t// text\n", e.String())
}

func TestBlank(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Blank()

	assert.Equal(t, "\n", e.String())
}

func TestLine(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Line("text")

	assert.Equal(t, "text\n", e.String())
}

func TestRaw(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Raw("raw")

	assert.Equal(t, "raw", e.String())
}

func TestStringAndBytes(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Line("hello")
	e.Line("world")

	expected := "hello\nworld\n"

	assert.Equal(t, expected, e.String())
	assert.Equal(t, []byte(expected), e.Bytes())
}

func TestLen(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	assert.Equal(t, 0, e.Len())

	e.Line("abc")

	assert.Equal(t, 4, e.Len())

	e.Instruction("RET")

	assert.Equal(t, 9, e.Len())
}

func TestReset(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Line("something")
	require.Greater(t, e.Len(), 0)

	e.Reset()
	assert.Equal(t, 0, e.Len())
	assert.Equal(t, "", e.String())
}

func TestAccumulation(t *testing.T) {
	t.Parallel()

	e := NewEmitter()
	e.Comment("header")
	e.Label("start")
	e.Instruction("MOVQ    AX, BX")
	e.Blank()
	e.IndentedComment("return")
	e.Instruction("RET")

	expected := "// header\n" +
		"start:\n" +
		"\tMOVQ    AX, BX\n" +
		"\n" +
		"\t// return\n" +
		"\tRET\n"

	assert.Equal(t, expected, e.String())
}

func TestNewEmitter(t *testing.T) {
	t.Parallel()

	e := NewEmitter()

	assert.Equal(t, 0, e.Len())
	assert.Equal(t, "", e.String())
	assert.Empty(t, e.Bytes())
}
