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

// This file contains LLM service related container methods.

import (
	"fmt"
	"time"

	llm_adapters_budget "piko.sh/piko/internal/llm/llm_adapters/budget_store/cache"
	llm_adapters_cache "piko.sh/piko/internal/llm/llm_adapters/cache"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/shutdown"
)

// fieldProvider is the structured log field key for provider names.
const fieldProvider = "provider"

// AddLLMProvider registers a named LLM provider for completions.
//
// If the provider implements a shutdown interface (Close, Shutdown, or Stop),
// it will be automatically registered for graceful shutdown.
//
// Takes name (string) which identifies the provider for later retrieval.
// Takes provider (llm_domain.LLMProviderPort) which handles LLM requests.
func (c *Container) AddLLMProvider(name string, provider llm_domain.LLMProviderPort) {
	if c.llmProviders == nil {
		c.llmProviders = make(map[string]llm_domain.LLMProviderPort)
	}
	c.llmProviders[name] = provider
	registerCloseableForShutdown(c.GetAppContext(), "LLMProvider-"+name, provider)
}

// SetLLMDefaultProvider sets the default LLM provider to use when none is
// specified.
//
// Takes name (string) which is the provider name to set as default.
func (c *Container) SetLLMDefaultProvider(name string) {
	c.llmDefaultProvider = name
}

// AddEmbeddingProvider registers a standalone embedding provider for
// embedding-only services such as Voyage AI. Unlike AddLLMProvider, this does
// not register a completion provider.
//
// If the provider implements a shutdown interface (Close, Shutdown, or Stop),
// it will be automatically registered for graceful shutdown.
//
// Takes name (string) which identifies the provider.
// Takes provider (llm_domain.EmbeddingProviderPort) which handles embedding
// requests.
func (c *Container) AddEmbeddingProvider(name string, provider llm_domain.EmbeddingProviderPort) {
	if c.llmEmbeddingProviders == nil {
		c.llmEmbeddingProviders = make(map[string]llm_domain.EmbeddingProviderPort)
	}
	c.llmEmbeddingProviders[name] = provider
	registerCloseableForShutdown(c.GetAppContext(), "EmbeddingProvider-"+name, provider)
}

// SetDefaultEmbeddingProvider sets the name of the default embedding provider.
// When set, this takes precedence over the auto-detected embedding support
// from the default LLM provider.
//
// Takes name (string) which is the provider name to set as default.
func (c *Container) SetDefaultEmbeddingProvider(name string) {
	c.llmDefaultEmbeddingProvider = name
}

// SetLLMService sets a custom LLM service, overriding the default.
//
// If the service implements a shutdown interface (Close, Shutdown, or Stop),
// it will be automatically registered for graceful shutdown.
//
// Takes service (llm_domain.Service) which is the custom service to use.
func (c *Container) SetLLMService(service llm_domain.Service) {
	c.llmServiceOverride = service
	registerCloseableForShutdown(c.GetAppContext(), "LLMService", service)
}

// GetLLMService returns the LLM service, initialising a default one if none
// was provided.
//
// Returns llm_domain.Service which is the configured LLM service.
// Returns error when the LLM service could not be created.
func (c *Container) GetLLMService() (llm_domain.Service, error) {
	c.llmOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.llmServiceOverride != nil {
			l.Internal("Using provided LLMService override.")
			c.llmService = c.llmServiceOverride
			return
		}
		c.createDefaultLLMService()
	})
	return c.llmService, c.llmErr
}

// createDefaultLLMService creates and sets up the default LLM service.
//
// It creates the service and registers any providers that were added via
// AddLLMProvider. Any errors are stored in c.llmErr rather than returned.
func (c *Container) createDefaultLLMService() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating default LLMService...")

	s := llm_domain.NewService(c.llmDefaultProvider)

	if err := c.registerLLMProviders(s, l); err != nil {
		c.llmErr = err
		return
	}

	c.registerStandaloneEmbeddingProviders(s, l)
	c.configureLLMDefaults(s, l)

	if err := c.configureLLMCache(s); err != nil {
		l.Warn("Failed to configure LLM cache, caching disabled",
			logger_domain.Error(err),
		)
	}

	if err := c.configureLLMBudget(s); err != nil {
		l.Warn("Failed to configure LLM budget store, budget tracking disabled",
			logger_domain.Error(err),
		)
	}

	shutdown.Register(c.GetAppContext(), "LLMService", s.Close)

	c.llmService = s
}

// registerLLMProviders registers all LLM providers and auto-registers any that
// also support embeddings.
//
// Takes s (llm_domain.Service) which is the service to register providers with.
// Takes l (logger_domain.Logger) which provides structured logging.
//
// Returns error when a provider fails to register.
func (c *Container) registerLLMProviders(s llm_domain.Service, l logger_domain.Logger) error {
	for name, provider := range c.llmProviders {
		if err := s.RegisterProvider(c.GetAppContext(), name, provider); err != nil {
			l.Error("Failed to register LLM provider",
				logger_domain.String(fieldProvider, name),
				logger_domain.Error(err),
			)
			return fmt.Errorf("registering LLM provider: %w", err)
		}

		if ep, ok := provider.(llm_domain.EmbeddingProviderPort); ok {
			if err := s.RegisterEmbeddingProvider(c.GetAppContext(), name, ep); err != nil {
				l.Warn("Failed to auto-register embedding provider",
					logger_domain.String(fieldProvider, name),
					logger_domain.Error(err),
				)
			}
		}
	}
	return nil
}

// registerStandaloneEmbeddingProviders registers all standalone embedding
// providers that were added via AddEmbeddingProvider.
//
// Takes s (llm_domain.Service) which is the service to register with.
// Takes l (logger_domain.Logger) which provides structured logging.
func (c *Container) registerStandaloneEmbeddingProviders(s llm_domain.Service, l logger_domain.Logger) {
	for name, provider := range c.llmEmbeddingProviders {
		if err := s.RegisterEmbeddingProvider(c.GetAppContext(), name, provider); err != nil {
			l.Warn("Failed to register standalone embedding provider",
				logger_domain.String(fieldProvider, name),
				logger_domain.Error(err),
			)
		}
	}
}

// configureLLMDefaults sets the default LLM and embedding providers on the
// service.
//
// Takes s (llm_domain.Service) which is the service to configure.
// Takes l (logger_domain.Logger) which provides structured logging.
func (c *Container) configureLLMDefaults(s llm_domain.Service, l logger_domain.Logger) {
	if c.llmDefaultProvider != "" && len(c.llmProviders) > 0 {
		if err := s.SetDefaultProvider(c.GetAppContext(), c.llmDefaultProvider); err != nil {
			l.Warn("Failed to set default LLM provider (provider may not exist)",
				logger_domain.String(fieldProvider, c.llmDefaultProvider),
				logger_domain.Error(err),
			)
		}
	}

	c.configureDefaultEmbeddingProvider(s, l)
}

// configureDefaultEmbeddingProvider sets the default embedding provider, either
// from an explicit setting or by auto-detecting from the default LLM provider.
//
// Takes s (llm_domain.Service) which is the service to configure.
// Takes l (logger_domain.Logger) which provides structured logging.
func (c *Container) configureDefaultEmbeddingProvider(s llm_domain.Service, l logger_domain.Logger) {
	if c.llmDefaultEmbeddingProvider != "" {
		if err := s.SetDefaultEmbeddingProvider(c.llmDefaultEmbeddingProvider); err != nil {
			l.Warn("Failed to set default embedding provider",
				logger_domain.String(fieldProvider, c.llmDefaultEmbeddingProvider),
				logger_domain.Error(err),
			)
		}
		return
	}

	if c.llmDefaultProvider == "" {
		return
	}
	provider, exists := c.llmProviders[c.llmDefaultProvider]
	if !exists {
		return
	}
	if _, ok := provider.(llm_domain.EmbeddingProviderPort); !ok {
		return
	}
	if err := s.SetDefaultEmbeddingProvider(c.llmDefaultProvider); err != nil {
		l.Warn("Failed to set default embedding provider",
			logger_domain.String(fieldProvider, c.llmDefaultProvider),
			logger_domain.Error(err),
		)
	}
}

// configureLLMCache sets up the cache manager for the LLM service.
//
// Takes s (llm_domain.Service) which is the service to configure.
//
// Returns error when cache setup fails.
func (c *Container) configureLLMCache(s llm_domain.Service) error {
	_, l := logger_domain.From(c.GetAppContext(), log)

	cacheService, err := c.GetCacheService()
	if err != nil {
		return fmt.Errorf("getting cache service for LLM: %w", err)
	}

	cacheStore, err := llm_adapters_cache.New(c.GetAppContext(), llm_adapters_cache.Config{
		CacheService: cacheService,
		Namespace:    "llm:cache",
		MaximumSize:  10000,
	})
	if err != nil {
		return fmt.Errorf("creating LLM cache store: %w", err)
	}

	const defaultCacheTTL = time.Hour
	cacheManager := llm_domain.NewCacheManager(cacheStore, defaultCacheTTL)
	s.SetCacheManager(cacheManager)

	l.Internal("LLM cache manager configured")
	return nil
}

// configureLLMBudget sets up the cache-backed budget store for the LLM
// service, enabling shared budget tracking across instances when backed by a
// distributed cache provider.
//
// Takes s (llm_domain.Service) which is the service to configure.
//
// Returns error when budget store creation fails.
func (c *Container) configureLLMBudget(s llm_domain.Service) error {
	_, l := logger_domain.From(c.GetAppContext(), log)

	cacheService, err := c.GetCacheService()
	if err != nil {
		return fmt.Errorf("getting cache service for LLM budget: %w", err)
	}

	budgetStore, err := llm_adapters_budget.New(c.GetAppContext(), llm_adapters_budget.Config{
		CacheService: cacheService,
	})
	if err != nil {
		return fmt.Errorf("creating LLM budget store: %w", err)
	}

	budgetManager := llm_domain.NewBudgetManager(budgetStore, s.GetCostCalculator())
	s.SetBudgetManager(budgetManager)

	l.Internal("LLM budget manager configured with cache-backed store")
	return nil
}
