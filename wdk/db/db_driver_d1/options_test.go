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

package db_driver_d1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewClientOptions(t *testing.T) {
	testCases := []struct {
		name     string
		options  []Option
		expected clientOptions
	}{
		{
			name:    "defaults",
			options: nil,
			expected: clientOptions{
				baseURL:           "",
				requestTimeout:    defaultRequestTimeout,
				maxResponseBytes:  defaultMaxResponseBytes,
				requestsPerSecond: defaultRequestsPerSecond,
			},
		},
		{
			name: "overrides",
			options: []Option{
				WithRequestTimeout(5 * time.Second),
				WithMaxResponseBytes(1024),
				WithRequestsPerSecond(10),
				withBaseURL("http://example.test"),
			},
			expected: clientOptions{
				baseURL:           "http://example.test",
				requestTimeout:    5 * time.Second,
				maxResponseBytes:  1024,
				requestsPerSecond: 10,
			},
		},
		{
			name: "non-positive values keep the defaults",
			options: []Option{
				WithRequestTimeout(0),
				WithMaxResponseBytes(-1),
				WithRequestsPerSecond(0),
			},
			expected: clientOptions{
				baseURL:           "",
				requestTimeout:    defaultRequestTimeout,
				maxResponseBytes:  defaultMaxResponseBytes,
				requestsPerSecond: defaultRequestsPerSecond,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, newClientOptions(testCase.options))
		})
	}
}
