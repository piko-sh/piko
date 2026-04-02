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

package llm_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

func TestServiceHealth_Name(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	assert.Equal(t, "LLMService", service.Name())
}

func TestServiceHealth_Check_Liveness(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	assert.Equal(t, "LLMService", status.Name)
	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
	assert.Equal(t, "LLM service is running", status.Message)
	assert.False(t, status.Timestamp.IsZero())
	assert.NotEmpty(t, status.Duration)
}

func TestServiceHealth_Check_Readiness_NoProviders(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	assert.Equal(t, "LLMService", status.Name)
	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
	assert.Equal(t, "No LLM providers configured", status.Message)
}

func TestServiceHealth_Check_Readiness_NoDefault(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	mock := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "openai", mock))

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	assert.Equal(t, healthprobe_dto.StateDegraded, status.State)
	assert.Contains(t, status.Message, "no default set")
	assert.Contains(t, status.Message, "1 provider(s) registered")
}

func TestServiceHealth_Check_Readiness_DefaultNotFound(t *testing.T) {
	service, ok := NewService("nonexistent").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	mock := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "openai", mock))

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
	assert.Contains(t, status.Message, "Default provider")
	assert.Contains(t, status.Message, "not found")
}

func TestServiceHealth_Check_Readiness_Healthy(t *testing.T) {
	service, ok := NewService("openai").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	mock := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "openai", mock))

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
	assert.Contains(t, status.Message, "Ready with 1 provider(s)")
	assert.Contains(t, status.Message, "default: openai")
}

func TestServiceHealth_Check_Readiness_MultipleProviders(t *testing.T) {
	service, ok := NewService("openai").(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	require.NoError(t, service.RegisterProvider(context.Background(), "openai", NewMockLLMProvider()))
	require.NoError(t, service.RegisterProvider(context.Background(), "anthropic", NewMockLLMProvider()))

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
	assert.Contains(t, status.Message, "Ready with 2 provider(s)")
}

func TestServiceHealth_Check_HasTimestamp(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	assert.False(t, status.Timestamp.IsZero())
}

func TestServiceHealth_Check_HasDuration(t *testing.T) {
	service, ok := NewService("").(*service)
	if !ok {
		t.Fatal("expected *service")
	}

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	assert.NotEmpty(t, status.Duration)
}
