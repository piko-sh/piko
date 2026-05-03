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

package storage_transformer_zstd_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/storage"
	stzstd "piko.sh/piko/wdk/storage/storage_transformer_zstd"
)

func roundTrip(t *testing.T, transformer storage.StreamTransformerPort, input []byte, options any) []byte {
	t.Helper()
	ctx := context.Background()

	compressed, err := transformer.Transform(ctx, bytes.NewReader(input), options)
	require.NoError(t, err)

	compressedBytes, err := io.ReadAll(compressed)
	require.NoError(t, err)

	decompressed, err := transformer.Reverse(ctx, bytes.NewReader(compressedBytes), nil)
	require.NoError(t, err)

	result, err := io.ReadAll(decompressed)
	require.NoError(t, err)

	if closer, ok := decompressed.(io.Closer); ok {
		require.NoError(t, closer.Close())
	}

	return result
}

type errReader struct {
	data     []byte
	position int
	failAt   int
	err      error
}

func (r *errReader) Read(p []byte) (int, error) {
	if r.position >= r.failAt {
		return 0, r.err
	}

	n := min(len(p), r.failAt-r.position, len(r.data)-r.position)

	copy(p, r.data[r.position:r.position+n])
	r.position += n

	if r.position >= r.failAt {
		return n, r.err
	}
	return n, nil
}

func TestDefaultConfig(t *testing.T) {
	config := stzstd.DefaultConfig()

	assert.Equal(t, "zstd", config.Name)
	assert.Equal(t, 100, config.Priority)
	assert.Equal(t, zstd.SpeedDefault, config.Level)
}

func TestNewTransformer(t *testing.T) {
	testCases := []struct {
		name     string
		config   stzstd.Config
		wantName string
		wantPri  int
	}{
		{
			name:     "zero value config uses defaults",
			config:   stzstd.Config{},
			wantName: "zstd",
			wantPri:  100,
		},
		{
			name:     "custom name preserved",
			config:   stzstd.Config{Name: "custom"},
			wantName: "custom",
			wantPri:  100,
		},
		{
			name:     "custom priority preserved",
			config:   stzstd.Config{Priority: 50},
			wantName: "zstd",
			wantPri:  50,
		},
		{
			name:   "SpeedFastest level",
			config: stzstd.Config{Level: zstd.SpeedFastest},
		},
		{
			name:   "SpeedBetterCompression level",
			config: stzstd.Config{Level: zstd.SpeedBetterCompression},
		},
		{
			name:   "SpeedBestCompression level",
			config: stzstd.Config{Level: zstd.SpeedBestCompression},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transformer, err := stzstd.NewZstdTransformer(tc.config)
			require.NoError(t, err)
			require.NotNil(t, transformer)

			if tc.wantName != "" {
				assert.Equal(t, tc.wantName, transformer.Name())
			}
			if tc.wantPri != 0 {
				assert.Equal(t, tc.wantPri, transformer.Priority())
			}
		})
	}
}

func TestAccessors(t *testing.T) {
	transformer, err := stzstd.NewZstdTransformer(stzstd.Config{
		Name:     "test-name",
		Priority: 42,
		Level:    zstd.SpeedFastest,
	})
	require.NoError(t, err)

	assert.Equal(t, "test-name", transformer.Name())
	assert.Equal(t, storage.TransformerCompression, transformer.Type())
	assert.Equal(t, 42, transformer.Priority())
}

func TestTransformRoundTrip(t *testing.T) {
	transformer, err := stzstd.NewZstdTransformer(stzstd.DefaultConfig())
	require.NoError(t, err)

	randomBytes := make([]byte, 4096)
	_, err = rand.Read(randomBytes)
	require.NoError(t, err)

	largePayload := bytes.Repeat([]byte("abcdefghijklmnop"), 4096)

	testCases := []struct {
		name  string
		input []byte
	}{
		{name: "simple text", input: []byte("Hello, World!")},
		{name: "empty data", input: []byte{}},
		{name: "single byte", input: []byte{0x42}},
		{name: "repeated bytes", input: bytes.Repeat([]byte{0xAA}, 1024)},
		{name: "random binary data", input: randomBytes},
		{name: "large payload", input: largePayload},
		{name: "unicode text", input: []byte("Héj världen! Bonjour le monde! Hallo Welt!")},
		{name: "null bytes", input: make([]byte, 256)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := roundTrip(t, transformer, tc.input, nil)
			assert.Equal(t, tc.input, result)
		})
	}
}

func TestTransformAtDifferentLevels(t *testing.T) {
	transformer, err := stzstd.NewZstdTransformer(stzstd.DefaultConfig())
	require.NoError(t, err)

	input := bytes.Repeat([]byte("compressible data "), 64)

	testCases := []struct {
		name  string
		level zstd.EncoderLevel
	}{
		{name: "SpeedFastest", level: zstd.SpeedFastest},
		{name: "SpeedDefault", level: zstd.SpeedDefault},
		{name: "SpeedBetterCompression", level: zstd.SpeedBetterCompression},
		{name: "SpeedBestCompression", level: zstd.SpeedBestCompression},
	}

	sizes := make(map[string]int)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			opts := map[string]any{"level": int(tc.level)}

			compressed, err := transformer.Transform(ctx, bytes.NewReader(input), opts)
			require.NoError(t, err)

			compressedBytes, err := io.ReadAll(compressed)
			require.NoError(t, err)
			sizes[tc.name] = len(compressedBytes)

			decompressed, err := transformer.Reverse(ctx, bytes.NewReader(compressedBytes), nil)
			require.NoError(t, err)

			result, err := io.ReadAll(decompressed)
			require.NoError(t, err)
			assert.Equal(t, input, result)
		})
	}

	t.Run("SpeedFastest output >= SpeedBestCompression output", func(t *testing.T) {
		if sizes["SpeedFastest"] == 0 || sizes["SpeedBestCompression"] == 0 {
			t.Skip("level tests did not run")
		}
		assert.GreaterOrEqual(t, sizes["SpeedFastest"], sizes["SpeedBestCompression"])
	})
}

func TestTransformOptionsOverride(t *testing.T) {
	transformer, err := stzstd.NewZstdTransformer(stzstd.Config{
		Level: zstd.SpeedFastest,
	})
	require.NoError(t, err)

	input := []byte("options override test data")

	testCases := []struct {
		name    string
		options any
	}{
		{name: "nil options uses config level", options: nil},
		{name: "non-map options ignored", options: "not a map"},
		{name: "map without level key ignored", options: map[string]any{"other": 1}},
		{name: "map with non-int level ignored", options: map[string]any{"level": "high"}},
		{name: "map with valid level overrides", options: map[string]any{"level": int(zstd.SpeedBestCompression)}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := roundTrip(t, transformer, input, tc.options)
			assert.Equal(t, input, result)
		})
	}
}

func TestReverseWithInvalidInput(t *testing.T) {
	transformer, err := stzstd.NewZstdTransformer(stzstd.DefaultConfig())
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("garbage data", func(t *testing.T) {
		reader, err := transformer.Reverse(ctx, bytes.NewReader([]byte("this is not zstd")), nil)
		if err != nil {

			assert.Contains(t, err.Error(), "failed to create zstd decoder")
			return
		}

		_, err = io.ReadAll(reader)
		assert.Error(t, err, "reading garbage data should produce an error")
	})

	t.Run("truncated stream", func(t *testing.T) {
		compressed, err := transformer.Transform(ctx, bytes.NewReader([]byte("valid data to compress")), nil)
		require.NoError(t, err)

		compressedBytes, err := io.ReadAll(compressed)
		require.NoError(t, err)

		truncated := compressedBytes[:len(compressedBytes)/2]

		reader, err := transformer.Reverse(ctx, bytes.NewReader(truncated), nil)
		if err != nil {
			return
		}

		_, err = io.ReadAll(reader)
		assert.Error(t, err, "reading truncated zstd stream should produce an error")
	})

	t.Run("empty input", func(t *testing.T) {
		reader, err := transformer.Reverse(ctx, bytes.NewReader([]byte{}), nil)
		if err != nil {

			return
		}

		result, err := io.ReadAll(reader)
		if err != nil {
			return
		}
		assert.Empty(t, result, "decompressing empty input should yield empty output")
	})
}

func TestTransformWithBrokenReader(t *testing.T) {
	transformer, err := stzstd.NewZstdTransformer(stzstd.DefaultConfig())
	require.NoError(t, err)

	ctx := context.Background()

	data := bytes.Repeat([]byte("broken reader test data "), 100)
	brokenErr := errors.New("simulated read failure")
	reader := &errReader{
		data:   data,
		failAt: len(data) / 2,
		err:    brokenErr,
	}

	compressed, err := transformer.Transform(ctx, reader, nil)
	require.NoError(t, err, "Transform itself should not error")

	_, err = io.ReadAll(compressed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compression error")
}

func TestTransformEdgeCases(t *testing.T) {
	transformer, err := stzstd.NewZstdTransformer(stzstd.DefaultConfig())
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("compressed smaller than input", func(t *testing.T) {
		input := bytes.Repeat([]byte("a"), 4096)

		compressed, err := transformer.Transform(ctx, bytes.NewReader(input), nil)
		require.NoError(t, err)

		compressedBytes, err := io.ReadAll(compressed)
		require.NoError(t, err)

		assert.Less(t, len(compressedBytes), len(input),
			"compressed output should be smaller than input for highly compressible data")
	})

	t.Run("output is valid standard zstd", func(t *testing.T) {
		input := []byte("verify standard zstd format output")

		compressed, err := transformer.Transform(ctx, bytes.NewReader(input), nil)
		require.NoError(t, err)

		compressedBytes, err := io.ReadAll(compressed)
		require.NoError(t, err)

		decoder, err := zstd.NewReader(bytes.NewReader(compressedBytes))
		require.NoError(t, err)
		defer decoder.Close()

		result, err := io.ReadAll(decoder)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("accepts standard zstd input", func(t *testing.T) {
		input := []byte("verify standard zstd format input")

		encoder, err := zstd.NewWriter(nil)
		require.NoError(t, err)
		compressedBytes := encoder.EncodeAll(input, nil)
		_ = encoder.Close()

		reader, err := transformer.Reverse(ctx, bytes.NewReader(compressedBytes), nil)
		require.NoError(t, err)

		result, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})
}

func TestTransformConcurrent(t *testing.T) {
	transformer, err := stzstd.NewZstdTransformer(stzstd.DefaultConfig())
	require.NoError(t, err)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	errs := make(chan error, goroutines)

	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			input := fmt.Appendf(nil, "goroutine-%d-payload-data", id)
			result := roundTrip(t, transformer, input, nil)
			if !bytes.Equal(input, result) {
				errs <- fmt.Errorf("goroutine %d: content mismatch", id)
				return
			}
			errs <- nil
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		assert.NoError(t, err)
	}
}

func TestReverseReaderClose(t *testing.T) {
	transformer, err := stzstd.NewZstdTransformer(stzstd.DefaultConfig())
	require.NoError(t, err)

	ctx := context.Background()

	input := []byte("close test data")
	compressed, err := transformer.Transform(ctx, bytes.NewReader(input), nil)
	require.NoError(t, err)

	compressedBytes, err := io.ReadAll(compressed)
	require.NoError(t, err)

	reader, err := transformer.Reverse(ctx, bytes.NewReader(compressedBytes), nil)
	require.NoError(t, err)

	closer, ok := reader.(io.ReadCloser)
	require.True(t, ok, "Reverse reader should implement io.ReadCloser")

	data, err := io.ReadAll(closer)
	require.NoError(t, err)
	assert.Equal(t, input, data)

	require.NoError(t, closer.Close())
}

func TestReverseReaderCloseBeforeFullRead(t *testing.T) {
	transformer, err := stzstd.NewZstdTransformer(stzstd.DefaultConfig())
	require.NoError(t, err)

	ctx := context.Background()

	input := bytes.Repeat([]byte("partial read "), 100)
	compressed, err := transformer.Transform(ctx, bytes.NewReader(input), nil)
	require.NoError(t, err)

	compressedBytes, err := io.ReadAll(compressed)
	require.NoError(t, err)

	reader, err := transformer.Reverse(ctx, bytes.NewReader(compressedBytes), nil)
	require.NoError(t, err)

	closer, ok := reader.(io.ReadCloser)
	require.True(t, ok)

	buffer := make([]byte, 10)
	_, err = closer.Read(buffer)
	require.NoError(t, err)

	require.NoError(t, closer.Close())
}

func TestReverseDecompressedBytesCap(t *testing.T) {
	t.Run("payload smaller than cap roundtrips cleanly", func(t *testing.T) {
		transformer, err := stzstd.NewZstdTransformer(
			stzstd.Config{},
			stzstd.WithMaxDecompressedBytes(64*1024),
		)
		require.NoError(t, err)

		input := bytes.Repeat([]byte("hello world\n"), 200)

		compressed, err := transformer.Transform(context.Background(), bytes.NewReader(input), nil)
		require.NoError(t, err)

		compressedBytes, err := io.ReadAll(compressed)
		require.NoError(t, err)

		decompressed, err := transformer.Reverse(context.Background(), bytes.NewReader(compressedBytes), nil)
		require.NoError(t, err)

		result, err := io.ReadAll(decompressed)
		require.NoError(t, err, "payload under cap should read cleanly")
		assert.Equal(t, input, result)

		if closer, ok := decompressed.(io.Closer); ok {
			require.NoError(t, closer.Close())
		}
	})

	t.Run("zip-bomb-style payload exceeding cap surfaces sentinel error", func(t *testing.T) {
		transformer, err := stzstd.NewZstdTransformer(
			stzstd.Config{},
			stzstd.WithMaxDecompressedBytes(1024),
		)
		require.NoError(t, err)

		bomb := make([]byte, 256*1024)

		var buf bytes.Buffer
		encoder, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
		require.NoError(t, err)
		_, err = encoder.Write(bomb)
		require.NoError(t, err)
		require.NoError(t, encoder.Close())

		assert.Less(t, buf.Len(), len(bomb)/100,
			"setup: highly redundant input should compress drastically (zip-bomb shape)")

		decompressed, err := transformer.Reverse(context.Background(), &buf, nil)
		require.NoError(t, err)

		_, err = io.ReadAll(decompressed)
		require.Error(t, err, "reading past the cap must surface an error")
		assert.True(t, errors.Is(err, stzstd.ErrDecompressedTooLarge),
			"expected ErrDecompressedTooLarge, got %v", err)

		if closer, ok := decompressed.(io.Closer); ok {
			_ = closer.Close()
		}
	})

	t.Run("WithMaxDecompressedBytes non-positive disables cap", func(t *testing.T) {
		transformer, err := stzstd.NewZstdTransformer(
			stzstd.Config{},
			stzstd.WithMaxDecompressedBytes(-1),
		)
		require.NoError(t, err)

		payload := bytes.Repeat([]byte("a"), 4*1024*1024)

		compressed, err := transformer.Transform(context.Background(), bytes.NewReader(payload), nil)
		require.NoError(t, err)

		compressedBytes, err := io.ReadAll(compressed)
		require.NoError(t, err)

		decompressed, err := transformer.Reverse(context.Background(), bytes.NewReader(compressedBytes), nil)
		require.NoError(t, err)

		result, err := io.ReadAll(decompressed)
		require.NoError(t, err, "negative cap must disable the cap")
		assert.Equal(t, len(payload), len(result))

		if closer, ok := decompressed.(io.Closer); ok {
			require.NoError(t, closer.Close())
		}
	})
}
