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
	"piko.sh/piko/internal/spamdetect/spamdetect_adapters/builtin_detectors"
	"piko.sh/piko/internal/spamdetect/spamdetect_domain"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
)

// AddSpamDetector registers a named spam detection detector.
//
// If the detector implements a shutdown interface (Close, Shutdown, or
// Stop), it will be automatically registered for graceful shutdown.
//
// Takes name (string) which identifies the detector.
// Takes detector (spamdetect_domain.Detector) which handles spam analysis.
func (c *Container) AddSpamDetector(name string, detector spamdetect_domain.Detector) {
	if c.spamdetectDetectors == nil {
		c.spamdetectDetectors = make(map[string]spamdetect_domain.Detector)
	}
	c.spamdetectDetectors[name] = detector
	registerCloseableForShutdown(c.GetAppContext(), "SpamDetector-"+name, detector)
}

// GetSpamDetectService returns the spam detection service, initialising a
// default one if none was provided.
//
// Returns spamdetect_domain.SpamDetectServicePort which is the configured
// service.
// Returns error when creation fails.
func (c *Container) GetSpamDetectService() (spamdetect_domain.SpamDetectServicePort, error) {
	c.spamdetectOnce.Do(func() {
		c.createDefaultSpamDetectService()
	})
	return c.spamdetectService, c.spamdetectErr
}

// SetSpamDetectService sets a pre-configured spam detection service on the
// container.
//
// Takes service (spamdetect_domain.SpamDetectServicePort) which is the
// pre-configured service.
func (c *Container) SetSpamDetectService(service spamdetect_domain.SpamDetectServicePort) {
	c.spamdetectOnce.Do(func() {
		c.spamdetectService = service
		registerCloseableForShutdown(c.GetAppContext(), "SpamDetectService", service)
	})
}

// SetSpamDetectFeedbackStore stores a feedback store for deferred
// application when the spam detection service is lazily created.
//
// Takes store (spamdetect_domain.FeedbackStore) which persists spam/ham
// feedback.
func (c *Container) SetSpamDetectFeedbackStore(store spamdetect_domain.FeedbackStore) {
	c.spamdetectFeedbackStore = store
}

// createDefaultSpamDetectService builds the spam detection service from
// detectors or config.
func (c *Container) createDefaultSpamDetectService() {
	if c.spamdetectService != nil {
		return
	}
	ctx := c.GetAppContext()
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Creating default SpamDetectService...")

	if len(c.spamdetectDetectors) > 0 {
		l.Internal("Using spam detection detectors registered via options")
		c.buildSpamDetectServiceFromDetectors(ctx)
		return
	}

	enabled := deref(c.config.ServerConfig.Security.SpamDetectEnabled, false)
	if !enabled {
		l.Internal("Spam detection not configured; spam detection service disabled.")
		c.spamdetectService = spamdetect_domain.NewDisabledSpamDetectService()
		return
	}

	l.Internal("Creating built-in detectors from config")
	c.buildSpamDetectServiceFromConfig(ctx)
}

// buildSpamDetectServiceFromDetectors creates the service using pre-registered detectors.
func (c *Container) buildSpamDetectServiceFromDetectors(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)

	serviceConfig := c.spamDetectServiceConfig()
	service, err := spamdetect_domain.NewSpamDetectService(serviceConfig)
	if err != nil {
		c.spamdetectErr = fmt.Errorf("creating spam detection service: %w", err)
		l.Error("Failed to create spam detection service", logger_domain.Error(c.spamdetectErr))
		return
	}

	for name, detector := range c.spamdetectDetectors {
		if err := service.RegisterDetector(ctx, name, detector); err != nil {
			c.spamdetectErr = fmt.Errorf("registering spam detection detector %q: %w", name, err)
			l.Error("Failed to register spam detection detector",
				logger_domain.String("detector", name),
				logger_domain.Error(c.spamdetectErr),
			)
			return
		}
		l.Internal("Registered spam detection detector", logger_domain.String("detector", name))
	}

	if c.spamdetectFeedbackStore != nil {
		service.SetFeedbackStore(c.spamdetectFeedbackStore)
	}

	c.spamdetectService = service
	registerCloseableForShutdown(c.GetAppContext(), "SpamDetectService", service)

	l.Internal("Spam detection service created",
		logger_domain.Int("detectors", len(c.spamdetectDetectors)))
}

// buildSpamDetectServiceFromConfig creates the service with built-in detectors from config.
func (c *Container) buildSpamDetectServiceFromConfig(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)

	serviceConfig := c.spamDetectServiceConfig()
	service, err := spamdetect_domain.NewSpamDetectService(serviceConfig)
	if err != nil {
		c.spamdetectErr = fmt.Errorf("creating spam detection service: %w", err)
		l.Error("Failed to create spam detection service", logger_domain.Error(c.spamdetectErr))
		return
	}

	blocklistPatterns := c.config.ServerConfig.Security.SpamDetectBlocklistPatterns

	err = builtin_detectors.RegisterDefaults(ctx, service, builtin_detectors.Config{
		BlocklistPatterns: blocklistPatterns,
	})
	if err != nil {
		c.spamdetectErr = fmt.Errorf("registering built-in detectors: %w", err)
		l.Error("Failed to register built-in detectors", logger_domain.Error(c.spamdetectErr))
		return
	}

	if c.spamdetectFeedbackStore != nil {
		service.SetFeedbackStore(c.spamdetectFeedbackStore)
	}

	c.spamdetectService = service
	registerCloseableForShutdown(c.GetAppContext(), "SpamDetectService", service)

	l.Internal("Spam detection service created with built-in detectors")
}

// spamDetectServiceConfig builds the service config from server config
// values.
//
// Returns *spamdetect_dto.ServiceConfig which contains the resolved
// service settings.
func (c *Container) spamDetectServiceConfig() *spamdetect_dto.ServiceConfig {
	defaultConfig := spamdetect_dto.DefaultServiceConfig()
	scoreThreshold := deref(c.config.ServerConfig.Security.SpamDetectScoreThreshold, defaultConfig.ScoreThreshold)

	return &spamdetect_dto.ServiceConfig{
		ScoreThreshold: scoreThreshold,
		Timeout:        defaultConfig.Timeout,
	}
}
