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

// This file contains provider configuration validation for the container.

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// ValidateProviderConfiguration checks that all default provider names match
// registered providers. This finds setup errors early, before services start.
//
// Returns error when a default provider is set but not registered.
func (c *Container) ValidateProviderConfiguration() error {
	var errs []string

	if err := c.validateEmailProviders(); err != "" {
		errs = append(errs, err)
	}
	if err := c.validateStorageProviders(); err != "" {
		errs = append(errs, err)
	}
	if err := c.validateCacheProviders(); err != "" {
		errs = append(errs, err)
	}
	if err := c.validateCryptoProviders(); err != "" {
		errs = append(errs, err)
	}
	if err := c.validateNotificationProviders(); err != "" {
		errs = append(errs, err)
	}
	if err := c.validateImageProviders(); err != "" {
		errs = append(errs, err)
	}
	if err := c.validateVideoProviders(); err != "" {
		errs = append(errs, err)
	}
	if err := c.validateLLMProviders(); err != "" {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("provider configuration errors:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// validateEmailProviders checks the email provider settings.
//
// Returns string which is empty when valid, or an error message when the
// default provider is not found in the list of registered providers.
func (c *Container) validateEmailProviders() string {
	if c.emailDefaultProvider == "" {
		return ""
	}
	if len(c.emailProviders) == 0 {
		return fmt.Sprintf("email default provider %q specified but no providers registered", c.emailDefaultProvider)
	}
	if _, ok := c.emailProviders[c.emailDefaultProvider]; !ok {
		return fmt.Sprintf("email default provider %q not found in registered providers: %s",
			c.emailDefaultProvider, providerNames(c.emailProviders))
	}
	return ""
}

// validateStorageProviders checks the storage provider settings.
//
// Returns string which is empty when valid, or a message describing the
// problem.
func (c *Container) validateStorageProviders() string {
	if c.storageDefaultProvider == "" {
		return ""
	}
	if len(c.storageProviders) == 0 {
		return fmt.Sprintf("storage default provider %q specified but no providers registered", c.storageDefaultProvider)
	}
	if _, ok := c.storageProviders[c.storageDefaultProvider]; !ok {
		return fmt.Sprintf("storage default provider %q not found in registered providers: %s",
			c.storageDefaultProvider, providerNames(c.storageProviders))
	}
	return ""
}

// validateCacheProviders checks the cache provider settings.
//
// Returns string which is empty when valid, or describes the problem found.
func (c *Container) validateCacheProviders() string {
	if c.cacheDefaultProvider == "" {
		return ""
	}
	if len(c.cacheProviders) == 0 {
		return fmt.Sprintf("cache default provider %q specified but no providers registered", c.cacheDefaultProvider)
	}
	if _, ok := c.cacheProviders[c.cacheDefaultProvider]; !ok {
		return fmt.Sprintf("cache default provider %q not found in registered providers: %s",
			c.cacheDefaultProvider, providerNames(c.cacheProviders))
	}
	return ""
}

// validateCryptoProviders checks the crypto provider settings.
//
// Returns string which holds an error message if the settings are not valid,
// or an empty string if all is well.
func (c *Container) validateCryptoProviders() string {
	if c.cryptoDefaultProvider == "" {
		return ""
	}
	if len(c.cryptoProviders) == 0 {
		return fmt.Sprintf("crypto default provider %q specified but no providers registered", c.cryptoDefaultProvider)
	}
	if _, ok := c.cryptoProviders[c.cryptoDefaultProvider]; !ok {
		return fmt.Sprintf("crypto default provider %q not found in registered providers: %s",
			c.cryptoDefaultProvider, providerNames(c.cryptoProviders))
	}
	return ""
}

// validateNotificationProviders checks the notification provider settings.
//
// Returns string which is empty if valid, or an error message if the default
// provider is not found in the list of registered providers.
func (c *Container) validateNotificationProviders() string {
	if c.notificationDefaultProvider == "" {
		return ""
	}
	if len(c.notificationProviders) == 0 {
		return fmt.Sprintf("notification default provider %q specified but no providers registered", c.notificationDefaultProvider)
	}
	if _, ok := c.notificationProviders[c.notificationDefaultProvider]; !ok {
		return fmt.Sprintf("notification default provider %q not found in registered providers: %s",
			c.notificationDefaultProvider, providerNames(c.notificationProviders))
	}
	return ""
}

// validateImageProviders checks the image transformer provider settings.
//
// Returns string which is empty if valid, or an error message describing the
// problem.
func (c *Container) validateImageProviders() string {
	if c.defaultImageTransformer == "" {
		return ""
	}
	if len(c.imageTransformers) == 0 {
		return fmt.Sprintf("image default transformer %q specified but no transformers registered", c.defaultImageTransformer)
	}
	if _, ok := c.imageTransformers[c.defaultImageTransformer]; !ok {
		return fmt.Sprintf("image default transformer %q not found in registered transformers: %s",
			c.defaultImageTransformer, providerNames(c.imageTransformers))
	}
	return ""
}

// validateVideoProviders checks the video transcoder provider settings.
//
// Returns string which is empty if valid, or an error message that describes
// what is wrong with the settings.
func (c *Container) validateVideoProviders() string {
	if c.defaultVideoTranscoder == "" {
		return ""
	}
	if len(c.videoTranscoders) == 0 {
		return fmt.Sprintf("video default transcoder %q specified but no transcoders registered", c.defaultVideoTranscoder)
	}
	if _, ok := c.videoTranscoders[c.defaultVideoTranscoder]; !ok {
		return fmt.Sprintf("video default transcoder %q not found in registered transcoders: %s",
			c.defaultVideoTranscoder, providerNames(c.videoTranscoders))
	}
	return ""
}

// validateLLMProviders checks that the LLM provider settings are correct.
//
// Returns string which is empty when valid, or an error message when the
// default provider is not found among the registered providers.
func (c *Container) validateLLMProviders() string {
	if c.llmDefaultProvider == "" {
		return ""
	}
	if len(c.llmProviders) == 0 {
		return fmt.Sprintf("llm default provider %q specified but no providers registered", c.llmDefaultProvider)
	}
	if _, ok := c.llmProviders[c.llmDefaultProvider]; !ok {
		return fmt.Sprintf("llm default provider %q not found in registered providers: %s",
			c.llmDefaultProvider, providerNames(c.llmProviders))
	}
	return ""
}

// providerNames gets the keys from a map and formats them as a list.
//
// Takes m (map[string]T) which is the map to get keys from.
//
// Returns string which contains the keys joined by commas, or "(none)" if
// the map is empty.
func providerNames[T any](m map[string]T) string {
	if len(m) == 0 {
		return "(none)"
	}
	return strings.Join(slices.Collect(maps.Keys(m)), ", ")
}
