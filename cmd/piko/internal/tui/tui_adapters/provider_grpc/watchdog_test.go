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

package provider_grpc

import (
	"context"
	"testing"
	"time"

	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
)

func TestWatchdogProviderName(t *testing.T) {
	provider := NewWatchdogProvider(nil, time.Second)
	if got := provider.Name(); got != "grpc-watchdog" {
		t.Errorf("Name = %q, want %q", got, "grpc-watchdog")
	}
}

func TestWatchdogProviderRefreshInterval(t *testing.T) {
	provider := NewWatchdogProvider(nil, 5*time.Second)
	if got := provider.RefreshInterval(); got != 5*time.Second {
		t.Errorf("RefreshInterval = %v, want 5s", got)
	}
}

func TestWatchdogProviderClose(t *testing.T) {
	provider := NewWatchdogProvider(nil, time.Second)
	if err := provider.Close(); err != nil {
		t.Errorf("Close = %v, want nil", err)
	}
}

func TestWatchdogProviderHealthErrorWithoutConnection(t *testing.T) {
	provider := NewWatchdogProvider(nil, time.Second)
	if err := provider.Health(context.Background()); err == nil {
		t.Errorf("Health expected error without connection")
	}
}

func TestWatchdogProviderRefreshErrorWithoutConnection(t *testing.T) {
	provider := NewWatchdogProvider(nil, time.Second)
	if err := provider.Refresh(context.Background()); err == nil {
		t.Errorf("Refresh expected error without connection")
	}
}

func TestWatchdogProviderListEventsErrorWithoutConnection(t *testing.T) {
	provider := NewWatchdogProvider(nil, time.Second)
	if _, err := provider.ListEvents(context.Background(), tui_domain.WatchdogEventQuery{}); err == nil {
		t.Errorf("ListEvents expected error without connection")
	}
}

func TestWatchdogProviderSubscribeEventsErrorWithoutConnection(t *testing.T) {
	provider := NewWatchdogProvider(nil, time.Second)
	_, _, err := provider.SubscribeEvents(context.Background(), time.Time{})
	if err == nil {
		t.Errorf("SubscribeEvents expected error without connection")
	}
}

func TestWatchdogProviderPruneErrorWithoutConnection(t *testing.T) {
	provider := NewWatchdogProvider(nil, time.Second)
	if _, err := provider.PruneProfiles(context.Background(), ""); err == nil {
		t.Errorf("PruneProfiles expected error without connection")
	}
}

func TestWatchdogProviderRunContentionDiagnosticErrorWithoutConnection(t *testing.T) {
	provider := NewWatchdogProvider(nil, time.Second)
	if err := provider.RunContentionDiagnostic(context.Background()); err == nil {
		t.Errorf("RunContentionDiagnostic expected error without connection")
	}
}

func TestWatchdogProviderDownloadProfileErrorWithoutConnection(t *testing.T) {
	provider := NewWatchdogProvider(nil, time.Second)
	if err := provider.DownloadProfile(context.Background(), "any.pb.gz", devNull{}); err == nil {
		t.Errorf("DownloadProfile expected error without connection")
	}
}

func TestWatchdogProviderDownloadSidecarErrorWithoutConnection(t *testing.T) {
	provider := NewWatchdogProvider(nil, time.Second)
	if _, _, err := provider.DownloadSidecar(context.Background(), "x"); err == nil {
		t.Errorf("DownloadSidecar expected error without connection")
	}
}

func TestWatchdogProviderEmptySnapshotsAreSafe(t *testing.T) {
	provider := NewWatchdogProvider(nil, time.Second)
	ctx := context.Background()

	if got, err := provider.GetStatus(ctx); err != nil || got != nil {
		t.Errorf("GetStatus before refresh = (%+v, %v)", got, err)
	}
	if got, err := provider.ListProfiles(ctx); err != nil || len(got) != 0 {
		t.Errorf("ListProfiles before refresh = (%+v, %v)", got, err)
	}
	if got, err := provider.GetStartupHistory(ctx); err != nil || len(got) != 0 {
		t.Errorf("GetStartupHistory before refresh = (%+v, %v)", got, err)
	}
}

func TestWatchdogProviderDroppedEventsStartsAtZero(t *testing.T) {
	provider := NewWatchdogProvider(nil, time.Second)
	if got := provider.DroppedEvents(); got != 0 {
		t.Errorf("DroppedEvents at construction = %d, want 0", got)
	}
}

type devNull struct{}

func (devNull) Write(p []byte) (int, error) { return len(p), nil }
