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

package logger_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	logger_adapters_handlers "piko.sh/piko/internal/logger/logger_adapters/driver_handlers"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/logger/logger_test"
)

var preMadeAttrs = []slog.Attr{
	slog.Int("user_id", 12345),
	slog.String("request_id", "a-b-c-d-e-f-g"),
	slog.Bool("is_admin", false),
}

func noopTracerProvider() trace.TracerProvider {
	return noop.NewTracerProvider()
}

type nullTransport struct{}

func (n *nullTransport) SendGroupedErrors(ctx context.Context, batch map[string]*logger_domain.GroupedError) error {
	return nil
}

func setupSlogBaseline(b *testing.B) *slog.Logger {
	b.Helper()
	return slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func setupGossrLogger(b *testing.B, handlers ...slog.Handler) logger_domain.Logger {
	b.Helper()
	otel.SetTracerProvider(noopTracerProvider())

	var baseHandler slog.Handler
	if len(handlers) == 1 {
		baseHandler = handlers[0]
	} else {
		baseHandler = slog.NewMultiHandler(handlers...)
	}

	log := slog.New(baseHandler)
	logger_test.InitDefaultFactory(log)
	return logger_domain.GetLogger("benchmark")
}

func BenchmarkSlogBaseline(b *testing.B) {
	log := setupSlogBaseline(b)
	b.ReportAllocs()
	b.ResetTimer()

	b.Run("Parallel_WithAttrs", func(b *testing.B) {
		ctx := context.Background()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				log.LogAttrs(ctx, slog.LevelInfo, "this is a benchmark log message", preMadeAttrs...)
			}
		})
	})
}

func BenchmarkFastPath_DisabledLevel(b *testing.B) {
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo})
	log := setupGossrLogger(b, handler)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Debug("this message should be discarded", preMadeAttrs...)
		}
	})
}

func BenchmarkSimpleHandler_Overhead(b *testing.B) {
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	log := setupGossrLogger(b, handler)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("this is a benchmark log message", preMadeAttrs...)
		}
	})
}

func BenchmarkWith_Contextual(b *testing.B) {
	handler := slog.NewJSONHandler(io.Discard, nil)
	log := setupGossrLogger(b, handler).With(slog.String("component", "api-server"))
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("request processed", preMadeAttrs...)
		}
	})
}

func BenchmarkWith_Chained(b *testing.B) {
	handler := slog.NewJSONHandler(io.Discard, nil)
	baseLogger := setupGossrLogger(b, handler)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log := baseLogger.
				With(slog.String("tenant_id", "tenant-123")).
				With(slog.String("data_center", "us-east-1"))
			log.Info("data ingested", preMadeAttrs...)
		}
	})
}

func BenchmarkMultiHandler_Scaling(b *testing.B) {
	runTest := func(b *testing.B, handlerCount int) {
		handlers := make([]slog.Handler, handlerCount)
		for i := range handlerCount {
			handlers[i] = slog.NewJSONHandler(io.Discard, nil)
		}
		log := setupGossrLogger(b, handlers...)

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				log.Error("critical system alert", preMadeAttrs...)
			}
		})
	}

	b.Run("2_Handlers", func(b *testing.B) { runTest(b, 2) })
	b.Run("5_Handlers", func(b *testing.B) { runTest(b, 5) })
	b.Run("10_Handlers", func(b *testing.B) { runTest(b, 10) })
}

func BenchmarkSpan_Overhead(b *testing.B) {
	handler := slog.NewJSONHandler(io.Discard, nil)
	log := setupGossrLogger(b, handler)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, span, _ := log.Span(ctx, "MySpan", preMadeAttrs...)

			span.End()
		}
	})
}

func BenchmarkNotificationHandler_Contention(b *testing.B) {
	nextHandler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})

	notificationHandler := logger_domain.NewNotificationHandler(
		nextHandler,
		&nullTransport{},
		slog.LevelError,
	)
	notificationHandler.SetDebounceDuration(1 * time.Hour)

	log := setupGossrLogger(b, notificationHandler)

	b.ReportAllocs()
	b.ResetTimer()

	errorPool := sync.Pool{
		New: func() any {
			return []slog.Attr{slog.Int("error_code", 1)}
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			attrs, ok := errorPool.Get().([]slog.Attr)
			if !ok {
				attrs = []slog.Attr{slog.Int("error_code", i)}
			}
			attrs[0].Value = slog.IntValue(i)
			log.Error("unique error event", attrs...)
			errorPool.Put(attrs)
			i++
		}
	})
}

func BenchmarkFileIO_Realistic(b *testing.B) {
	tmpFile, err := os.CreateTemp(b.TempDir(), "benchmark-log-*.log")
	if err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	handler := slog.NewJSONHandler(tmpFile, nil)
	log := setupGossrLogger(b, handler)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("this is a benchmark log message", preMadeAttrs...)
		}
	})
}

func BenchmarkFullStack_NoOp(b *testing.B) {
	destHandler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})

	wrappedHandler := logger_domain.NewOTelSlogHandler(destHandler)

	log := slog.New(wrappedHandler)
	logger_test.InitDefaultFactory(log)
	benchLogger := logger_domain.GetLogger("benchmark-fullstack")

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			benchLogger.Info("this is a benchmark log message", preMadeAttrs...)
		}
	})
}

func BenchmarkPrettyHandler_Overhead(b *testing.B) {
	handler := logger_adapters_handlers.NewPrettyHandler(io.Discard, &logger_adapters_handlers.Options{
		Level:    slog.LevelDebug,
		NoColour: true,
	})

	log := setupGossrLogger(b, handler)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("this is a benchmark log message", preMadeAttrs...)
		}
	})
}
