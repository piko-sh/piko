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

//go:build bench

package capabilities_test_bench

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/gzip"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/capabilities/capabilities_functions"
)

var (
	testDataHTML []byte
	testDataJSON []byte
	testDataBMP  []byte
	loadOnce     sync.Once
)

func loadTestData(b *testing.B) {
	b.Helper()
	var loadErr error
	loadOnce.Do(func() {

		dataPath := filepath.Join("testdata", "compression")

		testDataHTML, loadErr = os.ReadFile(filepath.Join(dataPath, "sample.html"))
		if loadErr != nil {
			return
		}
		testDataJSON, loadErr = os.ReadFile(filepath.Join(dataPath, "sample.json"))
		if loadErr != nil {
			return
		}
		testDataBMP, loadErr = os.ReadFile(filepath.Join(dataPath, "lena.bmp"))
		if loadErr != nil {

			loadErr = fmt.Errorf("failed to read 'lena.bmp' (is it missing from %s?): %w", dataPath, loadErr)
			return
		}
	})

	if loadErr != nil {
		b.Fatalf("Failed to load benchmark data: %v", loadErr)
	}
}

func runCompressionBench(b *testing.B, capFunc capabilities_domain.CapabilityFunc, data []byte, params capabilities_domain.CapabilityParams) {
	b.Helper()

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	ctx := context.Background()

	for b.Loop() {

		inputReader := bytes.NewReader(data)

		outputStream, err := capFunc(ctx, inputReader, params)
		if err != nil {
			b.Fatalf("capability function failed: %v", err)
		}

		if _, err := io.Copy(io.Discard, outputStream); err != nil {
			b.Fatalf("failed to read output stream: %v", err)
		}
	}
}

func BenchmarkGzip(b *testing.B) {
	loadTestData(b)
	gzipFunc := capabilities_functions.Gzip()

	scenarios := []struct {
		name string
		data []byte
	}{
		{name: "HTML", data: testDataHTML},
		{name: "JSON", data: testDataJSON},
		{name: "Incompressible_BMP", data: testDataBMP},
	}

	levels := []struct {
		name  string
		level int
	}{
		{name: "Level_BestSpeed", level: gzip.BestSpeed},
		{name: "Level_Default", level: gzip.DefaultCompression},
		{name: "Level_BestCompression", level: gzip.BestCompression},
	}

	for _, level := range levels {
		b.Run(level.name, func(b *testing.B) {
			params := capabilities_domain.CapabilityParams{
				"level": strconv.Itoa(level.level),
			}
			for _, s := range scenarios {
				b.Run(s.name, func(b *testing.B) {
					runCompressionBench(b, gzipFunc, s.data, params)
				})
			}
		})
	}
}

func BenchmarkBrotli(b *testing.B) {
	loadTestData(b)
	brotliFunc := capabilities_functions.Brotli()

	scenarios := []struct {
		name string
		data []byte
	}{
		{name: "HTML", data: testDataHTML},
		{name: "JSON", data: testDataJSON},
		{name: "Incompressible_BMP", data: testDataBMP},
	}

	levels := []struct {
		name  string
		level int
	}{
		{name: "Level_BestSpeed", level: brotli.BestSpeed},
		{name: "Level_Default", level: brotli.DefaultCompression},
		{name: "Level_BestCompression", level: brotli.BestCompression},
	}

	for _, level := range levels {
		b.Run(level.name, func(b *testing.B) {
			params := capabilities_domain.CapabilityParams{
				"level": strconv.Itoa(level.level),
			}
			for _, s := range scenarios {
				b.Run(s.name, func(b *testing.B) {
					runCompressionBench(b, brotliFunc, s.data, params)
				})
			}
		})
	}
}
