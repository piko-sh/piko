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

package cache_transformer_zstd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"piko.sh/piko/internal/cache/cache_dto"
)

func newTestTransformer(t *testing.T) *ZstdCacheTransformer {
	t.Helper()
	tr, err := NewZstdCacheTransformer(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create transformer: %v", err)
	}
	return tr
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Name != "zstd" {
		t.Errorf("expected name %q, got %q", "zstd", config.Name)
	}
	if config.Priority != 100 {
		t.Errorf("expected priority 100, got %d", config.Priority)
	}
	if config.Level != zstd.SpeedDefault {
		t.Errorf("expected level SpeedDefault (%d), got %d", zstd.SpeedDefault, config.Level)
	}
	if config.MaxDecompressedBytes != DefaultMaxDecompressedCacheBytes {
		t.Errorf("expected MaxDecompressedBytes %d, got %d",
			DefaultMaxDecompressedCacheBytes, config.MaxDecompressedBytes)
	}
}

func TestNewZstdCacheTransformer_DefaultConfig(t *testing.T) {
	tr := newTestTransformer(t)

	if tr.Name() != "zstd" {
		t.Errorf("expected name %q, got %q", "zstd", tr.Name())
	}
	if tr.Priority() != 100 {
		t.Errorf("expected priority 100, got %d", tr.Priority())
	}
	if tr.Type() != cache_dto.TransformerCompression {
		t.Errorf("expected type TransformerCompression, got %v", tr.Type())
	}
}

func TestNewZstdCacheTransformer_CustomConfig(t *testing.T) {
	config := Config{
		Name:     "custom-zstd",
		Priority: 50,
		Level:    zstd.SpeedBestCompression,
	}
	tr, err := NewZstdCacheTransformer(config)
	if err != nil {
		t.Fatalf("failed to create transformer: %v", err)
	}

	if tr.Name() != "custom-zstd" {
		t.Errorf("expected name %q, got %q", "custom-zstd", tr.Name())
	}
	if tr.Priority() != 50 {
		t.Errorf("expected priority 50, got %d", tr.Priority())
	}
}

func TestNewZstdCacheTransformer_ZeroValueDefaults(t *testing.T) {
	tr, err := NewZstdCacheTransformer(Config{})
	if err != nil {
		t.Fatalf("failed to create transformer: %v", err)
	}

	if tr.Name() != "zstd" {
		t.Errorf("expected default name %q, got %q", "zstd", tr.Name())
	}
	if tr.Priority() != 100 {
		t.Errorf("expected default priority 100, got %d", tr.Priority())
	}
}

func TestTransformReverse_RoundTrip(t *testing.T) {
	testCases := []struct {
		name  string
		input []byte
	}{
		{name: "small text", input: []byte("hello world")},
		{name: "large text", input: []byte(strings.Repeat("the quick brown fox jumps over the lazy dog. ", 1000))},
		{name: "binary data", input: func() []byte {
			b := make([]byte, 256)
			for i := range b {
				b[i] = byte(i)
			}
			return b
		}()},
		{name: "single byte", input: []byte{42}},
		{name: "empty input", input: []byte{}},
	}

	tr := newTestTransformer(t)
	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			compressed, err := tr.Transform(ctx, tc.input, nil)
			if err != nil {
				t.Fatalf("Transform failed: %v", err)
			}

			decompressed, err := tr.Reverse(ctx, compressed, nil)
			if err != nil {
				t.Fatalf("Reverse failed: %v", err)
			}

			if !bytes.Equal(decompressed, tc.input) {
				t.Errorf("round-trip mismatch: got %d bytes, want %d bytes", len(decompressed), len(tc.input))
			}
		})
	}
}

func TestTransformReverse_AllCompressionLevels(t *testing.T) {
	levels := []struct {
		name  string
		level zstd.EncoderLevel
	}{
		{name: "SpeedFastest", level: zstd.SpeedFastest},
		{name: "SpeedDefault", level: zstd.SpeedDefault},
		{name: "SpeedBetterCompression", level: zstd.SpeedBetterCompression},
		{name: "SpeedBestCompression", level: zstd.SpeedBestCompression},
	}

	input := []byte(strings.Repeat("compressible data with patterns ", 500))
	ctx := context.Background()

	for _, tc := range levels {
		t.Run(tc.name, func(t *testing.T) {
			tr, err := NewZstdCacheTransformer(Config{Level: tc.level})
			if err != nil {
				t.Fatalf("failed to create transformer: %v", err)
			}

			compressed, err := tr.Transform(ctx, input, nil)
			if err != nil {
				t.Fatalf("Transform failed: %v", err)
			}

			decompressed, err := tr.Reverse(ctx, compressed, nil)
			if err != nil {
				t.Fatalf("Reverse failed: %v", err)
			}

			if !bytes.Equal(decompressed, input) {
				t.Error("round-trip mismatch after decompression")
			}
		})
	}
}

func TestTransform_CompressesData(t *testing.T) {
	tr := newTestTransformer(t)
	ctx := context.Background()

	input := []byte(strings.Repeat("aaaaaaaaaa", 1000))
	compressed, err := tr.Transform(ctx, input, nil)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if len(compressed) >= len(input) {
		t.Errorf("expected compressed size (%d) < input size (%d)", len(compressed), len(input))
	}
}

func TestTransform_PerCallLevelOverride(t *testing.T) {
	tr := newTestTransformer(t)
	ctx := context.Background()

	input := []byte(strings.Repeat("test data for compression ", 500))
	options := map[string]any{"level": int(zstd.SpeedBestCompression)}

	compressed, err := tr.Transform(ctx, input, options)
	if err != nil {
		t.Fatalf("Transform failed with level override: %v", err)
	}

	decompressed, err := tr.Reverse(ctx, compressed, nil)
	if err != nil {
		t.Fatalf("Reverse failed: %v", err)
	}

	if !bytes.Equal(decompressed, input) {
		t.Error("round-trip failed with per-call level override")
	}
}

func TestTransform_InvalidOptionsFallback(t *testing.T) {
	testCases := []struct {
		options any
		name    string
	}{
		{name: "nil options", options: nil},
		{name: "wrong type", options: "not a map"},
		{name: "map without level", options: map[string]any{"other": 1}},
		{name: "level wrong type", options: map[string]any{"level": "high"}},
	}

	tr := newTestTransformer(t)
	ctx := context.Background()
	input := []byte("test data")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			compressed, err := tr.Transform(ctx, input, tc.options)
			if err != nil {
				t.Fatalf("Transform should not fail with invalid options: %v", err)
			}

			decompressed, err := tr.Reverse(ctx, compressed, nil)
			if err != nil {
				t.Fatalf("Reverse failed: %v", err)
			}

			if !bytes.Equal(decompressed, input) {
				t.Error("round-trip failed with fallback options")
			}
		})
	}
}

func TestReverse_CorruptData(t *testing.T) {
	tr := newTestTransformer(t)
	ctx := context.Background()

	testCases := []struct {
		name  string
		input []byte
	}{
		{name: "random bytes", input: []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x01, 0x02}},
		{name: "truncated compressed data", input: func() []byte {
			compressed, _ := tr.Transform(ctx, []byte("some data to compress"), nil)
			return compressed[:len(compressed)/2]
		}()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tr.Reverse(ctx, tc.input, nil)
			if err == nil {
				t.Error("expected error for corrupt data, got nil")
			}
		})
	}
}

func TestCreateTransformerFromConfig(t *testing.T) {
	testCases := []struct {
		name             string
		config           any
		expectedName     string
		expectedPriority int
	}{
		{
			name:             "nil config uses defaults",
			config:           nil,
			expectedName:     "zstd",
			expectedPriority: 100,
		},
		{
			name:             "Config struct",
			config:           Config{Name: "custom", Priority: 50, Level: zstd.SpeedFastest},
			expectedName:     "custom",
			expectedPriority: 50,
		},
		{
			name:             "map config",
			config:           map[string]any{"name": "mapped", "priority": 75},
			expectedName:     "mapped",
			expectedPriority: 75,
		},
		{
			name:             "unknown type uses defaults",
			config:           12345,
			expectedName:     "zstd",
			expectedPriority: 100,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tr, err := createTransformerFromConfig(tc.config)
			if err != nil {
				t.Fatalf("createTransformerFromConfig failed: %v", err)
			}
			if tr.Name() != tc.expectedName {
				t.Errorf("name: got %q, want %q", tr.Name(), tc.expectedName)
			}
			if tr.Priority() != tc.expectedPriority {
				t.Errorf("priority: got %d, want %d", tr.Priority(), tc.expectedPriority)
			}
		})
	}
}

func TestParseConfig(t *testing.T) {
	defaults := DefaultConfig()

	testCases := []struct {
		name             string
		config           any
		expectedName     string
		expectedPriority int
		expectedLevel    zstd.EncoderLevel
	}{
		{
			name:             "Config struct",
			config:           Config{Name: "direct", Priority: 42, Level: zstd.SpeedFastest},
			expectedName:     "direct",
			expectedPriority: 42,
			expectedLevel:    zstd.SpeedFastest,
		},
		{
			name:             "map with all fields",
			config:           map[string]any{"name": "mapped", "priority": 75, "level": 11},
			expectedName:     "mapped",
			expectedPriority: 75,
			expectedLevel:    zstd.EncoderLevel(11),
		},
		{
			name:             "map with partial fields",
			config:           map[string]any{"name": "partial"},
			expectedName:     "partial",
			expectedPriority: defaults.Priority,
			expectedLevel:    defaults.Level,
		},
		{
			name:             "unknown type returns defaults",
			config:           12345,
			expectedName:     defaults.Name,
			expectedPriority: defaults.Priority,
			expectedLevel:    defaults.Level,
		},
		{
			name:             "nil returns defaults",
			config:           nil,
			expectedName:     defaults.Name,
			expectedPriority: defaults.Priority,
			expectedLevel:    defaults.Level,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result Config
			if tc.config == nil {
				result = defaults
			} else {
				result = parseConfig(tc.config, defaults)
			}

			if result.Name != tc.expectedName {
				t.Errorf("name: got %q, want %q", result.Name, tc.expectedName)
			}
			if result.Priority != tc.expectedPriority {
				t.Errorf("priority: got %d, want %d", result.Priority, tc.expectedPriority)
			}
			if result.Level != tc.expectedLevel {
				t.Errorf("level: got %d, want %d", result.Level, tc.expectedLevel)
			}
		})
	}
}

func compressBytes(t *testing.T, payload []byte, level zstd.EncoderLevel) []byte {
	t.Helper()
	var buf bytes.Buffer
	encoder, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(level))
	if err != nil {
		t.Fatalf("failed to create test encoder: %v", err)
	}
	if _, err := encoder.Write(payload); err != nil {
		t.Fatalf("failed to write payload: %v", err)
	}
	if err := encoder.Close(); err != nil {
		t.Fatalf("failed to close encoder: %v", err)
	}
	return buf.Bytes()
}

func TestZstdCacheTransformer_Reverse_RejectsOversizeOutput(t *testing.T) {
	const byteCap = 1024
	tr, err := NewZstdCacheTransformer(
		Config{},
		WithMaxDecompressedCacheBytes(byteCap),
	)
	if err != nil {
		t.Fatalf("failed to create transformer: %v", err)
	}
	defer func() { _ = tr.Close() }()

	bomb := make([]byte, 256*1024)
	compressed := compressBytes(t, bomb, zstd.SpeedBestCompression)

	if len(compressed) > len(bomb)/100 {
		t.Logf("note: highly redundant payload compressed to %d bytes (zip-bomb shape)", len(compressed))
	}

	_, err = tr.Reverse(context.Background(), compressed, nil)
	if err == nil {
		t.Fatal("expected error for oversize decompression, got nil")
	}
	if !errors.Is(err, ErrDecompressedCacheTooLarge) {
		t.Errorf("expected ErrDecompressedCacheTooLarge, got %v", err)
	}
}

func TestZstdCacheTransformer_Reverse_AcceptsAtCap(t *testing.T) {
	const byteCap = 4096
	tr, err := NewZstdCacheTransformer(
		Config{},
		WithMaxDecompressedCacheBytes(byteCap),
	)
	if err != nil {
		t.Fatalf("failed to create transformer: %v", err)
	}
	defer func() { _ = tr.Close() }()

	payload := bytes.Repeat([]byte("x"), byteCap)
	compressed := compressBytes(t, payload, zstd.SpeedDefault)

	decompressed, err := tr.Reverse(context.Background(), compressed, nil)
	if err != nil {
		t.Fatalf("Reverse failed at cap: %v", err)
	}
	if len(decompressed) != byteCap {
		t.Errorf("decompressed length: got %d, want %d", len(decompressed), byteCap)
	}
	if !bytes.Equal(decompressed, payload) {
		t.Error("decompressed payload mismatch at cap")
	}
}

func TestZstdCacheTransformer_Reverse_DisabledCapAllowsLargeOutput(t *testing.T) {
	tr, err := NewZstdCacheTransformer(
		Config{},
		WithMaxDecompressedCacheBytes(-1),
	)
	if err != nil {
		t.Fatalf("failed to create transformer: %v", err)
	}
	defer func() { _ = tr.Close() }()

	payload := bytes.Repeat([]byte("a"), 4*1024*1024)
	compressed := compressBytes(t, payload, zstd.SpeedDefault)

	decompressed, err := tr.Reverse(context.Background(), compressed, nil)
	if err != nil {
		t.Fatalf("Reverse with disabled cap failed: %v", err)
	}
	if len(decompressed) != len(payload) {
		t.Errorf("expected %d bytes, got %d", len(payload), len(decompressed))
	}
}

func TestWithMaxDecompressedCacheBytes_AppliesOption(t *testing.T) {
	tr, err := NewZstdCacheTransformer(
		Config{},
		WithMaxDecompressedCacheBytes(2048),
	)
	if err != nil {
		t.Fatalf("failed to create transformer: %v", err)
	}
	defer func() { _ = tr.Close() }()

	if tr.maxDecompressedBytes != 2048 {
		t.Errorf("expected option to apply 2048, got %d", tr.maxDecompressedBytes)
	}
}

func TestWithMaxDecompressedCacheBytes_OverridesConfig(t *testing.T) {
	tr, err := NewZstdCacheTransformer(
		Config{MaxDecompressedBytes: 999},
		WithMaxDecompressedCacheBytes(4096),
	)
	if err != nil {
		t.Fatalf("failed to create transformer: %v", err)
	}
	defer func() { _ = tr.Close() }()

	if tr.maxDecompressedBytes != 4096 {
		t.Errorf("expected option to override config to 4096, got %d", tr.maxDecompressedBytes)
	}
}

func TestZstdCacheTransformer_Close_IsIdempotent(t *testing.T) {
	tr := newTestTransformer(t)

	if err := tr.Close(); err != nil {
		t.Fatalf("first Close returned error: %v", err)
	}
	if err := tr.Close(); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
	if err := tr.Close(); err != nil {
		t.Fatalf("third Close returned error: %v", err)
	}
}
