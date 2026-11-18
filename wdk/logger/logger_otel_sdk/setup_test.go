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

package logger_otel_sdk

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"

	"piko.sh/piko/internal/logger/logger_adapters/driver_handlers"
)

func TestCreateResource(t *testing.T) {
	t.Parallel()

	otelResource, err := createResource()
	require.NoError(t, err, "merging with the default resource must never conflict")

	defaultSchemaURL := resource.Default().SchemaURL()
	require.NotEmpty(t, defaultSchemaURL, "the default resource must declare a schema URL")
	assert.Equal(t, defaultSchemaURL, otelResource.SchemaURL(),
		"the merged resource must keep the default resource's schema URL")

	serviceName, ok := otelResource.Set().Value(semconv.ServiceNameKey)
	require.True(t, ok)
	assert.NotEmpty(t, serviceName.AsString())

	serviceVersion, ok := otelResource.Set().Value(semconv.ServiceVersionKey)
	require.True(t, ok)
	assert.NotEmpty(t, serviceVersion.AsString())
}

type recordingCloser struct {
	err    error
	closed atomic.Int32
}

func (c *recordingCloser) Close() error {
	c.closed.Add(1)
	return c.err
}

type keepingSpanExporter struct {
	*tracetest.InMemoryExporter
	shutdown atomic.Int32
}

func (e *keepingSpanExporter) Shutdown(_ context.Context) error {
	e.shutdown.Add(1)
	return nil
}

type recordingMetricExporter struct {
	exported atomic.Int32
	shutdown atomic.Int32
}

func (*recordingMetricExporter) Temporality(kind sdkmetric.InstrumentKind) metricdata.Temporality {
	return sdkmetric.DefaultTemporalitySelector(kind)
}

func (*recordingMetricExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(kind)
}

func (e *recordingMetricExporter) Export(_ context.Context, _ *metricdata.ResourceMetrics) error {
	e.exported.Add(1)
	return nil
}

func (*recordingMetricExporter) ForceFlush(_ context.Context) error {
	return nil
}

func (e *recordingMetricExporter) Shutdown(_ context.Context) error {
	e.shutdown.Add(1)
	return nil
}

type failingShutdownProcessor struct {
	sdktrace.SpanProcessor
}

func (failingShutdownProcessor) Shutdown(_ context.Context) error {
	return errProcessorShutdown
}

type failingShutdownReader struct {
	*sdkmetric.ManualReader
}

func (failingShutdownReader) Shutdown(_ context.Context) error {
	return errReaderShutdown
}

type fakeProtocol struct {
	traceExporter  any
	metricExporter any
	traceErr       error
	metricErr      error
	traceCloser    *recordingCloser
	metricCloser   *recordingCloser
}

var (
	errProcessorShutdown = errors.New("processor shutdown failed")
	errReaderShutdown    = errors.New("reader shutdown failed")
	errFactory           = errors.New("factory failed")
)

func registerFakeProtocol(t *testing.T, protocol *fakeProtocol) string {
	t.Helper()

	name := "test-" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	driver_handlers.RegisterOtlpProtocol(name, driver_handlers.OtlpProtocol{
		TraceExporterFactory: func(context.Context, driver_handlers.OtelSetupConfig) (any, io.Closer, error) {
			if protocol.traceErr != nil {
				return nil, nil, protocol.traceErr
			}
			return protocol.traceExporter, closerOrNil(protocol.traceCloser), nil
		},
		MetricExporterFactory: func(context.Context, driver_handlers.OtelSetupConfig) (any, io.Closer, error) {
			if protocol.metricErr != nil {
				return nil, nil, protocol.metricErr
			}
			return protocol.metricExporter, closerOrNil(protocol.metricCloser), nil
		},
	})

	return name
}

func closerOrNil(closer *recordingCloser) io.Closer {
	if closer == nil {
		return nil
	}
	return closer
}

func TestCreateProvidersLocalUsesSuppliedProcessorsAndReaders(t *testing.T) {
	t.Parallel()

	recorder := tracetest.NewSpanRecorder()
	reader := sdkmetric.NewManualReader()

	result, err := createProviders(t.Context(),
		driver_handlers.OtelSetupConfig{TraceSampleRate: 1},
		[]any{recorder, "not a processor"},
		[]any{reader, 42},
	)
	require.NoError(t, err)
	require.NotNil(t, result.ShutdownFunc)
	assert.Empty(t, result.Closers, "a local setup opens no connections")

	tracerProvider, ok := result.TracerProvider.(*sdktrace.TracerProvider)
	require.True(t, ok)
	_, span := tracerProvider.Tracer("test").Start(t.Context(), "operation")
	span.End()

	meterProvider, ok := result.MeterProvider.(*sdkmetric.MeterProvider)
	require.True(t, ok)
	counter, err := meterProvider.Meter("test").Int64Counter("requests")
	require.NoError(t, err)
	counter.Add(t.Context(), 3)

	var collected metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &collected))
	require.Len(t, collected.ScopeMetrics, 1)
	assert.Equal(t, "requests", collected.ScopeMetrics[0].Metrics[0].Name)

	require.Len(t, recorder.Ended(), 1)
	assert.Equal(t, "operation", recorder.Ended()[0].Name())

	require.NoError(t, result.ShutdownFunc(t.Context()))
}

func TestCreateProvidersOtlpExportsThroughRegisteredProtocol(t *testing.T) {
	t.Parallel()

	traceExporter := &keepingSpanExporter{InMemoryExporter: tracetest.NewInMemoryExporter()}
	metricExporter := &recordingMetricExporter{}
	protocol := &fakeProtocol{
		traceExporter:  traceExporter,
		metricExporter: metricExporter,
		traceCloser:    &recordingCloser{},
		metricCloser:   &recordingCloser{},
	}
	name := registerFakeProtocol(t, protocol)

	result, err := createProviders(t.Context(),
		driver_handlers.OtelSetupConfig{Enabled: true, Protocol: name, TraceSampleRate: 1},
		[]any{tracetest.NewSpanRecorder()},
		[]any{sdkmetric.NewManualReader()},
	)
	require.NoError(t, err)
	assert.Len(t, result.Closers, 2)

	tracerProvider, ok := result.TracerProvider.(*sdktrace.TracerProvider)
	require.True(t, ok)
	_, span := tracerProvider.Tracer("test").Start(t.Context(), "exported")
	span.End()

	require.NoError(t, result.ShutdownFunc(t.Context()))

	require.Len(t, traceExporter.GetSpans(), 1, "shutdown flushes the batch processor")
	assert.Equal(t, "exported", traceExporter.GetSpans()[0].Name)
	assert.Equal(t, int32(1), traceExporter.shutdown.Load())
	assert.Positive(t, metricExporter.exported.Load(), "shutdown runs a final metric export")
	assert.Equal(t, int32(1), metricExporter.shutdown.Load())
	assert.Equal(t, int32(1), protocol.traceCloser.closed.Load())
	assert.Equal(t, int32(1), protocol.metricCloser.closed.Load())
}

func TestCreateProvidersRejectsUnknownProtocol(t *testing.T) {
	t.Parallel()

	_, err := createProviders(t.Context(),
		driver_handlers.OtelSetupConfig{Enabled: true, Protocol: "no-such-protocol"}, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create tracer provider")
	assert.Contains(t, err.Error(), "unsupported OTLP protocol")
}

func TestCreateProvidersTraceExporterFailures(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		protocol     *fakeProtocol
		expectedText string
		wantClosed   int32
	}{
		{
			name:         "factory error",
			protocol:     &fakeProtocol{traceErr: errFactory},
			expectedText: errFactory.Error(),
		},
		{
			name:         "wrong exporter type closes the connection",
			protocol:     &fakeProtocol{traceExporter: "not an exporter", traceCloser: &recordingCloser{}},
			expectedText: "expected sdktrace.SpanExporter",
			wantClosed:   1,
		},
		{
			name:         "wrong exporter type without a connection",
			protocol:     &fakeProtocol{traceExporter: "not an exporter"},
			expectedText: "expected sdktrace.SpanExporter",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			name := registerFakeProtocol(t, testCase.protocol)

			_, err := createProviders(t.Context(),
				driver_handlers.OtelSetupConfig{Enabled: true, Protocol: name}, nil, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create tracer provider")
			assert.Contains(t, err.Error(), testCase.expectedText)
			if testCase.protocol.traceCloser != nil {
				assert.Equal(t, testCase.wantClosed, testCase.protocol.traceCloser.closed.Load())
			}
		})
	}
}

func TestCreateProvidersMeterFailureReleasesTracerResources(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		protocol     *fakeProtocol
		expectedText string
	}{
		{
			name:         "factory error",
			protocol:     &fakeProtocol{metricErr: errFactory},
			expectedText: errFactory.Error(),
		},
		{
			name: "wrong exporter type closes the connection",
			protocol: &fakeProtocol{
				metricExporter: "not an exporter",
				metricCloser:   &recordingCloser{},
			},
			expectedText: "expected sdkmetric.Exporter",
		},
		{
			name:         "wrong exporter type without a connection",
			protocol:     &fakeProtocol{metricExporter: "not an exporter"},
			expectedText: "expected sdkmetric.Exporter",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			traceExporter := &keepingSpanExporter{InMemoryExporter: tracetest.NewInMemoryExporter()}
			testCase.protocol.traceExporter = traceExporter
			testCase.protocol.traceCloser = &recordingCloser{}
			name := registerFakeProtocol(t, testCase.protocol)

			_, err := createProviders(t.Context(),
				driver_handlers.OtelSetupConfig{Enabled: true, Protocol: name, TraceSampleRate: 1}, nil, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create meter provider")
			assert.Contains(t, err.Error(), testCase.expectedText)

			assert.Equal(t, int32(1), testCase.protocol.traceCloser.closed.Load(),
				"the trace connection opened before the failure must be closed")
			assert.Equal(t, int32(1), traceExporter.shutdown.Load(),
				"the trace exporter created before the failure must be shut down")
			if testCase.protocol.metricCloser != nil {
				assert.Equal(t, int32(1), testCase.protocol.metricCloser.closed.Load())
			}
		})
	}
}

func TestBuildShutdownFuncJoinsEveryError(t *testing.T) {
	t.Parallel()

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(failingShutdownProcessor{SpanProcessor: tracetest.NewSpanRecorder()}))
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(failingShutdownReader{ManualReader: sdkmetric.NewManualReader()}))
	errClose := errors.New("connection close failed")
	failingCloser := &recordingCloser{err: errClose}
	healthyCloser := &recordingCloser{}

	err := buildShutdownFunc(tracerProvider, meterProvider, []io.Closer{failingCloser, healthyCloser})(t.Context())

	require.Error(t, err)
	require.ErrorIs(t, err, errProcessorShutdown)
	require.ErrorIs(t, err, errReaderShutdown)
	require.ErrorIs(t, err, errClose)
	assert.Contains(t, err.Error(), "tracerProvider shutdown")
	assert.Contains(t, err.Error(), "meterProvider shutdown")
	assert.Contains(t, err.Error(), "OTLP connection close")
	assert.Equal(t, int32(1), healthyCloser.closed.Load(), "one failing closer must not skip the rest")
}

func TestBuildShutdownFuncReportsCancelledContext(t *testing.T) {
	t.Parallel()

	errCause := errors.New("shutdown deadline passed")
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(errCause)

	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(tracetest.NewSpanRecorder()))

	err := buildShutdownFunc(tracerProvider, nil, nil)(ctx)

	require.ErrorIs(t, err, context.Canceled)
	assert.Contains(t, err.Error(), "tracerProvider shutdown")
	assert.ErrorIs(t, context.Cause(ctx), errCause)
}

func TestBuildShutdownFuncWithNothingToStop(t *testing.T) {
	t.Parallel()

	assert.NoError(t, buildShutdownFunc(nil, nil, nil)(t.Context()))
}

func TestCollectTypedSkipsOtherTypes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"a", "b"}, collectTyped[string]([]any{"a", 1, nil, "b", 2.5}))
	assert.Empty(t, collectTyped[string](nil))
}

func TestCreateProvidersReportsConnectionCloseFailure(t *testing.T) {
	t.Parallel()

	errClose := errors.New("connection close failed")
	testCases := []struct {
		protocol     *fakeProtocol
		name         string
		expectedText string
	}{
		{
			name:         "trace exporter of the wrong type",
			protocol:     &fakeProtocol{traceExporter: "not an exporter", traceCloser: &recordingCloser{err: errClose}},
			expectedText: "expected sdktrace.SpanExporter",
		},
		{
			name: "metric exporter of the wrong type",
			protocol: &fakeProtocol{
				traceExporter:  &keepingSpanExporter{InMemoryExporter: tracetest.NewInMemoryExporter()},
				traceCloser:    &recordingCloser{},
				metricExporter: "not an exporter",
				metricCloser:   &recordingCloser{err: errClose},
			},
			expectedText: "expected sdkmetric.Exporter",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			name := registerFakeProtocol(t, testCase.protocol)

			_, err := createProviders(t.Context(),
				driver_handlers.OtelSetupConfig{Enabled: true, Protocol: name, TraceSampleRate: 1}, nil, nil)

			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.expectedText)
			require.ErrorIs(t, err, errClose, "a failed close on the setup failure path must be reported")
		})
	}
}

func TestReleaseExporter(t *testing.T) {
	t.Parallel()

	errShutdown := errors.New("exporter shutdown failed")
	errClose := errors.New("connection close failed")
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(errors.New("test: caller gave up"))

	var shutdownCtxErr error
	closer := &recordingCloser{err: errClose}
	err := releaseExporter(ctx, func(shutdownCtx context.Context) error {
		shutdownCtxErr = shutdownCtx.Err()
		return errShutdown
	}, closer)

	require.ErrorIs(t, err, errShutdown)
	require.ErrorIs(t, err, errClose)
	require.NoError(t, shutdownCtxErr, "the exporter is shut down with a context that is not already cancelled")
	assert.Equal(t, int32(1), closer.closed.Load())

	require.NoError(t, releaseExporter(ctx, nil, nil), "nothing to release is not an error")
}
