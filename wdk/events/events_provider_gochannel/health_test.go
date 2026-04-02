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

package events_provider_gochannel

import (
	"context"
	"testing"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

func TestGoChannelProvider_Name(t *testing.T) {
	provider, err := NewGoChannelProvider(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	name := provider.Name()
	expected := "GoChannelProvider"

	if name != expected {
		t.Errorf("Expected name %q, got %q", expected, name)
	}
}

func TestGoChannelProvider_CheckLiveness_BeforeStart(t *testing.T) {
	provider, err := NewGoChannelProvider(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	status := provider.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	if status.Name != "GoChannelProvider" {
		t.Errorf("Expected name 'GoChannelProvider', got %q", status.Name)
	}

	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("Expected state Healthy for liveness (pubsub is initialised), got %v", status.State)
	}
}

func TestGoChannelProvider_CheckReadiness_BeforeStart(t *testing.T) {
	provider, err := NewGoChannelProvider(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	status := provider.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if status.Name != "GoChannelProvider" {
		t.Errorf("Expected name 'GoChannelProvider', got %q", status.Name)
	}

	if status.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("Expected state Unhealthy for readiness (router not started), got %v", status.State)
	}
}

func TestGoChannelProvider_CheckReadiness_AfterStart(t *testing.T) {
	provider, err := NewGoChannelProvider(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()

	if err := provider.Start(ctx); err != nil {
		t.Fatalf("Failed to start provider: %v", err)
	}
	defer func() { _ = provider.Close() }()

	status := provider.Check(ctx, healthprobe_dto.CheckTypeReadiness)

	if status.Name != "GoChannelProvider" {
		t.Errorf("Expected name 'GoChannelProvider', got %q", status.Name)
	}

	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("Expected state Healthy after start, got %v: %s", status.State, status.Message)
	}

	if len(status.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies (router, pubsub), got %d", len(status.Dependencies))
	}

	for _, dependency := range status.Dependencies {
		if dependency.State != healthprobe_dto.StateHealthy {
			t.Errorf("Expected dependency %q to be healthy, got %v: %s", dependency.Name, dependency.State, dependency.Message)
		}
	}
}

func TestGoChannelProvider_CheckReadiness_AfterClose(t *testing.T) {
	provider, err := NewGoChannelProvider(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()

	if err := provider.Start(ctx); err != nil {
		t.Fatalf("Failed to start provider: %v", err)
	}
	if err := provider.Close(); err != nil {
		t.Fatalf("Failed to close provider: %v", err)
	}

	status := provider.Check(ctx, healthprobe_dto.CheckTypeReadiness)

	if status.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("Expected state Unhealthy after close, got %v: %s", status.State, status.Message)
	}
}
