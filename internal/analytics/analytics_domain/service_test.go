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

package analytics_domain

import (
	"context"
	"sync"
	"testing"
	"time"

	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/wdk/maths"
)

type mockCollector struct {
	mu     sync.Mutex
	events []analytics_dto.Event
	name   string
}

func newMockCollector(name string) *mockCollector {
	return &mockCollector{name: name}
}

func (m *mockCollector) Collect(_ context.Context, ev *analytics_dto.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.events = append(m.events, *ev)
	return nil
}

func (m *mockCollector) Flush(_ context.Context) error { return nil }
func (m *mockCollector) Close(_ context.Context) error { return nil }
func (m *mockCollector) Name() string                  { return m.name }

func (m *mockCollector) collected() []analytics_dto.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]analytics_dto.Event, len(m.events))
	copy(out, m.events)
	return out
}

func TestService_SingleCollector(t *testing.T) {
	mc := newMockCollector("test")
	svc := NewService([]Collector{mc})
	svc.Start(context.Background())

	ev := analytics_dto.AcquireEvent()
	ev.Path = "/hello"
	ev.Type = analytics_dto.EventPageView
	svc.Track(ev)

	if err := svc.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	events := mc.collected()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Path != "/hello" {
		t.Errorf("expected path /hello, got %q", events[0].Path)
	}
	if events[0].Type != analytics_dto.EventPageView {
		t.Errorf("expected EventPageView, got %v", events[0].Type)
	}
}

func TestService_MultipleCollectors(t *testing.T) {
	mc1 := newMockCollector("alpha")
	mc2 := newMockCollector("beta")
	svc := NewService([]Collector{mc1, mc2})
	svc.Start(context.Background())

	ev := analytics_dto.AcquireEvent()
	ev.Path = "/multi"
	ev.ClientIP = "10.0.0.1"
	svc.Track(ev)

	if err := svc.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	for _, mc := range []*mockCollector{mc1, mc2} {
		events := mc.collected()
		if len(events) != 1 {
			t.Fatalf("collector %q: expected 1 event, got %d", mc.name, len(events))
		}
		if events[0].Path != "/multi" {
			t.Errorf("collector %q: expected path /multi, got %q", mc.name, events[0].Path)
		}
		if events[0].ClientIP != "10.0.0.1" {
			t.Errorf("collector %q: expected ClientIP 10.0.0.1, got %q", mc.name, events[0].ClientIP)
		}
	}
}

func TestService_DropsEventsWhenChannelFull(t *testing.T) {
	mc := newMockCollector("slow")
	svc := NewService([]Collector{mc}, WithChannelBufferSize(1))

	for range 5 {
		ev := analytics_dto.AcquireEvent()
		ev.Path = "/overflow"
		svc.Track(ev)
	}

	svc.Start(context.Background())
	if err := svc.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	events := mc.collected()
	if len(events) >= 5 {
		t.Errorf("expected some events to be dropped, got all %d", len(events))
	}
}

func TestService_TrackAfterClose(t *testing.T) {
	mc := newMockCollector("closed")
	svc := NewService([]Collector{mc})
	svc.Start(context.Background())
	if err := svc.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	ev := analytics_dto.AcquireEvent()
	ev.Path = "/late"
	svc.Track(ev)

	events := mc.collected()
	for _, e := range events {
		if e.Path == "/late" {
			t.Error("event sent after Close should not be delivered")
		}
	}
}

func TestService_CloseWithTimeout(t *testing.T) {
	mc := newMockCollector("timeout")
	svc := NewService([]Collector{mc})
	svc.Start(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := svc.Close(ctx); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestService_ZeroCollectors(t *testing.T) {
	svc := NewService(nil)
	svc.Start(context.Background())

	ev := analytics_dto.AcquireEvent()
	ev.Path = "/empty"
	svc.Track(ev)

	if err := svc.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestAcquireEventCopy(t *testing.T) {
	rev := maths.NewMoneyFromString("29.99", "GBP")
	src := analytics_dto.AcquireEvent()
	src.Path = "/original"
	src.ClientIP = "1.2.3.4"
	src.Hostname = "example.com"
	src.URL = "/original?ref=test"
	src.EventName = "purchase"
	src.Revenue = &rev
	src.Properties = map[string]string{"key": "value"}

	cp := AcquireEventCopy(src)

	if cp.Path != src.Path {
		t.Errorf("copy Path = %q, want %q", cp.Path, src.Path)
	}
	if cp.ClientIP != src.ClientIP {
		t.Errorf("copy ClientIP = %q, want %q", cp.ClientIP, src.ClientIP)
	}
	if cp.Hostname != src.Hostname {
		t.Errorf("copy Hostname = %q, want %q", cp.Hostname, src.Hostname)
	}
	if cp.URL != src.URL {
		t.Errorf("copy URL = %q, want %q", cp.URL, src.URL)
	}
	if cp.EventName != src.EventName {
		t.Errorf("copy EventName = %q, want %q", cp.EventName, src.EventName)
	}
	if cp.Properties["key"] != "value" {
		t.Error("copy Properties missing expected key")
	}
	if cp.Revenue == nil {
		t.Fatal("copy Revenue is nil, want non-nil")
	}
	if cp.Revenue == src.Revenue {
		t.Error("copy Revenue pointer is shared with source")
	}

	cp.Properties["key"] = "changed"
	if src.Properties["key"] != "value" {
		t.Error("mutating copy Properties affected source")
	}

	analytics_dto.ReleaseEvent(src)
	analytics_dto.ReleaseEvent(cp)
}

func TestAcquireEventCopy_NilRevenue(t *testing.T) {
	src := analytics_dto.AcquireEvent()
	src.Path = "/no-revenue"

	cp := AcquireEventCopy(src)

	if cp.Revenue != nil {
		t.Error("copy Revenue should be nil when source Revenue is nil")
	}

	analytics_dto.ReleaseEvent(src)
	analytics_dto.ReleaseEvent(cp)
}
