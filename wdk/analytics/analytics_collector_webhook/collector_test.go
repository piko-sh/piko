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

package analytics_collector_webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"piko.sh/piko/internal/analytics/analytics_dto"
)

func TestNewCollector_ReturnsWorkingCollector(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	collector, err := NewCollector(server.URL, WithBatchSize(1))
	if err != nil {
		t.Fatalf("NewCollector returned unexpected error: %v", err)
	}
	collector.Start(context.Background())

	event := &analytics_dto.Event{
		Path:       "/facade-test",
		Method:     http.MethodGet,
		StatusCode: 200,
		Timestamp:  time.Now(),
	}
	if err := collector.Collect(context.Background(), event); err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if err := collector.Flush(context.Background()); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}
	if err := collector.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestNewCollector_Name(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	collector, err := NewCollector(server.URL)
	if err != nil {
		t.Fatalf("NewCollector returned unexpected error: %v", err)
	}
	defer collector.Close(context.Background())

	if collector.Name() != "webhook" {
		t.Errorf("Name() = %q, want webhook", collector.Name())
	}
}

func TestNewCollector_AcceptsAllOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	headers := http.Header{}
	headers.Set("Authorization", "Bearer test")

	collector, err := NewCollector(server.URL,
		WithHeaders(headers),
		WithBatchSize(5),
		WithFlushInterval(10*time.Second),
		WithTimeout(30*time.Second),
	)
	if err != nil {
		t.Fatalf("NewCollector returned unexpected error: %v", err)
	}
	collector.Start(context.Background())

	if err := collector.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestNewCollector_EmptyURLReturnsError(t *testing.T) {
	_, err := NewCollector("")
	if err == nil {
		t.Fatal("expected error on empty URL")
	}
}
