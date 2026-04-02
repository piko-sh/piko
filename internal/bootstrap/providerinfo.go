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
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/provider/provider_domain"
)

// resourceDescriptorSources lists all services that may implement
// provider_domain.ResourceDescriptor. This table-driven approach follows the
// same pattern as healthprobe.go serviceProbes.
var resourceDescriptorSources = []probeRegistration{
	{name: "EmailService", getter: func(c *Container) (any, error) { return c.GetEmailService() }},
	{name: "StorageService", getter: func(c *Container) (any, error) { return c.GetStorageService() }},
	{name: "CacheService", getter: func(c *Container) (any, error) { return c.GetCacheService() }},
	{name: "CollectionService", getter: func(c *Container) (any, error) { return c.GetCollectionService() }},
	{name: "DatabaseService", getter: func(c *Container) (any, error) { return c.GetDatabaseService() }},
	{name: "LLMService", getter: func(c *Container) (any, error) { return c.GetLLMService() }},
}

// createProviderInfoAggregator discovers all services implementing
// ResourceDescriptor and builds an aggregator for monitoring.
//
// Takes c (*Container) which provides access to service instances.
//
// Returns monitoring_domain.ProviderInfoInspector which aggregates all
// discovered resource descriptors, or nil if none are found.
func createProviderInfoAggregator(c *Container) monitoring_domain.ProviderInfoInspector {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Discovering resource descriptors...")

	aggregator := monitoring_domain.NewProviderInfoAggregator()

	for _, src := range resourceDescriptorSources {
		service, err := src.getter(c)
		if err != nil {
			l.Internal("Skipping resource descriptor source (service unavailable)",
				logger_domain.String("service", src.name),
				logger_domain.String("reason", err.Error()))
			continue
		}
		if descriptor, ok := service.(provider_domain.ResourceDescriptor); ok {
			aggregator.Register(descriptor)
			l.Internal("Registered resource descriptor",
				logger_domain.String("service", src.name),
				logger_domain.String("resource_type", descriptor.ResourceType()))
		}
	}

	if !aggregator.HasDescriptors() {
		l.Internal("No resource descriptors found")
		return nil
	}

	l.Internal("Provider info aggregator initialised")
	return aggregator
}
