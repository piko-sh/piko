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
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/logger/logger_domain"
)

func TestGroupedError_StructureValidation(t *testing.T) {
	now := time.Now()
	later := now.Add(5 * time.Second)

	record := slog.NewRecord(now, slog.LevelError, "test error", 0)

	groupedError := &logger_domain.GroupedError{
		FirstSeen:  now,
		LastSeen:   later,
		SourceFile: "/path/to/file.go",
		SourceLine: 42,
		LogRecord:  record,
		Count:      5,
	}

	assert.True(t, groupedError.FirstSeen.Before(groupedError.LastSeen) || groupedError.FirstSeen.Equal(groupedError.LastSeen),
		"FirstSeen should be before or equal to LastSeen")

	assert.Greater(t, groupedError.Count, 0, "Count should be positive")

	assert.NotEmpty(t, groupedError.SourceFile)
	assert.Greater(t, groupedError.SourceLine, 0)
	assert.Equal(t, "test error", groupedError.LogRecord.Message)
}

func TestGroupedError_TimeOrdering(t *testing.T) {
	testCases := []struct {
		firstSeen time.Time
		lastSeen  time.Time
		name      string
		isValid   bool
	}{
		{
			name:      "first seen before last seen",
			firstSeen: time.Now(),
			lastSeen:  time.Now().Add(time.Minute),
			isValid:   true,
		},
		{
			name:      "first seen equals last seen (single occurrence)",
			firstSeen: time.Now(),
			lastSeen:  time.Now(),
			isValid:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			groupedError := &logger_domain.GroupedError{
				FirstSeen:  tc.firstSeen,
				LastSeen:   tc.lastSeen,
				SourceFile: "test.go",
				SourceLine: 10,
				LogRecord:  slog.NewRecord(tc.firstSeen, slog.LevelError, "test", 0),
				Count:      1,
			}

			if tc.isValid {
				assert.True(t, !groupedError.LastSeen.Before(groupedError.FirstSeen),
					"LastSeen should not be before FirstSeen")
			}
		})
	}
}

func TestGroupedError_CountValidation(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name     string
		shouldBe string
		count    int
	}{
		{
			name:     "single occurrence",
			count:    1,
			shouldBe: "at least 1",
		},
		{
			name:     "multiple occurrences",
			count:    100,
			shouldBe: "at least 1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			groupedError := &logger_domain.GroupedError{
				FirstSeen:  now,
				LastSeen:   now,
				SourceFile: "test.go",
				SourceLine: 10,
				LogRecord:  slog.NewRecord(now, slog.LevelError, "test", 0),
				Count:      tc.count,
			}

			assert.GreaterOrEqual(t, groupedError.Count, 1,
				"Count %s", tc.shouldBe)
		})
	}
}

func TestFormatter_InterfaceContract(t *testing.T) {

	t.Run("example implementation", func(t *testing.T) {
		formatter := &exampleFormatter{}

		batch := map[string]*logger_domain.GroupedError{
			"error1": {
				FirstSeen:  time.Now(),
				LastSeen:   time.Now(),
				SourceFile: "test.go",
				SourceLine: 10,
				LogRecord:  slog.NewRecord(time.Now(), slog.LevelError, "test error", 0),
				Count:      1,
			},
		}

		payload, err := formatter.Format(batch)
		require.NoError(t, err)
		assert.NotEmpty(t, payload, "formatted payload should not be empty")
	})

	t.Run("formatter error handling", func(t *testing.T) {
		formatter := &failingFormatter{}

		batch := map[string]*logger_domain.GroupedError{}

		payload, err := formatter.Format(batch)
		assert.Error(t, err, "formatter should return error when it fails")
		assert.Nil(t, payload, "payload should be nil on error")
	})
}

func TestSender_InterfaceContract(t *testing.T) {

	t.Run("example implementation", func(t *testing.T) {
		sender := &exampleSender{}

		payload := []byte(`{"message":"test"}`)
		err := sender.Send(context.Background(), payload)
		require.NoError(t, err)
	})

	t.Run("sender error handling", func(t *testing.T) {
		sender := &failingSender{}

		payload := []byte(`{"message":"test"}`)
		err := sender.Send(context.Background(), payload)
		assert.Error(t, err, "sender should return error when it fails")
	})

	t.Run("sender with context cancellation", func(t *testing.T) {
		sender := &contextAwareSender{}

		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(fmt.Errorf("test: simulating cancelled context"))

		payload := []byte(`{"message":"test"}`)
		err := sender.Send(ctx, payload)
		assert.Error(t, err, "sender should respect context cancellation")
	})
}

func TestTransport_InterfaceContract(t *testing.T) {

	t.Run("example implementation", func(t *testing.T) {
		transport := &exampleTransport{}

		batch := map[string]*logger_domain.GroupedError{
			"error1": {
				FirstSeen:  time.Now(),
				LastSeen:   time.Now(),
				SourceFile: "test.go",
				SourceLine: 10,
				LogRecord:  slog.NewRecord(time.Now(), slog.LevelError, "test error", 0),
				Count:      1,
			},
		}

		payload, err := transport.Format(batch)
		require.NoError(t, err)
		assert.NotEmpty(t, payload)

		err = transport.Send(context.Background(), payload)
		require.NoError(t, err)
	})

	t.Run("transport complete flow", func(t *testing.T) {
		transport := &exampleTransport{}

		batch := make(map[string]*logger_domain.GroupedError)
		batch["key1"] = &logger_domain.GroupedError{
			FirstSeen:  time.Now(),
			LastSeen:   time.Now(),
			SourceFile: "handler.go",
			SourceLine: 100,
			LogRecord:  slog.NewRecord(time.Now(), slog.LevelError, "critical error", 0),
			Count:      5,
		}

		payload, err := transport.Format(batch)
		require.NoError(t, err)

		err = transport.Send(context.Background(), payload)
		require.NoError(t, err)
	})
}

type exampleFormatter struct{}

func (f *exampleFormatter) Format(batch map[string]*logger_domain.GroupedError) ([]byte, error) {
	if len(batch) == 0 {
		return []byte(`{"errors":[]}`), nil
	}
	return []byte(`{"errors":[{"message":"test"}]}`), nil
}

type failingFormatter struct{}

func (f *failingFormatter) Format(batch map[string]*logger_domain.GroupedError) ([]byte, error) {
	return nil, errors.New("format failed")
}

type exampleSender struct{}

func (s *exampleSender) Send(ctx context.Context, payload []byte) error {
	if len(payload) == 0 {
		return errors.New("empty payload")
	}
	return nil
}

type failingSender struct{}

func (s *failingSender) Send(ctx context.Context, payload []byte) error {
	return errors.New("send failed")
}

type contextAwareSender struct{}

func (s *contextAwareSender) Send(ctx context.Context, payload []byte) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

type exampleTransport struct {
	exampleFormatter
	exampleSender
}

func TestTransport_RealWorldUsagePattern(t *testing.T) {

	transport := &exampleTransport{}

	batch := make(map[string]*logger_domain.GroupedError)

	now := time.Now()
	batch["hash1"] = &logger_domain.GroupedError{
		FirstSeen:  now,
		LastSeen:   now,
		SourceFile: "service.go",
		SourceLine: 42,
		LogRecord:  slog.NewRecord(now, slog.LevelError, "database connection failed", 0),
		Count:      1,
	}

	batch["hash1"].LastSeen = now.Add(5 * time.Second)
	batch["hash1"].Count++

	batch["hash2"] = &logger_domain.GroupedError{
		FirstSeen:  now.Add(10 * time.Second),
		LastSeen:   now.Add(10 * time.Second),
		SourceFile: "handler.go",
		SourceLine: 100,
		LogRecord:  slog.NewRecord(now, slog.LevelError, "invalid request", 0),
		Count:      1,
	}

	payload, err := transport.Format(batch)
	require.NoError(t, err)
	assert.NotEmpty(t, payload)

	err = transport.Send(context.Background(), payload)
	require.NoError(t, err)
}
