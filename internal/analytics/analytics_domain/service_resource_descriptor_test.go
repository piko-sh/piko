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
	"testing"
)

func TestService_ResourceType(t *testing.T) {
	svc := NewService(nil)
	if svc.ResourceType() != "analytics" {
		t.Errorf("ResourceType() = %q, want analytics", svc.ResourceType())
	}
}

func TestService_ResourceListColumns(t *testing.T) {
	svc := NewService(nil)
	columns := svc.ResourceListColumns()

	if len(columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(columns))
	}
	if columns[0].Header != "NAME" || columns[0].Key != "name" {
		t.Errorf("column 0 = %+v, want NAME/name", columns[0])
	}
	if columns[1].Header != "STATUS" || columns[1].Key != "status" {
		t.Errorf("column 1 = %+v, want STATUS/status", columns[1])
	}
	if columns[2].Header != "CHANNEL" || columns[2].Key != "channel" {
		t.Errorf("column 2 = %+v, want CHANNEL/channel", columns[2])
	}
}

func TestService_ResourceListProviders_Sorted(t *testing.T) {
	zebra := newMockCollector("zebra")
	alpha := newMockCollector("alpha")
	svc := NewService([]Collector{zebra, alpha})
	svc.Start(context.Background())

	entries := svc.ResourceListProviders(context.Background())

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "alpha" {
		t.Errorf("first entry = %q, want alpha", entries[0].Name)
	}
	if entries[1].Name != "zebra" {
		t.Errorf("second entry = %q, want zebra", entries[1].Name)
	}
	if entries[0].Values["status"] != "running" {
		t.Errorf("alpha status = %q, want running", entries[0].Values["status"])
	}

	if err := svc.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestService_ResourceDescribeProvider(t *testing.T) {
	mc := newMockCollector("webhook")
	svc := NewService([]Collector{mc})
	svc.Start(context.Background())

	detail, err := svc.ResourceDescribeProvider(context.Background(), "webhook")
	if err != nil {
		t.Fatalf("ResourceDescribeProvider: %v", err)
	}
	if detail.Name != "webhook" {
		t.Errorf("Name = %q, want webhook", detail.Name)
	}
	if len(detail.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(detail.Sections))
	}
	if detail.Sections[0].Title != "Overview" {
		t.Errorf("Section title = %q, want Overview", detail.Sections[0].Title)
	}
	if len(detail.Sections[0].Entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(detail.Sections[0].Entries))
	}

	if err := svc.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestService_ResourceDescribeProvider_NotFound(t *testing.T) {
	svc := NewService(nil)

	_, err := svc.ResourceDescribeProvider(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent collector")
	}
}
