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

// This file contains cache service related container methods.

import (
	"context"
	"fmt"

	cache_adapters_otter "piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/shutdown"
)

// AddCacheProvider registers a named cache provider.
//
// If the provider implements a shutdown interface (Close, Shutdown, or Stop),
// it will be automatically registered for graceful shutdown.
//
// Takes name (string) which identifies the provider for later retrieval.
// Takes provider (cache_domain.Provider) which creates cache instances.
func (c *Container) AddCacheProvider(name string, provider cache_domain.Provider) {
	if c.cacheProviders == nil {
		c.cacheProviders = make(map[string]cache_domain.Provider)
	}
	c.cacheProviders[name] = provider
	registerCloseableForShutdown(c.GetAppContext(), "CacheProvider-"+name, provider)
}

// SetCacheDefaultProvider sets the default cache provider.
//
// Takes name (string) which is the provider name to set as default.
func (c *Container) SetCacheDefaultProvider(name string) {
	c.cacheDefaultProvider = name
}

// GetCacheService returns the cache service, initialising a default one if
// none was provided.
//
// Returns cache_domain.Service which provides caching operations.
// Returns error when the cache service could not be initialised.
func (c *Container) GetCacheService() (cache_domain.Service, error) {
	c.cacheOnce.Do(func() {
		c.createDefaultCacheService()
	})
	return c.cacheService, c.cacheErr
}

// createDefaultCacheService sets up the cache service with user-provided
// providers, or with Otter as the default if none are given.
func (c *Container) createDefaultCacheService() {
	_, l := logger_domain.From(c.GetAppContext(), log)

	defaultProviderName := c.cacheDefaultProvider
	if defaultProviderName == "" && len(c.cacheProviders) > 0 {
		for name := range c.cacheProviders {
			defaultProviderName = name
			break
		}
	}

	c.cacheService = cache_domain.NewService(defaultProviderName)

	for name, provider := range c.cacheProviders {
		if err := c.cacheService.RegisterProvider(c.GetAppContext(), name, provider); err != nil {
			l.Error("Failed to register cache provider",
				logger_domain.String("provider_name", name),
				logger_domain.Error(err))
			c.cacheErr = err
			return
		}
	}

	if c.cacheDefaultProvider != "" {
		if err := c.cacheService.SetDefaultProvider(c.GetAppContext(), c.cacheDefaultProvider); err != nil {
			l.Error("Failed to set default cache provider",
				logger_domain.String("provider_name", c.cacheDefaultProvider),
				logger_domain.Error(err))
			c.cacheErr = err
			return
		}
	}

	if len(c.cacheProviders) == 0 {
		if err := c.registerDefaultOtterProvider(); err != nil {
			return
		}
	}

	shutdown.Register(c.GetAppContext(), "CacheService", func(ctx context.Context) error {
		return c.cacheService.Close(ctx)
	})

	l.Internal("Cache service created successfully")
}

// registerDefaultOtterProvider registers the built-in Otter provider as default.
//
// Returns error when provider registration or default setting fails.
func (c *Container) registerDefaultOtterProvider() error {
	_, l := logger_domain.From(c.GetAppContext(), log)

	otterProvider := cache_adapters_otter.NewOtterProvider()
	if err := c.cacheService.RegisterProvider(c.GetAppContext(), "otter", otterProvider); err != nil {
		_ = otterProvider.Close()
		l.Error("Failed to register Otter provider", logger_domain.Error(err))
		c.cacheErr = err
		return fmt.Errorf("registering default Otter cache provider: %w", err)
	}

	if err := c.cacheService.SetDefaultProvider(c.GetAppContext(), "otter"); err != nil {
		_ = otterProvider.Close()
		l.Error("Failed to set default Otter provider", logger_domain.Error(err))
		c.cacheErr = err
		return fmt.Errorf("setting default Otter cache provider: %w", err)
	}

	l.Internal("Using default Otter cache provider (no custom providers registered)")
	return nil
}

// SetCacheService allows builders to provide a pre-configured cache service
// to the container.
//
// If the provided service implements a shutdown interface (Close, Shutdown, or
// Stop), it will be automatically registered for graceful shutdown.
//
// Takes service (cache_domain.Service) which is the cache service to use.
func (c *Container) SetCacheService(service cache_domain.Service) {
	c.cacheService = service
	registerCloseableForShutdown(c.GetAppContext(), "CacheService", service)
}
