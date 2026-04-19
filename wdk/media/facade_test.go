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

package media_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/media"
)

func TestFitConstants_AreDistinct(t *testing.T) {
	t.Parallel()

	all := []media.FitMode{
		media.FitCover,
		media.FitContain,
		media.FitFill,
		media.FitInside,
		media.FitOutside,
	}

	seen := make(map[media.FitMode]struct{}, len(all))
	for _, mode := range all {
		_, dup := seen[mode]
		require.Falsef(t, dup, "duplicate fit mode: %v", mode)
		seen[mode] = struct{}{}
	}
}

func TestNewService_ReturnsService(t *testing.T) {
	t.Parallel()

	service := media.NewService("none")

	require.NotNil(t, service)
}

func TestNewTransformBuilder_ReturnsBuilder(t *testing.T) {
	t.Parallel()

	service := media.NewService("none")
	builder := media.NewTransformBuilder(service, strings.NewReader("fake-image"))

	require.NotNil(t, builder)
}

func TestImage_ReturnsConfigBuilder(t *testing.T) {
	t.Parallel()

	require.NotNil(t, media.Image())
}

func TestVariant_ReturnsVariantBuilder(t *testing.T) {
	t.Parallel()

	require.NotNil(t, media.Variant())
}

func TestDefaultTransformationSpec_ReturnsSpec(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		_ = media.DefaultTransformationSpec()
	})
}

func TestDefaultImageServiceConfig_ReturnsConfig(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		_ = media.DefaultImageServiceConfig()
	})
}

func TestGetDefaultService_FailsWhenFrameworkNotBootstrapped(t *testing.T) {
	t.Parallel()

	_, err := media.GetDefaultService()

	require.Error(t, err)
}

func TestGetVariantSpec_ReturnsFalseWhenFrameworkNotBootstrapped(t *testing.T) {
	t.Parallel()

	_, ok := media.GetVariantSpec("nonexistent-variant")

	require.False(t, ok)
}

func TestGetImageDimensions_FailsWhenFrameworkNotBootstrapped(t *testing.T) {
	t.Parallel()

	_, _, err := media.GetImageDimensions(t.Context(), strings.NewReader("fake-image"))

	require.Error(t, err)
}

func TestVideoErrorSentinels_AreNonNil(t *testing.T) {
	t.Parallel()

	cases := []error{
		media.ErrUnsupportedCodec,
		media.ErrUnsupportedFormat,
		media.ErrInvalidResolution,
		media.ErrInvalidBitrate,
		media.ErrInvalidFramerate,
		media.ErrDurationExceedsLimit,
		media.ErrFileSizeExceedsLimit,
		media.ErrResolutionExceedsLimit,
		media.ErrTranscodingFailed,
		media.ErrInvalidStream,
		media.ErrContextCancelled,
		media.ErrTimeout,
		media.ErrResourceExhausted,
		media.ErrInvalidHLSSpec,
		media.ErrSegmentationFailed,
	}

	for _, err := range cases {
		require.NotNil(t, err)
		require.NotEmpty(t, err.Error())
	}
}
