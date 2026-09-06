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

package pdf

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/fonts"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

const (
	testPdfPackagePath        = "piko.sh/piko/wdk/pdf/testdata/rendered"
	testFailingPdfPackagePath = "piko.sh/piko/wdk/pdf/testdata/failing"
)

type fakeManifestProvider struct {
	err      error
	manifest *generator_dto.Manifest
}

func (p *fakeManifestProvider) Load(context.Context) (*generator_dto.Manifest, error) {
	return p.manifest, p.err
}

var (
	registerTestTemplatesOnce sync.Once
)

func registerTestTemplates(t *testing.T) {
	t.Helper()
	registerTestTemplatesOnce.Do(func() {
		templater_domain.RegisterASTFunc(testPdfPackagePath, func(*templater_dto.RequestData, any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
			return &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{ast_domain.NewTextNode("rendered")}}, templater_dto.InternalMetadata{}, nil
		})
		templater_domain.RegisterASTFunc(testFailingPdfPackagePath, func(*templater_dto.RequestData, any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
			metadata := templater_dto.InternalMetadata{}
			metadata.RenderError = errors.New("template failed")
			return nil, metadata, nil
		})
	})
}

func TestBuildFontEntries(t *testing.T) {
	t.Parallel()

	extra := newStaticFontEntry("Extra", 300, 1, []byte{1})
	tests := []struct {
		name   string
		config serviceConfig
		want   []layouter_dto.FontEntry
	}{
		{
			name: "defaults include regular and bold",
			want: []layouter_dto.FontEntry{
				newStaticFontEntry(fonts.NotoSansFamilyName, defaultRegularFontWeight, int(layouter_domain.FontStyleNormal), fonts.NotoSansRegularTTF),
				newStaticFontEntry(fonts.NotoSansFamilyName, defaultBoldFontWeight, int(layouter_domain.FontStyleNormal), fonts.NotoSansBoldTTF),
			},
		},
		{
			name:   "excluded bold and extra fonts after the defaults",
			config: serviceConfig{excludeDefaultBold: true, extraFontEntries: []layouter_dto.FontEntry{extra}},
			want: []layouter_dto.FontEntry{
				newStaticFontEntry(fonts.NotoSansFamilyName, defaultRegularFontWeight, int(layouter_domain.FontStyleNormal), fonts.NotoSansRegularTTF),
				extra,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, buildFontEntries(&tt.config))
		})
	}
}

func TestNewStaticFontEntry(t *testing.T) {
	t.Parallel()

	entry := newStaticFontEntry("Serif", 500, 1, []byte{7})

	assert.Equal(t, layouter_dto.FontEntry{Family: "Serif", Weight: 500, Style: 1, Data: []byte{7}}, entry)
	assert.False(t, entry.IsVariable)
}

func TestServiceOptions(t *testing.T) {
	t.Parallel()

	config := &serviceConfig{}
	limits := LayoutLimits{MaxPages: 12}
	for _, option := range []ServiceOption{
		WithFont("Extra", 400, 0, []byte{1}),
		WithVariableFont("Variable", 100, 900, 1, []byte{2}),
		WithExcludeDefaultBold(),
		WithSVGVectorRendering(),
		WithLayoutLimits(limits),
		WithMaxImagePixels(99),
	} {
		option(config)
	}

	require.Len(t, config.extraFontEntries, 2)
	assert.Equal(t, 400, config.extraFontEntries[0].Weight)
	assert.True(t, config.extraFontEntries[1].IsVariable)
	assert.Equal(t, 100, config.extraFontEntries[1].WeightMin)
	assert.Equal(t, 900, config.extraFontEntries[1].WeightMax)
	assert.True(t, config.excludeDefaultBold)
	assert.True(t, config.svgVectorRendering)
	assert.Equal(t, limits, config.layoutLimits)
	assert.Equal(t, 99, config.maxImagePixels)
}

func TestNewServiceFromManifest(t *testing.T) {
	t.Parallel()

	t.Run("creates a service without reading the manifest", func(t *testing.T) {
		t.Parallel()
		service, err := NewServiceFromManifest(filepath.Join(t.TempDir(), "missing", "manifest.bin"), WithSVGVectorRendering())
		require.NoError(t, err)
		require.NotNil(t, service)
		assert.NotNil(t, service.NewRender())
	})

	t.Run("rejects an empty path", func(t *testing.T) {
		t.Parallel()
		_, err := NewServiceFromManifest("")
		assert.Error(t, err)
	})

	t.Run("rejects invalid font data", func(t *testing.T) {
		t.Parallel()
		_, err := NewServiceFromManifest("dist/manifest.bin", WithFont("Broken", 400, 0, []byte("not a font")))
		assert.Error(t, err)
	})
}

func TestManifestService_RenderBeforeLoad(t *testing.T) {
	t.Parallel()

	service, err := NewServiceFromManifest(filepath.Join(t.TempDir(), "manifest.bin"))
	require.NoError(t, err)

	_, err = service.NewRender().Template("pdfs/doc.pk").Do(context.Background())

	assert.ErrorIs(t, err, ErrManifestNotLoaded)
}

func TestManifestService_LoadErrors(t *testing.T) {
	t.Parallel()

	t.Run("missing directory", func(t *testing.T) {
		t.Parallel()
		service, err := NewServiceFromManifest(filepath.Join(t.TempDir(), "absent", "manifest.bin"))
		require.NoError(t, err)
		assert.Error(t, service.Load(context.Background()))
	})

	t.Run("missing manifest file", func(t *testing.T) {
		t.Parallel()
		service, err := NewServiceFromManifest(filepath.Join(t.TempDir(), "manifest.bin"))
		require.NoError(t, err)
		assert.Error(t, service.Load(context.Background()))
	})

	t.Run("corrupt manifest file", func(t *testing.T) {
		t.Parallel()
		directory := t.TempDir()
		path := filepath.Join(directory, "manifest.bin")
		require.NoError(t, os.WriteFile(path, []byte("not a manifest"), 0o600))
		service, err := NewServiceFromManifest(path)
		require.NoError(t, err)
		assert.Error(t, service.Load(context.Background()))
	})
}

func TestManifestTemplateRunner_RunPdfWithProps(t *testing.T) {
	t.Parallel()

	manifest := &generator_dto.Manifest{Pdfs: map[string]generator_dto.ManifestPdfEntry{
		"pdfs/doc.pk":    {PackagePath: testPdfPackagePath, OriginalSourcePath: "pdfs/doc.pk", StyleBlock: "p{}"},
		"pdfs/broken.pk": {PackagePath: testFailingPdfPackagePath, OriginalSourcePath: "pdfs/broken.pk"},
	}}
	registerTestTemplates(t)
	runner := &manifestTemplateRunner{}
	require.NoError(t, runner.load(context.Background(), &fakeManifestProvider{manifest: manifest}))

	t.Run("renders a known template", func(t *testing.T) {
		t.Parallel()
		tree, styling, err := runner.RunPdfWithProps(context.Background(), "pdfs/doc.pk", nil, map[string]string{"a": "b"})
		require.NoError(t, err)
		require.NotNil(t, tree)
		assert.Equal(t, "p{}", styling)
		require.Len(t, tree.RootNodes, 1)
		assert.Equal(t, "rendered", tree.RootNodes[0].TextContent)
	})

	t.Run("reports a render error", func(t *testing.T) {
		t.Parallel()
		_, _, err := runner.RunPdfWithProps(context.Background(), "pdfs/broken.pk", nil, nil)
		assert.ErrorContains(t, err, "template failed")
	})

	t.Run("reports an unknown template", func(t *testing.T) {
		t.Parallel()
		_, _, err := runner.RunPdfWithProps(context.Background(), "pdfs/unknown.pk", nil, nil)
		assert.ErrorContains(t, err, "PDF entry not found")
	})
}

func TestManifestTemplateRunner_LoadFailureKeepsPreviousStore(t *testing.T) {
	t.Parallel()

	runner := &manifestTemplateRunner{}
	err := runner.load(context.Background(), &fakeManifestProvider{err: errors.New("unreadable")})

	assert.Error(t, err)
	assert.Nil(t, runner.store.Load())
}
