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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testArch struct {
	architecture Architecture
	constraint   string
	header       string
}

func (a *testArch) Arch() Architecture                { return a.architecture }
func (a *testArch) BuildConstraint() string           { return a.constraint }
func (a *testArch) ArchitectureHeaderInclude() string { return a.header }

func newAMD64Arch() *testArch {
	return &testArch{
		architecture: ArchitectureAMD64,
		constraint:   " && amd64",
		header:       "test.h",
	}
}

func newARM64Arch() *testArch {
	return &testArch{
		architecture: ArchitectureARM64,
		constraint:   " && arm64",
		header:       "test_arm64.h",
	}
}

type memWriter struct {
	files map[string][]byte
}

func newMemWriter() *memWriter {
	return &memWriter{files: make(map[string][]byte)}
}

func (w *memWriter) WriteFile(path string, data []byte) error {
	w.files[path] = data
	return nil
}

type errWriter struct {
	err error
}

func (w *errWriter) WriteFile(_ string, _ []byte) error {
	return w.err
}

func minimalGroup(baseName string) FileGroup[*testArch] {
	return FileGroup[*testArch]{
		BaseName:        baseName,
		OutputDir:       "out",
		BuildConstraint: "!safe",
		Includes:        []string{"textflag.h"},
		Handlers: []HandlerDefinition[*testArch]{
			{
				Name:    "noop",
				Comment: "noop does nothing.",
				Emit:    func(e *Emitter, _ *testArch) { e.Instruction("RET") },
			},
		},
	}
}

func TestGenerateFiles_ProducesCorrectFilename(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := minimalGroup("foo")

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil)
	require.NoError(t, err)

	expectedPath := filepath.Join("out", "foo_amd64.s")
	_, exists := writer.files[expectedPath]
	assert.True(t, exists, "expected file %s to be written", expectedPath)
}

func TestGenerateFiles_LicenseHeaderPresent(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := minimalGroup("foo")

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil)
	require.NoError(t, err)

	content := string(writer.files[filepath.Join("out", "foo_amd64.s")])
	assert.True(t, strings.HasPrefix(content, "// Code generated"), "output should start with licence header")
}

func TestGenerateFiles_BuildConstraintPresent(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := minimalGroup("foo")

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil)
	require.NoError(t, err)

	content := string(writer.files[filepath.Join("out", "foo_amd64.s")])
	assert.Contains(t, content, "//go:build !safe && amd64")
}

func TestGenerateFiles_IncludesPresent(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := minimalGroup("foo")
	group.Includes = []string{"textflag.h", "offsets.h"}

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil)
	require.NoError(t, err)

	content := string(writer.files[filepath.Join("out", "foo_amd64.s")])
	assert.Contains(t, content, "#include \"textflag.h\"")
	assert.Contains(t, content, "#include \"offsets.h\"")
}

func TestGenerateFiles_HeaderComment(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := minimalGroup("foo")
	group.HeaderComment = "Arithmetic handlers for the dispatch loop."

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil)
	require.NoError(t, err)

	content := string(writer.files[filepath.Join("out", "foo_amd64.s")])
	assert.Contains(t, content, "// Arithmetic handlers for the dispatch loop.")
}

func TestGenerateFiles_HeaderCommentFunction(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := minimalGroup("foo")
	group.HeaderComment = "should be overridden"
	group.HeaderCommentFunction = func(arch Architecture) string {
		return "dynamic header for " + string(arch)
	}

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil)
	require.NoError(t, err)

	content := string(writer.files[filepath.Join("out", "foo_amd64.s")])
	assert.Contains(t, content, "// dynamic header for amd64")
	assert.NotContains(t, content, "should be overridden")
}

func TestGenerateFiles_HandlerEmitted(t *testing.T) {
	t.Parallel()

	emitCalled := false
	writer := newMemWriter()
	group := FileGroup[*testArch]{
		BaseName:        "bar",
		OutputDir:       "out",
		BuildConstraint: "!safe",
		Includes:        []string{"textflag.h"},
		Handlers: []HandlerDefinition[*testArch]{
			{
				Name:    "myHandler",
				Comment: "myHandler does a thing.",
				Emit: func(e *Emitter, _ *testArch) {
					emitCalled = true
					e.Instruction("MOVQ    AX, BX")
					e.Instruction("RET")
				},
			},
		},
	}

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil)
	require.NoError(t, err)

	assert.True(t, emitCalled, "handler Emit function should have been called")

	content := string(writer.files[filepath.Join("out", "bar_amd64.s")])
	assert.Contains(t, content, "// myHandler does a thing.")
	assert.Contains(t, content, "\tMOVQ    AX, BX\n")
	assert.Contains(t, content, "\tRET\n")
}

func TestGenerateFiles_TextDirectiveFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		handler  HandlerDefinition[*testArch]
		expected string
	}{
		{
			name: "with flags and frame size",
			handler: HandlerDefinition[*testArch]{
				Name:      "myFunc",
				Comment:   "myFunc comment.",
				Flags:     "NOSPLIT",
				FrameSize: "$0-8",
				Emit:      func(e *Emitter, _ *testArch) { e.Instruction("RET") },
			},

			expected: "TEXT \xc2\xb7myFunc(SB), NOSPLIT, $0-8",
		},
		{
			name: "no flags default frame size",
			handler: HandlerDefinition[*testArch]{
				Name:    "simple",
				Comment: "simple comment.",
				Emit:    func(e *Emitter, _ *testArch) { e.Instruction("RET") },
			},
			expected: "TEXT \xc2\xb7simple(SB), $0",
		},
		{
			name: "no flags custom frame size",
			handler: HandlerDefinition[*testArch]{
				Name:      "withFrame",
				Comment:   "withFrame comment.",
				FrameSize: "$16-24",
				Emit:      func(e *Emitter, _ *testArch) { e.Instruction("RET") },
			},
			expected: "TEXT \xc2\xb7withFrame(SB), $16-24",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			writer := newMemWriter()
			group := FileGroup[*testArch]{
				BaseName:        "txt",
				OutputDir:       "out",
				BuildConstraint: "!safe",
				Includes:        []string{"textflag.h"},
				Handlers:        []HandlerDefinition[*testArch]{tc.handler},
			}

			err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil)
			require.NoError(t, err)

			content := string(writer.files[filepath.Join("out", "txt_amd64.s")])
			assert.Contains(t, content, tc.expected)
		})
	}
}

func TestGenerateFiles_ArchitectureFilter(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := FileGroup[*testArch]{
		BaseName:        "filtered",
		OutputDir:       "out",
		BuildConstraint: "!safe",
		Includes:        []string{"textflag.h"},
		Handlers: []HandlerDefinition[*testArch]{
			{
				Name:          "arm64Only",
				Comment:       "arm64Only is ARM64 exclusive.",
				Architectures: []Architecture{ArchitectureARM64},
				Emit: func(e *Emitter, _ *testArch) {
					e.Instruction("RET")
				},
			},
			{
				Name:    "universal",
				Comment: "universal runs everywhere.",
				Emit: func(e *Emitter, _ *testArch) {
					e.Instruction("RET")
				},
			},
		},
	}

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil)
	require.NoError(t, err)

	content := string(writer.files[filepath.Join("out", "filtered_amd64.s")])
	assert.NotContains(t, content, "arm64Only")
	assert.Contains(t, content, "universal")
}

func TestGenerateFiles_CommentFunction(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := FileGroup[*testArch]{
		BaseName:        "cf",
		OutputDir:       "out",
		BuildConstraint: "!safe",
		Includes:        []string{"textflag.h"},
		Handlers: []HandlerDefinition[*testArch]{
			{
				Name:    "dynComment",
				Comment: "should be overridden",
				CommentFunction: func(arch Architecture) string {
					return "generated for " + string(arch)
				},
				Emit: func(e *Emitter, _ *testArch) { e.Instruction("RET") },
			},
		},
	}

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil)
	require.NoError(t, err)

	content := string(writer.files[filepath.Join("out", "cf_amd64.s")])
	assert.Contains(t, content, "// generated for amd64")
	assert.NotContains(t, content, "should be overridden")
}

func TestGenerateFiles_BlankLinesBetweenHandlers(t *testing.T) {
	t.Parallel()

	writer := newMemWriter()
	group := FileGroup[*testArch]{
		BaseName:        "multi",
		OutputDir:       "out",
		BuildConstraint: "!safe",
		Includes:        []string{"textflag.h"},
		Handlers: []HandlerDefinition[*testArch]{
			{
				Name:    "first",
				Comment: "first handler.",
				Emit:    func(e *Emitter, _ *testArch) { e.Instruction("RET") },
			},
			{
				Name:    "second",
				Comment: "second handler.",
				Emit:    func(e *Emitter, _ *testArch) { e.Instruction("RET") },
			},
		},
	}

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil)
	require.NoError(t, err)

	content := string(writer.files[filepath.Join("out", "multi_amd64.s")])

	assert.Contains(t, content, "\tRET\n\n// second handler.")
}

func TestGenerateFiles_HeaderFileGenerated(t *testing.T) {
	t.Parallel()

	emitCalled := false
	writer := newMemWriter()

	header := HeaderFile{
		Name: "constants.h",
		Dir:  "out",
		Emit: func(archs []ArchitecturePort) string {
			emitCalled = true
			return "// generated header\n#define FOO 42\n"
		},
	}

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, nil, []HeaderFile{header})
	require.NoError(t, err)

	assert.True(t, emitCalled, "HeaderFile Emit should have been called")

	content := string(writer.files[filepath.Join("out", "constants.h")])
	assert.Contains(t, content, "#define FOO 42")
}

func TestGenerateFiles_WriterErrorPropagates(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("disk full")
	writer := &errWriter{err: expectedError}
	group := minimalGroup("fail")

	err := GenerateFiles(writer, []*testArch{newAMD64Arch()}, []FileGroup[*testArch]{group}, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedError)
}

func TestGenerateAndValidate_NoMismatches(t *testing.T) {
	t.Parallel()

	temporaryDirectory := t.TempDir()

	arch := newAMD64Arch()
	group := minimalGroup("check")
	group.OutputDir = temporaryDirectory

	diskWriter := NewDiskWriter()
	err := GenerateFiles(diskWriter, []*testArch{arch}, []FileGroup[*testArch]{group}, nil)
	require.NoError(t, err)

	mismatches, err := GenerateAndValidate([]*testArch{arch}, []FileGroup[*testArch]{group}, nil)
	require.NoError(t, err)
	assert.Empty(t, mismatches)
}

func TestGenerateAndValidate_DetectsDifference(t *testing.T) {
	t.Parallel()

	temporaryDirectory := t.TempDir()

	arch := newAMD64Arch()
	group := minimalGroup("drift")
	group.OutputDir = temporaryDirectory

	diskWriter := NewDiskWriter()
	err := GenerateFiles(diskWriter, []*testArch{arch}, []FileGroup[*testArch]{group}, nil)
	require.NoError(t, err)

	filePath := filepath.Join(temporaryDirectory, "drift_amd64.s")
	original, err := os.ReadFile(filePath)
	require.NoError(t, err)

	tampered := strings.Replace(string(original), "noop does nothing.", "TAMPERED LINE", 1)
	require.NotEqual(t, string(original), tampered, "replacement should have changed the content")

	err = os.WriteFile(filePath, []byte(tampered), 0o600)
	require.NoError(t, err)

	mismatches, err := GenerateAndValidate([]*testArch{arch}, []FileGroup[*testArch]{group}, nil)
	require.NoError(t, err)
	require.Len(t, mismatches, 1)

	mismatch := mismatches[0]
	assert.Equal(t, filePath, mismatch.File)
	assert.Greater(t, mismatch.Line, 0, "line number should be positive")
	assert.Contains(t, mismatch.Expected, "TAMPERED LINE")
	assert.Contains(t, mismatch.Actual, "noop does nothing.")
}
