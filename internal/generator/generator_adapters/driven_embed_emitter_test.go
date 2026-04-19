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

package generator_adapters

import (
	"context"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/wdk/safedisk"
)

func newEmbedFixture(t *testing.T) (*DrivenEmbedEmitter, string) {
	t.Helper()
	root := t.TempDir()
	pikoDir := filepath.Join(root, ".piko")
	distDir := filepath.Join(root, "dist")

	files := map[string]string{
		".piko/blobs/aa/source.png":                        "source-bytes",
		".piko/blobs/bb/derived.webp":                      "derived-bytes",
		".piko/storage/uploads/readme.txt":                 "storage-bytes",
		".piko/wal/persistence/registry/snapshot.piko":     "registry-snapshot",
		".piko/wal/persistence/orchestrator/snapshot.piko": "orchestrator-snapshot",
		".piko/wal/persistence/registry/000001.wal":        "wal-segment",
		".piko/cache/some.cache":                           "cache-bytes",
		".piko/logs/app.log":                               "log-bytes",
		".piko/tmp/upload.part":                            "tmp-bytes",
		".piko/.lock":                                      "lock-bytes",
	}
	for name, content := range files {
		full := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o750), "fixture directories must create")
		require.NoError(t, os.WriteFile(full, []byte(content), 0o640), "fixture files must write")
	}
	require.NoError(t, os.MkdirAll(distDir, 0o750), "the dist fixture directory must create")

	factory, err := safedisk.NewCLIFactory(root)
	require.NoError(t, err, "the sandbox factory must create")
	source, err := factory.Create("embed-test-source", pikoDir, safedisk.ModeReadOnly)
	require.NoError(t, err, "the source sandbox must create")
	destination, err := factory.Create("embed-test-output", distDir, safedisk.ModeReadWrite)
	require.NoError(t, err, "the destination sandbox must create")

	return NewDrivenEmbedEmitter(source, destination), distDir
}

func listPayload(t *testing.T, distDir string) []string {
	t.Helper()
	var paths []string
	root := filepath.Join(distDir, "embed")
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			relative, relErr := filepath.Rel(distDir, path)
			require.NoError(t, relErr, "payload paths must be relativisable")
			paths = append(paths, filepath.ToSlash(relative))
		}
		return nil
	})
	require.NoError(t, err, "the payload must walk")
	return paths
}

func TestEmbedEmitter_CopiesExactlyTheAllowList(t *testing.T) {
	t.Parallel()

	emitter, distDir := newEmbedFixture(t)
	require.NoError(t, emitter.Emit(t.Context()), "the emit must succeed")

	assert.ElementsMatch(t, []string{
		"embed/piko/.pikoembed",
		"embed/piko/blobs/aa/source.png",
		"embed/piko/blobs/bb/derived.webp",
		"embed/piko/storage/uploads/readme.txt",
		"embed/piko/wal/persistence/registry/snapshot.piko",
	}, listPayload(t, distDir),
		"the payload must contain exactly the allow-listed trees, the registry snapshot, and the marker; never the orchestrator snapshot, WAL segments, cache, logs, tmp, or the lock file")
}

func TestEmbedEmitter_IsIdempotentAndClearsStalePayload(t *testing.T) {
	t.Parallel()

	emitter, distDir := newEmbedFixture(t)
	stale := filepath.Join(distDir, "embed", "piko", "stale.bin")
	require.NoError(t, os.MkdirAll(filepath.Dir(stale), 0o750), "the stale directory must create")
	require.NoError(t, os.WriteFile(stale, []byte("stale"), 0o640), "the stale file must write")

	require.NoError(t, emitter.Emit(t.Context()), "the first emit must succeed")
	require.NoError(t, emitter.Emit(t.Context()), "a rerun must succeed")

	for _, path := range listPayload(t, distDir) {
		assert.NotEqual(t, "embed/piko/stale.bin", path, "a rerun must clear content from a previous payload")
	}
}

func TestEmbedEmitter_ToleratesMissingSubtrees(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".piko"), 0o750), "the empty .piko must create")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "dist"), 0o750), "the dist directory must create")
	factory, err := safedisk.NewCLIFactory(root)
	require.NoError(t, err, "the sandbox factory must create")
	source, err := factory.Create("embed-test-source", filepath.Join(root, ".piko"), safedisk.ModeReadOnly)
	require.NoError(t, err, "the source sandbox must create")
	destination, err := factory.Create("embed-test-output", filepath.Join(root, "dist"), safedisk.ModeReadWrite)
	require.NoError(t, err, "the destination sandbox must create")

	emitter := NewDrivenEmbedEmitter(source, destination)
	require.NoError(t, emitter.Emit(t.Context()), "an empty project must still emit")

	assert.ElementsMatch(t, []string{"embed/piko/.pikoembed"}, listPayload(t, filepath.Join(root, "dist")),
		"an empty project's payload holds only the marker, which keeps the embed directive resolvable")
}

func TestEmbedEmitter_FailedEmitCleansUpPartialPayload(t *testing.T) {
	t.Parallel()

	emitter, distDir := newEmbedFixture(t)
	require.NoError(t, emitter.Emit(t.Context()), "the first emit must succeed")

	generatedPath := filepath.Join(distDir, "embed_gen.go")
	require.FileExists(t, generatedPath, "a successful emit must write embed_gen.go")

	cancelledContext, cancel := context.WithCancel(context.Background())
	cancel()
	require.Error(t, emitter.Emit(cancelledContext), "a cancelled emit must fail")

	assert.NoFileExists(t, generatedPath, "a failed emit must delete the stale generated file")
	assert.NoDirExists(t, filepath.Join(distDir, "embed"),
		"a failed emit must remove the partially copied payload")
}

func TestEmbedEmitter_WritesParseableTagGatedEmbedGen(t *testing.T) {
	t.Parallel()

	emitter, distDir := newEmbedFixture(t)
	require.NoError(t, emitter.Emit(t.Context()), "the emit must succeed")

	source, err := os.ReadFile(filepath.Join(distDir, "embed_gen.go"))
	require.NoError(t, err, "embed_gen.go must exist")
	text := string(source)

	assert.True(t, strings.HasPrefix(text, generator_dto.EmbedBuildConstraint), "the file must carry the piko_embed build constraint")
	assert.Contains(t, text, "//go:embed manifest.bin", "the manifest embed directive must be present")
	assert.Contains(t, text, "//go:embed all:embed/piko", "the payload embed directive must use the all: prefix so the dot-prefixed marker is included")
	assert.Contains(t, text, "RegisterEmbeddedRuntime", "the init must register the payload")

	fileSet := token.NewFileSet()
	_, err = parser.ParseFile(fileSet, "embed_gen.go", source, parser.AllErrors)
	require.NoError(t, err, "the generated file must parse as Go")
}

func TestEmbedBuildConstraint_CarriesBothTags(t *testing.T) {
	t.Parallel()

	assert.Contains(t, generator_dto.EmbedBuildConstraint, "piko_embed", "the payload must be opt-in via the piko_embed tag")
	assert.Contains(t, generator_dto.EmbedBuildConstraint, "!piko_analysis", "the payload must stay out of the analysis pass")
}
