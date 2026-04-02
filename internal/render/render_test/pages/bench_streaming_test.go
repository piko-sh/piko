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

package pages_test

import (
	"bytes"
	"context"
	"io"
	"maps"
	"net/http/httptest"
	"testing"
	"time"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/render/render_test/pages/fixtures"
	"piko.sh/piko/internal/render/render_test/pages/mocks"
	"piko.sh/piko/internal/templater/templater_dto"
)

type milestoneWriter struct {
	startTime        time.Time
	w                io.Writer
	milestones       map[string]time.Duration
	headCloseTag     []byte
	firstByteWritten bool
	headClosed       bool
}

func newMilestoneWriter(w io.Writer) *milestoneWriter {
	return &milestoneWriter{
		w:            w,
		startTime:    time.Now(),
		milestones:   make(map[string]time.Duration),
		headCloseTag: []byte("</head>"),
	}
}

func (mw *milestoneWriter) Write(p []byte) (n int, err error) {
	if !mw.firstByteWritten {
		if len(p) > 0 {
			mw.milestones["ttfb"] = time.Since(mw.startTime)
			mw.firstByteWritten = true
		}
	}

	if !mw.headClosed {
		if bytes.Contains(p, mw.headCloseTag) {
			mw.milestones["head"] = time.Since(mw.startTime)
			mw.headClosed = true
		}
	}
	return mw.w.Write(p)
}

func (mw *milestoneWriter) Milestones() map[string]time.Duration {
	result := make(map[string]time.Duration, len(mw.milestones))
	maps.Copy(result, mw.milestones)
	return result
}

func BenchmarkRenderOrchestrator_Streaming(b *testing.B) {
	mockRegistry := mocks.NewMockRegistry(b)
	mockCSRF := mocks.NewMockCSRF()
	orchestrator := render_domain.NewRenderOrchestrator(nil, nil, mockRegistry, mockCSRF)
	request := httptest.NewRequest("GET", "/", nil)

	mockRegistry.OnGetComponent("my-card", &render_dto.ComponentMetadata{TagName: "my-card", BaseJSPath: "/dist/my-card.js"})
	mockRegistry.OnGetComponent("another-component", &render_dto.ComponentMetadata{TagName: "another-component", BaseJSPath: "/dist/another.js"})
	mockRegistry.OnGetSVG("testmodule/lib/icon.svg", &render_domain.ParsedSvgData{InnerHTML: "<path d='...'></path>"})
	mockRegistry.OnGetSVG("testmodule/lib/logo.svg", &render_domain.ParsedSvgData{InnerHTML: `<path d="logo-path"></path>`})

	complexAST := fixtures.ComplexPageAST()
	metadataComplex := templater_dto.InternalMetadata{
		CustomTags: []string{"my-card", "another-component"},
	}

	megaComplexAST := fixtures.MegaComplexPageAST()
	metadataMega := templater_dto.InternalMetadata{
		CustomTags: []string{"my-card", "another-component"},
	}

	b.Run("ComplexFullPage", func(b *testing.B) {
		b.ReportAllocs()
		var totalTTFB, totalHeadTime time.Duration
		b.ResetTimer()
		for b.Loop() {
			b.StopTimer()
			writer := newMilestoneWriter(io.Discard)
			b.StartTimer()

			err := orchestrator.RenderAST(context.Background(), writer, nil, request, render_domain.RenderASTOptions{PageID: "bench-complex", Template: complexAST, Metadata: &metadataComplex, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}

			b.StopTimer()
			milestones := writer.Milestones()
			totalTTFB += milestones["ttfb"]
			totalHeadTime += milestones["head"]
			b.StartTimer()
		}
		b.StopTimer()

		if b.N > 0 {
			b.ReportMetric(float64(totalTTFB.Nanoseconds())/float64(b.N), "ttfb-ns/op")
			b.ReportMetric(float64(totalHeadTime.Nanoseconds())/float64(b.N), "head-ns/op")
		}
	})

	b.Run("ComplexFragmentPage", func(b *testing.B) {
		b.ReportAllocs()
		var totalTTFB time.Duration
		b.ResetTimer()
		for b.Loop() {
			b.StopTimer()
			writer := newMilestoneWriter(io.Discard)
			b.StartTimer()

			err := orchestrator.RenderAST(context.Background(), writer, nil, request, render_domain.RenderASTOptions{PageID: "bench-complex", Template: complexAST, Metadata: &metadataComplex, IsFragment: true, Styling: "", SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}

			b.StopTimer()
			milestones := writer.Milestones()
			totalTTFB += milestones["ttfb"]
			b.StartTimer()
		}
		b.StopTimer()

		if b.N > 0 {
			b.ReportMetric(float64(totalTTFB.Nanoseconds())/float64(b.N), "ttfb-ns/op")
		}
	})

	b.Run("MegaComplexFullPage", func(b *testing.B) {
		b.ReportAllocs()
		var totalTTFB, totalHeadTime time.Duration
		b.ResetTimer()
		for b.Loop() {
			b.StopTimer()
			writer := newMilestoneWriter(io.Discard)
			b.StartTimer()

			err := orchestrator.RenderAST(context.Background(), writer, nil, request, render_domain.RenderASTOptions{PageID: "bench-mega", Template: megaComplexAST, Metadata: &metadataMega, SiteConfig: &config.WebsiteConfig{}})
			if err != nil {
				b.Fatal(err)
			}

			b.StopTimer()
			milestones := writer.Milestones()
			totalTTFB += milestones["ttfb"]
			totalHeadTime += milestones["head"]
			b.StartTimer()
		}
		b.StopTimer()

		if b.N > 0 {
			b.ReportMetric(float64(totalTTFB.Nanoseconds())/float64(b.N), "ttfb-ns/op")
			b.ReportMetric(float64(totalHeadTime.Nanoseconds())/float64(b.N), "head-ns/op")
		}
	})
}
