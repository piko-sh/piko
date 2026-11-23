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

package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmitEmbeddedRuntime_RejectsUnsupportedScopes(t *testing.T) {
	t.Parallel()

	err := NewContainer(WithEmbedScope(EmbedSourceOnly)).EmitEmbeddedRuntime(context.Background())
	require.ErrorIs(t, err, ErrEmbedSourceOnlyUnimplemented)

	err = NewContainer(WithEmbedScope(EmbedScope(42))).EmitEmbeddedRuntime(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown embed scope 42")
}

func TestEmitEmbeddedRuntime_PayloadBootsAProductionDaemon(t *testing.T) {
	baseDir := newTestProject(t, map[string]string{"pages/index.pk": testProjectIndexPage})
	builder := newResolvedContainer(t, baseDir, WithReleaseID("release-embed"))
	require.NoError(t, BuildProject(context.Background(), GenerateModeAll, builder))
	require.NoError(t, builder.FlushPersistenceSnapshot(context.Background()))

	require.NoError(t, builder.EmitEmbeddedRuntime(context.Background()))

	embedDir := filepath.Join(baseDir, distDirName, "embed", "piko")
	require.DirExists(t, embedDir)
	manifest, err := os.ReadFile(filepath.Join(baseDir, distDirName, manifestFilenameBinary))
	require.NoError(t, err)

	runtime := newResolvedContainer(t, t.TempDir(),
		WithEmbeddedPikoFolder(os.DirFS(embedDir)),
		WithEmbeddedManifest(manifest),
		WithReleaseID("release-embed"),
	)
	daemon, err := Daemon(context.Background(), runModeProd, runtime, &Dependencies{AppRouter: chi.NewRouter()})

	require.NoError(t, err)
	assert.NotNil(t, daemon)
	waitForInitialTasks(t, runtime)
	assert.True(t, runtime.IsEmbeddedMode())
}

func TestCreateManifestProvider_RequiresTheManifestInEmbeddedMode(t *testing.T) {
	t.Parallel()

	c := NewContainer(WithEmbeddedPikoFolder(os.DirFS(t.TempDir())))

	provider, err := createManifestProvider(context.Background(), c)

	require.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "WithEmbeddedPikoFolder requires WithEmbeddedManifest")
}
