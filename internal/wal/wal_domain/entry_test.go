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

package wal_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOperation_String(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		op       Operation
	}{
		{
			name:     "OpSet",
			op:       OpSet,
			expected: "SET",
		},
		{
			name:     "OpDelete",
			op:       OpDelete,
			expected: "DELETE",
		},
		{
			name:     "OpClear",
			op:       OpClear,
			expected: "CLEAR",
		},
		{
			name:     "unknown operation",
			op:       Operation(99),
			expected: "UNKNOWN",
		},
		{
			name:     "zero operation",
			op:       Operation(0),
			expected: "UNKNOWN",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.op.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestOperation_IsValid(t *testing.T) {
	testCases := []struct {
		name     string
		op       Operation
		expected bool
	}{
		{
			name:     "OpSet is valid",
			op:       OpSet,
			expected: true,
		},
		{
			name:     "OpDelete is valid",
			op:       OpDelete,
			expected: true,
		},
		{
			name:     "OpClear is valid",
			op:       OpClear,
			expected: true,
		},
		{
			name:     "zero operation is invalid",
			op:       Operation(0),
			expected: false,
		},
		{
			name:     "high value operation is invalid",
			op:       Operation(99),
			expected: false,
		},
		{
			name:     "operation just above OpClear is invalid",
			op:       Operation(4),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.op.IsValid()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEntry_IsExpired(t *testing.T) {
	nowNano := int64(1000000000000)

	testCases := []struct {
		name      string
		expiresAt int64
		expected  bool
	}{
		{
			name:      "no expiration (zero)",
			expiresAt: 0,
			expected:  false,
		},
		{
			name:      "expires in the future",
			expiresAt: nowNano + 1000,
			expected:  false,
		},
		{
			name:      "expired in the past",
			expiresAt: nowNano - 1000,
			expected:  true,
		},
		{
			name:      "expires exactly at now",
			expiresAt: nowNano,
			expected:  false,
		},
		{
			name:      "expired one nanosecond ago",
			expiresAt: nowNano - 1,
			expected:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := Entry[string, string]{
				Key:       "test-key",
				Value:     "test-value",
				ExpiresAt: tc.expiresAt,
				Operation: OpSet,
			}
			result := entry.IsExpired(nowNano)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEntry_HasTags(t *testing.T) {
	testCases := []struct {
		name     string
		tags     []string
		expected bool
	}{
		{
			name:     "nil tags",
			tags:     nil,
			expected: false,
		},
		{
			name:     "empty tags slice",
			tags:     []string{},
			expected: false,
		},
		{
			name:     "single tag",
			tags:     []string{"tag1"},
			expected: true,
		},
		{
			name:     "multiple tags",
			tags:     []string{"tag1", "tag2", "tag3"},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := Entry[string, string]{
				Key:       "test-key",
				Value:     "test-value",
				Tags:      tc.tags,
				Operation: OpSet,
			}
			result := entry.HasTags()
			assert.Equal(t, tc.expected, result)
		})
	}
}
