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

package capabilities

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/compiler/compiler_dto"
	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/internal/image/image_dto"
	"piko.sh/piko/internal/video/video_domain"
	"piko.sh/piko/internal/video/video_dto"
)

type stubCompiler struct{}

func (s *stubCompiler) CompileSingle(_ context.Context, _ string) (*compiler_dto.CompiledArtefact, error) {
	return nil, nil
}

func (s *stubCompiler) CompileSFCBytes(_ context.Context, _ string, _ []byte) (*compiler_dto.CompiledArtefact, error) {
	return nil, nil
}

type stubImageService struct{}

func (s *stubImageService) Transform(_ io.Reader) *image_domain.TransformBuilder { return nil }
func (s *stubImageService) TransformStream(_ context.Context, _ io.Reader, _ image_dto.TransformationSpec) (*image_dto.TransformedImageResult, error) {
	return nil, nil
}
func (s *stubImageService) GenerateResponsiveVariants(_ context.Context, _ io.Reader, _ image_dto.TransformationSpec) ([]image_dto.ResponsiveVariant, error) {
	return nil, nil
}
func (s *stubImageService) GeneratePlaceholder(_ context.Context, _ io.Reader, _ image_dto.TransformationSpec) (string, error) {
	return "", nil
}
func (s *stubImageService) GetDimensions(_ context.Context, _ io.Reader) (int, int, error) {
	return 0, 0, nil
}

type stubVideoService struct{}

func (s *stubVideoService) Transcode(_ context.Context, _ io.Reader, _ map[string]string) (io.ReadCloser, error) {
	return nil, nil
}
func (s *stubVideoService) ExtractCapabilities(_ context.Context, _ io.Reader) (video_dto.VideoCapabilities, error) {
	return video_dto.VideoCapabilities{}, nil
}
func (s *stubVideoService) TranscodeHLS(_ context.Context, _ io.Reader, _ video_dto.HLSSpec) (video_dto.HLSResult, error) {
	return video_dto.HLSResult{}, nil
}
func (s *stubVideoService) ExtractThumbnail(_ context.Context, _ io.Reader, _ video_dto.ThumbnailSpec) (io.ReadCloser, error) {
	return nil, nil
}

func TestApplyOptions_Empty(t *testing.T) {
	t.Parallel()

	deps := applyOptions(nil)

	require.NotNil(t, deps)
	assert.Nil(t, deps.compiler)
	assert.Nil(t, deps.imageTransformer)
	assert.Nil(t, deps.videoTranscoder)
}

func TestApplyOptions_WithCompiler(t *testing.T) {
	t.Parallel()

	c := &stubCompiler{}
	deps := applyOptions([]Option{WithCompiler(c)})

	assert.Equal(t, compiler_domain.CompilerService(c), deps.compiler)
	assert.Nil(t, deps.imageTransformer)
	assert.Nil(t, deps.videoTranscoder)
}

func TestApplyOptions_WithImageProvider(t *testing.T) {
	t.Parallel()

	img := &stubImageService{}
	deps := applyOptions([]Option{WithImageProvider(img)})

	assert.Nil(t, deps.compiler)
	assert.Equal(t, image_domain.Service(img), deps.imageTransformer)
	assert.Nil(t, deps.videoTranscoder)
}

func TestApplyOptions_WithVideoProvider(t *testing.T) {
	t.Parallel()

	vid := &stubVideoService{}
	deps := applyOptions([]Option{WithVideoProvider(vid)})

	assert.Nil(t, deps.compiler)
	assert.Nil(t, deps.imageTransformer)
	assert.Equal(t, video_domain.Service(vid), deps.videoTranscoder)
}

func TestApplyOptions_Multiple(t *testing.T) {
	t.Parallel()

	c := &stubCompiler{}
	img := &stubImageService{}
	vid := &stubVideoService{}

	deps := applyOptions([]Option{
		WithCompiler(c),
		WithImageProvider(img),
		WithVideoProvider(vid),
	})

	assert.Equal(t, compiler_domain.CompilerService(c), deps.compiler)
	assert.Equal(t, image_domain.Service(img), deps.imageTransformer)
	assert.Equal(t, video_domain.Service(vid), deps.videoTranscoder)
}

func TestCalculateCapacity_NoDeps(t *testing.T) {
	t.Parallel()

	deps := &dependencies{}
	capacity := calculateCapacity(deps)

	assert.Equal(t, len(builtinCapabilities), capacity)
}

func TestCalculateCapacity_WithCompiler(t *testing.T) {
	t.Parallel()

	deps := &dependencies{compiler: &stubCompiler{}}
	capacity := calculateCapacity(deps)

	assert.Equal(t, len(builtinCapabilities)+1, capacity)
}

func TestCalculateCapacity_WithImage(t *testing.T) {
	t.Parallel()

	deps := &dependencies{imageTransformer: &stubImageService{}}
	capacity := calculateCapacity(deps)

	assert.Equal(t, len(builtinCapabilities)+1, capacity)
}

func TestCalculateCapacity_WithVideo(t *testing.T) {
	t.Parallel()

	deps := &dependencies{videoTranscoder: &stubVideoService{}}
	capacity := calculateCapacity(deps)

	assert.Equal(t, len(builtinCapabilities)+2, capacity)
}

func TestCalculateCapacity_AllDeps(t *testing.T) {
	t.Parallel()

	deps := &dependencies{
		compiler:         &stubCompiler{},
		imageTransformer: &stubImageService{},
		videoTranscoder:  &stubVideoService{},
	}
	capacity := calculateCapacity(deps)

	assert.Equal(t, len(builtinCapabilities)+4, capacity)
}

func TestNewServiceWithBuiltins_NoDeps(t *testing.T) {
	t.Parallel()

	service, err := NewServiceWithBuiltins()

	require.NoError(t, err)
	assert.NotNil(t, service)
}

func TestNewServiceWithBuiltins_AllDeps(t *testing.T) {
	t.Parallel()

	service, err := NewServiceWithBuiltins(
		WithCompiler(&stubCompiler{}),
		WithImageProvider(&stubImageService{}),
		WithVideoProvider(&stubVideoService{}),
	)

	require.NoError(t, err)
	assert.NotNil(t, service)
}
