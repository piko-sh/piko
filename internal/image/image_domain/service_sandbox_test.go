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

package image_domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/safedisk"
)

func TestLoadFallbackIcons_WithInjectedSandbox(t *testing.T) {
	t.Parallel()

	t.Run("loads icons with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/icons", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		require.NoError(t, sandbox.WriteFile("pdf.svg", []byte("<svg>pdf</svg>"), 0644))
		require.NoError(t, sandbox.WriteFile("default.svg", []byte("<svg>default</svg>"), 0644))

		paths := map[string]string{
			"application/pdf": "/icons/pdf.svg",
			"default":         "/icons/default.svg",
		}

		icons, err := loadFallbackIcons(paths, sandbox, nil)

		require.NoError(t, err)
		assert.Len(t, icons, 2)
		assert.Equal(t, []byte("<svg>pdf</svg>"), icons["application/pdf"])
		assert.Equal(t, []byte("<svg>default</svg>"), icons["default"])
	})

	t.Run("returns error when ReadFile fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/icons", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		require.NoError(t, sandbox.WriteFile("pdf.svg", []byte("<svg>pdf</svg>"), 0644))
		sandbox.ReadFileErr = errors.New("disk read error")

		paths := map[string]string{
			"application/pdf": "/icons/pdf.svg",
		}

		icons, err := loadFallbackIcons(paths, sandbox, nil)

		require.Error(t, err)
		assert.Nil(t, icons)
		assert.Contains(t, err.Error(), "could not read icon file")
		assert.Contains(t, err.Error(), "disk read error")
	})

	t.Run("returns error when file not found", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/icons", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()

		paths := map[string]string{
			"application/pdf": "/icons/nonexistent.svg",
		}

		icons, err := loadFallbackIcons(paths, sandbox, nil)

		require.Error(t, err)
		assert.Nil(t, icons)
		assert.Contains(t, err.Error(), "could not read icon file")
	})

	t.Run("handles empty paths map", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/icons", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()

		paths := map[string]string{}

		icons, err := loadFallbackIcons(paths, sandbox, nil)

		require.NoError(t, err)
		assert.Empty(t, icons)
	})

	t.Run("loads multiple icons successfully", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/icons", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		require.NoError(t, sandbox.WriteFile("pdf.svg", []byte("pdf-content"), 0644))
		require.NoError(t, sandbox.WriteFile("doc.svg", []byte("doc-content"), 0644))
		require.NoError(t, sandbox.WriteFile("image.svg", []byte("image-content"), 0644))

		paths := map[string]string{
			"application/pdf":    "/icons/pdf.svg",
			"application/msword": "/icons/doc.svg",
			"image/":             "/icons/image.svg",
		}

		icons, err := loadFallbackIcons(paths, sandbox, nil)

		require.NoError(t, err)
		assert.Len(t, icons, 3)
		assert.Equal(t, []byte("pdf-content"), icons["application/pdf"])
		assert.Equal(t, []byte("doc-content"), icons["application/msword"])
		assert.Equal(t, []byte("image-content"), icons["image/"])
	})
}

func TestNewService_WithFallbackIconSandbox(t *testing.T) {
	t.Parallel()

	t.Run("creates service with injected sandbox for fallback icons", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/icons", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		require.NoError(t, sandbox.WriteFile("default.svg", []byte("<svg/>"), 0644))

		mockTransformer := newMockTransformer()
		transformers := map[string]TransformerPort{
			"mock": mockTransformer,
		}

		config := ServiceConfig{
			FallbackIconPaths: map[string]string{
				"default": "/icons/default.svg",
			},
			FallbackIconSandbox: sandbox,
		}

		service, err := NewService(transformers, "mock", config)

		require.NoError(t, err)
		require.NotNil(t, service)
	})

	t.Run("returns error when fallback icon loading fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/icons", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.ReadFileErr = errors.New("icon read failed")

		mockTransformer := newMockTransformer()
		transformers := map[string]TransformerPort{
			"mock": mockTransformer,
		}

		config := ServiceConfig{
			FallbackIconPaths: map[string]string{
				"default": "/icons/default.svg",
			},
			FallbackIconSandbox: sandbox,
		}

		service, err := NewService(transformers, "mock", config)

		require.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "failed to load fallback icons")
	})
}
