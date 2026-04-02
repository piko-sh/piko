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

package pml_adapters

import (
	"strings"
	"testing"
)

func TestMSOConditionalCollector_EmptyCollector(t *testing.T) {
	collector := NewMSOConditionalCollector()
	result := collector.GenerateConditionalBlock()

	if result != "" {
		t.Errorf("Expected empty string for empty collector, got: %s", result)
	}
}

func TestMSOConditionalCollector_SingleRule(t *testing.T) {
	collector := NewMSOConditionalCollector()
	collector.RegisterStyle("ul", "margin: 0 !important;")

	result := collector.GenerateConditionalBlock()

	if !strings.Contains(result, "<!--[if mso]>") {
		t.Error("Expected MSO conditional opening tag")
	}
	if !strings.Contains(result, "<![endif]-->") {
		t.Error("Expected MSO conditional closing tag")
	}
	if !strings.Contains(result, "<style type=\"text/css\">") {
		t.Error("Expected style tag opening")
	}
	if !strings.Contains(result, "</style>") {
		t.Error("Expected style tag closing")
	}
	if !strings.Contains(result, "ul {margin: 0 !important;}") {
		t.Error("Expected ul style rule")
	}
}

func TestMSOConditionalCollector_MultipleRules(t *testing.T) {
	collector := NewMSOConditionalCollector()

	collector.RegisterStyle("ul", "margin: 0 !important;")
	collector.RegisterStyle("li", "margin-left: 40px !important;")
	collector.RegisterStyle("li.firstListItem", "margin-top: 20px !important;")
	collector.RegisterStyle("li.lastListItem", "margin-bottom: 20px !important;")

	result := collector.GenerateConditionalBlock()

	if !strings.Contains(result, "ul {margin: 0 !important;}") {
		t.Error("Expected ul style rule")
	}
	if !strings.Contains(result, "li {margin-left: 40px !important;}") {
		t.Error("Expected li style rule")
	}
	if !strings.Contains(result, "li.firstListItem {margin-top: 20px !important;}") {
		t.Error("Expected li.firstListItem style rule")
	}
	if !strings.Contains(result, "li.lastListItem {margin-bottom: 20px !important;}") {
		t.Error("Expected li.lastListItem style rule")
	}
}

func TestMSOConditionalCollector_Deduplication(t *testing.T) {
	collector := NewMSOConditionalCollector()

	collector.RegisterStyle("ul", "margin: 0 !important;")
	collector.RegisterStyle("ul", "margin: 0 !important;")
	collector.RegisterStyle("ul", "margin: 0 !important;")

	result := collector.GenerateConditionalBlock()

	count := strings.Count(result, "ul {margin: 0 !important;}")
	if count != 1 {
		t.Errorf("Expected exactly 1 occurrence of ul rule due to deduplication, got %d", count)
	}
}

func TestMSOConditionalCollector_IgnoresEmptyRegistrations(t *testing.T) {
	collector := NewMSOConditionalCollector()

	collector.RegisterStyle("", "margin: 0;")
	collector.RegisterStyle("ul", "")
	collector.RegisterStyle("", "")

	result := collector.GenerateConditionalBlock()

	if result != "" {
		t.Errorf("Expected empty string for collector with only invalid registrations, got: %s", result)
	}
}

func TestMSOConditionalCollector_SortedOutput(t *testing.T) {
	collector := NewMSOConditionalCollector()

	collector.RegisterStyle("ul", "margin: 0 !important;")
	collector.RegisterStyle("li.lastListItem", "margin-bottom: 20px !important;")
	collector.RegisterStyle("li.firstListItem", "margin-top: 20px !important;")
	collector.RegisterStyle("li", "margin-left: 40px !important;")

	result := collector.GenerateConditionalBlock()

	liIndex := strings.Index(result, "li {margin-left:")
	liFirstIndex := strings.Index(result, "li.firstListItem {margin-top:")
	liLastIndex := strings.Index(result, "li.lastListItem {margin-bottom:")
	ulIndex := strings.Index(result, "ul {margin:")

	if liIndex >= liFirstIndex || liFirstIndex >= liLastIndex || liLastIndex >= ulIndex {
		t.Error("Expected rules to be sorted alphabetically")
	}
}
