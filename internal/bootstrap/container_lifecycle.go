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

// This file contains lifecycle service related container methods. The lifecycle service
// manages asset seeding, file watching, and build notification handling.

import (
	"context"
	"fmt"

	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/wdk/clock"
)

// lifecycleServiceConfig holds configuration for creating a lifecycle service. Some
// dependencies like WatcherAdapter and RouterManager are daemon-specific and must be
// provided when building the service.
type lifecycleServiceConfig struct {
	// WatcherAdapter monitors file system changes; nil turns off file watching.
	WatcherAdapter lifecycle_domain.FileSystemWatcher

	// RouterManager handles router reloading; nil disables hot-reload.
	RouterManager lifecycle_domain.RouterReloadNotifier

	// TemplaterService swaps runners when switching to interpreted mode.
	TemplaterService lifecycle_domain.TemplaterRunnerSwapper

	// InterpretedOrchestrator manages interpreted builds; nil for compiled modes.
	InterpretedOrchestrator lifecycle_domain.InterpretedBuildOrchestrator

	// BuildCacheInvalidator clears cached builds; nil disables cache invalidation.
	BuildCacheInvalidator lifecycle_domain.BuildCacheInvalidator

	// DevEventNotifier broadcasts build-complete events to connected browsers via SSE. Nil
	// in production mode.
	DevEventNotifier lifecycle_domain.DevEventNotifier

	// Clock provides time functions; nil uses the real system clock.
	Clock clock.Clock

	// PathsConfig holds the resolved source directory paths for the lifecycle service. Each
	// builder populates this from its own config conversion.
	PathsConfig lifecycle_domain.LifecyclePathsConfig
}

// createLifecycleService creates a new lifecycle service with the container's services
// and the provided configuration.
//
// The lifecycle service is responsible for:
//   - Initial asset seeding and config loading
//   - File system watching (in dev modes)
//   - Build notification handling
//   - Post-startup garbage collection scheduling
//
// Takes config (*lifecycleServiceConfig) which provides daemon-specific dependencies.
//
// Returns lifecycle_domain.LifecycleService which manages the build-to-runtime lifecycle.
// Returns error when required services cannot be resolved.
func (c *Container) createLifecycleService(config *lifecycleServiceConfig) (lifecycle_domain.LifecycleService, error) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating LifecycleService...")

	registryService, err := c.GetRegistryService()
	if err != nil {
		return nil, fmt.Errorf("failed to get registry service for lifecycle: %w", err)
	}

	coordinatorService, err := c.GetCoordinatorService()
	if err != nil {
		return nil, fmt.Errorf("failed to get coordinator service for lifecycle: %w", err)
	}

	resolver, err := c.GetResolver()
	if err != nil {
		return nil, fmt.Errorf("failed to get resolver for lifecycle: %w", err)
	}

	deps := c.newLifecycleServiceDeps(config, registryService, coordinatorService, resolver)

	if captchaService, captchaErr := c.GetCaptchaService(); captchaErr == nil {
		deps.CaptchaService = captchaService
	} else {
		l.Internal("Captcha service not available for lifecycle", logger_domain.Error(captchaErr))
	}

	service := lifecycle_domain.NewLifecycleService(deps)

	shutdown.Register(c.GetAppContext(), "LifecycleService", func(ctx context.Context) error {
		return service.Stop(ctx)
	})

	l.Internal("LifecycleService created and registered for shutdown")
	return service, nil
}

// newLifecycleServiceDeps assembles the lifecycle service dependencies from the
// container's services and the daemon-specific configuration.
//
// Takes serviceConfig (*lifecycleServiceConfig) which provides the daemon-specific
// dependencies.
// Takes registryService (registry_domain.RegistryService) which manages artefacts.
// Takes coordinatorService (coordinator_domain.CoordinatorService) which coordinates
// builds.
// Takes resolver (resolver_domain.ResolverPort) which resolves import paths.
//
// Returns *lifecycle_domain.LifecycleServiceDeps which holds the dependencies, with no
// captcha service set.
func (c *Container) newLifecycleServiceDeps(
	serviceConfig *lifecycleServiceConfig,
	registryService registry_domain.RegistryService,
	coordinatorService coordinator_domain.CoordinatorService,
	resolver resolver_domain.ResolverPort,
) *lifecycle_domain.LifecycleServiceDeps {
	renderRegistry := c.GetRenderRegistry()
	renderer := c.GetRenderer()
	lifecycleClock := serviceConfig.Clock
	if lifecycleClock == nil {
		lifecycleClock = clock.RealClock()
	}

	return &lifecycle_domain.LifecycleServiceDeps{
		PathsConfig:             serviceConfig.PathsConfig,
		RegistryService:         registryService,
		CoordinatorService:      coordinatorService,
		Resolver:                resolver,
		RenderRegistryPort:      renderRegistry,
		Renderer:                renderer,
		Clock:                   lifecycleClock,
		WebsiteConfig:           c.websiteConfig,
		WatcherAdapter:          serviceConfig.WatcherAdapter,
		ProductionMode:          c.isProductionMode(),
		RegistryBlobsReadOnly:   c.registryBlobsReadOnly(),
		RouterManager:           serviceConfig.RouterManager,
		TemplaterService:        serviceConfig.TemplaterService,
		InterpretedOrchestrator: serviceConfig.InterpretedOrchestrator,
		BuildCacheInvalidator:   serviceConfig.BuildCacheInvalidator,
		DevEventNotifier:        serviceConfig.DevEventNotifier,
		ComponentRegistry:       c.GetComponentRegistry(),
		ExternalComponents:      c.externalComponents,
		AssetPipeline:           nil,
		FileSystem:              nil,
		CaptchaService:          nil,
	}
}
