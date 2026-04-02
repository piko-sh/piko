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

package logger_domain_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/logger/logger_domain"
)

func TestStandardisedKeys_AreUnique(t *testing.T) {
	keys := map[string]string{
		"KeyTime":      logger_domain.KeyTime,
		"KeyLevel":     logger_domain.KeyLevel,
		"KeyMessage":   logger_domain.KeyMessage,
		"KeySource":    logger_domain.KeySource,
		"KeyPID":       logger_domain.KeyPID,
		"KeyHost":      logger_domain.KeyHost,
		"KeyContext":   logger_domain.KeyContext,
		"KeyReference": logger_domain.KeyReference,
		"KeyMethod":    logger_domain.KeyMethod,
		"KeyTaskQueue": logger_domain.KeyTaskQueue,
		"KeyIPAddress": logger_domain.KeyIPAddress,
		"KeyAccountID": logger_domain.KeyAccountID,
		"KeyAttempt":   logger_domain.KeyAttempt,
		"KeyError":     logger_domain.KeyError,
	}

	seenValues := make(map[string]string)

	for constName, value := range keys {
		if existingConst, exists := seenValues[value]; exists {
			t.Errorf("Duplicate key value %q found in both %s and %s",
				value, constName, existingConst)
		}
		seenValues[value] = constName
	}
}

func TestStandardisedKeys_FollowNamingConvention(t *testing.T) {
	testCases := []struct {
		name              string
		key               string
		description       string
		expectedMinLength int
		expectedMaxLength int
	}{
		{
			name:              "KeyTime",
			key:               logger_domain.KeyTime,
			expectedMinLength: 1,
			expectedMaxLength: 10,
			description:       "timestamp key",
		},
		{
			name:              "KeyLevel",
			key:               logger_domain.KeyLevel,
			expectedMinLength: 1,
			expectedMaxLength: 10,
			description:       "log level key",
		},
		{
			name:              "KeyMessage",
			key:               logger_domain.KeyMessage,
			expectedMinLength: 1,
			expectedMaxLength: 10,
			description:       "message key",
		},
		{
			name:              "KeySource",
			key:               logger_domain.KeySource,
			expectedMinLength: 1,
			expectedMaxLength: 10,
			description:       "source location key",
		},
		{
			name:              "KeyPID",
			key:               logger_domain.KeyPID,
			expectedMinLength: 1,
			expectedMaxLength: 10,
			description:       "process ID key",
		},
		{
			name:              "KeyHost",
			key:               logger_domain.KeyHost,
			expectedMinLength: 1,
			expectedMaxLength: 10,
			description:       "hostname key",
		},
		{
			name:              "KeyContext",
			key:               logger_domain.KeyContext,
			expectedMinLength: 1,
			expectedMaxLength: 10,
			description:       "context key",
		},
		{
			name:              "KeyReference",
			key:               logger_domain.KeyReference,
			expectedMinLength: 1,
			expectedMaxLength: 10,
			description:       "reference ID key",
		},
		{
			name:              "KeyMethod",
			key:               logger_domain.KeyMethod,
			expectedMinLength: 1,
			expectedMaxLength: 10,
			description:       "HTTP method key",
		},
		{
			name:              "KeyTaskQueue",
			key:               logger_domain.KeyTaskQueue,
			expectedMinLength: 1,
			expectedMaxLength: 10,
			description:       "task queue key",
		},
		{
			name:              "KeyIPAddress",
			key:               logger_domain.KeyIPAddress,
			expectedMinLength: 1,
			expectedMaxLength: 10,
			description:       "IP address key",
		},
		{
			name:              "KeyAccountID",
			key:               logger_domain.KeyAccountID,
			expectedMinLength: 1,
			expectedMaxLength: 10,
			description:       "account ID key",
		},
		{
			name:              "KeyAttempt",
			key:               logger_domain.KeyAttempt,
			expectedMinLength: 1,
			expectedMaxLength: 10,
			description:       "retry attempt key",
		},
		{
			name:              "KeyError",
			key:               logger_domain.KeyError,
			expectedMinLength: 1,
			expectedMaxLength: 10,
			description:       "error key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			assert.NotEmpty(t, tc.key, "%s should not be empty", tc.name)

			assert.GreaterOrEqual(t, len(tc.key), tc.expectedMinLength,
				"%s should be at least %d characters", tc.name, tc.expectedMinLength)
			assert.LessOrEqual(t, len(tc.key), tc.expectedMaxLength,
				"%s should be at most %d characters", tc.name, tc.expectedMaxLength)

			for _, char := range tc.key {
				isValid := (char >= 'a' && char <= 'z') || char == '_'
				assert.True(t, isValid,
					"%s contains invalid character %q (only lowercase letters and underscores allowed)",
					tc.name, char)
			}
		})
	}
}

func TestStandardisedKeys_Values(t *testing.T) {

	expectedKeys := map[string]string{
		"KeyTime":      "time",
		"KeyLevel":     "level",
		"KeyMessage":   "msg",
		"KeySource":    "source",
		"KeyPID":       "pid",
		"KeyHost":      "host",
		"KeyContext":   "ctx",
		"KeyReference": "ref",
		"KeyMethod":    "mtd",
		"KeyTaskQueue": "tq",
		"KeyIPAddress": "ip",
		"KeyAccountID": "uid",
		"KeyAttempt":   "attempt",
		"KeyError":     "error",
	}

	assert.Equal(t, expectedKeys["KeyTime"], logger_domain.KeyTime)
	assert.Equal(t, expectedKeys["KeyLevel"], logger_domain.KeyLevel)
	assert.Equal(t, expectedKeys["KeyMessage"], logger_domain.KeyMessage)
	assert.Equal(t, expectedKeys["KeySource"], logger_domain.KeySource)
	assert.Equal(t, expectedKeys["KeyPID"], logger_domain.KeyPID)
	assert.Equal(t, expectedKeys["KeyHost"], logger_domain.KeyHost)
	assert.Equal(t, expectedKeys["KeyContext"], logger_domain.KeyContext)
	assert.Equal(t, expectedKeys["KeyReference"], logger_domain.KeyReference)
	assert.Equal(t, expectedKeys["KeyMethod"], logger_domain.KeyMethod)
	assert.Equal(t, expectedKeys["KeyTaskQueue"], logger_domain.KeyTaskQueue)
	assert.Equal(t, expectedKeys["KeyIPAddress"], logger_domain.KeyIPAddress)
	assert.Equal(t, expectedKeys["KeyAccountID"], logger_domain.KeyAccountID)
	assert.Equal(t, expectedKeys["KeyAttempt"], logger_domain.KeyAttempt)
	assert.Equal(t, expectedKeys["KeyError"], logger_domain.KeyError)
}

func TestStandardisedKeys_UsageExample(t *testing.T) {

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger := logger_domain.New(baseLogger, "test")

	logger.Info("User request received",
		logger_domain.String(logger_domain.KeyMethod, "GET"),
		logger_domain.String(logger_domain.KeyIPAddress, "192.168.1.1"),
		logger_domain.String(logger_domain.KeyAccountID, "usr_123"),
	)

	records := handler.GetRecords()
	assert.Len(t, records, 1)

	attrs := handler.GetRecordAttrs(records[0])
	assert.Equal(t, "GET", attrs[logger_domain.KeyMethod])
	assert.Equal(t, "192.168.1.1", attrs[logger_domain.KeyIPAddress])
	assert.Equal(t, "usr_123", attrs[logger_domain.KeyAccountID])
}
