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
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"piko.sh/piko/wdk/asmgen"

	interp_amd64 "piko.sh/piko/internal/interp/interp_domain/asm/asmgen_arch_amd64"
	interp_arm64 "piko.sh/piko/internal/interp/interp_domain/asm/asmgen_arch_arm64"
)

func TestInstructionParityWithOriginals(t *testing.T) {
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

	original_dir := ".."

	for generated_path, generated_data := range writer.files {
		filename := filepath.Base(generated_path)

		original_path := filepath.Join(original_dir, filename)
		original_data, err := os.ReadFile(original_path)
		if err != nil {
			t.Errorf("original file not found for generated file %s", filename)
			continue
		}

		generated_instructions := extractInstructions(string(generated_data))
		original_instructions := extractInstructions(string(original_data))

		generated_texts := extractTextNames(generated_instructions)
		original_texts := extractTextNames(original_instructions)

		for _, name := range original_texts {
			if !slices.Contains(generated_texts, name) {
				t.Errorf("%s: original TEXT block %q missing from generated output", filename, name)
			}
		}

		for _, name := range generated_texts {
			if !slices.Contains(original_texts, name) {
				t.Errorf("%s: generated output has extra TEXT block %q not present in original", filename, name)
			}
		}

		t.Logf("%s: original=%d instructions, generated=%d instructions, original TEXT blocks=%d, generated TEXT blocks=%d",
			filename, len(original_instructions), len(generated_instructions), len(original_texts), len(generated_texts))

		if len(original_instructions) > 0 {
			difference := math.Abs(float64(len(generated_instructions)-len(original_instructions))) / float64(len(original_instructions))
			if difference > 0.20 {
				t.Errorf("%s: instruction count differs by %.0f%% (original=%d, generated=%d), exceeds 20%% tolerance",
					filename, difference*100, len(original_instructions), len(generated_instructions))
			}
		}
	}
}

func extractInstructions(content string) []string {
	var result []string
	for line := range strings.SplitSeq(content, "\n") {
		stripped := stripInlineComment(line)
		trimmed := strings.TrimSpace(stripped)

		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		result = append(result, stripped)
	}
	return result
}

var inlineCommentPattern = regexp.MustCompile(`\s+//.*$`)

func stripInlineComment(line string) string {

	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
		return line
	}
	return inlineCommentPattern.ReplaceAllString(line, "")
}

func extractTextNames(lines []string) []string {
	var names []string
	for _, line := range lines {
		if strings.Contains(line, "TEXT") && strings.Contains(line, "(SB)") {

			start := strings.Index(line, "\xc2\xb7")
			if start < 0 {
				continue
			}
			start += 2
			end := strings.Index(line[start:], "(SB)")
			if end < 0 {
				continue
			}
			names = append(names, line[start:start+end])
		}
	}
	return names
}
