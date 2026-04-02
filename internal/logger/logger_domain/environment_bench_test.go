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

package logger_domain

import (
	"io"
	"log/slog"
	"testing"
)

func BenchmarkDetectEnvironment(b *testing.B) {
	b.Setenv("POD_NAME", "web-abc123")
	b.Setenv("POD_NAMESPACE", "production")
	b.Setenv("PIKO_ENVIRONMENT", "staging")

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		detectEnvironment()
	}
}

func BenchmarkLogWithEnvironmentAttrs(b *testing.B) {
	handler := slog.NewTextHandler(io.Discard, nil)
	logger := slog.New(handler)

	envAttrs := []slog.Attr{
		slog.String("k8s.pod.name", "test-pod"),
		slog.String("k8s.namespace.name", "production"),
		slog.String("deployment.environment.name", "staging"),
	}
	logger = logger.With(SlogAttrsToAny(envAttrs)...)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		logger.Info("request handled", slog.String("path", "/api/v1"))
	}
}

func BenchmarkLogWithoutEnvironmentAttrs(b *testing.B) {
	handler := slog.NewTextHandler(io.Discard, nil)
	logger := slog.New(handler)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		logger.Info("request handled", slog.String("path", "/api/v1"))
	}
}
