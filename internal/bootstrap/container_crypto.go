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

// This file contains crypto service related container methods.

import (
	"context"
	"fmt"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/crypto/crypto_adapters"
	"piko.sh/piko/internal/crypto/crypto_adapters/local_aes_gcm"
	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/crypto/crypto_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// AddCryptoProvider registers a named encryption provider for cryptographic
// operations.
//
// If the provider implements a shutdown interface (Close, Shutdown, or Stop),
// it will be automatically registered for graceful shutdown.
//
// Takes name (string) which identifies the provider for later retrieval.
// Takes provider (EncryptionProvider) which handles encryption and decryption.
func (c *Container) AddCryptoProvider(name string, provider crypto_domain.EncryptionProvider) {
	if c.cryptoProviders == nil {
		c.cryptoProviders = make(map[string]crypto_domain.EncryptionProvider)
	}
	c.cryptoProviders[name] = provider
	registerCloseableForShutdown(c.GetAppContext(), "CryptoProvider-"+name, provider)
}

// SetCryptoDefaultProvider sets the default encryption provider to use when
// none is specified.
//
// Takes name (string) which identifies the provider to use as the default.
func (c *Container) SetCryptoDefaultProvider(name string) {
	c.cryptoDefaultProvider = name
}

// GetCryptoService returns the crypto service, initialising a default one if
// none was provided.
//
// Returns crypto_domain.CryptoServicePort which provides cryptographic
// operations.
// Returns error when the default crypto service fails to initialise.
func (c *Container) GetCryptoService() (crypto_domain.CryptoServicePort, error) {
	c.cryptoOnce.Do(func() {
		c.createDefaultCryptoService()
	})
	return c.cryptoService, c.cryptoErr
}

// createDefaultCryptoService sets up the crypto service using default settings.
func (c *Container) createDefaultCryptoService() {
	ctx := c.GetAppContext()
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Creating default CryptoService...")

	securityConfig := c.config.ServerConfig.Security

	providerName, baseProvider, activeKeyID, err := c.selectCryptoProvider(ctx, &securityConfig)
	if err != nil {
		c.cryptoErr = err
		l.Error("Failed to create crypto provider", logger_domain.Error(c.cryptoErr))
		return
	}

	if providerName == "disabled" {
		l.Internal("Encryption not configured; crypto service disabled. Set security.encryptionKey to enable.")
		c.cryptoService = crypto_domain.NewDisabledCryptoService()
		return
	}

	cacheService, cacheServiceErr := c.GetCacheService()
	if cacheServiceErr != nil {
		l.Warn("Cache service not available for crypto service; data key caching disabled",
			logger_domain.Error(cacheServiceErr))
		cacheService = nil
	}

	cryptoService, err := c.buildCryptoService(ctx, baseProvider, cacheService, activeKeyID, &securityConfig)
	if err != nil {
		c.cryptoErr = err
		l.Error("Failed to create crypto service", logger_domain.Error(c.cryptoErr))
		return
	}
	c.cryptoService = cryptoService

	l.Internal("Crypto service created",
		logger_domain.String("provider", providerName),
		logger_domain.String("active_key", activeKeyID),
		logger_domain.Bool("data_key_caching_enabled", cacheService != nil && deref(securityConfig.DataKeyCacheTTL, 0) > 0))
}

// selectCryptoProvider selects the appropriate crypto provider based on options
// or config.
//
// Takes ctx (context.Context) which carries deadlines and cancellation
// signals.
// Takes securityConfig (*config.SecurityConfig) which provides the security
// settings including the provider type and encryption key.
//
// Returns string which is the selected provider name.
// Returns crypto_domain.EncryptionProvider which provides the
// encryption operations.
// Returns string which is the active key identifier.
// Returns error when the provider cannot be selected or created.
func (c *Container) selectCryptoProvider(ctx context.Context, securityConfig *config.SecurityConfig) (providerName string, provider crypto_domain.EncryptionProvider, activeKeyID string, err error) {
	ctx, l := logger_domain.From(ctx, log)
	if len(c.cryptoProviders) > 0 {
		l.Internal("Using crypto provider registered via options")

		if c.cryptoDefaultProvider != "" {
			providerName = c.cryptoDefaultProvider
			provider = c.cryptoProviders[providerName]
			if provider == nil {
				return "", nil, "", fmt.Errorf("crypto default provider %q not registered", providerName)
			}
		} else {
			for n, p := range c.cryptoProviders {
				providerName, provider = n, p
				break
			}
		}

		activeKeyID = "default"

		l.Internal("Crypto provider selected from options",
			logger_domain.String("provider", providerName))
		return providerName, provider, activeKeyID, nil
	}

	l.Internal("No crypto providers registered via options; creating from config")
	return c.createProviderFromConfig(ctx, securityConfig)
}

// createProviderFromConfig creates a crypto provider based on config
// settings. This fallback only supports the local_aes_gcm provider
// for simple config-based initialisation.
//
// For cloud providers (AWS KMS, GCP KMS) or custom providers, use
// the option-based approach:
// import "piko.sh/piko/wdk/crypto/crypto_provider_aws_kms"
// provider, _ := crypto_provider_aws_kms.NewAWSKMSProvider(ctx, config)
// server := piko.New(
//
//	piko.WithCryptoProvider("aws_kms", provider),
//	piko.WithDefaultCryptoProvider("aws_kms"),
//
// )
//
// Takes securityConfig (*config.SecurityConfig) which provides the security
// settings including the provider type and encryption key.
//
// Returns string which is the selected provider name.
// Returns crypto_domain.EncryptionProvider which provides the
// encryption operations.
// Returns string which is the active key identifier.
// Returns error when the provider type is unsupported or creation
// fails.
func (c *Container) createProviderFromConfig(_ context.Context, securityConfig *config.SecurityConfig) (providerName string, provider crypto_domain.EncryptionProvider, activeKeyID string, err error) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	providerType := deref(securityConfig.CryptoProvider, "local_aes_gcm")

	l.Internal("Creating crypto provider", logger_domain.String("provider", providerType))

	switch providerType {
	case "local_aes_gcm":
		return c.createLocalAESGCMProvider(securityConfig)

	case "aws_kms", "gcp_kms":
		return "", nil, "", c.cloudProviderConfigError(providerType)

	default:
		return "", nil, "", fmt.Errorf(
			"unknown crypto provider '%s'.\n\n"+
				"Supported config provider: local_aes_gcm\n"+
				"For cloud providers (aws_kms, gcp_kms) or custom providers, use the option-based approach.\n\n"+
				"See: crypto/README.md for examples",
			providerType,
		)
	}
}

// createLocalAESGCMProvider creates a local AES-GCM provider from config.
//
// Takes securityConfig (*config.SecurityConfig) which provides the encryption key
// for AES-GCM operations.
//
// Returns string which is the provider name ("local_aes_gcm" or
// "disabled").
// Returns crypto_domain.EncryptionProvider which provides the AES-GCM
// encryption operations, or nil when disabled.
// Returns string which is the active key identifier.
// Returns error when the base64 key is invalid or provider creation
// fails.
func (*Container) createLocalAESGCMProvider(securityConfig *config.SecurityConfig) (providerName string, provider crypto_domain.EncryptionProvider, activeKeyID string, err error) {
	encKey := deref(securityConfig.EncryptionKey, "")
	if encKey == "" {
		return "disabled", nil, "", nil
	}

	provider, err = crypto_adapters.CreateProviderFromBase64Key(encKey, "piko-default-key")
	if err != nil {
		return "", nil, "", fmt.Errorf("failed to create local encryption provider: %w", err)
	}

	return "local_aes_gcm", provider, "piko-default-key", nil
}

// cloudProviderConfigError returns an error for cloud providers that cannot
// be set up via a config file.
//
// Takes providerType (string) which identifies the cloud provider type.
//
// Returns error when a cloud provider needs option-based setup.
func (*Container) cloudProviderConfigError(providerType string) error {
	providerName := map[string]string{"aws_kms": "AWSKMS", "gcp_kms": "GCPKMS"}[providerType]
	return fmt.Errorf(
		"crypto provider '%s' cannot be configured via config file.\n\n"+
			"Please use the option-based approach for better type safety and clarity:\n\n"+
			"  import (\n"+
			"      \"piko.sh/piko\"\n"+
			"      \"piko.sh/piko/wdk/crypto/crypto_provider_%s\"\n"+
			"  )\n\n"+
			"  provider, _ := crypto_provider_%s.New%sProvider(ctx, config)\n"+
			"  server := piko.New(\n"+
			"      piko.WithCryptoProvider(\"%s\", provider),\n"+
			"      piko.WithDefaultCryptoProvider(\"%s\"),\n"+
			"  )\n\n"+
			"See: crypto/README.md for detailed examples",
		providerType, providerType, providerType, providerName, providerType, providerType,
	)
}

// buildCryptoService creates the crypto service with the selected provider.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes baseProvider (crypto_domain.EncryptionProvider) which provides the
// underlying encryption operations.
// Takes cacheService (cache_domain.Service) which handles data key caching.
// Takes activeKeyID (string) which identifies the current encryption key.
// Takes securityConfig (*config.SecurityConfig) which specifies
// cache TTL and deprecated key settings.
//
// Returns crypto_domain.CryptoServicePort which is the configured crypto
// service ready for use.
// Returns error when the crypto service cannot be created.
func (*Container) buildCryptoService(
	ctx context.Context,
	baseProvider crypto_domain.EncryptionProvider,
	cacheService cache_domain.Service,
	activeKeyID string,
	securityConfig *config.SecurityConfig,
) (crypto_domain.CryptoServicePort, error) {
	serviceConfig := crypto_dto.DefaultServiceConfig()
	serviceConfig.ActiveKeyID = activeKeyID
	serviceConfig.DeprecatedKeyIDs = securityConfig.DeprecatedKeyIDs

	if ttl := deref(securityConfig.DataKeyCacheTTL, 0); ttl > 0 {
		serviceConfig.DataKeyCacheTTL = ttl
	}
	if maxSize := deref(securityConfig.DataKeyCacheMaxSize, 0); maxSize > 0 {
		serviceConfig.DataKeyCacheMaxSize = maxSize
	}

	localFactory := local_aes_gcm.NewFactory()

	service, err := crypto_domain.NewCryptoService(
		ctx,
		cacheService,
		serviceConfig,
		crypto_domain.WithLocalProviderFactory(localFactory),
	)
	if err != nil {
		return nil, fmt.Errorf("creating crypto service: %w", err)
	}

	providerName := string(baseProvider.Type())
	if err := service.RegisterProvider(ctx, providerName, baseProvider); err != nil {
		return nil, fmt.Errorf("registering crypto provider: %w", err)
	}

	if err := service.SetDefaultProvider(providerName); err != nil {
		return nil, fmt.Errorf("setting default crypto provider: %w", err)
	}

	return service, nil
}

// SetCryptoService sets a pre-configured crypto service on the container.
//
// If the service implements a shutdown interface (Close, Shutdown, or Stop),
// it will be automatically registered for graceful shutdown.
//
// Takes service (crypto_domain.CryptoServicePort) which is the crypto service
// to use. Allows builders to provide their own implementation.
func (c *Container) SetCryptoService(service crypto_domain.CryptoServicePort) {
	c.cryptoService = service
	registerCloseableForShutdown(c.GetAppContext(), "CryptoService", service)
}
