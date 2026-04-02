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

//go:build integration

package llm_integration_test

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
)

func TestGolden_DumpChunks(t *testing.T) {
	goldenDir := filepath.Join("golden")

	_ = os.RemoveAll(goldenDir)
	require.NoError(t, os.MkdirAll(goldenDir, 0o755))

	docsPath := filepath.Join("..", "..", "..", "docs")

	loader := llm_domain.NewRecursiveFSLoader(os.DirFS(docsPath), "*.md")
	docs, err := loader.Load(context.Background())
	require.NoError(t, err, "loading docs")
	require.NotEmpty(t, docs, "expected at least one document")

	t.Logf("Loaded %d documents", len(docs))

	frontmatter := llm_domain.ExtractFrontmatter()
	for i, doc := range docs {
		docs[i] = frontmatter(doc)
	}

	splitter, err := llm_domain.NewMarkdownSplitter(3000, 100, llm_domain.WithMinChunkSize(400))
	require.NoError(t, err)

	var allChunks []llm_domain.Document
	for _, doc := range docs {
		chunks := splitter.Split(doc)
		allChunks = append(allChunks, chunks...)
	}

	slices.SortFunc(allChunks, func(a, b llm_domain.Document) int {
		return cmp.Compare(a.ID, b.ID)
	})

	t.Logf("Generated %d chunks from %d documents", len(allChunks), len(docs))

	for i, chunk := range allChunks {
		filename := fmt.Sprintf("%04d_%s.md", i, sanitiseFilename(chunk.ID))
		path := filepath.Join(goldenDir, filename)

		var buffer strings.Builder
		buffer.WriteString("---\n")
		fmt.Fprintf(&buffer, "id: %s\n", chunk.ID)
		fmt.Fprintf(&buffer, "size: %d bytes\n", len(chunk.Content))
		if heading, ok := chunk.Metadata["heading"].(string); ok && heading != "" {
			fmt.Fprintf(&buffer, "heading: %s\n", heading)
		}
		if source, ok := chunk.Metadata["source"].(string); ok && source != "" {
			fmt.Fprintf(&buffer, "source: %s\n", source)
		}
		if title, ok := chunk.Metadata["title"].(string); ok && title != "" {
			fmt.Fprintf(&buffer, "title: %s\n", title)
		}
		buffer.WriteString("---\n\n")
		buffer.WriteString(chunk.Content)
		buffer.WriteByte('\n')

		require.NoError(t, os.WriteFile(path, []byte(buffer.String()), 0o644),
			"writing golden file %s", filename)
	}

	t.Logf("Wrote %d golden files to %s", len(allChunks), goldenDir)
}

func sanitiseFilename(id string) string {
	r := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		" ", "_",
		".md", "",
	)
	name := r.Replace(id)

	if len(name) > 100 {
		name = name[:100]
	}
	return name
}
