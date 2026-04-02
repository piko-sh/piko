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

package bootstrap

import (
	"context"
	"fmt"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_domain"
)

// InitialiseHeadless bootstraps Piko's global services for headless use cases
// such as CLI tools, background workers, and microservices that need framework
// services (image processing, storage, cache, persistence) without running an
// HTTP server.
//
// Unlike ConfigAndContainer, this function:
//   - Does not require an AppRouter or Dependencies struct
//   - Does not load configuration files (piko.yaml, config.json)
//   - Does not initialise the logger from configuration
//   - Does not set up frontend assets or the .piko directory
//
// The options passed here configure the container identically to piko.New().
// For example, WithImageProvider and WithStorageProvider will register real
// providers that are accessible via the global service functions such as
// media.GetImageDimensions and storage.GetDefaultService.
//
// This function must be called before any code that uses global service access.
// It can only be called once per process (subsequent calls are no-ops due to
// sync.Once in initialiseGlobalServices).
//
// Takes opts which configure the container with providers.
//
// Returns *Container which can be used for direct service access.
// Returns error when provider validation fails.
func InitialiseHeadless(opts ...Option) (*Container, error) {
	_, l := logger_domain.From(context.Background(), log)
	l.Internal("Initialising Piko in headless mode...")

	configProvider := config.NewConfigProvider()
	container := NewContainer(configProvider, opts...)

	l.Internal("Validating provider configuration...")
	if err := container.ValidateProviderConfiguration(); err != nil {
		return nil, fmt.Errorf("validating provider configuration: %w", err)
	}

	initialiseGlobalServices(container)

	l.Internal("Headless initialisation complete")
	return container, nil
}
