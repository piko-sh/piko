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

package agents_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	canonicalSkillPath   = "SKILL.md"
	marketplaceSkillPath = "skills/piko/SKILL.md"
	frontmatterDelimiter = "---\n"
	generatedNotice      = "\n<!-- Generated from plugins/agents/SKILL.md by `go test ./plugins/agents/ -update`. Do not edit by hand. -->\n"
)

var (
	update                     = flag.Bool("update", false, "regenerate skills/piko/SKILL.md from SKILL.md")
	rootReferencePrefix        = []byte("`references/")
	marketplaceReferencePrefix = []byte("`../../references/")
)

func TestMarketplaceSkillMatchesCanonicalSkill(t *testing.T) {
	canonical, err := os.ReadFile(canonicalSkillPath)
	require.NoError(t, err)

	want := deriveMarketplaceSkill(t, canonical)

	if *update {
		require.NoError(t, os.MkdirAll(filepath.Dir(marketplaceSkillPath), 0o755))
		require.NoError(t, os.WriteFile(marketplaceSkillPath, want, 0o644))
	}

	got, err := os.ReadFile(marketplaceSkillPath)
	require.NoError(t, err, "run `go test ./plugins/agents/ -update` to generate %s", marketplaceSkillPath)
	assert.Equal(t, string(want), string(got),
		"%s has drifted from %s; edit %s and run `go test ./plugins/agents/ -update`",
		marketplaceSkillPath, canonicalSkillPath, canonicalSkillPath)
}

func TestMarketplaceSkillReferencesResolve(t *testing.T) {
	derived, err := os.ReadFile(marketplaceSkillPath)
	require.NoError(t, err)

	for line := range bytes.Lines(derived) {
		index := bytes.Index(line, marketplaceReferencePrefix)
		if index < 0 {
			continue
		}
		rest := line[index+1:]
		end := bytes.IndexByte(rest, '`')
		require.Positive(t, end, "unterminated reference in %q", line)
		target := filepath.Join(filepath.Dir(marketplaceSkillPath), string(rest[:end]))
		_, statErr := os.Stat(target)
		assert.NoError(t, statErr, "reference %s does not resolve from %s", rest[:end], marketplaceSkillPath)
	}
}

func deriveMarketplaceSkill(t *testing.T, canonical []byte) []byte {
	t.Helper()

	require.True(t, bytes.HasPrefix(canonical, []byte(frontmatterDelimiter)), "SKILL.md must open with YAML frontmatter")
	body := canonical[len(frontmatterDelimiter):]
	closing := bytes.Index(body, []byte("\n"+frontmatterDelimiter))
	require.GreaterOrEqual(t, closing, 0, "SKILL.md frontmatter is not closed")

	frontmatterEnd := len(frontmatterDelimiter) + closing + 1 + len(frontmatterDelimiter)

	var derived bytes.Buffer
	derived.Write(canonical[:frontmatterEnd])
	derived.WriteString(generatedNotice)
	derived.Write(bytes.ReplaceAll(canonical[frontmatterEnd:], rootReferencePrefix, marketplaceReferencePrefix))
	return derived.Bytes()
}
