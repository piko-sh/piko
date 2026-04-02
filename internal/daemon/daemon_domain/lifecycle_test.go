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

package daemon_domain

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"piko.sh/piko/internal/tlscert"
)

func TestStartHTTPServers_StartsMainServer(t *testing.T) {
	t.Parallel()

	serverStarted := make(chan bool, 1)
	mockServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			serverStarted <- true
			return http.ErrServerClosed
		},
	}

	deps := &DaemonServiceDeps{
		DaemonConfig: testDaemonConfig(),
		Server:       mockServer,
		FinalRouter:  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	ctx, cancel := context.WithTimeoutCause(context.Background(), time.Second, fmt.Errorf("test: startHTTPServers exceeded %s timeout", time.Second))
	defer cancel()

	errChan := service.startHTTPServers(ctx)

	select {
	case <-serverStarted:

	case err := <-errChan:
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	case <-ctx.Done():
		t.Error("Timeout waiting for server to start")
	}
}

func TestStartHTTPServers_StartsHealthServer_WhenConfigured(t *testing.T) {
	t.Parallel()

	mainServerStarted := make(chan bool, 1)
	healthServerStarted := make(chan bool, 1)

	mockServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			mainServerStarted <- true
			return http.ErrServerClosed
		},
	}
	mockHealthServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			healthServerStarted <- true
			return http.ErrServerClosed
		},
	}

	deps := &DaemonServiceDeps{
		DaemonConfig: testDaemonConfigWithHealthProbe(),
		Server:       mockServer,
		FinalRouter:  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
		HealthServer: mockHealthServer,
		HealthRouter: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	ctx, cancel := context.WithTimeoutCause(context.Background(), time.Second, fmt.Errorf("test: startHTTPServers health exceeded %s timeout", time.Second))
	defer cancel()

	_ = service.startHTTPServers(ctx)

	var mainStarted, healthStarted bool
	timeout := time.After(time.Second)

	for !mainStarted || !healthStarted {
		select {
		case <-mainServerStarted:
			mainStarted = true
		case <-healthServerStarted:
			healthStarted = true
		case <-timeout:
			if !healthStarted {
				t.Error("Health server was not started")
			}
			return
		}
	}
}

func TestStartHTTPServers_SkipsHealthServer_WhenNotConfigured(t *testing.T) {
	t.Parallel()

	mockServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			return http.ErrServerClosed
		},
	}
	mockHealthServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			t.Error("Health server should not be started")
			return http.ErrServerClosed
		},
	}

	daemonConfig := testDaemonConfig()
	daemonConfig.HealthEnabled = false

	deps := &DaemonServiceDeps{
		DaemonConfig: daemonConfig,
		Server:       mockServer,
		FinalRouter:  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
		HealthServer: mockHealthServer,
		HealthRouter: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 100*time.Millisecond, fmt.Errorf("test: SkipsHealthServer timeout"))
	defer cancel()

	errChan := service.startHTTPServers(ctx)

	select {
	case <-errChan:

	case <-ctx.Done():

	}
}

func TestStartServer_UsesConfiguredPort(t *testing.T) {
	t.Parallel()

	var usedAddress string
	mockServer := &MockServerAdapter{
		ListenAndServeFunc: func(addr string, _ http.Handler) error {
			usedAddress = addr
			return http.ErrServerClosed
		},
	}

	daemonConfig := testDaemonConfig()
	daemonConfig.NetworkPort = "9999"

	deps := &DaemonServiceDeps{
		DaemonConfig: daemonConfig,
		Server:       mockServer,
		FinalRouter:  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	_ = service.startServer(context.Background(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))

	if usedAddress != ":9999" {
		t.Errorf("Expected address ':9999', got %q", usedAddress)
	}
}

func TestStartServer_RetriesOnPortInUse_WhenAutoNextPortEnabled(t *testing.T) {
	t.Parallel()

	var attempts int
	var mu sync.Mutex

	mockServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			mu.Lock()
			attempts++
			currentAttempt := attempts
			mu.Unlock()

			if currentAttempt < 3 {
				return newPortInUseError()
			}
			return http.ErrServerClosed
		},
	}

	daemonConfig := testDaemonConfig()
	daemonConfig.NetworkPort = "8080"
	daemonConfig.NetworkAutoNextPort = true

	deps := &DaemonServiceDeps{
		DaemonConfig: daemonConfig,
		Server:       mockServer,
		FinalRouter:  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	err := service.startServer(context.Background(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	mu.Lock()
	finalAttempts := attempts
	mu.Unlock()

	if finalAttempts < 3 {
		t.Errorf("Expected at least 3 attempts, got %d", finalAttempts)
	}
}

func TestStartServer_FailsImmediately_WhenAutoNextPortDisabled(t *testing.T) {
	t.Parallel()

	mockServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			return newPortInUseError()
		},
	}

	daemonConfig := testDaemonConfig()
	daemonConfig.NetworkPort = "8080"
	daemonConfig.NetworkAutoNextPort = false

	deps := &DaemonServiceDeps{
		DaemonConfig: daemonConfig,
		Server:       mockServer,
		FinalRouter:  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	err := service.startServer(context.Background(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))

	if err == nil {
		t.Error("Expected error when port in use and AutoNextPort disabled")
	}
}

func TestStartServer_ReturnsError_AfterMaxRetries(t *testing.T) {
	t.Parallel()

	mockServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			return newPortInUseError()
		},
	}

	daemonConfig := testDaemonConfig()
	daemonConfig.NetworkPort = "8080"
	daemonConfig.NetworkAutoNextPort = true

	deps := &DaemonServiceDeps{
		DaemonConfig: daemonConfig,
		Server:       mockServer,
		FinalRouter:  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	err := service.startServer(context.Background(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))

	if err == nil {
		t.Error("Expected error after max retries")
	}
}

func TestStartServer_ReturnsNil_OnGracefulShutdown(t *testing.T) {
	t.Parallel()

	mockServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			return http.ErrServerClosed
		},
	}

	deps := &DaemonServiceDeps{
		DaemonConfig: testDaemonConfig(),
		Server:       mockServer,
		FinalRouter:  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	err := service.startServer(context.Background(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))

	if err != nil {
		t.Errorf("Expected nil error on graceful shutdown, got %v", err)
	}
}

func TestStartServer_ReturnsError_OnInvalidPort(t *testing.T) {
	t.Parallel()

	mockServer := &MockServerAdapter{}

	daemonConfig := testDaemonConfig()
	daemonConfig.NetworkPort = "invalid"

	deps := &DaemonServiceDeps{
		DaemonConfig: daemonConfig,
		Server:       mockServer,
		FinalRouter:  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	err := service.startServer(context.Background(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))

	if err == nil {
		t.Error("Expected error for invalid port")
	}
}

func TestStartHealthServer_StartsOnConfiguredAddress(t *testing.T) {
	t.Parallel()

	var usedAddress string
	mockHealthServer := &MockServerAdapter{
		ListenAndServeFunc: func(addr string, _ http.Handler) error {
			usedAddress = addr
			return http.ErrServerClosed
		},
	}

	daemonConfig := testDaemonConfigWithHealthProbe()
	daemonConfig.HealthBindAddress = "0.0.0.0"
	daemonConfig.HealthPort = "9090"

	deps := &DaemonServiceDeps{
		DaemonConfig: daemonConfig,
		HealthServer: mockHealthServer,
		HealthRouter: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	_ = service.startHealthServer(context.Background())

	if usedAddress != "0.0.0.0:9090" {
		t.Errorf("Expected address '0.0.0.0:9090', got %q", usedAddress)
	}
}

func TestStartHealthServer_ReturnsNil_OnGracefulShutdown(t *testing.T) {
	t.Parallel()

	mockHealthServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			return http.ErrServerClosed
		},
	}

	deps := &DaemonServiceDeps{
		DaemonConfig: testDaemonConfigWithHealthProbe(),
		HealthServer: mockHealthServer,
		HealthRouter: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	err := service.startHealthServer(context.Background())

	if err != nil {
		t.Errorf("Expected nil error on graceful shutdown, got %v", err)
	}
}

func TestStartHealthServer_ReturnsError_OnFailure(t *testing.T) {
	t.Parallel()

	healthErr := errors.New("health server failed")
	mockHealthServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			return healthErr
		},
	}

	deps := &DaemonServiceDeps{
		DaemonConfig: testDaemonConfigWithHealthProbe(),
		HealthServer: mockHealthServer,
		HealthRouter: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	err := service.startHealthServer(context.Background())

	if err == nil {
		t.Error("Expected error on health server failure")
	}
}

func TestWaitForShutdown_ReturnsNil_OnContextDone(t *testing.T) {
	t.Parallel()

	deps := &DaemonServiceDeps{}

	service := mustBuildDaemonService(t, deps)

	ctx, cancel := context.WithCancelCause(context.Background())
	serverErrChan := make(chan error, 1)

	var wg sync.WaitGroup
	var result error

	wg.Go(func() {
		result = service.waitForShutdown(ctx, getNoopSpan(), serverErrChan)
	})

	cancel(fmt.Errorf("test: simulating cancelled context"))
	wg.Wait()

	if result != nil {
		t.Errorf("Expected nil error, got %v", result)
	}
}

func TestWaitForShutdown_ReturnsError_OnServerError(t *testing.T) {
	t.Parallel()

	deps := &DaemonServiceDeps{}

	service := mustBuildDaemonService(t, deps)

	serverErrChan := make(chan error, 1)

	serverErr := errors.New("server error")
	serverErrChan <- serverErr

	result := service.waitForShutdown(context.Background(), getNoopSpan(), serverErrChan)

	if !errors.Is(result, serverErr) {
		t.Errorf("Expected server error, got %v", result)
	}
}

func TestWaitForShutdown_ReturnsNil_OnStopChan(t *testing.T) {
	t.Parallel()

	deps := &DaemonServiceDeps{}

	service := mustBuildDaemonService(t, deps)

	serverErrChan := make(chan error, 1)

	var wg sync.WaitGroup
	var result error

	wg.Go(func() {
		result = service.waitForShutdown(context.Background(), getNoopSpan(), serverErrChan)
	})

	close(service.stopChan)
	wg.Wait()

	if result != nil {
		t.Errorf("Expected nil error, got %v", result)
	}
}

func TestShutdown_ShutsDownMainServer(t *testing.T) {
	t.Parallel()

	mockServer := &MockServerAdapter{}

	deps := &DaemonServiceDeps{
		Server: mockServer,
	}

	service := mustBuildDaemonService(t, deps)

	_ = service.shutdown(context.Background())

	if atomic.LoadInt64(&mockServer.ShutdownCallCount) == 0 {
		t.Error("Expected Shutdown to be called on main server")
	}
}

func TestShutdown_ShutsDownHealthServer(t *testing.T) {
	t.Parallel()

	mockHealthServer := &MockServerAdapter{}

	deps := &DaemonServiceDeps{
		HealthServer: mockHealthServer,
	}

	service := mustBuildDaemonService(t, deps)

	_ = service.shutdown(context.Background())

	if atomic.LoadInt64(&mockHealthServer.ShutdownCallCount) == 0 {
		t.Error("Expected Shutdown to be called on health server")
	}
}

func TestShutdown_ReturnsMainServerError(t *testing.T) {
	t.Parallel()

	shutdownErr := errors.New("main server shutdown failed")
	mockServer := &MockServerAdapter{
		ShutdownFunc: func(_ context.Context) error {
			return shutdownErr
		},
	}

	deps := &DaemonServiceDeps{
		Server: mockServer,
	}

	service := mustBuildDaemonService(t, deps)

	err := service.shutdown(context.Background())

	if !errors.Is(err, shutdownErr) {
		t.Errorf("Expected main server error, got %v", err)
	}
}

func TestShutdown_ReturnsNil_OnSuccess(t *testing.T) {
	t.Parallel()

	mockServer := &MockServerAdapter{
		ShutdownFunc: func(_ context.Context) error {
			return nil
		},
	}
	mockHealthServer := &MockServerAdapter{
		ShutdownFunc: func(_ context.Context) error {
			return nil
		},
	}

	deps := &DaemonServiceDeps{
		Server:       mockServer,
		HealthServer: mockHealthServer,
	}

	service := mustBuildDaemonService(t, deps)

	err := service.shutdown(context.Background())

	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func TestShouldStartHealthServer_ReturnsTrue_WhenFullyConfigured(t *testing.T) {
	t.Parallel()

	mockHealthServer := &MockServerAdapter{}

	deps := &DaemonServiceDeps{
		DaemonConfig: testDaemonConfigWithHealthProbe(),
		HealthServer: mockHealthServer,
		HealthRouter: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	result := service.shouldStartHealthServer()

	if !result {
		t.Error("Expected true when health server fully configured")
	}
}

func TestShouldStartHealthServer_ReturnsFalse_WhenDisabled(t *testing.T) {
	t.Parallel()

	mockHealthServer := &MockServerAdapter{}

	daemonConfig := testDaemonConfig()
	daemonConfig.HealthEnabled = false

	deps := &DaemonServiceDeps{
		DaemonConfig: daemonConfig,
		HealthServer: mockHealthServer,
		HealthRouter: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	result := service.shouldStartHealthServer()

	if result {
		t.Error("Expected false when health probe disabled")
	}
}

func TestShouldStartHealthServer_ReturnsFalse_WhenServerNil(t *testing.T) {
	t.Parallel()

	deps := &DaemonServiceDeps{
		DaemonConfig: testDaemonConfigWithHealthProbe(),
		HealthServer: nil,
		HealthRouter: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	result := service.shouldStartHealthServer()

	if result {
		t.Error("Expected false when health server is nil")
	}
}

func TestShouldStartHealthServer_ReturnsFalse_WhenRouterNil(t *testing.T) {
	t.Parallel()

	mockHealthServer := &MockServerAdapter{}

	deps := &DaemonServiceDeps{
		DaemonConfig: testDaemonConfigWithHealthProbe(),
		HealthServer: mockHealthServer,
		HealthRouter: nil,
	}

	service := mustBuildDaemonService(t, deps)

	result := service.shouldStartHealthServer()

	if result {
		t.Error("Expected false when health router is nil")
	}
}

func TestHandleServerError_ReturnsNil_ForNilError(t *testing.T) {
	t.Parallel()

	service := &daemonService{}

	result := service.handleServerError(context.Background(), getNoopSpan(), nil)

	if result != nil {
		t.Errorf("Expected nil for nil error, got %v", result)
	}
}

func TestHandleServerError_ReturnsNil_ForServerClosed(t *testing.T) {
	t.Parallel()

	service := &daemonService{}

	result := service.handleServerError(context.Background(), getNoopSpan(), http.ErrServerClosed)

	if result != nil {
		t.Errorf("Expected nil for ErrServerClosed, got %v", result)
	}
}

func TestHandleServerError_ReturnsError_ForOtherErrors(t *testing.T) {
	t.Parallel()

	service := &daemonService{}
	otherErr := errors.New("some other error")

	result := service.handleServerError(context.Background(), getNoopSpan(), otherErr)

	if !errors.Is(result, otherErr) {
		t.Errorf("Expected error to be passed through, got %v", result)
	}
}

func newPortInUseError() error {
	return &net.OpError{
		Op:  "listen",
		Net: "tcp",
		Err: errors.New("address already in use"),
	}
}

func getNoopSpan() trace.Span {
	tracer := noop.NewTracerProvider().Tracer("test")
	_, span := tracer.Start(context.Background(), "test")
	return span
}

func mustBuildDaemonService(t *testing.T, deps *DaemonServiceDeps) *daemonService {
	t.Helper()

	service := NewService(context.Background(), deps)
	ds, ok := service.(*daemonService)
	if !ok {
		t.Fatalf("expected *daemonService, got %T", service)
	}
	return ds
}

func TestExtractTraceContext_ExtractsHeadersToCarrier(t *testing.T) {
	t.Parallel()

	request, _ := http.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	request.Header.Set("tracestate", "congo=t61rcWkgMzE")

	ctx := extractTraceContext(request)

	if ctx == nil {
		t.Error("extractTraceContext returned nil context")
	}
}

func TestExtractTraceContext_HandlesEmptyHeaders(t *testing.T) {
	t.Parallel()

	request, _ := http.NewRequest(http.MethodGet, "/test", nil)

	ctx := extractTraceContext(request)

	if ctx == nil {
		t.Error("extractTraceContext returned nil context for empty headers")
	}
}

func TestExtractTraceContext_HandlesMultiValueHeaders(t *testing.T) {
	t.Parallel()

	request, _ := http.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Add("X-Custom-Header", "value1")
	request.Header.Add("X-Custom-Header", "value2")

	ctx := extractTraceContext(request)

	if ctx == nil {
		t.Error("extractTraceContext returned nil context")
	}
}

func TestCreateTracingHandler_WrapsRouter(t *testing.T) {
	t.Parallel()

	routerCalled := false
	mockRouter := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		routerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	service := &daemonService{
		finalRouter: mockRouter,
	}

	handler := service.createTracingHandler()

	request, _ := http.NewRequest(http.MethodGet, "/test", nil)
	rr := &mockResponseWriter{headers: make(http.Header)}

	handler.ServeHTTP(rr, request)

	if !routerCalled {
		t.Error("createTracingHandler should call the final router")
	}
}

func TestCreateTracingHandler_PassesContextToRouter(t *testing.T) {
	t.Parallel()

	var receivedCtx context.Context
	mockRouter := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		receivedCtx = r.Context()
	})

	service := &daemonService{
		finalRouter: mockRouter,
	}

	handler := service.createTracingHandler()

	request, _ := http.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	rr := &mockResponseWriter{headers: make(http.Header)}

	handler.ServeHTTP(rr, request)

	if receivedCtx == nil {
		t.Error("createTracingHandler should pass context to router")
	}
}

type mockResponseWriter struct {
	headers    http.Header
	body       []byte
	statusCode int
}

func (m *mockResponseWriter) Header() http.Header {
	return m.headers
}

func (m *mockResponseWriter) Write(data []byte) (int, error) {
	m.body = append(m.body, data...)
	return len(data), nil
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

func TestHandleServerListenResult_Success_ReturnsNil(t *testing.T) {
	t.Parallel()

	service := &daemonService{}

	err := service.handleServerListenResult(context.Background(), getNoopSpan(), nil, ":8080", true, 1, serverKindMain)

	if err != nil {
		t.Errorf("handleServerListenResult should return nil on success, got %v", err)
	}
}

func TestHandleServerListenResult_ServerClosed_ReturnsNil(t *testing.T) {
	t.Parallel()

	service := &daemonService{}

	err := service.handleServerListenResult(context.Background(), getNoopSpan(), http.ErrServerClosed, ":8080", true, 1, serverKindMain)

	if err != nil {
		t.Errorf("handleServerListenResult should return nil for ErrServerClosed, got %v", err)
	}
}

func TestHandleServerListenResult_PortInUse_WithAutoNext_ReturnsContinueRetry(t *testing.T) {
	t.Parallel()

	service := &daemonService{}

	portErr := newPortInUseError()
	err := service.handleServerListenResult(context.Background(), getNoopSpan(), portErr, ":8080", true, 1, serverKindMain)

	if !errors.Is(err, errContinueRetry) {
		t.Errorf("handleServerListenResult should return errContinueRetry when port in use with autoNextPort, got %v", err)
	}
}

func TestHandleServerListenResult_PortInUse_WithoutAutoNext_ReturnsError(t *testing.T) {
	t.Parallel()

	service := &daemonService{}

	portErr := newPortInUseError()
	err := service.handleServerListenResult(context.Background(), getNoopSpan(), portErr, ":8080", false, 1, serverKindMain)

	if err == nil {
		t.Error("handleServerListenResult should return error when port in use without autoNextPort")
	}
}

func TestHandleServerListenResult_OtherError_ReturnsError(t *testing.T) {
	t.Parallel()

	service := &daemonService{}

	otherErr := errors.New("some other error")
	err := service.handleServerListenResult(context.Background(), getNoopSpan(), otherErr, ":8080", true, 1, serverKindMain)

	if !errors.Is(err, otherErr) {
		t.Errorf("handleServerListenResult should return the original error, got %v", err)
	}
}

func TestRunDaemonMain_GracefulShutdown(t *testing.T) {
	t.Parallel()

	mockServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			return http.ErrServerClosed
		},
	}

	deps := &DaemonServiceDeps{
		DaemonConfig: testDaemonConfig(),
		Server:       mockServer,
		FinalRouter:  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	err := service.runDaemonMain(context.Background())

	if err != nil {
		t.Errorf("expected nil error on graceful shutdown, got %v", err)
	}
}

func TestRunDaemonMain_PropagatesServerError(t *testing.T) {
	t.Parallel()

	serverErr := errors.New("server failed")
	mockServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			return serverErr
		},
	}

	deps := &DaemonServiceDeps{
		DaemonConfig: testDaemonConfig(),
		Server:       mockServer,
		FinalRouter:  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	err := service.runDaemonMain(context.Background())

	if !errors.Is(err, serverErr) {
		t.Errorf("expected server error, got %v", err)
	}
}

func TestRunDaemonMain_StopsOnStopChan(t *testing.T) {
	t.Parallel()

	mockServer := newMockServerAdapter()
	t.Cleanup(func() { _ = mockServer.Shutdown(context.Background()) })
	mockServer.ListenAndServeFunc = func(_ string, _ http.Handler) error {
		select {
		case <-mockServer.shutdownCh:
			return http.ErrServerClosed
		case <-time.After(5 * time.Second):
			return http.ErrServerClosed
		}
	}

	deps := &DaemonServiceDeps{
		DaemonConfig: testDaemonConfig(),
		Server:       mockServer,
		FinalRouter:  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	done := make(chan error, 1)
	go func() {
		done <- service.runDaemonMain(context.Background())
	}()

	time.Sleep(50 * time.Millisecond)

	close(service.stopChan)

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected nil error on stop, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("runDaemonMain did not return after stopChan close")
	}
}

func TestReportNoHealthPortAvailable_ReturnsError(t *testing.T) {
	t.Parallel()

	service := &daemonService{}

	err := service.reportNoHealthPortAvailable(context.Background(), getNoopSpan(), 9090)

	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "health server") {
		t.Errorf("expected error to mention 'health server', got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "9090") {
		t.Errorf("expected error to mention port 9090, got %q", err.Error())
	}
}

func TestReportNoPortAvailable_ReturnsError(t *testing.T) {
	t.Parallel()

	service := &daemonService{}

	err := service.reportNoPortAvailable(context.Background(), getNoopSpan(), 8080)

	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "8080") {
		t.Errorf("expected error to mention port 8080, got %q", err.Error())
	}
}

func TestStartHealthServer_ReturnsError_OnInvalidPort(t *testing.T) {
	t.Parallel()

	mockHealthServer := &MockServerAdapter{}

	daemonConfig := testDaemonConfigWithHealthProbe()
	daemonConfig.HealthPort = "invalid"

	deps := &DaemonServiceDeps{
		DaemonConfig: daemonConfig,
		HealthServer: mockHealthServer,
		HealthRouter: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	err := service.startHealthServer(context.Background())

	if err == nil {
		t.Error("expected error for invalid health port")
	}
}

func TestStartHealthServer_RetriesOnPortInUse_WhenAutoNextPortEnabled(t *testing.T) {
	t.Parallel()

	var attempts int
	var mu sync.Mutex

	mockHealthServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			mu.Lock()
			attempts++
			current := attempts
			mu.Unlock()

			if current < 3 {
				return newPortInUseError()
			}
			return http.ErrServerClosed
		},
	}

	daemonConfig := testDaemonConfigWithHealthProbe()
	daemonConfig.HealthAutoNextPort = true

	deps := &DaemonServiceDeps{
		DaemonConfig: daemonConfig,
		HealthServer: mockHealthServer,
		HealthRouter: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	err := service.startHealthServer(context.Background())

	if err != nil {
		t.Errorf("expected nil error after retry, got %v", err)
	}

	mu.Lock()
	finalAttempts := attempts
	mu.Unlock()

	if finalAttempts < 3 {
		t.Errorf("expected at least 3 attempts, got %d", finalAttempts)
	}
}

func TestHandleServerPortInUse_HealthServerKind(t *testing.T) {
	t.Parallel()

	service := &daemonService{}
	portErr := newPortInUseError()

	err := service.handleServerPortInUse(context.Background(), getNoopSpan(), portErr, "127.0.0.1:9090", true, 0, serverKindHealth)

	if !errors.Is(err, errContinueRetry) {
		t.Errorf("expected errContinueRetry, got %v", err)
	}

	err = service.handleServerPortInUse(context.Background(), getNoopSpan(), portErr, "127.0.0.1:9090", false, 0, serverKindHealth)

	if err == nil {
		t.Error("expected error when autoNextPort is disabled")
	}
	if errors.Is(err, errContinueRetry) {
		t.Error("should not return errContinueRetry when autoNextPort is disabled")
	}
}

func TestShutdownServers_BothFail_ReturnsMainError(t *testing.T) {
	t.Parallel()

	mainErr := errors.New("main shutdown failed")
	healthErr := errors.New("health shutdown failed")

	mockServer := &MockServerAdapter{
		ShutdownFunc: func(_ context.Context) error {
			return mainErr
		},
	}
	mockHealthServer := &MockServerAdapter{
		ShutdownFunc: func(_ context.Context) error {
			return healthErr
		},
	}

	deps := &DaemonServiceDeps{
		Server:       mockServer,
		HealthServer: mockHealthServer,
	}

	service := mustBuildDaemonService(t, deps)

	err := service.shutdown(context.Background())

	if !errors.Is(err, mainErr) {
		t.Errorf("expected main server error, got %v", err)
	}
}

func TestShutdown_ReturnsHealthServerError_WhenMainSucceeds(t *testing.T) {
	t.Parallel()

	healthErr := errors.New("health shutdown failed")

	mockServer := &MockServerAdapter{
		ShutdownFunc: func(_ context.Context) error {
			return nil
		},
	}
	mockHealthServer := &MockServerAdapter{
		ShutdownFunc: func(_ context.Context) error {
			return healthErr
		},
	}

	deps := &DaemonServiceDeps{
		Server:       mockServer,
		HealthServer: mockHealthServer,
	}

	service := mustBuildDaemonService(t, deps)

	err := service.shutdown(context.Background())

	if !errors.Is(err, healthErr) {
		t.Errorf("expected health server error, got %v", err)
	}
}

func TestHandleServerListenResult_HealthKind_Success(t *testing.T) {
	t.Parallel()

	service := &daemonService{}

	err := service.handleServerListenResult(context.Background(), getNoopSpan(), http.ErrServerClosed, "127.0.0.1:9090", false, 0, serverKindHealth)

	if err != nil {
		t.Errorf("expected nil for health server graceful close, got %v", err)
	}
}

func TestHandleServerListenResult_HealthKind_OtherError(t *testing.T) {
	t.Parallel()

	service := &daemonService{}
	otherErr := errors.New("health server error")

	err := service.handleServerListenResult(context.Background(), getNoopSpan(), otherErr, "127.0.0.1:9090", false, 0, serverKindHealth)

	if !errors.Is(err, otherErr) {
		t.Errorf("expected original error, got %v", err)
	}
}

func TestStartMainServer_GracefulShutdown(t *testing.T) {
	t.Parallel()

	mockServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			return http.ErrServerClosed
		},
	}

	deps := &DaemonServiceDeps{
		DaemonConfig: testDaemonConfig(),
		Server:       mockServer,
		FinalRouter:  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	err := service.startMainServer(context.Background())

	if err != nil {
		t.Errorf("expected nil error on graceful shutdown, got %v", err)
	}
}

func TestStartMainServer_PropagatesError(t *testing.T) {
	t.Parallel()

	serverErr := errors.New("main server failed")
	mockServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			return serverErr
		},
	}

	deps := &DaemonServiceDeps{
		DaemonConfig: testDaemonConfig(),
		Server:       mockServer,
		FinalRouter:  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	err := service.startMainServer(context.Background())

	if !errors.Is(err, serverErr) {
		t.Errorf("expected server error, got %v", err)
	}
}

func TestStartMainServer_WithTLS_SkipsH2C(t *testing.T) {
	t.Parallel()

	var receivedHandler http.Handler
	mockServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, h http.Handler) error {
			receivedHandler = h
			return http.ErrServerClosed
		},
	}

	daemonConfig := testDaemonConfig()
	daemonConfig.TLS = tlscert.TLSValues{Mode: tlscert.TLSModeCertFile}

	deps := &DaemonServiceDeps{
		DaemonConfig: daemonConfig,
		Server:       mockServer,
		FinalRouter:  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	_ = service.startMainServer(context.Background())

	if receivedHandler == nil {
		t.Fatal("expected handler to be set")
	}

	handlerType := fmt.Sprintf("%T", receivedHandler)
	if strings.Contains(handlerType, "h2c") {
		t.Errorf("expected non-h2c handler when TLS is enabled, got %s", handlerType)
	}
}

func TestShutdown_SignalsDrainBeforeShuttingDown(t *testing.T) {
	t.Parallel()

	var events []string
	var mu sync.Mutex
	record := func(event string) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	}

	drainSignaller := &MockDrainSignaller{}
	mockServer := &MockServerAdapter{
		ShutdownFunc: func(_ context.Context) error {
			if atomic.LoadInt64(&drainSignaller.SignalDrainCallCount) == 0 {
				t.Error("Expected drain to be signalled before main server shutdown")
			}
			record("main_shutdown")
			return nil
		},
	}

	deps := &DaemonServiceDeps{
		Server:         mockServer,
		DrainSignaller: drainSignaller,
	}

	service := mustBuildDaemonService(t, deps)

	_ = service.shutdown(context.Background())

	assert.Greater(t, atomic.LoadInt64(&drainSignaller.SignalDrainCallCount), int64(0))
}

func TestShutdown_WaitsDrainDelay(t *testing.T) {
	t.Parallel()

	drainSignaller := &MockDrainSignaller{}
	mockServer := &MockServerAdapter{}

	daemonConfig := testDaemonConfig()
	daemonConfig.ShutdownDrainDelay = 200 * time.Millisecond

	deps := &DaemonServiceDeps{
		DaemonConfig:   daemonConfig,
		Server:         mockServer,
		DrainSignaller: drainSignaller,
	}

	service := mustBuildDaemonService(t, deps)

	start := time.Now()
	_ = service.shutdown(context.Background())
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed, 150*time.Millisecond, "expected shutdown to wait at least ~200ms for drain delay")
}

func TestShutdown_HealthServerShutsDownAfterMainServer(t *testing.T) {
	t.Parallel()

	var events []string
	var mu sync.Mutex
	record := func(event string) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	}

	mockServer := &MockServerAdapter{
		ShutdownFunc: func(_ context.Context) error {
			record("main_shutdown")
			return nil
		},
	}
	mockHealthServer := &MockServerAdapter{
		ShutdownFunc: func(_ context.Context) error {
			record("health_shutdown")
			return nil
		},
	}

	deps := &DaemonServiceDeps{
		Server:       mockServer,
		HealthServer: mockHealthServer,
	}

	service := mustBuildDaemonService(t, deps)

	_ = service.shutdown(context.Background())

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, events, 2)
	assert.Equal(t, "main_shutdown", events[0])
	assert.Equal(t, "health_shutdown", events[1])
}

func TestShutdown_SkipsDrainDelay_WhenZero(t *testing.T) {
	t.Parallel()

	mockServer := &MockServerAdapter{}
	drainSignaller := &MockDrainSignaller{}

	daemonConfig := testDaemonConfig()
	daemonConfig.ShutdownDrainDelay = 0

	deps := &DaemonServiceDeps{
		DaemonConfig:   daemonConfig,
		Server:         mockServer,
		DrainSignaller: drainSignaller,
	}

	service := mustBuildDaemonService(t, deps)

	start := time.Now()
	_ = service.shutdown(context.Background())
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 50*time.Millisecond, "expected no drain delay when set to 0")
	assert.Greater(t, atomic.LoadInt64(&drainSignaller.SignalDrainCallCount), int64(0))
}

func TestShutdown_NoPanic_WhenDrainSignallerNil(t *testing.T) {
	t.Parallel()

	mockServer := &MockServerAdapter{}

	deps := &DaemonServiceDeps{
		Server:         mockServer,
		DrainSignaller: nil,
	}

	service := mustBuildDaemonService(t, deps)

	assert.NotPanics(t, func() {
		_ = service.shutdown(context.Background())
	})
}

func TestShutdown_DrainWaitRespectsContextDeadline(t *testing.T) {
	t.Parallel()

	drainSignaller := &MockDrainSignaller{}
	mockServer := &MockServerAdapter{}

	daemonConfig := testDaemonConfig()
	daemonConfig.ShutdownDrainDelay = 10 * time.Second

	deps := &DaemonServiceDeps{
		DaemonConfig:   daemonConfig,
		Server:         mockServer,
		DrainSignaller: drainSignaller,
	}

	service := mustBuildDaemonService(t, deps)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 100*time.Millisecond,
		fmt.Errorf("test: short context deadline for drain"))
	defer cancel()

	start := time.Now()
	_ = service.shutdown(ctx)
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 2*time.Second, "drain wait should respect context deadline, not wait full 10s")
}

func TestStartMainServer_WithoutTLS_UsesH2C(t *testing.T) {
	t.Parallel()

	var receivedHandler http.Handler
	mockServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, h http.Handler) error {
			receivedHandler = h
			return http.ErrServerClosed
		},
	}

	daemonConfig := testDaemonConfig()

	deps := &DaemonServiceDeps{
		DaemonConfig: daemonConfig,
		Server:       mockServer,
		FinalRouter:  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
	}

	service := mustBuildDaemonService(t, deps)

	_ = service.startMainServer(context.Background())

	if receivedHandler == nil {
		t.Fatal("expected handler to be set")
	}

	handlerType := fmt.Sprintf("%T", receivedHandler)
	if !strings.Contains(handlerType, "h2c") {
		t.Errorf("expected h2c handler when TLS is off, got %s", handlerType)
	}
}
