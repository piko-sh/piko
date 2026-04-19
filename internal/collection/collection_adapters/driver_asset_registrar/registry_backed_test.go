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

package driver_asset_registrar

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewRegistryBackedRegistrar_AppliesDefaults(t *testing.T) {
	r := NewRegistryBackedRegistrar(&render_domain.MockRegistryPort{})
	assert.Equal(t, defaultMaxAssetSizeBytes, r.maxAssetSizeBytes)
}

func TestNewRegistryBackedRegistrar_WithMaxAssetSizeBytes(t *testing.T) {
	r := NewRegistryBackedRegistrar(&render_domain.MockRegistryPort{}, WithMaxAssetSizeBytes(1024))
	assert.Equal(t, int64(1024), r.maxAssetSizeBytes)
}

func TestNewRegistryBackedRegistrar_WithMaxAssetSizeBytes_IgnoresNonPositive(t *testing.T) {
	r := NewRegistryBackedRegistrar(
		&render_domain.MockRegistryPort{},
		WithMaxAssetSizeBytes(0),
		WithMaxAssetSizeBytes(-5),
	)
	assert.Equal(t, defaultMaxAssetSizeBytes, r.maxAssetSizeBytes)
}

func TestRegisterCollectionAsset_NilRegistry(t *testing.T) {
	r := &RegistryBackedRegistrar{}
	_, err := r.RegisterCollectionAsset(context.Background(), newTestSandbox(t, nil), "a.svg", "docs")
	assert.ErrorIs(t, err, ErrRegistryNotConfigured)
}

func TestRegisterCollectionAsset_NilSandbox(t *testing.T) {
	r := NewRegistryBackedRegistrar(&render_domain.MockRegistryPort{})
	_, err := r.RegisterCollectionAsset(context.Background(), nil, "a.svg", "docs")
	assert.ErrorIs(t, err, ErrSandboxRequired)
}

func TestRegisterCollectionAsset_EmptyPath(t *testing.T) {
	r := NewRegistryBackedRegistrar(&render_domain.MockRegistryPort{})
	_, err := r.RegisterCollectionAsset(context.Background(), newTestSandbox(t, nil), "", "docs")
	assert.ErrorIs(t, err, ErrEmptyPath)
}

func TestRegisterCollectionAsset_EmptyCollectionName(t *testing.T) {
	r := NewRegistryBackedRegistrar(&render_domain.MockRegistryPort{})
	_, err := r.RegisterCollectionAsset(context.Background(), newTestSandbox(t, nil), "a.svg", "")
	assert.ErrorIs(t, err, ErrEmptyCollectionName)
}

func TestRegisterCollectionAsset_InvalidCollectionName(t *testing.T) {
	r := NewRegistryBackedRegistrar(&render_domain.MockRegistryPort{})
	badNames := []string{"docs/blog", "docs\\blog", "docs/..", "..", "../../etc"}
	for _, name := range badNames {
		t.Run(name, func(t *testing.T) {
			_, err := r.RegisterCollectionAsset(context.Background(), newTestSandbox(t, nil), "a.svg", name)
			assert.ErrorIs(t, err, ErrInvalidCollectionName)
		})
	}
}

func TestRegisterCollectionAsset_HappyPath(t *testing.T) {
	sandbox := newTestSandbox(t, map[string][]byte{
		"diagrams/bar.svg": []byte("<svg></svg>"),
	})

	var (
		seenArtefactID  atomic.Value
		seenSourcePath  atomic.Value
		seenBackendID   atomic.Value
		seenRawContents atomic.Value
	)

	mock := &render_domain.MockRegistryPort{
		UpsertArtefactFunc: func(
			_ context.Context,
			artefactID string,
			sourcePath string,
			sourceData io.Reader,
			storageBackendID string,
			_ []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			data, readErr := io.ReadAll(sourceData)
			require.NoError(t, readErr)
			seenArtefactID.Store(artefactID)
			seenSourcePath.Store(sourcePath)
			seenBackendID.Store(storageBackendID)
			seenRawContents.Store(string(data))
			return &registry_dto.ArtefactMeta{ID: artefactID}, nil
		},
		GetArtefactServePathFunc: func(_ context.Context, artefactID string) string {
			return "/_piko/assets/" + artefactID
		},
	}

	r := NewRegistryBackedRegistrar(mock)
	serveURL, err := r.RegisterCollectionAsset(context.Background(), sandbox, "diagrams/bar.svg", "docs")
	require.NoError(t, err)
	assert.Equal(t, "/_piko/assets/collection/docs/diagrams/bar.svg", serveURL)
	assert.Equal(t, "collection/docs/diagrams/bar.svg", seenArtefactID.Load())
	assert.Equal(t, "diagrams/bar.svg", seenSourcePath.Load())
	assert.Equal(t, defaultStorageBackendID, seenBackendID.Load())
	assert.Equal(t, "<svg></svg>", seenRawContents.Load())
}

func TestRegisterCollectionAsset_ExceedsMaxSize(t *testing.T) {
	bigPayload := make([]byte, 2048)
	for i := range bigPayload {
		bigPayload[i] = 'X'
	}
	sandbox := newTestSandbox(t, map[string][]byte{"big.svg": bigPayload})

	mock := &render_domain.MockRegistryPort{
		UpsertArtefactFunc: func(
			_ context.Context, _ string, _ string, _ io.Reader, _ string, _ []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			t.Fatal("UpsertArtefact must not be called when asset exceeds size cap")
			return nil, nil
		},
	}

	r := NewRegistryBackedRegistrar(mock, WithMaxAssetSizeBytes(1024))
	_, err := r.RegisterCollectionAsset(context.Background(), sandbox, "big.svg", "docs")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAssetTooLarge)
}

func TestRegisterCollectionAsset_ExactlyAtMaxSizeSucceeds(t *testing.T) {
	const limit int64 = 32
	payload := make([]byte, limit)
	sandbox := newTestSandbox(t, map[string][]byte{"edge.svg": payload})

	mock := &render_domain.MockRegistryPort{
		UpsertArtefactFunc: func(
			_ context.Context, artefactID string, _ string, _ io.Reader, _ string, _ []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{ID: artefactID}, nil
		},
		GetArtefactServePathFunc: func(_ context.Context, artefactID string) string {
			return "/_piko/assets/" + artefactID
		},
	}

	r := NewRegistryBackedRegistrar(mock, WithMaxAssetSizeBytes(limit))
	_, err := r.RegisterCollectionAsset(context.Background(), sandbox, "edge.svg", "docs")
	require.NoError(t, err)
}

func TestRegisterCollectionAsset_FileMissing(t *testing.T) {
	sandbox := newTestSandbox(t, nil)
	r := NewRegistryBackedRegistrar(&render_domain.MockRegistryPort{})
	_, err := r.RegisterCollectionAsset(context.Background(), sandbox, "missing.svg", "docs")
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "opening asset"), "error should mention open failure")
}

func TestRegisterCollectionAsset_UpsertFailure(t *testing.T) {
	sandbox := newTestSandbox(t, map[string][]byte{"a.svg": []byte("<svg/>")})
	boom := errors.New("registry is down")
	mock := &render_domain.MockRegistryPort{
		UpsertArtefactFunc: func(
			_ context.Context, _ string, _ string, _ io.Reader, _ string, _ []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			return nil, boom
		},
	}
	r := NewRegistryBackedRegistrar(mock)
	_, err := r.RegisterCollectionAsset(context.Background(), sandbox, "a.svg", "docs")
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestRegisterCollectionAsset_NoServableVariant(t *testing.T) {
	sandbox := newTestSandbox(t, map[string][]byte{"a.svg": []byte("<svg/>")})
	mock := &render_domain.MockRegistryPort{
		UpsertArtefactFunc: func(
			_ context.Context, artefactID string, _ string, _ io.Reader, _ string, _ []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{ID: artefactID}, nil
		},
		GetArtefactServePathFunc: func(_ context.Context, _ string) string {
			return ""
		},
	}
	r := NewRegistryBackedRegistrar(mock)
	_, err := r.RegisterCollectionAsset(context.Background(), sandbox, "a.svg", "docs")
	assert.ErrorIs(t, err, ErrNoServableVariant)
}

func TestBuildArtefactID_Deterministic(t *testing.T) {
	first := buildArtefactID("docs", "images/foo.svg")
	second := buildArtefactID("docs", "images/foo.svg")
	assert.Equal(t, first, second)
	assert.Equal(t, "collection/docs/images/foo.svg", first)
}

func TestBuildArtefactID_StripsLeadingSlashes(t *testing.T) {
	a := buildArtefactID("/docs/", "/images/foo.svg")
	b := buildArtefactID("docs", "images/foo.svg")
	assert.Equal(t, b, a)
}

func TestBuildArtefactID_EmptyCollectionName(t *testing.T) {
	id := buildArtefactID("", "images/foo.svg")
	assert.Equal(t, "collection/images/foo.svg", id)
}

func newTestSandbox(t *testing.T, files map[string][]byte) safedisk.Sandbox {
	t.Helper()
	directory := t.TempDir()
	for relativePath, data := range files {
		full := filepath.Join(directory, relativePath)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, data, 0o644))
	}
	sandbox, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadOnly)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })
	return sandbox
}
