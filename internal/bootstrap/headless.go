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

	"piko.sh/piko/internal/logger/logger_domain"
)

// InitialiseHeadless initialises global services without an HTTP server or frontend
// assets.
//
// Call it before using global services. Only the first call initialises services; options
// configure providers as they do for piko.New.
//
// Takes opts (...Option) which configure the container with providers.
//
// Returns *Container which can be used for direct service access.
// Returns error when provider validation fails.
func InitialiseHeadless(opts ...Option) (*Container, error) {
	_, l := logger_domain.From(context.Background(), log)
	l.Internal("Initialising Piko in headless mode...")

	container := NewContainer(opts...)

	l.Internal("Validating provider configuration...")
	if err := container.ValidateProviderConfiguration(); err != nil {
		return nil, fmt.Errorf("validating provider configuration: %w", err)
	}

	initialiseGlobalServices(container)

	l.Internal("Headless initialisation complete")
	return container, nil
}
