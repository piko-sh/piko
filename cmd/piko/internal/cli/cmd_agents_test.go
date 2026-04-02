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

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"piko.sh/piko/cmd/piko/internal/wizard/templates"
	"piko.sh/piko/wdk/safedisk"
)

func TestCopyClaudeCodeSkill_WritesCorrectFiles(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	if err := templates.CopyClaudeCodeSkill(directory); err != nil {
		t.Fatalf("CopyClaudeCodeSkill() = %v", err)
	}

	if _, err := os.Stat(filepath.Join(directory, "SKILL.md")); err != nil {
		t.Error("SKILL.md not created")
	}

	if _, err := os.Stat(filepath.Join(directory, "references", "pk-file-format.md")); err != nil {
		t.Error("references/pk-file-format.md not created")
	}
	if _, err := os.Stat(filepath.Join(directory, "references", "template-syntax.md")); err != nil {
		t.Error("references/template-syntax.md not created")
	}

	if _, err := os.Stat(filepath.Join(directory, "AGENTS.md")); err == nil {
		t.Error("AGENTS.md should not be written by CopyClaudeCodeSkill")
	}

	if _, err := os.Stat(filepath.Join(directory, ".claude-plugin")); err == nil {
		t.Error(".claude-plugin/ should not be written by CopyClaudeCodeSkill")
	}

	if _, err := os.Stat(filepath.Join(directory, ".lsp.json")); err == nil {
		t.Error(".lsp.json should not be written by CopyClaudeCodeSkill")
	}
}

func TestCopyProjectAgents_WritesCorrectFiles(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	if err := templates.CopyProjectAgents(directory); err != nil {
		t.Fatalf("CopyProjectAgents() = %v", err)
	}

	if _, err := os.Stat(filepath.Join(directory, "AGENTS.md")); err != nil {
		t.Error("AGENTS.md not created")
	}

	if _, err := os.Stat(filepath.Join(directory, "references", "pk-file-format.md")); err != nil {
		t.Error("references/pk-file-format.md not created")
	}

	if _, err := os.Stat(filepath.Join(directory, "SKILL.md")); err == nil {
		t.Error("SKILL.md should not be written by CopyProjectAgents")
	}

	if _, err := os.Stat(filepath.Join(directory, ".claude-plugin")); err == nil {
		t.Error(".claude-plugin/ should not be written by CopyProjectAgents")
	}
}

func TestCopyAgentFiles_SkipsGoFiles(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	if err := templates.CopyClaudeCodeSkill(directory); err != nil {
		t.Fatalf("CopyClaudeCodeSkill() = %v", err)
	}

	if _, err := os.Stat(filepath.Join(directory, "embed.go")); err == nil {
		t.Error("embed.go should not be written to the destination")
	}
}

func TestCopyClaudeCodeSkill_OverwritesExisting(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()

	skillPath := filepath.Join(directory, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte("old content"), 0640); err != nil {
		t.Fatal(err)
	}

	if err := templates.CopyClaudeCodeSkill(directory); err != nil {
		t.Fatalf("CopyClaudeCodeSkill() = %v", err)
	}

	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) == "old content" {
		t.Error("SKILL.md should have been overwritten but still contains old content")
	}
}

func TestCopyClaudeCodeSkill_RemovesStaleFiles(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()

	refsDir := filepath.Join(directory, "references")
	if err := os.MkdirAll(refsDir, 0750); err != nil {
		t.Fatal(err)
	}
	stalePath := filepath.Join(refsDir, "old-removed-reference.md")
	if err := os.WriteFile(stalePath, []byte("stale"), 0640); err != nil {
		t.Fatal(err)
	}

	if err := templates.CopyClaudeCodeSkill(directory); err != nil {
		t.Fatalf("CopyClaudeCodeSkill() = %v", err)
	}

	if _, err := os.Stat(stalePath); err == nil {
		t.Error("stale file should have been removed but still exists")
	}
}

func TestCopyProjectAgents_OverwritesExisting(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()

	agentsPath := filepath.Join(directory, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("old content"), 0640); err != nil {
		t.Fatal(err)
	}

	if err := templates.CopyProjectAgents(directory); err != nil {
		t.Fatalf("CopyProjectAgents() = %v", err)
	}

	content, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) == "old content" {
		t.Error("AGENTS.md should have been overwritten but still contains old content")
	}
}

func TestCopyProjectAgents_RemovesStaleReferences(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()

	refsDir := filepath.Join(directory, "references")
	if err := os.MkdirAll(refsDir, 0750); err != nil {
		t.Fatal(err)
	}
	stalePath := filepath.Join(refsDir, "old-removed-reference.md")
	if err := os.WriteFile(stalePath, []byte("stale"), 0640); err != nil {
		t.Fatal(err)
	}

	if err := templates.CopyProjectAgents(directory); err != nil {
		t.Fatalf("CopyProjectAgents() = %v", err)
	}

	if _, err := os.Stat(stalePath); err == nil {
		t.Error("stale reference file should have been removed but still exists")
	}

	if _, err := os.Stat(filepath.Join(refsDir, "pk-file-format.md")); err != nil {
		t.Error("references/pk-file-format.md should exist after update")
	}
}

func TestNewAgentsUninstallModel_OmitsAbsentAgentsMD(t *testing.T) {
	orig, _ := os.Getwd()
	directory := t.TempDir()
	_ = os.Chdir(directory)
	defer func() { _ = os.Chdir(orig) }()

	factory, err := safedisk.NewCLIFactory(".")
	if err != nil {
		t.Fatal(err)
	}

	m := newAgentsUninstallModel(factory)

	for _, tgt := range m.targets {
		if tgt.name == "AGENTS.md" {
			t.Error("AGENTS.md target should not appear when file does not exist")
		}
	}
	if m.Step != agentsUninstallStepSelect {
		t.Errorf("step = %d, want agentsUninstallStepSelect (%d)", m.Step, agentsUninstallStepSelect)
	}
}

func TestNewAgentsUninstallModel_DetectsAgentsMD(t *testing.T) {
	orig, _ := os.Getwd()
	directory := t.TempDir()
	_ = os.Chdir(directory)
	defer func() { _ = os.Chdir(orig) }()

	if err := os.WriteFile(filepath.Join(directory, "AGENTS.md"), []byte("agents"), 0640); err != nil {
		t.Fatal(err)
	}

	factory, err := safedisk.NewCLIFactory(".")
	if err != nil {
		t.Fatal(err)
	}

	m := newAgentsUninstallModel(factory)

	found := false
	for _, tgt := range m.targets {
		if tgt.name == "AGENTS.md" {
			found = true
			break
		}
	}
	if !found {
		t.Error("AGENTS.md target should be detected when file exists")
	}
}

func TestRemoveGitignoreEntries_RemovesExactBlock(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	content := "node_modules/\ndist/\n" + gitignoreEntries + "*.log\n"
	if err := os.WriteFile(filepath.Join(directory, ".gitignore"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	builder, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer builder.Close()

	if err := removeGitignoreEntries(builder); err != nil {
		t.Fatalf("removeGitignoreEntries() = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(directory, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	result := string(got)
	if strings.Contains(result, "AGENTS.md") {
		t.Error("AGENTS.md should have been removed from .gitignore")
	}
	if !strings.Contains(result, "node_modules/") {
		t.Error("existing entries should be preserved")
	}
	if !strings.Contains(result, "*.log") {
		t.Error("entries after the agent block should be preserved")
	}
}

func TestRemoveGitignoreEntries_RemovesIndividualLines(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	content := "node_modules/\nAGENTS.md\nreferences/\n*.log\n"
	if err := os.WriteFile(filepath.Join(directory, ".gitignore"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	builder, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer builder.Close()

	if err := removeGitignoreEntries(builder); err != nil {
		t.Fatalf("removeGitignoreEntries() = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(directory, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	result := string(got)
	if strings.Contains(result, "AGENTS.md") {
		t.Error("AGENTS.md should have been removed")
	}
	if strings.Contains(result, "references/") {
		t.Error("references/ should have been removed")
	}
	if !strings.Contains(result, "node_modules/") {
		t.Error("unrelated entries should be preserved")
	}
	if !strings.Contains(result, "*.log") {
		t.Error("unrelated entries should be preserved")
	}
}

func TestRemoveGitignoreEntries_NoopWhenNotPresent(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	content := "node_modules/\ndist/\n"
	if err := os.WriteFile(filepath.Join(directory, ".gitignore"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	builder, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer builder.Close()

	if err := removeGitignoreEntries(builder); err != nil {
		t.Fatalf("removeGitignoreEntries() = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(directory, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != content {
		t.Errorf("file should be unchanged, got %q", string(got))
	}
}

func TestRemoveGitignoreEntries_NoopWhenMissing(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	builder, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer builder.Close()

	if err := removeGitignoreEntries(builder); err != nil {
		t.Fatalf("removeGitignoreEntries() = %v", err)
	}
}

func TestNewAgentsModel_Initialisation(t *testing.T) {
	t.Parallel()

	factory, err := safedisk.NewCLIFactory(".")
	if err != nil {
		t.Fatal(err)
	}

	m := newAgentsModel(factory)

	if len(m.targets) != 2 {
		t.Errorf("len(targets) = %d, want 2", len(m.targets))
	}
	if len(m.Selected) != 2 {
		t.Errorf("len(selected) = %d, want 2", len(m.Selected))
	}
	if m.Step != agentsStepSelect {
		t.Errorf("step = %d, want agentsStepSelect (%d)", m.Step, agentsStepSelect)
	}
	if m.Cursor != len(m.targets) {
		t.Errorf("cursor = %d, want %d (Continue button)", m.Cursor, len(m.targets))
	}
}

func TestAppendGitignoreEntries_CreatesFile(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	builder, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer builder.Close()

	if err := appendGitignoreEntries(builder); err != nil {
		t.Fatalf("appendGitignoreEntries() = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(directory, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	got := string(content)
	if !strings.Contains(got, "AGENTS.md") {
		t.Error(".gitignore should contain AGENTS.md entry")
	}
	if !strings.Contains(got, "references/") {
		t.Error(".gitignore should contain references/ entry")
	}
}

func TestAppendGitignoreEntries_AppendsToExisting(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	existing := "node_modules/\ndist/\n"
	if err := os.WriteFile(filepath.Join(directory, ".gitignore"), []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	builder, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer builder.Close()

	if err := appendGitignoreEntries(builder); err != nil {
		t.Fatalf("appendGitignoreEntries() = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(directory, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	got := string(content)
	if !strings.Contains(got, "node_modules/") {
		t.Error("existing .gitignore content should be preserved")
	}
	if !strings.Contains(got, "AGENTS.md") {
		t.Error(".gitignore should contain AGENTS.md entry")
	}
}

func TestAppendGitignoreEntries_SkipsIfAlreadyPresent(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	existing := "AGENTS.md\nreferences/\n"
	if err := os.WriteFile(filepath.Join(directory, ".gitignore"), []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	builder, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer builder.Close()

	if err := appendGitignoreEntries(builder); err != nil {
		t.Fatalf("appendGitignoreEntries() = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(directory, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	if strings.Count(string(content), "AGENTS.md") > 1 {
		t.Error("AGENTS.md should not be duplicated in .gitignore")
	}
}
