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
	"strings"
	"testing"

	"piko.sh/piko/wdk/asmgen"

	interp_amd64 "piko.sh/piko/internal/interp/interp_domain/asm/asmgen_arch_amd64"
	interp_arm64 "piko.sh/piko/internal/interp/interp_domain/asm/asmgen_arch_arm64"
)

var exactMatchBaseNames = map[string]bool{
	"asm_vm_dispatch_string": true,
	"asm_vm_dispatch_init":   true,
	"asm_vm_dispatch_inline": true,
}

func TestDeepInstructionComparison(t *testing.T) {
	architectures := []BytecodeArchitecturePort{
		interp_amd64.New(),
		interp_arm64.New(),
	}

	writer := &memWriter{files: make(map[string][]byte)}
	err := asmgen.GenerateFiles(writer, architectures, FileGroups(), HeaderFiles())
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	for generatedPath, generatedData := range writer.files {
		filename := filepath.Base(generatedPath)
		if !strings.HasSuffix(filename, ".s") {
			continue
		}

		originalName := strings.TrimPrefix(filename, "asm_")
		originalPath := filepath.Join("..", originalName)
		originalData, err := os.ReadFile(originalPath)
		if err != nil {
			continue
		}

		oldBlocks := extractTextBlocks(string(originalData))
		newBlocks := extractTextBlocks(string(generatedData))

		requiresExactMatch := false
		for baseName := range exactMatchBaseNames {
			if strings.Contains(filename, strings.TrimPrefix(baseName, "asm_")) {
				requiresExactMatch = true
				break
			}
		}

		for name, oldInstructions := range oldBlocks {
			newInstructions, exists := newBlocks[name]
			if !exists {
				t.Errorf("%s: TEXT block %s missing from generated output", filename, name)
				continue
			}

			oldNormalised := normaliseInstructions(oldInstructions)
			newNormalised := normaliseInstructions(newInstructions)

			if requiresExactMatch {

				if len(oldNormalised) != len(newNormalised) {
					t.Errorf("%s:%s instruction count mismatch: old=%d new=%d",
						filename, name, len(oldNormalised), len(newNormalised))
					continue
				}
				for i := range oldNormalised {
					if oldNormalised[i] != newNormalised[i] {
						t.Errorf("%s:%s instruction %d differs:\n  old: %s\n  new: %s",
							filename, name, i+1, oldNormalised[i], newNormalised[i])
					}
				}
			} else {

				if len(oldNormalised) == len(newNormalised) {
					for i := range oldNormalised {
						if oldNormalised[i] != newNormalised[i] {
							t.Logf("%s:%s instruction %d differs:\n  old: %s\n  new: %s",
								filename, name, i+1, oldNormalised[i], newNormalised[i])
						}
					}
				} else {
					t.Logf("%s:%s instruction count: old=%d new=%d",
						filename, name, len(oldNormalised), len(newNormalised))
				}

				if len(oldNormalised) > 0 {
					difference := math.Abs(float64(len(newNormalised)-len(oldNormalised))) / float64(len(oldNormalised))
					if difference > 0.20 {
						t.Errorf("%s:%s instruction count differs by %.0f%% (old=%d, new=%d), exceeds 20%% tolerance",
							filename, name, difference*100, len(oldNormalised), len(newNormalised))
					}
				}
			}
		}
	}
}

var trailingComment = regexp.MustCompile(`\s+//.*$`)

func extractTextBlocks(content string) map[string][]string {
	blocks := make(map[string][]string)
	var currentName string
	for line := range strings.SplitSeq(content, "\n") {
		if strings.Contains(line, "TEXT") && strings.Contains(line, "(SB)") {
			start := strings.Index(line, "\xc2\xb7")
			if start >= 0 {
				start += 2
				end := strings.Index(line[start:], "(SB)")
				if end >= 0 {
					currentName = line[start : start+end]
					blocks[currentName] = nil
					continue
				}
			}
		}
		if currentName != "" {
			stripped := stripAssemblyLine(line)
			if stripped != "" {
				blocks[currentName] = append(blocks[currentName], stripped)
			}
		}
	}
	return blocks
}

func stripAssemblyLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || trimmed == "" {
		return ""
	}
	stripped := trailingComment.ReplaceAllString(line, "")
	return strings.TrimRight(stripped, " \t")
}

func normaliseInstructions(instructions []string) []string {
	result := make([]string, len(instructions))
	for i, instr := range instructions {
		result[i] = strings.Join(strings.Fields(instr), " ")
	}
	return result
}
