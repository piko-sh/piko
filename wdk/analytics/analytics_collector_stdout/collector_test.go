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

package analytics_collector_stdout

import (
	"context"
	"net/http"
	"testing"
	"time"

	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/wdk/maths"
)

func TestCollector_Name(t *testing.T) {
	c := NewCollector()
	if c.Name() != "stdout" {
		t.Errorf("Name() = %q, want stdout", c.Name())
	}
}

func TestCollector_CollectReturnsNil(t *testing.T) {
	c := NewCollector()
	event := &analytics_dto.Event{
		Path:       "/test",
		Method:     http.MethodGet,
		StatusCode: 200,
		Type:       analytics_dto.EventPageView,
		Timestamp:  time.Now(),
	}
	if err := c.Collect(context.Background(), event); err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
}

func TestCollector_CollectWithAllFields(t *testing.T) {
	c := NewCollector()
	event := &analytics_dto.Event{
		Hostname:   "example.com",
		URL:        "/checkout?ref=email",
		Path:       "/checkout",
		Method:     http.MethodPost,
		StatusCode: 200,
		Type:       analytics_dto.EventCustom,
		EventName:  "purchase",
		ActionName: "cart.Purchase",
		UserID:     "user-123",
		ClientIP:   "192.168.1.1",
		Referrer:   "https://google.com",
		Duration:   150 * time.Millisecond,
		Revenue:    new(maths.NewMoneyFromString("49.99", "GBP")),
		Properties: map[string]string{"plan": "pro", "source": "organic"},
		Timestamp:  time.Now(),
	}
	if err := c.Collect(context.Background(), event); err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
}

func TestCollector_CollectMinimalEvent(t *testing.T) {
	c := NewCollector()
	event := &analytics_dto.Event{
		Path:       "/",
		Method:     http.MethodGet,
		StatusCode: 200,
		Type:       analytics_dto.EventPageView,
		Timestamp:  time.Now(),
	}
	if err := c.Collect(context.Background(), event); err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
}

func TestCollector_FlushReturnsNil(t *testing.T) {
	c := NewCollector()
	if err := c.Flush(context.Background()); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}
}

func TestCollector_CloseReturnsNil(t *testing.T) {
	c := NewCollector()
	if err := c.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestCollector_StartIsNoOp(t *testing.T) {
	c := NewCollector()
	c.Start(context.Background())
}

func TestAppendCoreFields(t *testing.T) {
	event := &analytics_dto.Event{
		Path:       "/products",
		Method:     http.MethodGet,
		StatusCode: 200,
		Type:       analytics_dto.EventPageView,
	}
	fields := appendCoreFields(nil, event)
	if len(fields) != 4 {
		t.Fatalf("expected 4 core fields, got %d", len(fields))
	}
}

func TestAppendOptionalFields_Empty(t *testing.T) {
	event := &analytics_dto.Event{}
	fields := appendOptionalFields(nil, event)
	if len(fields) != 0 {
		t.Fatalf("expected 0 optional fields for empty event, got %d", len(fields))
	}
}

func TestAppendOptionalFields_AllSet(t *testing.T) {
	event := &analytics_dto.Event{
		Hostname:   "example.com",
		URL:        "/page",
		EventName:  "signup",
		ActionName: "auth.Register",
		UserID:     "u1",
		ClientIP:   "10.0.0.1",
		Referrer:   "https://ref.com",
		Duration:   100 * time.Millisecond,
		Revenue:    new(maths.NewMoneyFromString("10.00", "USD")),
	}
	fields := appendOptionalFields(nil, event)
	if len(fields) != 9 {
		t.Fatalf("expected 9 optional fields, got %d", len(fields))
	}
}

func TestAppendPropertyFields(t *testing.T) {
	event := &analytics_dto.Event{
		Properties: map[string]string{"a": "1", "b": "2", "c": "3"},
	}
	fields := appendPropertyFields(nil, event)
	if len(fields) != 3 {
		t.Fatalf("expected 3 property fields, got %d", len(fields))
	}
}

func TestAppendPropertyFields_Empty(t *testing.T) {
	event := &analytics_dto.Event{}
	fields := appendPropertyFields(nil, event)
	if len(fields) != 0 {
		t.Fatalf("expected 0 property fields, got %d", len(fields))
	}
}
