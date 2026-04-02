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

package tui_domain

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultStatusBarConfig(t *testing.T) {
	config := DefaultStatusBarConfig()

	result := config.Style.Render("test")
	if result == "" {
		t.Error("expected style to render content")
	}
}

func TestStatusBar(t *testing.T) {
	config := DefaultStatusBarConfig()

	testCases := []struct {
		name string
		data StatusBarData
	}{
		{
			name: "empty data",
			data: StatusBarData{},
		},
		{
			name: "with keybindings",
			data: StatusBarData{
				KeyBindings: []KeyBinding{
					{Key: "q", Description: "Quit"},
					{Key: "?", Description: "Help"},
				},
			},
		},
		{
			name: "with providers",
			data: StatusBarData{
				Providers: []ProviderInfo{
					{Name: "Provider1", Status: ProviderStatusConnected},
					{Name: "Provider2", Status: ProviderStatusError},
				},
			},
		},
		{
			name: "with last refresh",
			data: StatusBarData{
				Now:         time.Now(),
				LastRefresh: time.Now().Add(-5 * time.Second),
			},
		},
		{
			name: "full data",
			data: StatusBarData{
				Now:          time.Now(),
				LastRefresh:  time.Now().Add(-10 * time.Second),
				CurrentPanel: "Metrics",
				KeyBindings: []KeyBinding{
					{Key: "r", Description: "Refresh"},
				},
				Providers: []ProviderInfo{
					{Name: "Local", Status: ProviderStatusConnected},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := StatusBar(tc.data, &config, 80)
			if result == "" {
				t.Error("expected non-empty status bar")
			}
		})
	}
}

func TestStatusBar_ProviderStatus(t *testing.T) {
	data := StatusBarData{
		Providers: []ProviderInfo{
			{Name: "Connected", Status: ProviderStatusConnected},
			{Name: "Disconnected", Status: ProviderStatusDisconnected},
		},
	}

	result := StatusBar(data, new(DefaultStatusBarConfig()), 100)

	if !strings.Contains(result, "Connected") {
		t.Error("expected provider name in output")
	}
}

func TestMiniStatusBar(t *testing.T) {
	config := DefaultStatusBarConfig()

	testCases := []struct {
		name          string
		panelName     string
		itemCount     int
		selectedIndex int
	}{
		{
			name:          "no items",
			panelName:     "Metrics",
			itemCount:     0,
			selectedIndex: 0,
		},
		{
			name:          "with items",
			panelName:     "Traces",
			itemCount:     10,
			selectedIndex: 3,
		},
		{
			name:          "first item",
			panelName:     "Resources",
			itemCount:     5,
			selectedIndex: 0,
		},
		{
			name:          "last item",
			panelName:     "Health",
			itemCount:     5,
			selectedIndex: 4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := MiniStatusBar(tc.panelName, tc.itemCount, tc.selectedIndex, &config, 80)
			if result == "" {
				t.Error("expected non-empty mini status bar")
			}
			if !strings.Contains(result, tc.panelName) {
				t.Errorf("expected panel name %q in output", tc.panelName)
			}
		})
	}
}

func TestHelpBar(t *testing.T) {
	config := DefaultStatusBarConfig()

	testCases := []struct {
		name     string
		bindings []KeyBinding
		width    int
	}{
		{
			name:     "empty bindings",
			bindings: nil,
			width:    80,
		},
		{
			name: "few bindings",
			bindings: []KeyBinding{
				{Key: "q", Description: "Quit"},
				{Key: "?", Description: "Help"},
			},
			width: 80,
		},
		{
			name: "many bindings narrow width",
			bindings: []KeyBinding{
				{Key: "q", Description: "Quit"},
				{Key: "?", Description: "Help"},
				{Key: "r", Description: "Refresh"},
				{Key: "Tab", Description: "Next"},
				{Key: "Enter", Description: "Select"},
			},
			width: 40,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := HelpBar(tc.bindings, &config, tc.width)

			if len(tc.bindings) > 0 && result == "" {
				t.Error("expected non-empty help bar with bindings")
			}
		})
	}
}

func TestHelpBar_Truncation(t *testing.T) {
	bindings := []KeyBinding{
		{Key: "very-long-key-1", Description: "Very long description 1"},
		{Key: "very-long-key-2", Description: "Very long description 2"},
		{Key: "very-long-key-3", Description: "Very long description 3"},
	}

	result := HelpBar(bindings, new(DefaultStatusBarConfig()), 30)

	if result == "" {
		t.Error("expected non-empty result even when truncated")
	}
}

func TestStatusBar_NarrowWidth(t *testing.T) {
	data := StatusBarData{
		KeyBindings: []KeyBinding{
			{Key: "q", Description: "Quit"},
		},
		Providers: []ProviderInfo{
			{Name: "Provider", Status: ProviderStatusConnected},
		},
	}

	result := StatusBar(data, new(DefaultStatusBarConfig()), 20)
	if result == "" {
		t.Error("expected status bar even at narrow width")
	}
}
