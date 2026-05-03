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

package analytics_dto

import (
	"net/http"
	"testing"
	"time"

	"piko.sh/piko/wdk/maths"
)

func TestEventType_String(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		input    EventType
	}{
		{name: "page view", input: EventPageView, expected: "pageview"},
		{name: "action", input: EventAction, expected: "action"},
		{name: "custom", input: EventCustom, expected: "custom"},
		{name: "out of range", input: EventType(99), expected: "unknown"},
		{name: "negative value", input: EventType(-1), expected: "unknown"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.input.String()
			if got != tc.expected {
				t.Errorf("EventType(%d).String() = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestAcquireEvent_ReturnsZeroedEvent(t *testing.T) {
	ev := AcquireEvent()
	defer ReleaseEvent(ev)

	if ev.Request != nil {
		t.Error("expected Request to be nil")
	}
	if ev.Revenue != nil {
		t.Error("expected Revenue to be nil")
	}
	if ev.ClientIP != "" {
		t.Error("expected ClientIP to be empty")
	}
	if ev.Hostname != "" {
		t.Error("expected Hostname to be empty")
	}
	if ev.URL != "" {
		t.Error("expected URL to be empty")
	}
	if ev.EventName != "" {
		t.Error("expected EventName to be empty")
	}
	if ev.StatusCode != 0 {
		t.Error("expected StatusCode to be zero")
	}
	if ev.Properties != nil {
		t.Error("expected Properties to be nil")
	}
	if ev.Type != EventPageView {
		t.Errorf("expected Type to be EventPageView (0), got %d", ev.Type)
	}
}

func TestAcquireEvent_ResetsFieldsFromPreviousUse(t *testing.T) {
	ev := AcquireEvent()
	ev.ClientIP = "1.2.3.4"
	ev.Hostname = "example.com"
	ev.URL = "/test?utm_source=twitter"
	ev.EventName = "purchase"
	ev.Path = "/test"
	ev.StatusCode = 200
	ev.Properties = map[string]string{"key": "value"}
	ev.Request = &http.Request{}
	ev.Revenue = new(maths.NewMoneyFromString("29.99", "GBP"))
	ev.Timestamp = time.Now()
	ev.Duration = 5 * time.Second
	ev.Type = EventCustom
	ReleaseEvent(ev)

	ev2 := AcquireEvent()
	defer ReleaseEvent(ev2)

	if ev2.ClientIP != "" {
		t.Errorf("expected ClientIP to be reset, got %q", ev2.ClientIP)
	}
	if ev2.Hostname != "" {
		t.Errorf("expected Hostname to be reset, got %q", ev2.Hostname)
	}
	if ev2.URL != "" {
		t.Errorf("expected URL to be reset, got %q", ev2.URL)
	}
	if ev2.EventName != "" {
		t.Errorf("expected EventName to be reset, got %q", ev2.EventName)
	}
	if ev2.Path != "" {
		t.Errorf("expected Path to be reset, got %q", ev2.Path)
	}
	if ev2.StatusCode != 0 {
		t.Errorf("expected StatusCode to be reset, got %d", ev2.StatusCode)
	}
	if ev2.Properties != nil {
		t.Error("expected Properties to be nil after reset")
	}
	if ev2.Request != nil {
		t.Error("expected Request to be nil after reset")
	}
	if ev2.Revenue != nil {
		t.Error("expected Revenue to be nil after reset")
	}
	if !ev2.Timestamp.IsZero() {
		t.Error("expected Timestamp to be zero after reset")
	}
	if ev2.Duration != 0 {
		t.Error("expected Duration to be zero after reset")
	}
	if ev2.Type != EventPageView {
		t.Errorf("expected Type to be reset to EventPageView, got %d", ev2.Type)
	}
}

func TestReleaseEvent_NilSafe(t *testing.T) {

	ReleaseEvent(nil)
}
