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

package render_test_bench

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

type BenchmarkRegistry struct {
	components map[string]*render_dto.ComponentMetadata
	svgAssets  map[string]*render_domain.ParsedSvgData
}

func NewBenchmarkRegistry() *BenchmarkRegistry {
	r := &BenchmarkRegistry{
		components: make(map[string]*render_dto.ComponentMetadata),
		svgAssets:  make(map[string]*render_domain.ParsedSvgData),
	}

	r.components["my-card"] = &render_dto.ComponentMetadata{
		TagName:    "my-card",
		BaseJSPath: "/dist/my-card.js",
	}
	r.components["another-component"] = &render_dto.ComponentMetadata{
		TagName:    "another-component",
		BaseJSPath: "/dist/another.js",
	}
	r.components["custom-button"] = &render_dto.ComponentMetadata{
		TagName:    "custom-button",
		BaseJSPath: "/dist/custom-button.js",
	}

	svgIcons := []string{
		"testmodule/lib/icon.svg",
		"testmodule/lib/logo.svg",
		"testmodule/lib/icon-home.svg",
		"testmodule/lib/icon-settings.svg",
		"testmodule/lib/icon-user.svg",
		"testmodule/lib/icon-search.svg",
		"testmodule/lib/icon-menu.svg",
	}

	for _, icon := range svgIcons {
		parsedData := &render_domain.ParsedSvgData{
			InnerHTML: `<path d="M12 2L2 7v10l10 5 10-5V7l-10-5z"></path>`,
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "viewBox", Value: "0 0 24 24"},
				{Name: "fill", Value: "currentColor"},
			},
		}
		parsedData.CachedSymbol = render_domain.ComputeSymbolString(icon, parsedData)
		r.svgAssets[icon] = parsedData
	}

	return r
}

func (r *BenchmarkRegistry) AddComponent(tagName string, metadata *render_dto.ComponentMetadata) {
	r.components[tagName] = metadata
}

func (r *BenchmarkRegistry) AddSVG(assetID string, data *render_domain.ParsedSvgData) {
	r.svgAssets[assetID] = data
}

func (r *BenchmarkRegistry) GetComponentMetadata(_ context.Context, componentType string) (*render_dto.ComponentMetadata, error) {
	if meta, ok := r.components[componentType]; ok {
		return meta, nil
	}
	return nil, fmt.Errorf("component '%s' not found", componentType)
}

func (r *BenchmarkRegistry) GetAssetRawSVG(_ context.Context, assetID string) (*render_domain.ParsedSvgData, error) {
	if data, ok := r.svgAssets[assetID]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("SVG asset '%s' not found", assetID)
}

func (r *BenchmarkRegistry) BulkGetAssetRawSVG(_ context.Context, assetIDs []string) (map[string]*render_domain.ParsedSvgData, error) {
	results := make(map[string]*render_domain.ParsedSvgData, len(assetIDs))
	for _, id := range assetIDs {
		if data, ok := r.svgAssets[id]; ok {
			results[id] = data
		}
	}
	return results, nil
}

func (r *BenchmarkRegistry) BulkGetComponentMetadata(_ context.Context, componentTypes []string) (map[string]*render_dto.ComponentMetadata, error) {
	results := make(map[string]*render_dto.ComponentMetadata, len(componentTypes))
	for _, ct := range componentTypes {
		if meta, ok := r.components[ct]; ok {
			results[ct] = meta
		}
	}
	return results, nil
}

func (r *BenchmarkRegistry) GetStats() render_domain.RegistryAdapterStats {
	return render_domain.RegistryAdapterStats{
		ComponentCacheSize: len(r.components),
		SVGCacheSize:       len(r.svgAssets),
	}
}

func (r *BenchmarkRegistry) ClearComponentCache(_ context.Context, _ string) {}

func (r *BenchmarkRegistry) ClearSvgCache(_ context.Context, _ string) {}

func (r *BenchmarkRegistry) GetArtefactServePath(_ context.Context, _ string) string { return "" }

func (r *BenchmarkRegistry) UpsertArtefact(_ context.Context, artefactID string, _ string, _ io.Reader, _ string, desiredProfiles []registry_dto.NamedProfile) (*registry_dto.ArtefactMeta, error) {
	return &registry_dto.ArtefactMeta{
		ID:              artefactID,
		DesiredProfiles: desiredProfiles,
	}, nil
}

type BenchmarkCSRFService struct{}

func (s *BenchmarkCSRFService) Name() string {
	return "BenchmarkCSRFService"
}

func (s *BenchmarkCSRFService) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	return healthprobe_dto.Status{
		Name:    "BenchmarkCSRFService",
		State:   healthprobe_dto.StateHealthy,
		Message: "Mock CSRF service",
	}
}

func (s *BenchmarkCSRFService) GenerateCSRFPair(_ http.ResponseWriter, _ *http.Request, buffer *bytes.Buffer) (security_dto.CSRFPair, error) {
	buffer.Reset()
	buffer.WriteString("benchmark-action-token-payload^signature")
	return security_dto.CSRFPair{
		RawEphemeralToken: "benchmark-ephemeral-token-12345678",
		ActionToken:       buffer.Bytes(),
	}, nil
}

func (s *BenchmarkCSRFService) ValidateCSRFPair(_ *http.Request, _ string, _ []byte) (bool, error) {
	return true, nil
}

type MetricsWriter struct {
	startTime        time.Time
	w                io.Writer
	headCloseTag     []byte
	bytesWritten     int64
	writeCount       int
	ttfb             time.Duration
	headTime         time.Duration
	totalTime        time.Duration
	mu               sync.Mutex
	firstByteWritten bool
	headClosed       bool
}

func NewMetricsWriter(w io.Writer) *MetricsWriter {
	return &MetricsWriter{
		w:            w,
		startTime:    time.Now(),
		headCloseTag: []byte("</head>"),
	}
}

func (mw *MetricsWriter) Write(p []byte) (n int, err error) {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	if !mw.firstByteWritten && len(p) > 0 {
		mw.ttfb = time.Since(mw.startTime)
		mw.firstByteWritten = true
	}

	if !mw.headClosed && bytes.Contains(p, mw.headCloseTag) {
		mw.headTime = time.Since(mw.startTime)
		mw.headClosed = true
	}

	mw.writeCount++
	n, err = mw.w.Write(p)
	mw.bytesWritten += int64(n)
	return n, err
}

func (mw *MetricsWriter) Finish() {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	mw.totalTime = time.Since(mw.startTime)
}

func (mw *MetricsWriter) Metrics() WriterMetrics {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	return WriterMetrics{
		BytesWritten: mw.bytesWritten,
		WriteCount:   mw.writeCount,
		TTFB:         mw.ttfb,
		HeadTime:     mw.headTime,
		TotalTime:    mw.totalTime,
	}
}

type WriterMetrics struct {
	BytesWritten int64
	WriteCount   int
	TTFB         time.Duration
	HeadTime     time.Duration
	TotalTime    time.Duration
}

type CountingWriter struct {
	BytesWritten int64
	WriteCount   int
}

func (cw *CountingWriter) Write(p []byte) (n int, err error) {
	cw.BytesWritten += int64(len(p))
	cw.WriteCount++
	return len(p), nil
}

func NewTestOrchestrator() *render_domain.RenderOrchestrator {
	registry := NewBenchmarkRegistry()
	csrf := &BenchmarkCSRFService{}
	return render_domain.NewRenderOrchestrator(nil, nil, registry, csrf)
}

func NewTestOrchestratorWithRegistry(registry *BenchmarkRegistry) *render_domain.RenderOrchestrator {
	csrf := &BenchmarkCSRFService{}
	return render_domain.NewRenderOrchestrator(nil, nil, registry, csrf)
}

func NewTestRequest() *http.Request {
	return httptest.NewRequest("GET", "/benchmark", nil)
}

func NewTestResponseWriter() http.ResponseWriter {
	return httptest.NewRecorder()
}

type BenchmarkResult struct {
	Name           string
	Iterations     int
	NsPerOp        float64
	BytesPerOp     int64
	AllocsPerOp    int64
	BytesOutput    int64
	TTFB           time.Duration
	HeadTime       time.Duration
	ThroughputMBps float64
}

func ReportCustomMetrics(b *testing.B, metrics WriterMetrics) {
	if b.N > 0 {
		b.ReportMetric(float64(metrics.TTFB.Nanoseconds())/float64(b.N), "ttfb-ns/op")
		if metrics.HeadTime > 0 {
			b.ReportMetric(float64(metrics.HeadTime.Nanoseconds())/float64(b.N), "head-ns/op")
		}
		b.ReportMetric(float64(metrics.BytesWritten)/float64(b.N), "bytes/op-output")
		b.ReportMetric(float64(metrics.WriteCount)/float64(b.N), "writes/op")

		if metrics.TotalTime > 0 {
			throughputMBps := float64(metrics.BytesWritten) / metrics.TotalTime.Seconds() / (1024 * 1024)
			b.ReportMetric(throughputMBps, "MB/s")
		}
	}
}

func WarmUpOrchestrator(orchestrator *render_domain.RenderOrchestrator, ast *ast_domain.TemplateAST) {
	ctx := context.Background()
	request := NewTestRequest()
	response := httptest.NewRecorder()
	metadata := &templater_dto.InternalMetadata{}
	websiteConfig := &config.WebsiteConfig{}
	for range 10 {
		_ = orchestrator.RenderAST(ctx, io.Discard, response, request, render_domain.RenderASTOptions{
			PageID:     "warmup",
			Template:   ast,
			Metadata:   metadata,
			SiteConfig: websiteConfig,
		})
	}
}
