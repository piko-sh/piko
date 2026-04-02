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

package lifecycle_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/config"
)

func Test_AssetPipelineOrchestrator_generateVideoProfiles(t *testing.T) {
	t.Parallel()

	t.Run("generates profiles for default qualities", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewAssetPipelineOrchestrator(nil, nil)

		asset := &annotator_dto.FinalAssetDependency{
			SourcePath:           "videos/hero.mp4",
			AssetType:            "video",
			TransformationParams: map[string][]string{},
		}

		profiles := orchestrator.generateVideoProfiles(asset)

		require.Len(t, profiles, 3)
		profileNames := make([]string, len(profiles))
		for i, p := range profiles {
			profileNames[i] = p.Name
		}
		assert.Contains(t, profileNames, "hls_1080p")
		assert.Contains(t, profileNames, "hls_720p")
		assert.Contains(t, profileNames, "hls_480p")
	})

	t.Run("generates profiles with explicit qualities", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewAssetPipelineOrchestrator(nil, nil)

		asset := &annotator_dto.FinalAssetDependency{
			SourcePath: "videos/clip.mp4",
			AssetType:  "video",
			TransformationParams: map[string][]string{
				"qualities": {"720p,480p"},
			},
		}

		profiles := orchestrator.generateVideoProfiles(asset)

		require.Len(t, profiles, 2)
		profileNames := make([]string, len(profiles))
		for i, p := range profiles {
			profileNames[i] = p.Name
		}
		assert.Contains(t, profileNames, "hls_720p")
		assert.Contains(t, profileNames, "hls_480p")
	})

	t.Run("generates profiles with custom segment duration", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewAssetPipelineOrchestrator(nil, nil)

		asset := &annotator_dto.FinalAssetDependency{
			SourcePath: "videos/clip.mp4",
			AssetType:  "video",
			TransformationParams: map[string][]string{
				"segment-duration": {"6"},
				"qualities":        {"480p"},
			},
		}

		profiles := orchestrator.generateVideoProfiles(asset)

		require.Len(t, profiles, 1)
		segDuration, ok := profiles[0].Profile.Params.GetByName("segment_duration")
		assert.True(t, ok)
		assert.Equal(t, "6", segDuration)
	})

	t.Run("processes video assets through ProcessBuildResult", func(t *testing.T) {
		t.Parallel()

		mockRegistry, calls := newPipelineRegistryMock(nil)
		orchestrator := NewAssetPipelineOrchestrator(mockRegistry, nil)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
				{
					SourcePath: "videos/intro.mp4",
					AssetType:  "video",
					TransformationParams: map[string][]string{
						"qualities": {"1080p"},
					},
				},
			},
		}

		err := orchestrator.ProcessBuildResult(context.Background(), result)
		assert.NoError(t, err)
		require.Len(t, *calls, 1)
		assert.Equal(t, "videos/intro.mp4", (*calls)[0].artefactID)
		assert.Len(t, (*calls)[0].profiles, 1)
	})
}

func Test_AssetPipelineOrchestrator_parseVideoQualities(t *testing.T) {
	t.Parallel()

	t.Run("returns fallback qualities when no params", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewAssetPipelineOrchestrator(nil, nil)

		result := orchestrator.parseVideoQualities(map[string][]string{})

		assert.Equal(t, []string{"1080p", "720p", "480p"}, result)
	})

	t.Run("returns config default qualities when available", func(t *testing.T) {
		t.Parallel()

		assetsConfig := &config.AssetsConfig{
			Image: config.ImageAssetsConfig{},
			Video: config.VideoAssetsConfig{
				DefaultQualities:       []string{"720p", "360p"},
				DefaultSegmentDuration: 0,
			},
		}
		orchestrator := NewAssetPipelineOrchestrator(nil, assetsConfig)

		result := orchestrator.parseVideoQualities(map[string][]string{})

		assert.Equal(t, []string{"720p", "360p"}, result)
	})

	t.Run("parses explicit qualities from params", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewAssetPipelineOrchestrator(nil, nil)

		params := map[string][]string{
			"qualities": {"1080p,720p,480p"},
		}

		result := orchestrator.parseVideoQualities(params)

		assert.Equal(t, []string{"1080p", "720p", "480p"}, result)
	})

	t.Run("filters out unknown quality levels", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewAssetPipelineOrchestrator(nil, nil)

		params := map[string][]string{
			"qualities": {"1080p,unknown,480p"},
		}

		result := orchestrator.parseVideoQualities(params)

		assert.Equal(t, []string{"1080p", "480p"}, result)
	})

	t.Run("falls back to defaults when all qualities unknown", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewAssetPipelineOrchestrator(nil, nil)

		params := map[string][]string{
			"qualities": {"8k,16k"},
		}

		result := orchestrator.parseVideoQualities(params)

		assert.Equal(t, []string{"1080p", "720p", "480p"}, result)
	})

	t.Run("trims whitespace around quality names", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewAssetPipelineOrchestrator(nil, nil)

		params := map[string][]string{
			"qualities": {" 720p , 480p "},
		}

		result := orchestrator.parseVideoQualities(params)

		assert.Equal(t, []string{"720p", "480p"}, result)
	})

	t.Run("handles single quality value", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewAssetPipelineOrchestrator(nil, nil)

		params := map[string][]string{
			"qualities": {"360p"},
		}

		result := orchestrator.parseVideoQualities(params)

		assert.Equal(t, []string{"360p"}, result)
	})
}

func Test_AssetPipelineOrchestrator_parseSegmentDuration(t *testing.T) {
	t.Parallel()

	t.Run("returns fallback when no params and no config", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewAssetPipelineOrchestrator(nil, nil)

		result := orchestrator.parseSegmentDuration(map[string][]string{})

		assert.Equal(t, 10, result)
	})

	t.Run("returns config default when available", func(t *testing.T) {
		t.Parallel()

		assetsConfig := &config.AssetsConfig{
			Image: config.ImageAssetsConfig{},
			Video: config.VideoAssetsConfig{
				DefaultQualities:       nil,
				DefaultSegmentDuration: 8,
			},
		}
		orchestrator := NewAssetPipelineOrchestrator(nil, assetsConfig)

		result := orchestrator.parseSegmentDuration(map[string][]string{})

		assert.Equal(t, 8, result)
	})

	t.Run("parses explicit segment duration from params", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewAssetPipelineOrchestrator(nil, nil)

		params := map[string][]string{
			"segment-duration": {"6"},
		}

		result := orchestrator.parseSegmentDuration(params)

		assert.Equal(t, 6, result)
	})

	t.Run("falls back on invalid segment duration", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewAssetPipelineOrchestrator(nil, nil)

		params := map[string][]string{
			"segment-duration": {"abc"},
		}

		result := orchestrator.parseSegmentDuration(params)

		assert.Equal(t, 10, result)
	})

	t.Run("falls back on zero segment duration", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewAssetPipelineOrchestrator(nil, nil)

		params := map[string][]string{
			"segment-duration": {"0"},
		}

		result := orchestrator.parseSegmentDuration(params)

		assert.Equal(t, 10, result)
	})

	t.Run("falls back on negative segment duration", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewAssetPipelineOrchestrator(nil, nil)

		params := map[string][]string{
			"segment-duration": {"-5"},
		}

		result := orchestrator.parseSegmentDuration(params)

		assert.Equal(t, 10, result)
	})

	t.Run("param takes precedence over config default", func(t *testing.T) {
		t.Parallel()

		assetsConfig := &config.AssetsConfig{
			Image: config.ImageAssetsConfig{},
			Video: config.VideoAssetsConfig{
				DefaultQualities:       nil,
				DefaultSegmentDuration: 15,
			},
		}
		orchestrator := NewAssetPipelineOrchestrator(nil, assetsConfig)

		params := map[string][]string{
			"segment-duration": {"4"},
		}

		result := orchestrator.parseSegmentDuration(params)

		assert.Equal(t, 4, result)
	})
}

func Test_AssetPipelineOrchestrator_buildVideoProfiles(t *testing.T) {
	t.Parallel()

	t.Run("builds profiles for valid qualities", func(t *testing.T) {
		t.Parallel()

		orchestrator := &AssetPipelineOrchestrator{
			registryService: nil,
			assetsConfig:    nil,
		}

		profiles := orchestrator.buildVideoProfiles([]string{"1080p", "720p"}, 10)

		require.Len(t, profiles, 2)
		assert.Equal(t, "hls_1080p", profiles[0].Name)
		assert.Equal(t, "hls_720p", profiles[1].Name)
	})

	t.Run("sets correct profile parameters", func(t *testing.T) {
		t.Parallel()

		orchestrator := &AssetPipelineOrchestrator{
			registryService: nil,
			assetsConfig:    nil,
		}

		profiles := orchestrator.buildVideoProfiles([]string{"1080p"}, 10)

		require.Len(t, profiles, 1)

		resolution, ok := profiles[0].Profile.Params.GetByName("resolution")
		assert.True(t, ok)
		assert.Equal(t, "1920x1080", resolution)

		bitrate, ok := profiles[0].Profile.Params.GetByName("bitrate")
		assert.True(t, ok)
		assert.Equal(t, "5000k", bitrate)

		segDuration, ok := profiles[0].Profile.Params.GetByName("segment_duration")
		assert.True(t, ok)
		assert.Equal(t, "10", segDuration)
	})

	t.Run("sets correct tags", func(t *testing.T) {
		t.Parallel()

		orchestrator := &AssetPipelineOrchestrator{
			registryService: nil,
			assetsConfig:    nil,
		}

		profiles := orchestrator.buildVideoProfiles([]string{"720p"}, 6)

		require.Len(t, profiles, 1)

		tagType, ok := profiles[0].Profile.ResultingTags.GetByName("Type")
		assert.True(t, ok)
		assert.Equal(t, "video-variant", tagType)

		quality, ok := profiles[0].Profile.ResultingTags.GetByName("quality")
		assert.True(t, ok)
		assert.Equal(t, "720p", quality)

		mimeType, ok := profiles[0].Profile.ResultingTags.GetByName("mimeType")
		assert.True(t, ok)
		assert.Equal(t, "application/x-mpegURL", mimeType)
	})

	t.Run("skips unknown quality levels", func(t *testing.T) {
		t.Parallel()

		orchestrator := &AssetPipelineOrchestrator{
			registryService: nil,
			assetsConfig:    nil,
		}

		profiles := orchestrator.buildVideoProfiles([]string{"1080p", "unknown", "480p"}, 10)

		require.Len(t, profiles, 2)
		assert.Equal(t, "hls_1080p", profiles[0].Name)
		assert.Equal(t, "hls_480p", profiles[1].Name)
	})

	t.Run("returns empty for no valid qualities", func(t *testing.T) {
		t.Parallel()

		orchestrator := &AssetPipelineOrchestrator{
			registryService: nil,
			assetsConfig:    nil,
		}

		profiles := orchestrator.buildVideoProfiles([]string{"8k", "16k"}, 10)

		assert.Empty(t, profiles)
	})

	t.Run("returns empty for nil qualities", func(t *testing.T) {
		t.Parallel()

		orchestrator := &AssetPipelineOrchestrator{
			registryService: nil,
			assetsConfig:    nil,
		}

		profiles := orchestrator.buildVideoProfiles(nil, 10)

		assert.Empty(t, profiles)
	})

	t.Run("builds all four quality levels", func(t *testing.T) {
		t.Parallel()

		orchestrator := &AssetPipelineOrchestrator{
			registryService: nil,
			assetsConfig:    nil,
		}

		profiles := orchestrator.buildVideoProfiles([]string{"1080p", "720p", "480p", "360p"}, 10)

		require.Len(t, profiles, 4)
		assert.Equal(t, "hls_1080p", profiles[0].Name)
		assert.Equal(t, "hls_720p", profiles[1].Name)
		assert.Equal(t, "hls_480p", profiles[2].Name)
		assert.Equal(t, "hls_360p", profiles[3].Name)

		res360, _ := profiles[3].Profile.Params.GetByName("resolution")
		assert.Equal(t, "640x360", res360)

		br360, _ := profiles[3].Profile.Params.GetByName("bitrate")
		assert.Equal(t, "500k", br360)
	})

	t.Run("uses custom segment duration", func(t *testing.T) {
		t.Parallel()

		orchestrator := &AssetPipelineOrchestrator{
			registryService: nil,
			assetsConfig:    nil,
		}

		profiles := orchestrator.buildVideoProfiles([]string{"480p"}, 4)

		require.Len(t, profiles, 1)
		segDuration, ok := profiles[0].Profile.Params.GetByName("segment_duration")
		assert.True(t, ok)
		assert.Equal(t, "4", segDuration)
	})

	t.Run("sets correct capability name", func(t *testing.T) {
		t.Parallel()

		orchestrator := &AssetPipelineOrchestrator{
			registryService: nil,
			assetsConfig:    nil,
		}

		profiles := orchestrator.buildVideoProfiles([]string{"720p"}, 10)

		require.Len(t, profiles, 1)
		assert.Equal(t, "video-transcode", profiles[0].Profile.CapabilityName)
	})
}
