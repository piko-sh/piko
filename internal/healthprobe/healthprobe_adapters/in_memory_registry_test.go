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

package healthprobe_adapters

import (
	"context"
	"sync"
	"testing"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

type testProbe struct {
	name  string
	state healthprobe_dto.State
}

func (t *testProbe) Name() string {
	return t.name
}

func (t *testProbe) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	return healthprobe_dto.Status{
		Name:      t.name,
		State:     t.state,
		Message:   "Test probe",
		Timestamp: time.Now(),
		Duration:  "0ms",
	}
}

func TestNewInMemoryRegistry(t *testing.T) {
	registry := NewInMemoryRegistry()

	if registry == nil {
		t.Fatal("NewInMemoryRegistry returned nil")
	}

	probes := registry.GetAll()
	if len(probes) != 0 {
		t.Errorf("New registry should have 0 probes, got %d", len(probes))
	}
}

func TestInMemoryRegistry_Register_Single(t *testing.T) {
	registry := NewInMemoryRegistry()

	probe := &testProbe{name: "TestProbe", state: healthprobe_dto.StateHealthy}
	registry.Register(probe)

	probes := registry.GetAll()
	if len(probes) != 1 {
		t.Fatalf("Expected 1 probe, got %d", len(probes))
	}

	if probes[0].Name() != "TestProbe" {
		t.Errorf("Expected probe name 'TestProbe', got %s", probes[0].Name())
	}
}

func TestInMemoryRegistry_Register_Multiple(t *testing.T) {
	registry := NewInMemoryRegistry()

	probe1 := &testProbe{name: "Probe1", state: healthprobe_dto.StateHealthy}
	probe2 := &testProbe{name: "Probe2", state: healthprobe_dto.StateHealthy}
	probe3 := &testProbe{name: "Probe3", state: healthprobe_dto.StateDegraded}

	registry.Register(probe1)
	registry.Register(probe2)
	registry.Register(probe3)

	probes := registry.GetAll()
	if len(probes) != 3 {
		t.Fatalf("Expected 3 probes, got %d", len(probes))
	}

	expectedNames := []string{"Probe1", "Probe2", "Probe3"}
	for i, probe := range probes {
		if probe.Name() != expectedNames[i] {
			t.Errorf("Expected probe %d to be %s, got %s", i, expectedNames[i], probe.Name())
		}
	}
}

func TestInMemoryRegistry_GetAll_ReturnsCopy(t *testing.T) {
	registry := NewInMemoryRegistry()

	probe := &testProbe{name: "TestProbe", state: healthprobe_dto.StateHealthy}
	registry.Register(probe)

	probes1 := registry.GetAll()
	probes2 := registry.GetAll()

	if &probes1[0] == &probes2[0] {
		t.Error("GetAll should return a copy, not the same slice")
	}

	if len(probes1) != len(probes2) {
		t.Errorf("Both calls should return same number of probes")
	}
}

func TestInMemoryRegistry_ThreadSafety(t *testing.T) {
	registry := NewInMemoryRegistry()

	var wg sync.WaitGroup
	probeCount := 100

	for i := range probeCount {
		index := i
		wg.Go(func() {
			probe := &testProbe{
				name:  "Probe" + string(rune('A'+index%26)),
				state: healthprobe_dto.StateHealthy,
			}
			registry.Register(probe)
		})
	}

	for range 50 {
		wg.Go(func() {
			_ = registry.GetAll()
		})
	}

	wg.Wait()

	probes := registry.GetAll()
	if len(probes) != probeCount {
		t.Errorf("Expected %d probes after concurrent registration, got %d", probeCount, len(probes))
	}
}

func TestInMemoryRegistry_GetAll_EmptyRegistry(t *testing.T) {
	registry := NewInMemoryRegistry()

	probes := registry.GetAll()

	if probes == nil {
		t.Error("GetAll should return empty slice, not nil")
	}

	if len(probes) != 0 {
		t.Errorf("Expected 0 probes, got %d", len(probes))
	}
}
