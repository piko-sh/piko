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

package render_domain

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestExtractPikoVideoAttrs_BasicAttributes(t *testing.T) {
	testCases := []struct {
		name       string
		attributes []ast_domain.HTMLAttribute
		expected   pikoVideoAttrs
	}{
		{
			name: "src only",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "videos/intro.mp4"},
			},
			expected: pikoVideoAttrs{
				src: "videos/intro.mp4",
			},
		},
		{
			name: "src with poster",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "videos/movie.mp4"},
				{Name: "poster", Value: "images/poster.jpg"},
			},
			expected: pikoVideoAttrs{
				src:    "videos/movie.mp4",
				poster: "images/poster.jpg",
			},
		},
		{
			name: "all quality attributes",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "videos/movie.mp4"},
				{Name: "qualities", Value: "1080p,720p"},
				{Name: "segment-duration", Value: "6"},
			},
			expected: pikoVideoAttrs{
				src:             "videos/movie.mp4",
				qualities:       "1080p,720p",
				segmentDuration: "6",
			},
		},
		{
			name: "boolean attributes",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "videos/hero.mp4"},
				{Name: "controls", Value: ""},
				{Name: "autoplay", Value: ""},
				{Name: "muted", Value: ""},
				{Name: "loop", Value: ""},
				{Name: "playsinline", Value: ""},
			},
			expected: pikoVideoAttrs{
				src:         "videos/hero.mp4",
				controls:    true,
				autoplay:    true,
				muted:       true,
				loop:        true,
				playsInline: true,
			},
		},
		{
			name: "dimension attributes",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "videos/clip.mp4"},
				{Name: "width", Value: "1280"},
				{Name: "height", Value: "720"},
			},
			expected: pikoVideoAttrs{
				src:    "videos/clip.mp4",
				width:  "1280",
				height: "720",
			},
		},
		{
			name: "preload attribute",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "videos/clip.mp4"},
				{Name: "preload", Value: "auto"},
			},
			expected: pikoVideoAttrs{
				src:     "videos/clip.mp4",
				preload: "auto",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node := &ast_domain.TemplateNode{
				TagName:    "piko:video",
				Attributes: tc.attributes,
			}

			result := extractPikoVideoAttrs(node)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractPikoVideoAttrs_LowercaseAttrs(t *testing.T) {

	node := &ast_domain.TemplateNode{
		TagName: "piko:video",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "videos/movie.mp4"},
			{Name: "poster", Value: "images/poster.jpg"},
			{Name: "controls", Value: ""},
			{Name: "autoplay", Value: ""},
		},
	}

	result := extractPikoVideoAttrs(node)

	assert.Equal(t, "videos/movie.mp4", result.src)
	assert.Equal(t, "images/poster.jpg", result.poster)
	assert.True(t, result.controls)
	assert.True(t, result.autoplay)
}

func TestPikoVideoAttrs_HasQualityProfile(t *testing.T) {
	testCases := []struct {
		name     string
		attrs    pikoVideoAttrs
		expected bool
	}{
		{
			name:     "empty attrs",
			attrs:    pikoVideoAttrs{},
			expected: false,
		},
		{
			name:     "src only",
			attrs:    pikoVideoAttrs{src: "videos/movie.mp4"},
			expected: false,
		},
		{
			name:     "with qualities",
			attrs:    pikoVideoAttrs{src: "videos/movie.mp4", qualities: "1080p,720p"},
			expected: true,
		},
		{
			name:     "with segment-duration",
			attrs:    pikoVideoAttrs{src: "videos/movie.mp4", segmentDuration: "10"},
			expected: true,
		},
		{
			name:     "with both",
			attrs:    pikoVideoAttrs{src: "videos/movie.mp4", qualities: "1080p", segmentDuration: "6"},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.attrs.hasQualityProfile()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPikoVideoAttrs_GetQualities(t *testing.T) {
	testCases := []struct {
		name     string
		attrs    pikoVideoAttrs
		expected []string
	}{
		{
			name:     "default when empty",
			attrs:    pikoVideoAttrs{},
			expected: []string{"1080p", "720p", "480p"},
		},
		{
			name:     "single quality",
			attrs:    pikoVideoAttrs{qualities: "720p"},
			expected: []string{"720p"},
		},
		{
			name:     "multiple qualities comma-separated",
			attrs:    pikoVideoAttrs{qualities: "1080p,720p,480p"},
			expected: []string{"1080p", "720p", "480p"},
		},
		{
			name:     "multiple qualities space-separated",
			attrs:    pikoVideoAttrs{qualities: "1080p 720p"},
			expected: []string{"1080p", "720p"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.attrs.getQualities()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPikoVideoAttrs_GetSegmentDuration(t *testing.T) {
	testCases := []struct {
		name     string
		attrs    pikoVideoAttrs
		expected int
	}{
		{
			name:     "default when empty",
			attrs:    pikoVideoAttrs{},
			expected: 10,
		},
		{
			name:     "custom duration",
			attrs:    pikoVideoAttrs{segmentDuration: "6"},
			expected: 6,
		},
		{
			name:     "invalid falls back to default",
			attrs:    pikoVideoAttrs{segmentDuration: "invalid"},
			expected: 10,
		},
		{
			name:     "negative falls back to default",
			attrs:    pikoVideoAttrs{segmentDuration: "-5"},
			expected: 10,
		},
		{
			name:     "zero falls back to default",
			attrs:    pikoVideoAttrs{segmentDuration: "0"},
			expected: 10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.attrs.getSegmentDuration()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsPikoVideoSpecialAttr(t *testing.T) {
	testCases := []struct {
		name          string
		attributeName string
		expected      bool
	}{

		{name: "qualities", attributeName: "qualities", expected: true},
		{name: "segment-duration", attributeName: "segment-duration", expected: true},
		{name: "poster-widths", attributeName: "poster-widths", expected: true},
		{name: "thumbnail", attributeName: "thumbnail", expected: true},
		{name: "src is not special", attributeName: "src", expected: false},
		{name: "controls is not special", attributeName: "controls", expected: false},
		{name: "class is not special", attributeName: "class", expected: false},
		{name: "poster is not special", attributeName: "poster", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isPikoVideoSpecialAttr(tc.attributeName)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRenderPikoVideo_BasicRendering(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:video",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "videos/intro.mp4"},
			{Name: "controls", Value: ""},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoVideo(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, "<video")
	assert.Contains(t, output, "id=\"piko-video-")
	assert.Contains(t, output, "controls")
	assert.Contains(t, output, "</video>")
	assert.Contains(t, output, "<script>")
	assert.Contains(t, output, "Hls")
}

func TestRenderPikoVideo_WithAllAttributes(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:video",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "videos/movie.mp4"},
			{Name: "poster", Value: "images/poster.jpg"},
			{Name: "width", Value: "1280"},
			{Name: "height", Value: "720"},
			{Name: "preload", Value: "auto"},
			{Name: "controls", Value: ""},
			{Name: "autoplay", Value: ""},
			{Name: "muted", Value: ""},
			{Name: "loop", Value: ""},
			{Name: "playsinline", Value: ""},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoVideo(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, `width="1280"`)
	assert.Contains(t, output, `height="720"`)
	assert.Contains(t, output, `poster="/_piko/assets/images/poster.jpg"`)
	assert.Contains(t, output, `preload="auto"`)
	assert.Contains(t, output, " controls")
	assert.Contains(t, output, " autoplay")
	assert.Contains(t, output, " muted")
	assert.Contains(t, output, " loop")
	assert.Contains(t, output, " playsinline")
}

func TestRenderPikoVideo_MissingSrc(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:video",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "controls", Value: ""},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoVideo(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, "<!-- piko:video error: 'src' attribute is missing -->")
	assert.Contains(t, output, "<div")
}

func TestRenderPikoVideo_FiltersSpecialAttributes(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:video",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "videos/movie.mp4"},
			{Name: "qualities", Value: "1080p,720p"},
			{Name: "segment-duration", Value: "6"},
			{Name: "class", Value: "video-player"},
			{Name: "controls", Value: ""},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoVideo(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.NotContains(t, output, `qualities=`)
	assert.NotContains(t, output, `segment-duration=`)
	assert.Contains(t, output, `class="video-player"`)
}

func TestRenderPikoVideo_HLSManifestURL(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:video",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "videos/movie.mp4"},
			{Name: "controls", Value: ""},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoVideo(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, "/_piko/video/")
	assert.Contains(t, output, "/master.m3u8")
	assert.Contains(t, output, `type="application/x-mpegURL"`)
}

func TestRenderPikoVideo_DefaultPreload(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:video",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "videos/movie.mp4"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoVideo(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, `preload="metadata"`)
}

func TestBuildVideoDesiredProfiles(t *testing.T) {
	testCases := []struct {
		name            string
		qualities       []string
		expectedNames   []string
		segmentDuration int
		expectedCount   int
	}{
		{
			name:            "single quality",
			qualities:       []string{"1080p"},
			segmentDuration: 10,
			expectedCount:   1,
			expectedNames:   []string{"hls_1080p"},
		},
		{
			name:            "multiple qualities",
			qualities:       []string{"1080p", "720p", "480p"},
			segmentDuration: 10,
			expectedCount:   3,
			expectedNames:   []string{"hls_1080p", "hls_720p", "hls_480p"},
		},
		{
			name:            "unknown quality ignored",
			qualities:       []string{"1080p", "unknown", "720p"},
			segmentDuration: 10,
			expectedCount:   2,
			expectedNames:   []string{"hls_1080p", "hls_720p"},
		},
		{
			name:            "all unknown qualities",
			qualities:       []string{"unknown1", "unknown2"},
			segmentDuration: 10,
			expectedCount:   0,
			expectedNames:   []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			profiles := buildVideoDesiredProfiles(tc.qualities, tc.segmentDuration)
			assert.Len(t, profiles, tc.expectedCount)

			for i, expectedName := range tc.expectedNames {
				if i < len(profiles) {
					assert.Equal(t, expectedName, profiles[i].Name)
					assert.Equal(t, "video.encode.hls", profiles[i].Profile.CapabilityName)
				}
			}
		})
	}
}

func TestBuildVideoDesiredProfiles_ParamsCorrect(t *testing.T) {
	profiles := buildVideoDesiredProfiles([]string{"1080p"}, 6)
	require.Len(t, profiles, 1)

	profile := profiles[0]
	assert.Equal(t, "hls_1080p", profile.Name)

	resolution, _ := profile.Profile.Params.GetByName("resolution")
	assert.Equal(t, "1920x1080", resolution)

	bitrate, _ := profile.Profile.Params.GetByName("bitrate")
	assert.Equal(t, "5000k", bitrate)

	segDuration, _ := profile.Profile.Params.GetByName("segment_duration")
	assert.Equal(t, "6", segDuration)

	quality, _ := profile.Profile.ResultingTags.GetByName("quality")
	assert.Equal(t, "1080p", quality)

	bandwidth, _ := profile.Profile.ResultingTags.GetByName("bandwidth")
	assert.Equal(t, "5000000", bandwidth)
}

func TestBuildManifestURL(t *testing.T) {
	testCases := []struct {
		name      string
		videoPath string
		expected  string
	}{
		{
			name:      "simple path",
			videoPath: "videos/movie.mp4",
			expected:  "/_piko/video/videos/movie.mp4/master.m3u8",
		},
		{
			name:      "transformed path",
			videoPath: "/_piko/assets/videos/movie.mp4",
			expected:  "/_piko/video//_piko/assets/videos/movie.mp4/master.m3u8",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildManifestURL(tc.videoPath)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildVideoCacheKey(t *testing.T) {
	testCases := []struct {
		name            string
		src             string
		expected        string
		qualities       []string
		segmentDuration int
	}{
		{
			name:            "basic key",
			src:             "videos/movie.mp4",
			qualities:       []string{"1080p", "720p"},
			segmentDuration: 10,
			expected:        "video:videos/movie.mp4:1080p,720p:10",
		},
		{
			name:            "single quality",
			src:             "clip.mp4",
			qualities:       []string{"480p"},
			segmentDuration: 6,
			expected:        "video:clip.mp4:480p:6",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildVideoCacheKey(tc.src, tc.qualities, tc.segmentDuration)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGenerateVideoID_UniqueIDs(t *testing.T) {
	seen := make(map[string]bool)

	for i := range 100 {
		id := generateVideoID()
		assert.True(t, strings.HasPrefix(id, "piko-video-"), "ID should have correct prefix, iteration %d", i)
		assert.False(t, seen[id], "ID should be unique, iteration %d", i)
		seen[id] = true
	}
}

func TestGenerateVideoID_Format(t *testing.T) {
	id := generateVideoID()
	assert.True(t, strings.HasPrefix(id, "piko-video-"))
	assert.Len(t, id, len("piko-video-")+16)
}

func TestBuildThumbnailURL(t *testing.T) {
	testCases := []struct {
		name          string
		videoPath     string
		thumbnailTime string
		expected      string
	}{
		{
			name:      "basic path without time",
			videoPath: "videos/intro.mp4",
			expected:  "/_piko/video/videos/intro.mp4/thumbnail.jpg",
		},
		{
			name:          "path with time parameter",
			videoPath:     "videos/clip.mp4",
			thumbnailTime: "5.0",
			expected:      "/_piko/video/videos/clip.mp4/thumbnail.jpg?t=5.0",
		},
		{
			name:      "nested path without time",
			videoPath: "assets/media/hero.webm",
			expected:  "/_piko/video/assets/media/hero.webm/thumbnail.jpg",
		},
		{
			name:          "empty time is omitted",
			videoPath:     "v.mp4",
			thumbnailTime: "",
			expected:      "/_piko/video/v.mp4/thumbnail.jpg",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildThumbnailURL(tc.videoPath, tc.thumbnailTime)
			assert.Equal(t, tc.expected, result)
		})
	}
}
