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
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/capabilities/capabilities_functions"
)

var (
	testDataCSS     []byte
	testDataJS      []byte
	testDataSVG     []byte
	loadMinDataOnce sync.Once
)

func loadMinificationData(b *testing.B) {
	b.Helper()
	var loadErr error
	loadMinDataOnce.Do(func() {
		dataPath := filepath.Join("testdata", "minification")

		testDataCSS, loadErr = os.ReadFile(filepath.Join(dataPath, "sample.css"))
		if loadErr != nil {
			return
		}
		testDataJS, loadErr = os.ReadFile(filepath.Join(dataPath, "sample.js"))
		if loadErr != nil {
			return
		}
		testDataSVG, loadErr = os.ReadFile(filepath.Join(dataPath, "sample.svg"))
		if loadErr != nil {
			return
		}
	})

	if loadErr != nil {
		b.Fatalf("Failed to load benchmark data: %v", loadErr)
	}
}

func runMinificationBench(b *testing.B, capFunc capabilities_domain.CapabilityFunc, data []byte) {
	b.Helper()

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	ctx := context.Background()

	for b.Loop() {
		inputReader := bytes.NewReader(data)

		outputStream, err := capFunc(ctx, inputReader, nil)
		if err != nil {
			b.Fatalf("capability function failed: %v", err)
		}

		if _, err := io.Copy(io.Discard, outputStream); err != nil {
			b.Fatalf("failed to read output stream: %v", err)
		}
	}
}

func BenchmarkMinifyCSS(b *testing.B) {
	loadMinificationData(b)
	cssFunc := capabilities_functions.MinifyCSS()

	b.Run("Realistic_CSS", func(b *testing.B) {
		runMinificationBench(b, cssFunc, testDataCSS)
	})
}

func BenchmarkMinifyJS(b *testing.B) {
	loadMinificationData(b)
	jsFunc := capabilities_functions.MinifyJavascript()

	b.Run("Realistic_JS", func(b *testing.B) {
		runMinificationBench(b, jsFunc, testDataJS)
	})
}

func BenchmarkMinifySVG(b *testing.B) {
	loadMinificationData(b)
	svgFunc := capabilities_functions.MinifySVG()

	b.Run("Realistic_SVG", func(b *testing.B) {
		runMinificationBench(b, svgFunc, testDataSVG)
	})
}
