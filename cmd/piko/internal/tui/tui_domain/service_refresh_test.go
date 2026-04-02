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

package tui_domain

import (
	"context"
	"testing"
	"time"
)

func TestNewRefreshOrchestrator(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)
	service := &Service{
		config: config,
		model:  model,
	}

	orch := newRefreshOrchestrator(service)

	if orch == nil {
		t.Fatal("expected non-nil orchestrator")
	}
	if orch.service != service {
		t.Error("expected service reference")
	}
	if orch.cancel != nil {
		t.Error("expected nil cancel before Start")
	}
}

func TestRefreshOrchestrator_StartStop(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)
	service := &Service{
		config: config,
		model:  model,
	}

	orch := newRefreshOrchestrator(service)

	ctx := context.Background()
	orch.Start(ctx)

	if orch.cancel == nil {
		t.Error("expected cancel to be set after Start")
	}

	orch.Stop()
}

func TestRefreshOrchestrator_Stop_NilCancel(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)
	service := &Service{
		config: config,
		model:  model,
	}

	orch := newRefreshOrchestrator(service)

	orch.Stop()
}

func TestRefreshOrchestrator_ForceRefresh(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)
	service := &Service{
		config: config,
		model:  model,
	}

	orch := newRefreshOrchestrator(service)

	ctx := context.Background()
	command := orch.ForceRefresh(ctx)

	if command == nil {
		t.Error("expected non-nil command")
	}

	message := command()
	if _, ok := message.(dataUpdatedMessage); !ok {
		t.Errorf("expected dataUpdatedMessage, got %T", message)
	}
}

func TestDataUpdatedMessage(t *testing.T) {
	testTime := time.Now()
	message := dataUpdatedMessage{time: testTime}

	if !message.time.Equal(testTime) {
		t.Error("expected time to match")
	}
}

func TestRefreshOrchestrator_UpdateProviderStatus(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)
	service := &Service{
		config:  config,
		model:   model,
		program: nil,
	}

	orch := newRefreshOrchestrator(service)

	orch.updateProviderStatus("test", ProviderStatusConnected, nil)
	orch.updateProviderStatus("test", ProviderStatusError, ErrConnectionFailed)
}
