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
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/seo/seo_dto"
)

func TestNewService_DefaultsToFallbackSignalNotifier(t *testing.T) {
	t.Parallel()

	deps := &DaemonServiceDeps{
		SignalNotifier: nil,
	}

	service := mustBuildDaemonService(t, deps)

	if service.signalNotifier == nil {
		t.Error("Expected signalNotifier to be set to fallback when nil provided")
	}
}

func TestNewService_UsesProvidedSignalNotifier(t *testing.T) {
	t.Parallel()

	mockNotifier := NewMockSignalNotifier()

	deps := &DaemonServiceDeps{
		SignalNotifier: mockNotifier,
	}

	service := mustBuildDaemonService(t, deps)

	if service.signalNotifier != mockNotifier {
		t.Error("Expected service to use provided signal notifier")
	}
}

func TestNewService_InitialisesStopChannel(t *testing.T) {
	t.Parallel()

	deps := &DaemonServiceDeps{}

	service := mustBuildDaemonService(t, deps)

	if service.stopChan == nil {
		t.Error("Expected stopChan to be initialised")
	}
}

func TestStop_ClosesStopChannel(t *testing.T) {
	t.Parallel()

	deps := &DaemonServiceDeps{}

	service := mustBuildDaemonService(t, deps)

	err := service.Stop(context.Background())

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	select {
	case <-service.stopChan:

	default:
		t.Error("Expected stopChan to be closed")
	}
}

func TestStop_IsIdempotent(t *testing.T) {
	t.Parallel()

	deps := &DaemonServiceDeps{}

	service := mustBuildDaemonService(t, deps)

	err1 := service.Stop(context.Background())
	err2 := service.Stop(context.Background())
	err3 := service.Stop(context.Background())

	if err1 != nil || err2 != nil || err3 != nil {
		t.Error("Expected Stop to be idempotent without errors")
	}
}

func TestStop_CallsServerShutdown(t *testing.T) {
	t.Parallel()

	mockServer := &MockServerAdapter{
		ShutdownFunc: func(_ context.Context) error {
			return nil
		},
	}

	deps := &DaemonServiceDeps{
		Server: mockServer,
	}

	service := mustBuildDaemonService(t, deps)

	_ = service.Stop(context.Background())

	if atomic.LoadInt64(&mockServer.ShutdownCallCount) == 0 {
		t.Error("Expected server Shutdown to be called")
	}
}

func TestHandleBuildNotifications_IgnoresNilResult(t *testing.T) {
	t.Parallel()

	mockSEO := &MockSEOService{}

	deps := &DaemonServiceDeps{
		SEOService: mockSEO,
	}

	service := mustBuildDaemonService(t, deps)

	notification := coordinator_domain.BuildNotification{
		CausationID: "test",
		Result:      nil,
	}

	service.processNotification(context.Background(), &notification)

	if atomic.LoadInt64(&mockSEO.GenerateArtefactsCallCount) != 0 {
		t.Error("Expected GenerateArtefacts NOT to be called for nil result")
	}
}

func TestProcessSEOArtefacts_Skips_WhenSEOServiceNil(t *testing.T) {
	t.Parallel()

	deps := &DaemonServiceDeps{
		SEOService: nil,
	}

	service := mustBuildDaemonService(t, deps)

	result := &annotator_dto.ProjectAnnotationResult{}

	service.processSEOArtefacts(result)
}

func TestSubscribeToCoordinator_ReturnsNoOp_WhenCoordinatorNil(t *testing.T) {
	t.Parallel()

	deps := &DaemonServiceDeps{
		CoordinatorService: nil,
	}

	service := mustBuildDaemonService(t, deps)

	unsubscribe := service.subscribeToCoordinator(context.Background())

	unsubscribe()
}

func TestSubscribeToCoordinator_SubscribesToCoordinator(t *testing.T) {
	t.Parallel()

	mockCoordinator := &coordinator_domain.MockCoordinatorService{}

	deps := &DaemonServiceDeps{
		CoordinatorService: mockCoordinator,
	}

	service := mustBuildDaemonService(t, deps)

	ctx := t.Context()

	_ = service.subscribeToCoordinator(ctx)

	if atomic.LoadInt64(&mockCoordinator.SubscribeCallCount) == 0 {
		t.Error("Expected Subscribe to be called on coordinator")
	}
}

func TestProcessNotification_CallsSEO_WhenResultPresent(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})
	mockSEO := &MockSEOService{
		GenerateArtefactsFunc: func(_ context.Context, _ *seo_dto.ProjectView) error {
			close(done)
			return nil
		},
	}

	service := &daemonService{
		seoService: mockSEO,
	}

	result := &annotator_dto.ProjectAnnotationResult{}

	notification := coordinator_domain.BuildNotification{
		CausationID: "test-123",
		Result:      result,
	}

	service.processNotification(context.Background(), &notification)

	select {
	case <-done:

	case <-time.After(time.Second):
		t.Error("processNotification did not call SEO service")
	}
}

func TestProcessSEOArtefacts_CallsService_WhenPresent(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})
	mockSEO := &MockSEOService{
		GenerateArtefactsFunc: func(_ context.Context, _ *seo_dto.ProjectView) error {
			close(done)
			return nil
		},
	}

	service := &daemonService{
		seoService: mockSEO,
	}

	result := &annotator_dto.ProjectAnnotationResult{}

	service.processSEOArtefacts(result)

	select {
	case <-done:

	case <-time.After(time.Second):
		t.Error("processSEOArtefacts did not call SEO service")
	}
}

func TestProcessSEOArtefacts_HandlesError_WhenServiceFails(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})
	mockSEO := &MockSEOService{
		GenerateArtefactsFunc: func(_ context.Context, _ *seo_dto.ProjectView) error {
			defer close(done)
			return errors.New("seo generation failed")
		},
	}

	service := &daemonService{
		seoService: mockSEO,
	}

	result := &annotator_dto.ProjectAnnotationResult{}

	service.processSEOArtefacts(result)

	select {
	case <-done:

	case <-time.After(time.Second):
		t.Error("processSEOArtefacts goroutine did not complete")
	}
}

func TestHandleBuildNotifications_StopsOnContextDone(t *testing.T) {
	t.Parallel()

	deps := &DaemonServiceDeps{}

	service := mustBuildDaemonService(t, deps)

	notifications := make(chan coordinator_domain.BuildNotification, 10)

	ctx, cancel := context.WithCancelCause(context.Background())

	var wg sync.WaitGroup
	wg.Go(func() {
		service.handleBuildNotifications(ctx, notifications)
	})

	cancel(fmt.Errorf("test: simulating cancelled context"))

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:

	case <-time.After(time.Second):
		t.Error("handleBuildNotifications did not exit on context cancellation")
	}
}

func TestHandleBuildNotifications_StopsOnChannelClose(t *testing.T) {
	t.Parallel()

	deps := &DaemonServiceDeps{}

	service := mustBuildDaemonService(t, deps)

	notifications := make(chan coordinator_domain.BuildNotification, 10)

	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Go(func() {
		service.handleBuildNotifications(ctx, notifications)
	})

	close(notifications)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:

	case <-time.After(time.Second):
		t.Error("handleBuildNotifications did not exit on channel close")
	}
}

func TestGetHandler_ReturnsFinalRouter(t *testing.T) {
	t.Parallel()

	expectedRouter := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	deps := &DaemonServiceDeps{
		FinalRouter: expectedRouter,
	}

	service := mustBuildDaemonService(t, deps)

	handler := service.GetHandler()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestRunDev_GracefulShutdown(t *testing.T) {
	t.Parallel()

	notifier := NewMockSignalNotifier()

	mockServer := newMockServerAdapter()
	mockServer.ListenAndServeFunc = func(_ string, _ http.Handler) error {
		select {
		case <-mockServer.shutdownCh:
			return http.ErrServerClosed
		case <-time.After(15 * time.Second):
			t.Error("ListenAndServeFunc: shutdown channel never closed within 15s")
			return http.ErrServerClosed
		}
	}

	deps := &DaemonServiceDeps{
		DaemonConfig:   testDaemonConfig(),
		Server:         mockServer,
		FinalRouter:    http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
		SignalNotifier: notifier,
	}

	service := mustBuildDaemonService(t, deps)

	done := make(chan error, 1)
	go func() {
		done <- service.RunDev(context.Background())
	}()

	select {
	case <-notifier.AwaitNotifyContext():
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for NotifyContext to be called")
	}

	notifier.Trigger()

	select {
	case err := <-done:
		require.NoError(t, err, "expected nil error on graceful shutdown")
	case <-time.After(15 * time.Second):
		t.Fatal("RunDev did not return after signal trigger")
	}
}

func TestRunDev_ServerError(t *testing.T) {
	t.Parallel()

	notifier := NewMockSignalNotifier()
	serverErr := errors.New("bind failed")

	mockServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			return serverErr
		},
	}

	deps := &DaemonServiceDeps{
		DaemonConfig:   testDaemonConfig(),
		Server:         mockServer,
		FinalRouter:    http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
		SignalNotifier: notifier,
	}

	service := mustBuildDaemonService(t, deps)

	done := make(chan error, 1)
	go func() {
		done <- service.RunDev(context.Background())
	}()

	select {
	case err := <-done:
		if !errors.Is(err, serverErr) {
			t.Errorf("expected server error, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RunDev did not return after server error")
	}
}

func TestRunDev_SubscribesToCoordinator(t *testing.T) {
	t.Parallel()

	notifier := NewMockSignalNotifier()
	mockCoordinator := &coordinator_domain.MockCoordinatorService{}

	mockServer := newMockServerAdapter()
	mockServer.ListenAndServeFunc = func(_ string, _ http.Handler) error {
		select {
		case <-mockServer.shutdownCh:
			return http.ErrServerClosed
		case <-time.After(5 * time.Second):
			return http.ErrServerClosed
		}
	}

	deps := &DaemonServiceDeps{
		DaemonConfig:       testDaemonConfig(),
		Server:             mockServer,
		FinalRouter:        http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
		SignalNotifier:     notifier,
		CoordinatorService: mockCoordinator,
	}

	service := mustBuildDaemonService(t, deps)

	done := make(chan error, 1)
	go func() {
		done <- service.RunDev(context.Background())
	}()

	if !waitForCondition(2*time.Second, func() bool {
		return atomic.LoadInt64(&mockCoordinator.SubscribeCallCount) > 0
	}) {
		t.Fatal("timed out waiting for Subscribe to be called on coordinator")
	}

	notifier.Trigger()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("RunDev did not return")
	}
}

func TestRunProd_GracefulShutdown(t *testing.T) {
	t.Parallel()

	notifier := NewMockSignalNotifier()

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
		DaemonConfig:   testDaemonConfig(),
		Server:         mockServer,
		FinalRouter:    http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
		SignalNotifier: notifier,
	}

	service := mustBuildDaemonService(t, deps)

	done := make(chan error, 1)
	go func() {
		done <- service.RunProd(context.Background())
	}()

	select {
	case <-notifier.AwaitNotifyContext():
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for NotifyContext to be called")
	}

	notifier.Trigger()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected nil error on graceful shutdown, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RunProd did not return after signal trigger")
	}
}

func TestRunProd_ServerError(t *testing.T) {
	t.Parallel()

	notifier := NewMockSignalNotifier()
	serverErr := errors.New("bind failed")

	mockServer := &MockServerAdapter{
		ListenAndServeFunc: func(_ string, _ http.Handler) error {
			return serverErr
		},
	}

	deps := &DaemonServiceDeps{
		DaemonConfig:   testDaemonConfig(),
		Server:         mockServer,
		FinalRouter:    http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
		SignalNotifier: notifier,
	}

	service := mustBuildDaemonService(t, deps)

	done := make(chan error, 1)
	go func() {
		done <- service.RunProd(context.Background())
	}()

	select {
	case err := <-done:
		if !errors.Is(err, serverErr) {
			t.Errorf("expected server error, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RunProd did not return after server error")
	}
}

func TestRunProd_DisablesWatchMode(t *testing.T) {
	t.Parallel()

	notifier := NewMockSignalNotifier()

	mockServer := newMockServerAdapter()
	t.Cleanup(func() { _ = mockServer.Shutdown(context.Background()) })

	mockServer.ListenAndServeFunc = func(_ string, _ http.Handler) error {
		<-mockServer.shutdownCh
		return http.ErrServerClosed
	}

	watchMode := true

	deps := &DaemonServiceDeps{
		DaemonConfig:   testDaemonConfig(),
		WatchMode:      &watchMode,
		Server:         mockServer,
		FinalRouter:    http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
		SignalNotifier: notifier,
	}

	service := mustBuildDaemonService(t, deps)

	done := make(chan error, 1)
	go func() {
		done <- service.RunProd(context.Background())
	}()

	select {
	case <-notifier.AwaitNotifyContext():
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for NotifyContext to be called")
	}

	notifier.Trigger()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("RunProd did not return")
	}

	if watchMode {
		t.Error("expected WatchMode to be disabled in prod mode")
	}
}

func TestLaunchDaemonProcess_ReturnsErrChan(t *testing.T) {
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

	errChan := service.launchDaemonProcess(context.Background())
	if errChan == nil {
		t.Fatal("expected non-nil error channel")
	}

	select {
	case <-errChan:

	case <-time.After(5 * time.Second):
		t.Fatal("error channel was not closed")
	}
}

func TestLaunchDaemonProcess_PropagatesServerError(t *testing.T) {
	t.Parallel()

	serverErr := errors.New("server crashed")

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

	errChan := service.launchDaemonProcess(context.Background())

	select {
	case err := <-errChan:
		if !errors.Is(err, serverErr) {
			t.Errorf("expected server error, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("did not receive error from channel")
	}
}

func TestAwaitShutdownDev_ContextCancelled(t *testing.T) {
	t.Parallel()

	deps := &DaemonServiceDeps{}

	service := mustBuildDaemonService(t, deps)
	serverErrChan := make(chan error, 1)

	ctx, cancel := context.WithCancelCause(context.Background())

	var wg sync.WaitGroup
	var result error

	wg.Go(func() {
		result = service.awaitShutdownDev(ctx, getNoopSpan(), serverErrChan)
	})

	cancel(fmt.Errorf("test: simulating cancelled context"))
	wg.Wait()

	if result != nil {
		t.Errorf("expected nil error on context cancellation, got %v", result)
	}
}

func TestAwaitShutdownDev_ServerError(t *testing.T) {
	t.Parallel()

	deps := &DaemonServiceDeps{}

	service := mustBuildDaemonService(t, deps)

	serverErr := errors.New("server failed")
	serverErrChan := make(chan error, 1)
	serverErrChan <- serverErr

	result := service.awaitShutdownDev(context.Background(), getNoopSpan(), serverErrChan)

	if !errors.Is(result, serverErr) {
		t.Errorf("expected server error, got %v", result)
	}
}

func TestMockSignalNotifier_WasTriggered(t *testing.T) {
	t.Parallel()

	notifier := NewMockSignalNotifier()

	if notifier.WasTriggered() {
		t.Error("expected WasTriggered=false before trigger")
	}

	notifier.NotifyContext(context.Background())
	notifier.Trigger()

	if !notifier.WasTriggered() {
		t.Error("expected WasTriggered=true after trigger")
	}
}

func TestMockSignalNotifier_Reset(t *testing.T) {
	t.Parallel()

	notifier := NewMockSignalNotifier()

	notifier.NotifyContext(context.Background())
	notifier.Trigger()
	notifier.Reset()

	if notifier.WasTriggered() {
		t.Error("expected WasTriggered=false after reset")
	}
}

func TestStop_StopsOrchestratorService(t *testing.T) {
	t.Parallel()

	mockOrchestrator := &orchestrator_domain.MockOrchestratorService{}

	deps := &DaemonServiceDeps{
		OrchestratorService: mockOrchestrator,
	}

	service := mustBuildDaemonService(t, deps)

	_ = service.Stop(context.Background())

	if atomic.LoadInt64(&mockOrchestrator.StopCallCount) == 0 {
		t.Error("expected orchestrator Stop() to be called")
	}
}

func TestNewService_DefaultsSEOSemaphoreCapacity(t *testing.T) {
	t.Parallel()

	deps := &DaemonServiceDeps{}

	service := mustBuildDaemonService(t, deps)

	if service.seoSemaphore == nil {
		t.Fatal("expected SEO semaphore to be initialised")
	}

	if cap(service.seoSemaphore) <= 0 {
		t.Errorf("expected positive default semaphore capacity, got %d", cap(service.seoSemaphore))
	}
}

func TestNewService_HonoursMaxConcurrentSEOJobsConfig(t *testing.T) {
	t.Parallel()

	cfg := testDaemonConfig()
	cfg.MaxConcurrentSEOJobs = 3

	deps := &DaemonServiceDeps{
		DaemonConfig: cfg,
	}

	service := mustBuildDaemonService(t, deps)

	if got := cap(service.seoSemaphore); got != 3 {
		t.Errorf("expected semaphore capacity 3, got %d", got)
	}
}

func TestProcessSEOArtefacts_BoundsConcurrencyToSemaphoreCapacity(t *testing.T) {
	t.Parallel()

	const burst = 8
	const slots = 2

	release := make(chan struct{})
	var inFlight int64
	var peak int64

	mockSEO := &MockSEOService{
		GenerateArtefactsFunc: func(_ context.Context, _ *seo_dto.ProjectView) error {
			current := atomic.AddInt64(&inFlight, 1)
			defer atomic.AddInt64(&inFlight, -1)

			for {
				existing := atomic.LoadInt64(&peak)
				if current <= existing || atomic.CompareAndSwapInt64(&peak, existing, current) {
					break
				}
			}

			<-release
			return nil
		},
	}

	cfg := testDaemonConfig()
	cfg.MaxConcurrentSEOJobs = slots

	deps := &DaemonServiceDeps{
		DaemonConfig: cfg,
		SEOService:   mockSEO,
	}

	service := mustBuildDaemonService(t, deps)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range burst {
			service.processSEOArtefacts(&annotator_dto.ProjectAnnotationResult{})
		}
	}()

	if !waitForCondition(2*time.Second, func() bool {
		return atomic.LoadInt64(&inFlight) >= int64(slots)
	}) {
		t.Fatal("timed out waiting for semaphore to fill")
	}

	close(release)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("processSEOArtefacts loop did not return")
	}

	service.seoWg.Wait()

	if got := atomic.LoadInt64(&peak); got > int64(slots) {
		t.Errorf("expected peak in-flight <= %d, got %d", slots, got)
	}

	if calls := atomic.LoadInt64(&mockSEO.GenerateArtefactsCallCount); calls != burst {
		t.Errorf("expected %d generate calls, got %d", burst, calls)
	}
}

func TestProcessSEOArtefacts_SkipsWhenSEOContextCancelled(t *testing.T) {
	t.Parallel()

	mockSEO := &MockSEOService{
		GenerateArtefactsFunc: func(_ context.Context, _ *seo_dto.ProjectView) error {
			return nil
		},
	}

	cfg := testDaemonConfig()
	cfg.MaxConcurrentSEOJobs = 1

	deps := &DaemonServiceDeps{
		DaemonConfig: cfg,
		SEOService:   mockSEO,
	}

	service := mustBuildDaemonService(t, deps)

	service.seoSemaphore <- struct{}{}

	service.seoCancel(errors.New("test: cancelling SEO context"))

	service.processSEOArtefacts(&annotator_dto.ProjectAnnotationResult{})

	service.releaseSEOSlot()

	if calls := atomic.LoadInt64(&mockSEO.GenerateArtefactsCallCount); calls != 0 {
		t.Errorf("expected GenerateArtefacts NOT to be called when context cancelled, got %d", calls)
	}
}
