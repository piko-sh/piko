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

//go:build !bench

package logger_domain_test

import (
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/logger/logger_domain"
)

func TestInitDefaultFactory(t *testing.T) {

	originalFactory := logger_domain.GetDefaultFactory()
	t.Cleanup(func() {
		logger_domain.SetDefaultFactory(originalFactory)
	})

	testCases := []struct {
		baseLogger   *slog.Logger
		setupFunc    func()
		name         string
		expectNonNil bool
	}{
		{
			name:         "initialises with valid logger",
			baseLogger:   slog.Default(),
			setupFunc:    func() { logger_domain.SetDefaultFactory(nil) },
			expectNonNil: true,
		},
		{
			name:         "uses slog.Default when nil logger provided",
			baseLogger:   nil,
			setupFunc:    func() { logger_domain.SetDefaultFactory(nil) },
			expectNonNil: true,
		},
		{
			name:       "reconfigures existing factory",
			baseLogger: slog.New(NewRecordingHandler()),
			setupFunc: func() {

				logger_domain.InitDefaultFactory(slog.Default())
			},
			expectNonNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupFunc != nil {
				tc.setupFunc()
			}

			logger_domain.InitDefaultFactory(tc.baseLogger)

			if tc.expectNonNil {
				assert.NotNil(t, logger_domain.GetDefaultFactory())
			}
		})
	}
}

func TestGetLogger_CreatesLoggerWithName(t *testing.T) {

	originalFactory := logger_domain.GetDefaultFactory()
	t.Cleanup(func() {
		logger_domain.SetDefaultFactory(originalFactory)
	})

	testCases := []struct {
		name         string
		packageName  string
		setupFactory bool
		expectNonNil bool
	}{
		{
			name:         "creates logger with standard package name",
			packageName:  "github.com/test/package",
			setupFactory: true,
			expectNonNil: true,
		},
		{
			name:         "creates logger with short name",
			packageName:  "test",
			setupFactory: true,
			expectNonNil: true,
		},
		{
			name:         "creates logger with empty name",
			packageName:  "",
			setupFactory: true,
			expectNonNil: true,
		},
		{
			name:         "creates logger with special characters",
			packageName:  "pkg/sub-pkg/module_v2",
			setupFactory: true,
			expectNonNil: true,
		},
		{
			name:         "creates logger when factory is nil",
			packageName:  "test",
			setupFactory: false,
			expectNonNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupFactory {
				handler := NewRecordingHandler()
				logger_domain.InitDefaultFactory(slog.New(handler))
			} else {
				logger_domain.SetDefaultFactory(nil)
			}

			logger := logger_domain.GetLogger(tc.packageName)

			if tc.expectNonNil {
				require.NotNil(t, logger)
				assert.NotNil(t, logger.GetContext())
			}
		})
	}
}

func TestGetLogger_UsesFactoryConfiguration(t *testing.T) {

	originalFactory := logger_domain.GetDefaultFactory()
	originalDefault := slog.Default()
	t.Cleanup(func() {
		logger_domain.SetDefaultFactory(originalFactory)
		slog.SetDefault(originalDefault)
	})

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger_domain.InitDefaultFactory(baseLogger)
	slog.SetDefault(baseLogger)

	logger := logger_domain.GetLogger("test-package")
	require.NotNil(t, logger)

	logger.Info("test message", logger_domain.String("key", "value"))

	records := handler.GetRecords()
	require.Len(t, records, 1)
	assert.Equal(t, "test message", records[0].Message)
}

func TestGetLoggerForPackage(t *testing.T) {

	originalDefault := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(originalDefault)
	})

	testCases := []struct {
		name        string
		packageName string
	}{
		{
			name:        "creates logger for package",
			packageName: "github.com/test/package",
		},
		{
			name:        "creates logger for empty package",
			packageName: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewRecordingHandler()
			baseLogger := slog.New(handler)
			logger_domain.InitDefaultFactory(baseLogger)
			slog.SetDefault(baseLogger)

			factory := logger_domain.GetDefaultFactory()
			logger := logger_domain.GetLoggerForPackage(factory, tc.packageName)

			require.NotNil(t, logger)
			assert.NotNil(t, logger.GetContext())

			logger.Info("test")
			assert.Positive(t, handler.Count())
		})
	}
}

func TestGetLogger_MultipleCallsCreateDifferentInstances(t *testing.T) {

	originalFactory := logger_domain.GetDefaultFactory()
	t.Cleanup(func() {
		logger_domain.SetDefaultFactory(originalFactory)
	})

	logger_domain.InitDefaultFactory(slog.Default())

	logger1 := logger_domain.GetLogger("test-package")
	logger2 := logger_domain.GetLogger("test-package")

	require.NotNil(t, logger1)
	require.NotNil(t, logger2)

	logger1WithAttr := logger1.With(logger_domain.String("key1", "value1"))
	logger2WithAttr := logger2.With(logger_domain.String("key2", "value2"))

	require.NotNil(t, logger1WithAttr)
	require.NotNil(t, logger2WithAttr)
}

func TestGetLogger_ThreadSafety(t *testing.T) {

	originalFactory := logger_domain.GetDefaultFactory()
	originalDefault := slog.Default()
	t.Cleanup(func() {
		logger_domain.SetDefaultFactory(originalFactory)
		slog.SetDefault(originalDefault)
	})

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger_domain.InitDefaultFactory(baseLogger)
	slog.SetDefault(baseLogger)

	const goroutines = 100
	const logsPerGoroutine = 10

	RunConcurrentTest(t, goroutines, func(id int) {
		packageName := "test-package"
		logger := logger_domain.GetLogger(packageName)

		for i := range logsPerGoroutine {
			logger.Info("concurrent test",
				logger_domain.Int("goroutine_id", id),
				logger_domain.Int("log_num", i),
			)
		}
	})

	totalExpectedLogs := goroutines * logsPerGoroutine
	assert.Equal(t, totalExpectedLogs, handler.Count())
}

func TestInitDefaultFactory_ThreadSafety(t *testing.T) {

	originalFactory := logger_domain.GetDefaultFactory()
	t.Cleanup(func() {
		logger_domain.SetDefaultFactory(originalFactory)
	})

	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			handler := NewRecordingHandler()
			logger_domain.InitDefaultFactory(slog.New(handler))
		}(i)
	}

	wg.Wait()

	assert.NotNil(t, logger_domain.GetDefaultFactory())

	logger := logger_domain.GetLogger("test")
	assert.NotNil(t, logger)
}

func TestGetLogger_AfterFactoryReconfiguration(t *testing.T) {

	originalFactory := logger_domain.GetDefaultFactory()
	originalDefault := slog.Default()
	t.Cleanup(func() {
		logger_domain.SetDefaultFactory(originalFactory)
		slog.SetDefault(originalDefault)
	})

	handler1 := NewRecordingHandler()
	baseLogger1 := slog.New(handler1)
	logger_domain.InitDefaultFactory(baseLogger1)
	slog.SetDefault(baseLogger1)

	logger1 := logger_domain.GetLogger("test-package")
	logger1.Info("message from handler1")

	handler2 := NewRecordingHandler()
	baseLogger2 := slog.New(handler2)
	logger_domain.InitDefaultFactory(baseLogger2)
	slog.SetDefault(baseLogger2)

	logger2 := logger_domain.GetLogger("test-package")
	logger2.Info("message from handler2")

	assert.Equal(t, 1, handler1.Count())

	assert.Equal(t, 1, handler2.Count())

	records1 := handler1.GetRecords()
	records2 := handler2.GetRecords()
	assert.Equal(t, "message from handler1", records1[0].Message)
	assert.Equal(t, "message from handler2", records2[0].Message)
}

func TestLogFactory_DirectInstantiation(t *testing.T) {
	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)

	originalFactory := logger_domain.GetDefaultFactory()
	originalDefault := slog.Default()
	logger_domain.InitDefaultFactory(baseLogger)
	slog.SetDefault(baseLogger)

	t.Cleanup(func() {
		logger_domain.SetDefaultFactory(originalFactory)
		slog.SetDefault(originalDefault)
	})

	factory := logger_domain.GetDefaultFactory()
	logger := logger_domain.GetLoggerForPackage(factory, "test")
	require.NotNil(t, logger)

	logger.Info("test message")
	assert.Equal(t, 1, handler.Count())
}
