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

package annotator_domain

import (
	"context"
	"errors"
	goast "go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

type stubKeyGenerator struct {
	err error
	key string
}

func (s *stubKeyGenerator) Generate(_ context.Context, _ inspector_dto.Config, _ map[string][]byte, _ map[string]string) (string, error) {
	return s.key, s.err
}

type stubParser struct {
	files map[string]*goast.File
	err   error
}

func (s *stubParser) Parse(_ context.Context, _ map[string][]byte, _ int) (map[string]*goast.File, error) {
	return s.files, s.err
}

type stubLoader struct {
	err  error
	pkgs []*packages.Package
}

func (s *stubLoader) Load(_ context.Context, _ inspector_dto.Config, _ map[string][]byte) ([]*packages.Package, error) {
	return s.pkgs, s.err
}

type stubEncoder struct {
	data *inspector_dto.TypeData
	err  error
}

func (s *stubEncoder) Encode(_ []*packages.Package, _ string) (*inspector_dto.TypeData, error) {
	return s.data, s.err
}

func newTestTypeBuilder() *inspector_domain.TypeBuilder {
	return inspector_domain.NewTypeBuilder(
		inspector_dto.Config{
			BaseDir:    "/tmp/test",
			ModuleName: "example.com/test",
		},
		inspector_domain.WithBuilderCacheKeyGenerator(&stubKeyGenerator{key: "test-key"}),
		inspector_domain.WithParser(&stubParser{files: map[string]*goast.File{}}),
		inspector_domain.WithBuilderPackageLoader(&stubLoader{pkgs: []*packages.Package{}}),
		inspector_domain.WithEncoder(&stubEncoder{
			data: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{},
			},
		}),
	)
}

func newFailingTestTypeBuilder(buildErr error) *inspector_domain.TypeBuilder {
	return inspector_domain.NewTypeBuilder(
		inspector_dto.Config{
			BaseDir:    "/tmp/test",
			ModuleName: "example.com/test",
		},
		inspector_domain.WithBuilderCacheKeyGenerator(&stubKeyGenerator{key: "fail-key"}),
		inspector_domain.WithParser(&stubParser{files: map[string]*goast.File{}}),
		inspector_domain.WithBuilderPackageLoader(&stubLoader{err: buildErr}),
		inspector_domain.WithEncoder(&stubEncoder{
			data: &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{},
			},
		}),
	)
}

func TestNewTypeInspectorBuilderAdapter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		builder *inspector_domain.TypeBuilder
		name    string
	}{
		{
			name:    "wraps a valid TypeBuilder",
			builder: newTestTypeBuilder(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			adapter := NewTypeInspectorBuilderAdapter(tc.builder)

			require.NotNil(t, adapter)
			assert.Equal(t, tc.builder, adapter.builder)
		})
	}
}

func TestTypeInspectorBuilderAdapter_ImplementsPort(t *testing.T) {
	t.Parallel()

	var _ TypeInspectorBuilderPort = (*TypeInspectorBuilderAdapter)(nil)
}

func TestTypeInspectorBuilderAdapter_SetConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		config inspector_dto.Config
	}{
		{
			name: "delegates config with base directory and module name",
			config: inspector_dto.Config{
				BaseDir:    "/home/user/project",
				ModuleName: "example.com/myapp",
			},
		},
		{
			name:   "delegates empty config without panicking",
			config: inspector_dto.Config{},
		},
		{
			name: "delegates config with all fields populated",
			config: inspector_dto.Config{
				BaseDir:    "/srv/app",
				ModuleName: "example.com/fullapp",
				GOOS:       "linux",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			builder := newTestTypeBuilder()
			adapter := NewTypeInspectorBuilderAdapter(builder)

			assert.NotPanics(t, func() {
				adapter.SetConfig(tc.config)
			})
		})
	}
}

func TestTypeInspectorBuilderAdapter_Build(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		builder       *inspector_domain.TypeBuilder
		sourceOverlay map[string][]byte
		scriptHashes  map[string]string
		name          string
		errSubstring  string
		wantErr       bool
	}{
		{
			name:          "successful build delegates and returns nil error",
			builder:       newTestTypeBuilder(),
			sourceOverlay: map[string][]byte{},
			scriptHashes:  map[string]string{},
			wantErr:       false,
		},
		{
			name:          "build with nil overlays delegates without error",
			builder:       newTestTypeBuilder(),
			sourceOverlay: nil,
			scriptHashes:  nil,
			wantErr:       false,
		},
		{
			name:          "build propagates error from underlying builder",
			builder:       newFailingTestTypeBuilder(errors.New("loader failed: connection refused")),
			sourceOverlay: map[string][]byte{},
			scriptHashes:  map[string]string{},
			wantErr:       true,
			errSubstring:  "loader failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			adapter := NewTypeInspectorBuilderAdapter(tc.builder)
			err := adapter.Build(context.Background(), tc.sourceOverlay, tc.scriptHashes)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstring)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTypeInspectorBuilderAdapter_GetQuerier(t *testing.T) {
	t.Parallel()

	t.Run("returns nil and false before Build is called", func(t *testing.T) {
		t.Parallel()

		builder := newTestTypeBuilder()
		adapter := NewTypeInspectorBuilderAdapter(builder)

		querier, ok := adapter.GetQuerier()

		assert.Nil(t, querier)
		assert.False(t, ok)
	})

	t.Run("returns querier and true after successful Build", func(t *testing.T) {
		t.Parallel()

		builder := newTestTypeBuilder()
		adapter := NewTypeInspectorBuilderAdapter(builder)

		err := adapter.Build(context.Background(), map[string][]byte{}, map[string]string{})
		require.NoError(t, err)

		querier, ok := adapter.GetQuerier()

		require.True(t, ok)
		assert.NotNil(t, querier)

		_ = querier
	})

	t.Run("returns nil and false after failed Build", func(t *testing.T) {
		t.Parallel()

		builder := newFailingTestTypeBuilder(errors.New("intentional build failure"))
		adapter := NewTypeInspectorBuilderAdapter(builder)

		err := adapter.Build(context.Background(), map[string][]byte{}, map[string]string{})
		require.Error(t, err)

		querier, ok := adapter.GetQuerier()

		assert.Nil(t, querier)
		assert.False(t, ok)
	})
}
