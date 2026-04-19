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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/cmd/piko/internal/wizard/templates"
	"piko.sh/piko/wdk/safedisk"
)

func TestCopyClaudeCodePlugin_WritesCorrectFiles(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	err := templates.CopyClaudeCodePlugin(directory)
	require.NoError(t, err, "CopyClaudeCodePlugin() = %v", err)

	_, err = os.Stat(filepath.Join(directory, "SKILL.md"))
	assert.NoError(t, err, "SKILL.md not created")

	_, err = os.Stat(filepath.Join(directory, "references", "pk-file-format.md"))
	assert.NoError(t, err, "references/pk-file-format.md not created")
	_, err = os.Stat(filepath.Join(directory, "references", "template-syntax.md"))
	assert.NoError(t, err, "references/template-syntax.md not created")

	_, err = os.Stat(filepath.Join(directory, ".claude-plugin", "plugin.json"))
	assert.NoError(t, err, ".claude-plugin/plugin.json not created")

	lspConfig, err := os.ReadFile(filepath.Join(directory, ".lsp.json"))
	if assert.NoError(t, err, ".lsp.json not created: %v", err) {
		assert.Contains(t, string(lspConfig), "pikopls", ".lsp.json should configure the pikopls command")
	}

	_, err = os.Stat(filepath.Join(directory, "AGENTS.md"))
	assert.Error(t, err, "AGENTS.md should not be written by CopyClaudeCodePlugin")
}

func TestCopyProjectAgents_WritesCorrectFiles(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	err := templates.CopyProjectAgents(directory)
	require.NoError(t, err, "CopyProjectAgents() = %v", err)

	_, err = os.Stat(filepath.Join(directory, "AGENTS.md"))
	assert.NoError(t, err, "AGENTS.md not created")

	_, err = os.Stat(filepath.Join(directory, "references", "pk-file-format.md"))
	assert.NoError(t, err, "references/pk-file-format.md not created")

	_, err = os.Stat(filepath.Join(directory, "SKILL.md"))
	assert.Error(t, err, "SKILL.md should not be written by CopyProjectAgents")

	_, err = os.Stat(filepath.Join(directory, ".claude-plugin"))
	assert.Error(t, err, ".claude-plugin/ should not be written by CopyProjectAgents")
}

func TestCopyAgentFiles_SkipsGoFiles(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	err := templates.CopyClaudeCodePlugin(directory)
	require.NoError(t, err, "CopyClaudeCodePlugin() = %v", err)

	_, err = os.Stat(filepath.Join(directory, "embed.go"))
	assert.Error(t, err, "embed.go should not be written to the destination")
}

func TestCopyClaudeCodePlugin_OverwritesExisting(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()

	skillPath := filepath.Join(directory, "SKILL.md")
	err := os.WriteFile(skillPath, []byte("old content"), 0640)
	require.NoError(t, err)

	err = templates.CopyClaudeCodePlugin(directory)
	require.NoError(t, err, "CopyClaudeCodePlugin() = %v", err)

	content, err := os.ReadFile(skillPath)
	require.NoError(t, err)

	assert.NotEqual(t, "old content", string(content), "SKILL.md should have been overwritten but still contains old content")
}

func TestCopyClaudeCodePlugin_RemovesStaleFiles(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()

	refsDir := filepath.Join(directory, "references")
	err := os.MkdirAll(refsDir, 0750)
	require.NoError(t, err)
	stalePath := filepath.Join(refsDir, "old-removed-reference.md")
	err = os.WriteFile(stalePath, []byte("stale"), 0640)
	require.NoError(t, err)

	err = templates.CopyClaudeCodePlugin(directory)
	require.NoError(t, err, "CopyClaudeCodePlugin() = %v", err)

	_, err = os.Stat(stalePath)
	assert.Error(t, err, "stale file should have been removed but still exists")
}

func TestCopyProjectAgents_OverwritesExisting(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()

	agentsPath := filepath.Join(directory, "AGENTS.md")
	err := os.WriteFile(agentsPath, []byte("old content"), 0640)
	require.NoError(t, err)

	err = templates.CopyProjectAgents(directory)
	require.NoError(t, err, "CopyProjectAgents() = %v", err)

	content, err := os.ReadFile(agentsPath)
	require.NoError(t, err)

	assert.NotEqual(t, "old content", string(content), "AGENTS.md should have been overwritten but still contains old content")
}

func TestCopyProjectAgents_RemovesStaleReferences(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()

	refsDir := filepath.Join(directory, "references")
	err := os.MkdirAll(refsDir, 0750)
	require.NoError(t, err)
	stalePath := filepath.Join(refsDir, "old-removed-reference.md")
	err = os.WriteFile(stalePath, []byte("stale"), 0640)
	require.NoError(t, err)

	err = templates.CopyProjectAgents(directory)
	require.NoError(t, err, "CopyProjectAgents() = %v", err)

	_, err = os.Stat(stalePath)
	assert.Error(t, err, "stale reference file should have been removed but still exists")

	_, err = os.Stat(filepath.Join(refsDir, "pk-file-format.md"))
	assert.NoError(t, err, "references/pk-file-format.md should exist after update")
}

func TestNewAgentsUninstallModel_OmitsAbsentAgentsMD(t *testing.T) {
	orig, _ := os.Getwd()
	directory := t.TempDir()
	_ = os.Chdir(directory)
	defer func() { _ = os.Chdir(orig) }()

	factory, err := safedisk.NewCLIFactory(".")
	require.NoError(t, err)

	m := newAgentsUninstallModel(factory)

	for _, tgt := range m.targets {
		assert.NotEqual(t, "AGENTS.md", tgt.name, "AGENTS.md target should not appear when file does not exist")
	}
	assert.Equal(t, agentsUninstallStepSelect, m.Step, "step = %d, want agentsUninstallStepSelect (%d)", m.Step, agentsUninstallStepSelect)
}

func TestNewAgentsUninstallModel_DetectsAgentsMD(t *testing.T) {
	orig, _ := os.Getwd()
	directory := t.TempDir()
	_ = os.Chdir(directory)
	defer func() { _ = os.Chdir(orig) }()

	err := os.WriteFile(filepath.Join(directory, "AGENTS.md"), []byte("agents"), 0640)
	require.NoError(t, err)

	factory, err := safedisk.NewCLIFactory(".")
	require.NoError(t, err)

	m := newAgentsUninstallModel(factory)

	found := false
	for _, tgt := range m.targets {
		if tgt.name == "AGENTS.md" {
			found = true
			break
		}
	}
	assert.True(t, found, "AGENTS.md target should be detected when file exists")
}

func TestRemoveGitignoreEntries_RemovesExactBlock(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	content := "node_modules/\ndist/\n" + gitignoreEntries + "*.log\n"
	err := os.WriteFile(filepath.Join(directory, ".gitignore"), []byte(content), 0600)
	require.NoError(t, err)

	builder, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
	require.NoError(t, err)
	defer builder.Close()

	err = removeGitignoreEntries(builder)
	require.NoError(t, err, "removeGitignoreEntries() = %v", err)

	got, err := os.ReadFile(filepath.Join(directory, ".gitignore"))
	require.NoError(t, err)

	result := string(got)
	assert.NotContains(t, result, "AGENTS.md", "AGENTS.md should have been removed from .gitignore")
	assert.Contains(t, result, "node_modules/", "existing entries should be preserved")
	assert.Contains(t, result, "*.log", "entries after the agent block should be preserved")
}

func TestRemoveGitignoreEntries_RemovesIndividualLines(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	content := "node_modules/\nAGENTS.md\nreferences/\n*.log\n"
	err := os.WriteFile(filepath.Join(directory, ".gitignore"), []byte(content), 0600)
	require.NoError(t, err)

	builder, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
	require.NoError(t, err)
	defer builder.Close()

	err = removeGitignoreEntries(builder)
	require.NoError(t, err, "removeGitignoreEntries() = %v", err)

	got, err := os.ReadFile(filepath.Join(directory, ".gitignore"))
	require.NoError(t, err)

	result := string(got)
	assert.NotContains(t, result, "AGENTS.md", "AGENTS.md should have been removed")
	assert.NotContains(t, result, "references/", "references/ should have been removed")
	assert.Contains(t, result, "node_modules/", "unrelated entries should be preserved")
	assert.Contains(t, result, "*.log", "unrelated entries should be preserved")
}

func TestRemoveGitignoreEntries_NoopWhenNotPresent(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	content := "node_modules/\ndist/\n"
	err := os.WriteFile(filepath.Join(directory, ".gitignore"), []byte(content), 0600)
	require.NoError(t, err)

	builder, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
	require.NoError(t, err)
	defer builder.Close()

	err = removeGitignoreEntries(builder)
	require.NoError(t, err, "removeGitignoreEntries() = %v", err)

	got, err := os.ReadFile(filepath.Join(directory, ".gitignore"))
	require.NoError(t, err)

	assert.Equal(t, content, string(got), "file should be unchanged, got %q", string(got))
}

func TestRemoveGitignoreEntries_NoopWhenMissing(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	builder, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
	require.NoError(t, err)
	defer builder.Close()

	err = removeGitignoreEntries(builder)
	require.NoError(t, err, "removeGitignoreEntries() = %v", err)
}

func TestNewAgentsModel_Initialisation(t *testing.T) {
	t.Parallel()

	factory, err := safedisk.NewCLIFactory(".")
	require.NoError(t, err)

	m := newAgentsModel(factory, "0.0.0-test")

	assert.Len(t, m.targets, 2, "len(targets) = %d, want 2", len(m.targets))
	assert.Len(t, m.Selected, 2, "len(selected) = %d, want 2", len(m.Selected))
	assert.Equal(t, agentsStepSelect, m.Step, "step = %d, want agentsStepSelect (%d)", m.Step, agentsStepSelect)
	assert.Len(t, m.targets, m.Cursor, "cursor = %d, want %d (Continue button)", m.Cursor, len(m.targets))
}

func TestAppendGitignoreEntries_CreatesFile(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	builder, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
	require.NoError(t, err)
	defer builder.Close()

	err = appendGitignoreEntries(builder)
	require.NoError(t, err, "appendGitignoreEntries() = %v", err)

	content, err := os.ReadFile(filepath.Join(directory, ".gitignore"))
	require.NoError(t, err)

	got := string(content)
	assert.Contains(t, got, "AGENTS.md", ".gitignore should contain AGENTS.md entry")
	assert.Contains(t, got, "references/", ".gitignore should contain references/ entry")
}

func TestAppendGitignoreEntries_AppendsToExisting(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	existing := "node_modules/\ndist/\n"
	err := os.WriteFile(filepath.Join(directory, ".gitignore"), []byte(existing), 0600)
	require.NoError(t, err)

	builder, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
	require.NoError(t, err)
	defer builder.Close()

	err = appendGitignoreEntries(builder)
	require.NoError(t, err, "appendGitignoreEntries() = %v", err)

	content, err := os.ReadFile(filepath.Join(directory, ".gitignore"))
	require.NoError(t, err)

	got := string(content)
	assert.Contains(t, got, "node_modules/", "existing .gitignore content should be preserved")
	assert.Contains(t, got, "AGENTS.md", ".gitignore should contain AGENTS.md entry")
}

func TestAppendGitignoreEntries_SkipsIfAlreadyPresent(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	existing := "AGENTS.md\nreferences/\n"
	err := os.WriteFile(filepath.Join(directory, ".gitignore"), []byte(existing), 0600)
	require.NoError(t, err)

	builder, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
	require.NoError(t, err)
	defer builder.Close()

	err = appendGitignoreEntries(builder)
	require.NoError(t, err, "appendGitignoreEntries() = %v", err)

	content, err := os.ReadFile(filepath.Join(directory, ".gitignore"))
	require.NoError(t, err)

	assert.False(t, strings.Count(string(content), "AGENTS.md") > 1, "AGENTS.md should not be duplicated in .gitignore")
}
