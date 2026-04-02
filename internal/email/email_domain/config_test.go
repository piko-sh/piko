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

package email_domain

import "testing"

func TestDefaultServiceConfig(t *testing.T) {
	config := defaultServiceConfig()
	if config.MaxTotalRecipients != defaultMaxTotalRecipients {
		t.Errorf("MaxTotalRecipients = %d, want %d", config.MaxTotalRecipients, defaultMaxTotalRecipients)
	}
	if config.MaxPayloadSizeBytes != defaultMaxPayloadSizeBytes {
		t.Errorf("MaxPayloadSizeBytes = %d, want %d", config.MaxPayloadSizeBytes, defaultMaxPayloadSizeBytes)
	}
	if config.MaxRetryHeapSize != defaultServiceRetryHeapMax {
		t.Errorf("MaxRetryHeapSize = %d, want %d", config.MaxRetryHeapSize, defaultServiceRetryHeapMax)
	}
}

func TestWithMaxTotalRecipients(t *testing.T) {
	testCases := []struct {
		name     string
		input    int
		expected int
	}{
		{name: "positive value", input: 50, expected: 50},
		{name: "zero ignored", input: 0, expected: defaultMaxTotalRecipients},
		{name: "negative ignored", input: -1, expected: defaultMaxTotalRecipients},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := defaultServiceConfig()
			WithMaxTotalRecipients(tc.input)(&config)
			if config.MaxTotalRecipients != tc.expected {
				t.Errorf("MaxTotalRecipients = %d, want %d", config.MaxTotalRecipients, tc.expected)
			}
		})
	}
}

func TestWithMaxPayloadSizeBytes(t *testing.T) {
	testCases := []struct {
		name     string
		input    int64
		expected int64
	}{
		{name: "positive value", input: 1024, expected: 1024},
		{name: "zero ignored", input: 0, expected: defaultMaxPayloadSizeBytes},
		{name: "negative ignored", input: -100, expected: defaultMaxPayloadSizeBytes},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := defaultServiceConfig()
			WithMaxPayloadSizeBytes(tc.input)(&config)
			if config.MaxPayloadSizeBytes != tc.expected {
				t.Errorf("MaxPayloadSizeBytes = %d, want %d", config.MaxPayloadSizeBytes, tc.expected)
			}
		})
	}
}

func TestWithMaxRetryHeapSize(t *testing.T) {
	testCases := []struct {
		name     string
		input    int
		expected int
	}{
		{name: "positive value", input: 1000, expected: 1000},
		{name: "zero ignored", input: 0, expected: defaultServiceRetryHeapMax},
		{name: "negative ignored", input: -5, expected: defaultServiceRetryHeapMax},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := defaultServiceConfig()
			WithMaxRetryHeapSize(tc.input)(&config)
			if config.MaxRetryHeapSize != tc.expected {
				t.Errorf("MaxRetryHeapSize = %d, want %d", config.MaxRetryHeapSize, tc.expected)
			}
		})
	}
}

func TestServiceOptions_Chained(t *testing.T) {
	config := defaultServiceConfig()
	opts := []ServiceOption{
		WithMaxTotalRecipients(25),
		WithMaxPayloadSizeBytes(1024 * 1024),
		WithMaxRetryHeapSize(500),
	}
	for _, opt := range opts {
		opt(&config)
	}
	if config.MaxTotalRecipients != 25 {
		t.Errorf("MaxTotalRecipients = %d, want 25", config.MaxTotalRecipients)
	}
	if config.MaxPayloadSizeBytes != 1024*1024 {
		t.Errorf("MaxPayloadSizeBytes = %d, want %d", config.MaxPayloadSizeBytes, 1024*1024)
	}
	if config.MaxRetryHeapSize != 500 {
		t.Errorf("MaxRetryHeapSize = %d, want 500", config.MaxRetryHeapSize)
	}
}
