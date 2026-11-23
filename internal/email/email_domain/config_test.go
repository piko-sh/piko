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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultServiceConfig(t *testing.T) {
	config := defaultServiceConfig()

	assert.Equal(t, defaultMaxTotalRecipients, config.MaxTotalRecipients)
	assert.Equal(t, int64(defaultMaxPayloadSizeBytes), config.MaxPayloadSizeBytes)
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
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			config := defaultServiceConfig()
			WithMaxTotalRecipients(testCase.input)(&config)
			assert.Equal(t, testCase.expected, config.MaxTotalRecipients)
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
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			config := defaultServiceConfig()
			WithMaxPayloadSizeBytes(testCase.input)(&config)
			assert.Equal(t, testCase.expected, config.MaxPayloadSizeBytes)
		})
	}
}

func TestServiceOptionsChain(t *testing.T) {
	config := defaultServiceConfig()
	for _, option := range []ServiceOption{
		WithMaxTotalRecipients(25),
		WithMaxPayloadSizeBytes(1024 * 1024),
	} {
		option(&config)
	}

	assert.Equal(t, 25, config.MaxTotalRecipients)
	assert.Equal(t, int64(1024*1024), config.MaxPayloadSizeBytes)
}
