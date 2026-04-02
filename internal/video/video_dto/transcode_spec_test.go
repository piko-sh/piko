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

package video_dto

import (
	"strings"
	"testing"
)

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]string
		key     string
		want    int
		wantErr bool
	}{
		{name: "present", params: map[string]string{"w": "1920"}, key: "w", want: 1920, wantErr: false},
		{name: "missing", params: map[string]string{}, key: "w", want: 0, wantErr: false},
		{name: "empty", params: map[string]string{"w": ""}, key: "w", want: 0, wantErr: false},
		{name: "invalid", params: map[string]string{"w": "abc"}, key: "w", want: 0, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIntParam(tt.params, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseIntPtrParam(t *testing.T) {
	got, err := parseIntPtrParam(map[string]string{"crf": "23"}, "crf")
	if err != nil || got == nil || *got != 23 {
		t.Errorf("parseIntPtrParam(23) = %v, %v", got, err)
	}

	got, err = parseIntPtrParam(map[string]string{}, "crf")
	if err != nil || got != nil {
		t.Errorf("parseIntPtrParam(missing) = %v, %v; want nil, nil", got, err)
	}

	_, err = parseIntPtrParam(map[string]string{"crf": "bad"}, "crf")
	if err == nil {
		t.Error("expected error for invalid value")
	}
}

func TestParseFloatParam(t *testing.T) {
	got, err := parseFloatParam(map[string]string{"fps": "29.97"}, "fps")
	if err != nil || got != 29.97 {
		t.Errorf("parseFloatParam(29.97) = %v, %v", got, err)
	}

	got, err = parseFloatParam(map[string]string{}, "fps")
	if err != nil || got != 0 {
		t.Errorf("parseFloatParam(missing) = %v, %v", got, err)
	}

	_, err = parseFloatParam(map[string]string{"fps": "bad"}, "fps")
	if err == nil {
		t.Error("expected error for invalid value")
	}
}

func TestParseLowerStringParam(t *testing.T) {
	if got := parseLowerStringParam(map[string]string{"c": "H264"}, "c"); got != "h264" {
		t.Errorf("got %q, want %q", got, "h264")
	}
	if got := parseLowerStringParam(map[string]string{}, "c"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestParseTranscodeSpec(t *testing.T) {
	params := map[string]string{
		"codec":       "H264",
		"width":       "1920",
		"height":      "1080",
		"bitrate":     "5000000",
		"framerate":   "30",
		"preset":      "fast",
		"description": "HD transcode",
	}

	spec, err := ParseTranscodeSpec(params)
	if err != nil {
		t.Fatalf("ParseTranscodeSpec: %v", err)
	}
	if spec.Codec != "h264" {
		t.Errorf("Codec = %q, want h264", spec.Codec)
	}
	if spec.Width != 1920 || spec.Height != 1080 {
		t.Errorf("dimensions = %dx%d, want 1920x1080", spec.Width, spec.Height)
	}
	if spec.Bitrate != 5000000 {
		t.Errorf("Bitrate = %d, want 5000000", spec.Bitrate)
	}
	if spec.Description != "HD transcode" {
		t.Errorf("Description = %q", spec.Description)
	}
}

func TestParseTranscodeSpec_MissingCodec(t *testing.T) {
	_, err := ParseTranscodeSpec(map[string]string{})
	if err == nil {
		t.Error("expected error for missing codec")
	}
}

func TestParseTranscodeSpec_InvalidParam(t *testing.T) {
	_, err := ParseTranscodeSpec(map[string]string{"codec": "h264", "width": "bad"})
	if err == nil {
		t.Error("expected error for invalid width")
	}
}

func TestTranscodeSpec_InferFormat(t *testing.T) {
	tests := []struct {
		codec  string
		format string
		want   string
	}{
		{codec: "h264", format: "", want: "mp4"},
		{codec: "vp9", format: "", want: "webm"},
		{codec: "av1", format: "", want: "webm"},
		{codec: "h264", format: "mkv", want: "mkv"},
	}
	for _, tt := range tests {
		spec := &TranscodeSpec{Codec: tt.codec, Format: tt.format}
		if got := spec.InferFormat(); got != tt.want {
			t.Errorf("InferFormat(codec=%s, format=%s) = %q, want %q", tt.codec, tt.format, got, tt.want)
		}
	}
}

func TestTranscodeSpec_InferAudioCodec(t *testing.T) {
	tests := []struct {
		codec      string
		audioCodec string
		want       string
	}{
		{codec: "h264", audioCodec: "", want: "aac"},
		{codec: "vp9", audioCodec: "", want: "opus"},
		{codec: "h264", audioCodec: "flac", want: "flac"},
	}
	for _, tt := range tests {
		spec := &TranscodeSpec{Codec: tt.codec, AudioCodec: tt.audioCodec}
		if got := spec.InferAudioCodec(); got != tt.want {
			t.Errorf("InferAudioCodec(codec=%s, audio=%s) = %q, want %q", tt.codec, tt.audioCodec, got, tt.want)
		}
	}
}

func TestTranscodeSpec_Validate(t *testing.T) {
	valid := &TranscodeSpec{Codec: "h264", Width: 1920, Height: 1080}
	if err := valid.Validate(); err != nil {
		t.Errorf("valid spec: %v", err)
	}
}

func TestTranscodeSpec_Validate_Errors(t *testing.T) {
	tests := []struct {
		name string
		want string
		spec TranscodeSpec
	}{
		{name: "bad codec", want: "unsupported codec", spec: TranscodeSpec{Codec: "bad"}},
		{name: "negative width", want: "non-negative", spec: TranscodeSpec{Codec: "h264", Width: -1}},
		{name: "width without height", want: "together", spec: TranscodeSpec{Codec: "h264", Width: 100}},
		{name: "negative bitrate", want: "bitrate", spec: TranscodeSpec{Codec: "h264", Bitrate: -1}},
		{name: "negative framerate", want: "framerate", spec: TranscodeSpec{Codec: "h264", Framerate: -1}},
		{name: "negative audio bitrate", want: "audio bitrate", spec: TranscodeSpec{Codec: "h264", AudioBitrate: -1}},
		{name: "bad preset", want: "invalid preset", spec: TranscodeSpec{Codec: "h264", Preset: "invalid"}},
		{name: "crf too high", want: "CRF", spec: TranscodeSpec{Codec: "h264", CRF: new(99)}},
		{name: "negative segment", want: "segment duration", spec: TranscodeSpec{Codec: "h264", SegmentDuration: new(-1)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.want)) {
				t.Errorf("error %q should contain %q", err.Error(), tt.want)
			}
		})
	}
}
