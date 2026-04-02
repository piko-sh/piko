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

package asm

import (
	"strings"
	"testing"

	"piko.sh/piko/wdk/asmgen"

	interp_amd64 "piko.sh/piko/internal/interp/interp_domain/asm/asmgen_arch_amd64"
	interp_arm64 "piko.sh/piko/internal/interp/interp_domain/asm/asmgen_arch_arm64"
)

func TestGenerateProducesNonEmptyOutput(t *testing.T) {
	architectures := []BytecodeArchitecturePort{
		interp_amd64.New(),
		interp_arm64.New(),
	}

	groups := FileGroups()
	writer := &memWriter{files: make(map[string][]byte)}

	err := asmgen.GenerateFiles(writer, architectures, groups, nil)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	if len(writer.files) == 0 {
		t.Fatal("no files generated")
	}

	for path, data := range writer.files {
		if len(data) == 0 {
			t.Errorf("empty file: %s", path)
		}
	}
}

func TestEachFileGroupGeneratesForBothArchitectures(t *testing.T) {
	architectures := []BytecodeArchitecturePort{
		interp_amd64.New(),
		interp_arm64.New(),
	}

	groups := FileGroups()
	writer := &memWriter{files: make(map[string][]byte)}

	err := asmgen.GenerateFiles(writer, architectures, groups, nil)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	for _, group := range groups {
		for _, arch := range architectures {
			filename := group.BaseName + "_" + string(arch.Arch()) + ".s"
			found := false
			for path := range writer.files {
				if strings.HasSuffix(path, filename) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("missing generated file: %s", filename)
			}
		}
	}
}

func TestGeneratedFilesContainTextDirectives(t *testing.T) {
	architectures := []BytecodeArchitecturePort{
		interp_amd64.New(),
		interp_arm64.New(),
	}

	groups := FileGroups()
	writer := &memWriter{files: make(map[string][]byte)}

	err := asmgen.GenerateFiles(writer, architectures, groups, nil)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	for path, data := range writer.files {
		content := string(data)
		if !strings.Contains(content, "TEXT") {
			t.Errorf("file %s contains no TEXT directives", path)
		}
		if !strings.Contains(content, "//go:build") {
			t.Errorf("file %s contains no build constraint", path)
		}
		if !strings.Contains(content, "#include") {
			t.Errorf("file %s contains no #include directives", path)
		}
	}
}

func TestGeneratedArithmeticFileContainsExpectedHandlers(t *testing.T) {
	architectures := []BytecodeArchitecturePort{interp_amd64.New()}
	groups := FileGroups()
	writer := &memWriter{files: make(map[string][]byte)}

	err := asmgen.GenerateFiles(writer, architectures, groups, nil)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	var arithContent string
	for path, data := range writer.files {
		if strings.Contains(path, "vm_dispatch_arith_amd64") {
			arithContent = string(data)
			break
		}
	}

	if arithContent == "" {
		t.Fatal("no vm_dispatch_arith_amd64.s generated")
	}

	expectedHandlers := []string{
		"handlerNop", "handlerMoveInt", "handlerMoveFloat",
		"handlerAddInt", "handlerSubInt", "handlerMulInt",
		"handlerDivInt", "handlerRemInt", "handlerNegInt",
		"handlerIncInt", "handlerDecInt",
		"handlerBitAnd", "handlerBitOr", "handlerBitXor",
		"handlerBitAndNot", "handlerBitNot",
		"handlerShiftLeft", "handlerShiftRight",
		"handlerAddFloat", "handlerSubFloat",
		"handlerMulFloat", "handlerDivFloat", "handlerNegFloat",
	}

	for _, handler := range expectedHandlers {
		if !strings.Contains(arithContent, handler) {
			t.Errorf("missing handler in arith file: %s", handler)
		}
	}
}

func TestGeneratedComparisonFileContainsExpectedHandlers(t *testing.T) {
	architectures := []BytecodeArchitecturePort{interp_amd64.New()}
	groups := FileGroups()
	writer := &memWriter{files: make(map[string][]byte)}

	err := asmgen.GenerateFiles(writer, architectures, groups, nil)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	var cmpContent string
	for path, data := range writer.files {
		if strings.Contains(path, "vm_dispatch_cmp_amd64") {
			cmpContent = string(data)
			break
		}
	}

	if cmpContent == "" {
		t.Fatal("no vm_dispatch_cmp_amd64.s generated")
	}

	expectedHandlers := []string{
		"handlerEqInt", "handlerNeInt", "handlerLtInt",
		"handlerLeInt", "handlerGtInt", "handlerGeInt",
		"handlerEqFloat", "handlerNeFloat",
		"handlerIntToFloat", "handlerFloatToInt",
		"handlerMathSqrt", "handlerMathAbs",
		"handlerNot", "handlerJump",
		"handlerJumpIfTrue", "handlerJumpIfFalse",
	}

	for _, handler := range expectedHandlers {
		if !strings.Contains(cmpContent, handler) {
			t.Errorf("missing handler in cmp file: %s", handler)
		}
	}
}

func TestGenerateOutputSample(t *testing.T) {

	architectures := []BytecodeArchitecturePort{interp_amd64.New()}
	groups := FileGroups()
	writer := &memWriter{files: make(map[string][]byte)}

	err := asmgen.GenerateFiles(writer, architectures, groups, nil)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	for path, data := range writer.files {
		if strings.Contains(path, "vm_dispatch_arith_amd64") {
			lines := strings.Split(string(data), "\n")
			limit := min(50, len(lines))
			t.Logf("=== %s (first %d lines) ===", path, limit)
			for i := range limit {
				t.Logf("%3d: %s", i+1, lines[i])
			}
			break
		}
	}

	arm64Archs := []BytecodeArchitecturePort{interp_arm64.New()}
	arm64Writer := &memWriter{files: make(map[string][]byte)}
	err = asmgen.GenerateFiles(arm64Writer, arm64Archs, groups, nil)
	if err != nil {
		t.Fatalf("arm64 generate error: %v", err)
	}

	for path, data := range arm64Writer.files {
		if strings.Contains(path, "vm_dispatch_arith_arm64") {
			lines := strings.Split(string(data), "\n")
			limit := min(50, len(lines))
			t.Logf("=== %s (first %d lines) ===", path, limit)
			for i := range limit {
				t.Logf("%3d: %s", i+1, lines[i])
			}
			break
		}
	}
}

func TestHeaderFilesGenerate(t *testing.T) {
	architectures := []BytecodeArchitecturePort{
		interp_amd64.New(),
		interp_arm64.New(),
	}

	headers := HeaderFiles()
	writer := &memWriter{files: make(map[string][]byte)}

	err := asmgen.GenerateFiles(writer, architectures, nil, headers)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	expectedFiles := []string{"dispatch_offsets.h", "dispatch_amd64.h", "dispatch_arm64.h"}
	for _, name := range expectedFiles {
		found := false
		for path, data := range writer.files {
			if strings.HasSuffix(path, name) {
				found = true
				if len(data) == 0 {
					t.Errorf("header %s is empty", name)
				}
				content := string(data)
				if !strings.Contains(content, "#define") {
					t.Errorf("header %s contains no #define directives", name)
				}
				t.Logf("%s: %d bytes", name, len(data))
				break
			}
		}
		if !found {
			t.Errorf("missing header file: %s", name)
		}
	}
}

type memWriter struct {
	files map[string][]byte
}

func (w *memWriter) WriteFile(path string, data []byte) error {
	w.files[path] = data
	return nil
}
