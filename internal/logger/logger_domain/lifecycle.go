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

package logger_domain

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

// lifecycleManager handles shutdown hooks and closable resources for the
// logger subsystem. It implements handlerShutdown and contextShutdown.
type lifecycleManager struct {
	// shutdownHooks stores functions to run during shutdown in LIFO order.
	shutdownHooks []func()

	// closableOutputs holds io.Closer resources that are closed during shutdown.
	closableOutputs []io.Closer

	// mu guards shutdownHooks and closableOutputs during concurrent access.
	mu sync.Mutex
}

// defaultLifecycleManager is the global lifecycle manager used by package-level
// functions. It maintains backward compatibility with existing code.
var defaultLifecycleManager = newLifecycleManager()

// RegisterShutdownHook adds a function to be called when this manager shuts
// down. Hooks are called in reverse order of registration (LIFO).
//
// Takes hook (func()) which is the function to call during shutdown.
//
// Safe for concurrent use.
func (lm *lifecycleManager) RegisterShutdownHook(hook func()) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.shutdownHooks = append(lm.shutdownHooks, hook)
}

// RegisterClosable adds an io.Closer to be closed when the manager shuts down.
// This is typically used for log file handles or network connections.
//
// Takes c (io.Closer) which is the resource to close during shutdown.
//
// Safe for concurrent use.
func (lm *lifecycleManager) RegisterClosable(c io.Closer) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.closableOutputs = append(lm.closableOutputs, c)
}

// Shutdown stops the lifecycle manager by running all shutdown hooks in
// reverse order and closing all registered resources.
//
// Returns error when any resource fails to close.
//
// Safe for concurrent use.
func (lm *lifecycleManager) Shutdown(_ context.Context) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	shouldLog := len(lm.shutdownHooks) > 0 || len(lm.closableOutputs) > 0

	if shouldLog {
		shutdownLogger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		shutdownLogger.Info("Shutting down logger subsystems...")
		defer shutdownLogger.Info("Logger subsystems shut down successfully.")
	}

	for i := len(lm.shutdownHooks) - 1; i >= 0; i-- {
		lm.shutdownHooks[i]()
	}
	lm.shutdownHooks = nil

	var allErrors []error
	for _, closer := range lm.closableOutputs {
		if err := closer.Close(); err != nil {
			allErrors = append(allErrors, err)
		}
	}
	lm.closableOutputs = nil

	if len(allErrors) > 0 {
		if shouldLog {
			shutdownLogger := slog.New(slog.NewTextHandler(os.Stderr, nil))
			shutdownLogger.Error("Errors occurred during logger shutdown", "errors", allErrors)
		}
		return fmt.Errorf("errors during logger shutdown: %v", allErrors)
	}

	return nil
}

// RegisterShutdownHook registers a function to be called during logger
// shutdown. Hooks are called in reverse order of registration (LIFO).
//
// Takes hook (func()) which is the function to call during shutdown.
func RegisterShutdownHook(hook func()) {
	defaultLifecycleManager.RegisterShutdownHook(hook)
}

// RegisterClosable registers an io.Closer to be closed during logger shutdown
// using the defaultLifecycleManager. This is typically used for log file
// handles or network connections.
//
// Takes c (io.Closer) which is the resource to close during shutdown.
func RegisterClosable(c io.Closer) {
	defaultLifecycleManager.RegisterClosable(c)
}

// ClearLifecycle closes all registered closable resources and then removes all
// shutdown hooks and closables from the default lifecycle manager.
//
// This is for test cleanup to ensure tests do not affect each other. Closable
// resources (e.g. log rotators) are closed before being removed so their
// background goroutines are stopped.
//
// Safe for concurrent use.
func ClearLifecycle() {
	defaultLifecycleManager.mu.Lock()
	defer defaultLifecycleManager.mu.Unlock()

	for _, closer := range defaultLifecycleManager.closableOutputs {
		_ = closer.Close()
	}

	defaultLifecycleManager.shutdownHooks = nil
	defaultLifecycleManager.closableOutputs = nil
}

// Shutdown stops the logger subsystem by calling all registered shutdown hooks and
// closing all registered closable resources. Uses the defaultLifecycleManager.
//
// Returns error when any closable resource fails to close.
func Shutdown(ctx context.Context) error {
	return defaultLifecycleManager.Shutdown(ctx)
}

// newLifecycleManager creates a new lifecycle manager for testing.
// In production code, use defaultLifecycleManager or the package-level
// functions instead.
//
// Returns *lifecycleManager which manages shutdown hooks and closable outputs.
func newLifecycleManager() *lifecycleManager {
	return &lifecycleManager{
		shutdownHooks:   []func(){},
		closableOutputs: []io.Closer{},
		mu:              sync.Mutex{},
	}
}
