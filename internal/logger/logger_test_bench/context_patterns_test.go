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
	"runtime"
	"testing"

	"go.opentelemetry.io/otel"
	"piko.sh/piko/internal/caller"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/logger/logger_test"
)

func setupDiscardLogger(b *testing.B) logger_domain.Logger {
	b.Helper()
	otel.SetTracerProvider(noopTracerProvider())
	handler := slog.NewJSONHandler(io.Discard, nil)
	baseLog := slog.New(handler)

	slog.SetDefault(baseLog)
	logger_test.InitDefaultFactory(baseLog)
	return logger_domain.GetLogger("bench")
}

func getPackageLogger(b *testing.B) logger_domain.Logger {
	b.Helper()
	otel.SetTracerProvider(noopTracerProvider())
	handler := slog.NewJSONHandler(io.Discard, nil)
	return logger_domain.New(slog.New(handler), "bench/pkg")
}

func BenchmarkPattern_PackageGlobal_Log(b *testing.B) {
	pkgLog := getPackageLogger(b)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		pkgLog.Info("message", slog.String("key", "value"))
	}
}

func doWorkWithLoggerArg(log logger_domain.Logger) {
	log.Info("message", slog.String("key", "value"))
}

func BenchmarkPattern_LoggerArg_Log(b *testing.B) {
	log := setupDiscardLogger(b)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		doWorkWithLoggerArg(log)
	}
}

func doWorkWithContextLogger(ctx context.Context) {
	_, l := logger_domain.From(ctx, nil)
	l.Info("message", slog.String("key", "value"))
}

func BenchmarkPattern_FromContext_Log(b *testing.B) {
	log := setupDiscardLogger(b)
	ctx := logger_domain.WithLogger(context.Background(), log)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		doWorkWithContextLogger(ctx)
	}
}

func doWorkWithContextLoggerFallback(ctx context.Context, fallback logger_domain.Logger) {
	_, l := logger_domain.From(ctx, fallback)
	l.Info("message", slog.String("key", "value"))
}

func BenchmarkPattern_FromContextFallback_Log(b *testing.B) {
	log := setupDiscardLogger(b)
	ctx := logger_domain.WithLogger(context.Background(), log)
	fallback := logger_domain.GetLogger("fallback")
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		doWorkWithContextLoggerFallback(ctx, fallback)
	}
}

func BenchmarkFrom_LoggerPresent(b *testing.B) {
	log := setupDiscardLogger(b)
	ctx := logger_domain.WithLogger(context.Background(), log)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = logger_domain.From(ctx, nil)
	}
}

func BenchmarkFrom_LoggerAbsent_NoFallback(b *testing.B) {
	setupDiscardLogger(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = logger_domain.From(ctx, nil)
	}
}

func BenchmarkFrom_LoggerAbsent_WithFallback(b *testing.B) {
	log := setupDiscardLogger(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = logger_domain.From(ctx, log)
	}
}

type contextDepthKey int

func buildDeepContext(depth int, log logger_domain.Logger) context.Context {
	ctx := logger_domain.WithLogger(context.Background(), log)
	for i := range depth {
		ctx = context.WithValue(ctx, contextDepthKey(i), i)
	}
	return ctx
}

func BenchmarkFrom_ContextDepth(b *testing.B) {
	depths := []int{0, 5, 10, 20, 50}

	for _, depth := range depths {
		b.Run(depthName(depth), func(b *testing.B) {
			log := setupDiscardLogger(b)
			ctx := buildDeepContext(depth, log)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_, _ = logger_domain.From(ctx, nil)
			}
		})
	}
}

func depthName(depth int) string {
	switch depth {
	case 0:
		return "depth_0"
	case 5:
		return "depth_5"
	case 10:
		return "depth_10"
	case 20:
		return "depth_20"
	case 50:
		return "depth_50"
	default:
		return "depth_unknown"
	}
}

func BenchmarkWithLogger(b *testing.B) {
	log := setupDiscardLogger(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = logger_domain.WithLogger(ctx, log)
	}
}

func BenchmarkRequest_PackageGlobal(b *testing.B) {
	pkgLog := getPackageLogger(b)
	ctx := context.Background()

	var requestHandler func(context.Context)
	var service func(context.Context)
	var repo func(context.Context)

	requestHandler = func(_ context.Context) {
		pkgLog.Info("handler", slog.String("step", "1"))
		service(ctx)
	}
	service = func(_ context.Context) {
		pkgLog.Info("service", slog.String("step", "2"))
		repo(ctx)
	}
	repo = func(_ context.Context) {
		pkgLog.Info("repo", slog.String("step", "3"))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		requestHandler(ctx)
	}
}

func requestHandler_LoggerArg(ctx context.Context, log logger_domain.Logger) {
	log.Info("handler", slog.String("step", "1"))
	service_LoggerArg(ctx, log)
}

func service_LoggerArg(ctx context.Context, log logger_domain.Logger) {
	log.Info("service", slog.String("step", "2"))
	repo_LoggerArg(ctx, log)
}

func repo_LoggerArg(ctx context.Context, log logger_domain.Logger) {
	log.Info("repo", slog.String("step", "3"))
}

func BenchmarkRequest_LoggerArg(b *testing.B) {
	log := setupDiscardLogger(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		requestHandler_LoggerArg(ctx, log)
	}
}

func requestHandler_FromContext(ctx context.Context) {
	_, l := logger_domain.From(ctx, nil)
	l.Info("handler", slog.String("step", "1"))
	service_FromContext(ctx)
}

func service_FromContext(ctx context.Context) {
	_, l := logger_domain.From(ctx, nil)
	l.Info("service", slog.String("step", "2"))
	repo_FromContext(ctx)
}

func repo_FromContext(ctx context.Context) {
	_, l := logger_domain.From(ctx, nil)
	l.Info("repo", slog.String("step", "3"))
}

func BenchmarkRequest_FromContext(b *testing.B) {
	log := setupDiscardLogger(b)
	ctx := logger_domain.WithLogger(context.Background(), log)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		requestHandler_FromContext(ctx)
	}
}

func BenchmarkRequest_WithEnrichment_FromContext(b *testing.B) {
	baseLog := setupDiscardLogger(b)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {

		enrichedLog := baseLog.With(
			slog.String("request_id", "request-123"),
			slog.String("user_id", "usr-456"),
		)
		ctx := logger_domain.WithLogger(context.Background(), enrichedLog)

		requestHandler_FromContext(ctx)
	}
}

func BenchmarkRequest_WithEnrichment_LoggerArg(b *testing.B) {
	baseLog := setupDiscardLogger(b)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {

		enrichedLog := baseLog.With(
			slog.String("request_id", "request-123"),
			slog.String("user_id", "usr-456"),
		)
		ctx := context.Background()

		requestHandler_LoggerArg(ctx, enrichedLog)
	}
}

func BenchmarkHasLogger_Present(b *testing.B) {
	log := setupDiscardLogger(b)
	ctx := logger_domain.WithLogger(context.Background(), log)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = logger_domain.HasLogger(ctx)
	}
}

func BenchmarkHasLogger_Absent(b *testing.B) {
	setupDiscardLogger(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = logger_domain.HasLogger(ctx)
	}
}

func BenchmarkCaller(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = logger_domain.Caller()
	}
}

func BenchmarkCaller_VsManualKeyMethod(b *testing.B) {
	b.Run("Caller_Auto", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = logger_domain.Caller()
		}
	})

	b.Run("String_Manual", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = logger_domain.String(logger_domain.KeyMethod, "Container.StartMonitoringService")
		}
	})
}

func BenchmarkCallerComponents(b *testing.B) {
	b.Run("RuntimeCaller_Only", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			pc, _, _, _ := runtime.Caller(1)
			_ = pc
		}
	})

	b.Run("RuntimeCaller_Plus_FuncForPC", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			pc, _, _, _ := runtime.Caller(1)
			runtimeFunction := runtime.FuncForPC(pc)
			_ = runtimeFunction.Name()
		}
	})

	b.Run("CallerPackage_Only", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			pc := caller.Caller(1)
			_ = pc
		}
	})

	b.Run("CallerPackage_Plus_NameFileLine", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			pc := caller.Caller(1)
			name, _, _ := pc.NameFileLine()
			_ = name
		}
	})

	b.Run("Full_CallerInfoAtSkip", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = logger_domain.Caller()
		}
	})
}

func BenchmarkPCCapture_ForSlogRecord(b *testing.B) {
	const skipFrames = 3

	b.Run("RuntimeCallers_Current", func(b *testing.B) {

		b.ReportAllocs()
		for b.Loop() {
			var pcs [1]uintptr
			runtime.Callers(skipFrames, pcs[:])
			_ = pcs[0]
		}
	})

	b.Run("CallerPackage_Proposed", func(b *testing.B) {

		b.ReportAllocs()
		for b.Loop() {
			pc := caller.Caller(skipFrames - 1)
			_ = uintptr(pc)
		}
	})

	b.Run("RuntimeCaller_Single", func(b *testing.B) {

		b.ReportAllocs()
		for b.Loop() {
			pc, _, _, _ := runtime.Caller(skipFrames - 1)
			_ = pc
		}
	})
}

func BenchmarkStackTraceCapture(b *testing.B) {
	const maxFrames = 32
	const skipFrames = 3

	b.Run("RuntimeCallers_Current", func(b *testing.B) {

		b.ReportAllocs()
		for b.Loop() {
			pcs := make([]uintptr, maxFrames)
			n := runtime.Callers(skipFrames, pcs)
			frames := runtime.CallersFrames(pcs[:n])
			for {
				frame, more := frames.Next()
				_ = frame.File
				_ = frame.Line
				if !more {
					break
				}
			}
		}
	})

	b.Run("CallerPackages_Proposed", func(b *testing.B) {

		b.ReportAllocs()
		for b.Loop() {
			pcs := caller.Callers(skipFrames-1, maxFrames)
			for _, pc := range pcs {
				name, file, line := pc.NameFileLine()
				_ = name
				_ = file
				_ = line
			}
		}
	})

	b.Run("CallerPackages_NoIteration", func(b *testing.B) {

		b.ReportAllocs()
		for b.Loop() {
			pcs := caller.Callers(skipFrames-1, maxFrames)
			_ = pcs
		}
	})

	b.Run("RuntimeCallers_NoIteration", func(b *testing.B) {

		b.ReportAllocs()
		for b.Loop() {
			pcs := make([]uintptr, maxFrames)
			n := runtime.Callers(skipFrames, pcs)
			_ = pcs[:n]
		}
	})

}

func BenchmarkAutoCaller_Enabled(b *testing.B) {

	log := setupDiscardLogger(b)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		log.Info("message", slog.String("key", "value"))
	}
}

func BenchmarkAutoCaller_Disabled(b *testing.B) {

	log := setupDiscardLogger(b).WithoutAutoCaller()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		log.Info("message", slog.String("key", "value"))
	}
}

func BenchmarkAutoCaller_DisabledLevel(b *testing.B) {

	otel.SetTracerProvider(noopTracerProvider())
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo})
	baseLog := slog.New(handler)
	slog.SetDefault(baseLog)
	logger_test.InitDefaultFactory(baseLog)
	log := logger_domain.GetLogger("bench")
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		log.Debug("this should be skipped entirely")
	}
}
