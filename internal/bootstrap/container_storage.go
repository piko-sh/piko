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

// This file contains storage service related container methods.

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/internal/storage/storage_adapters/provider_disk"
	"piko.sh/piko/internal/storage/storage_adapters/provider_fs"
	"piko.sh/piko/internal/storage/storage_adapters/transformer_crypto"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/wdk/safedisk"
)

// cryptoTransformerPriority is the priority level for the crypto transformer.
const cryptoTransformerPriority = 100

// AddStorageProvider registers a named storage provider for file operations.
//
// If the provider implements a shutdown interface (Close, Shutdown, or Stop),
// it will be automatically registered for graceful shutdown.
//
// Takes name (string) which identifies the provider for later retrieval.
// Takes provider (StorageProviderPort) which handles storage operations.
func (c *Container) AddStorageProvider(name string, provider storage_domain.StorageProviderPort) {
	if c.storageProviders == nil {
		c.storageProviders = make(map[string]storage_domain.StorageProviderPort)
	}
	c.storageProviders[name] = provider
	registerCloseableForShutdown(c.GetAppContext(), "StorageProvider-"+name, provider)
}

// SetStorageDefaultProvider sets the default storage provider to use when
// none is specified.
//
// Takes name (string) which is the provider name to use as the default.
func (c *Container) SetStorageDefaultProvider(name string) {
	c.storageDefaultProvider = name
}

// SetStorageDispatcherConfig configures the storage dispatcher for async file
// operations.
//
// Takes dispatcherConfig (*storage_domain.DispatcherConfig)
// which specifies the dispatcher settings.
func (c *Container) SetStorageDispatcherConfig(dispatcherConfig *storage_domain.DispatcherConfig) {
	c.storageDispatcherConfig = dispatcherConfig
	c.hasStorageDispatcher = true
}

// SetStoragePresignBaseURL sets the base URL for presigned storage URLs, which
// is essential for headless CMS scenarios where the frontend is on a different
// host than the storage service.
//
// Takes baseURL (string) which is the full base URL including scheme and host,
// e.g., "http://localhost:8080" or "https://cms.example.com".
func (c *Container) SetStoragePresignBaseURL(baseURL string) {
	c.storagePresignBaseURL = baseURL
}

// SetStoragePublicBaseURL sets the base URL for public storage URLs, causing
// them to be generated as absolute URLs (e.g.,
// "http://localhost:8080/_piko/storage/public/...") instead of relative paths
// when the website and CMS/API run on different ports or hosts.
//
// Takes baseURL (string) which is the full base URL including scheme and host,
// e.g., "http://localhost:8080" or "https://cms.example.com".
func (c *Container) SetStoragePublicBaseURL(baseURL string) {
	c.storagePublicBaseURL = baseURL
}

// SetStorageService allows users to provide a custom storage service
// implementation.
//
// If the service implements a shutdown interface (Close, Shutdown, or Stop),
// it will be automatically registered for graceful shutdown.
//
// Takes service (storage_domain.Service) which is the custom storage service to
// use instead of the default.
func (c *Container) SetStorageService(service storage_domain.Service) {
	c.storageServiceOverride = service
	c.storageService = service
	registerCloseableForShutdown(c.GetAppContext(), "StorageService", service)
}

// GetStorageService returns the storage service, initialising a default one
// if none was provided.
//
// Returns storage_domain.Service which provides storage operations.
// Returns error when the default storage service cannot be created.
func (c *Container) GetStorageService() (storage_domain.Service, error) {
	c.storageOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.storageServiceOverride != nil {
			l.Internal("Using provided StorageService override.")
			c.storageService = c.storageServiceOverride
			return
		}
		c.createDefaultStorageService()
	})
	return c.storageService, c.storageErr
}

// createDefaultStorageService sets up the default storage service for the
// container. It selects a storage provider, creates a dispatcher, and
// registers the service for shutdown.
func (c *Container) createDefaultStorageService() {
	ctx := c.GetAppContext()
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Creating default StorageService...")

	baseName, baseProvider, err := c.selectStorageBaseProvider()
	if err != nil {
		c.storageErr = err
		return
	}

	dispatcher, err := c.createStorageDispatcher(ctx, baseProvider, baseName)
	if err != nil {
		c.storageErr = err
		return
	}

	serviceOpts := c.buildStorageServiceOpts()
	s := storage_domain.NewService(c.GetAppContext(), serviceOpts...)

	if err := c.registerStorageProviders(s, baseName, baseProvider); err != nil {
		c.storageErr = err
		return
	}

	if err := c.startStorageDispatcher(ctx, s, dispatcher); err != nil {
		c.storageErr = err
		return
	}

	c.registerStorageCryptoTransformer(s)

	shutdown.Register(ctx, "StorageService", func(shutdownCtx context.Context) error {
		return s.Close(shutdownCtx)
	})

	c.storageService = s
}

// buildStorageServiceOpts builds the storage service options from the
// container's presign and public base URL settings.
//
// Returns []storage_domain.ServiceOption which contains the resolved options.
func (c *Container) buildStorageServiceOpts() []storage_domain.ServiceOption {
	_, l := logger_domain.From(c.GetAppContext(), log)
	var opts []storage_domain.ServiceOption

	if c.embeddedPikoFS == nil {
		tempSandbox, err := c.createSandbox("storage-temp", filepath.Join(deref(c.config.ServerConfig.Paths.BaseDir, "."), ".piko", "tmp"), safedisk.ModeReadWrite)
		if err != nil {
			l.Warn("Failed to create storage temp sandbox, using fallback",
				logger_domain.Error(err))
		} else {
			opts = append(opts, storage_domain.WithTempSandbox(tempSandbox))
		}
	}

	presignBaseURL := c.storagePresignBaseURL
	if presignBaseURL == "" && c.config != nil {
		if configURL := deref(c.config.ServerConfig.Storage.Presign.BaseURL, ""); configURL != "" {
			presignBaseURL = configURL
		}
	}
	if presignBaseURL != "" {
		opts = append(opts, storage_domain.WithPresignFallbackBaseURL(presignBaseURL))
	}

	publicBaseURL := c.storagePublicBaseURL
	if publicBaseURL == "" && c.config != nil {
		if configURL := deref(c.config.ServerConfig.Storage.PublicBaseURL, ""); configURL != "" {
			publicBaseURL = configURL
		}
	}
	if publicBaseURL != "" {
		opts = append(opts, storage_domain.WithPublicFallbackBaseURL(publicBaseURL))
	}

	return opts
}

// startStorageDispatcher registers and starts the storage dispatcher if one was
// configured.
//
// Takes ctx (context.Context) which carries the application context.
// Takes s (storage_domain.Service) which is the storage service.
// Takes dispatcher (storage_domain.StorageDispatcherPort) which is the
// dispatcher to start, or nil if none configured.
//
// Returns error when registration or startup fails.
func (*Container) startStorageDispatcher(ctx context.Context, s storage_domain.Service, dispatcher storage_domain.StorageDispatcherPort) error {
	if dispatcher == nil {
		return nil
	}
	if err := s.RegisterDispatcher(ctx, dispatcher); err != nil {
		return fmt.Errorf("failed to register and start storage dispatcher: %w", err)
	}
	if err := dispatcher.Start(ctx); err != nil {
		return fmt.Errorf("failed to start storage dispatcher: %w", err)
	}
	return nil
}

// selectStorageBaseProvider selects the base storage provider based on
// configuration.
//
// Returns string which is the name of the selected provider.
// Returns storage_domain.StorageProviderPort which is the selected
// storage provider.
// Returns error when the configured provider is not registered or the
// default disk provider fails to initialise.
func (c *Container) selectStorageBaseProvider() (string, storage_domain.StorageProviderPort, error) {
	if len(c.storageProviders) > 0 {
		return c.selectExplicitStorageProvider()
	}
	if c.embeddedPikoFS != nil {
		return c.selectEmbeddedStorageProvider()
	}
	return c.selectDiskStorageProvider()
}

// selectExplicitStorageProvider picks a user-registered storage provider.
func (c *Container) selectExplicitStorageProvider() (string, storage_domain.StorageProviderPort, error) {
	if c.storageDefaultProvider != "" {
		provider := c.storageProviders[c.storageDefaultProvider]
		if provider == nil {
			return "", nil, fmt.Errorf("storage default provider %q not registered", c.storageDefaultProvider)
		}
		return c.storageDefaultProvider, provider, nil
	}
	if provider, ok := c.storageProviders[storage_dto.StorageProviderDefault]; ok {
		return storage_dto.StorageProviderDefault, provider, nil
	}
	for name, provider := range c.storageProviders {
		return name, provider, nil
	}
	return "", nil, errors.New("no storage providers registered")
}

// selectEmbeddedStorageProvider creates a read-only provider from the
// embedded .piko filesystem.
func (c *Container) selectEmbeddedStorageProvider() (string, storage_domain.StorageProviderPort, error) {
	storageSubFS, subErr := fs.Sub(c.embeddedPikoFS, "storage")
	if subErr != nil {
		return "", nil, fmt.Errorf("failed to create storage sub-fs: %w", subErr)
	}
	provider, err := provider_fs.NewFSProvider(storageSubFS)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create embedded fs storage provider: %w", err)
	}
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Using embedded fs.FS storage provider (embedded mode)")
	return storage_dto.StorageProviderDefault, provider, nil
}

// selectDiskStorageProvider creates the default disk-backed storage provider.
func (c *Container) selectDiskStorageProvider() (string, storage_domain.StorageProviderPort, error) {
	storageDir := filepath.Join(deref(c.config.ServerConfig.Paths.BaseDir, "."), config.PikoInternalPath, "storage")
	storageSandbox, sandboxErr := c.createSandbox("storage-disk-provider", storageDir, safedisk.ModeReadWrite)
	if sandboxErr != nil {
		return "", nil, fmt.Errorf("failed to create storage sandbox: %w", sandboxErr)
	}
	provider, err := provider_disk.NewDiskProvider(provider_disk.Config{
		BaseDirectory: storageDir,
		Sandbox:       storageSandbox,
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to initialise default disk storage provider: %w", err)
	}
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Using default disk storage provider (no custom providers registered)",
		logger_domain.String("storage_dir", storageDir))
	return storage_dto.StorageProviderDefault, provider, nil
}

// createStorageDispatcher creates a storage dispatcher if one is set up.
//
// Takes baseProvider (storage_domain.StorageProviderPort) which provides the
// underlying storage operations.
// Takes baseName (string) which specifies the base name for the dispatcher.
//
// Returns storage_domain.StorageDispatcherPort which is the set up dispatcher,
// or nil if no dispatcher is set up.
// Returns error when dispatcher creation fails.
func (c *Container) createStorageDispatcher(_ context.Context, baseProvider storage_domain.StorageProviderPort, baseName string) (storage_domain.StorageDispatcherPort, error) {
	if !c.hasStorageDispatcher {
		return nil, nil
	}

	dispatcherConfig := c.storageDispatcherConfig
	if dispatcherConfig == nil {
		dispatcherConfig = new(storage_domain.DefaultDispatcherConfig())
	}
	return storage_domain.NewStorageDispatcher(baseProvider, baseName, *dispatcherConfig), nil
}

// registerStorageProviders registers all storage providers with the service.
//
// Takes s (storage_domain.Service) which receives the provider registrations.
// Takes baseName (string) which identifies the primary storage provider.
// Takes baseProvider (storage_domain.StorageProviderPort) which is the primary
// provider to register.
//
// Returns error when registration fails or the default provider cannot be set.
func (c *Container) registerStorageProviders(s storage_domain.Service, baseName string, baseProvider storage_domain.StorageProviderPort) error {
	ctx := c.GetAppContext()
	if err := s.RegisterProvider(ctx, baseName, baseProvider); err != nil {
		return fmt.Errorf("failed to register base storage provider %q: %w", baseName, err)
	}

	for name, provider := range c.storageProviders {
		if name != baseName {
			if err := s.RegisterProvider(ctx, name, provider); err != nil {
				return fmt.Errorf("failed to register storage provider %q: %w", name, err)
			}
		}
	}

	if c.storageDefaultProvider != "" {
		if err := s.SetDefaultProvider(c.storageDefaultProvider); err != nil {
			return fmt.Errorf("failed to set default storage provider to %q: %w", c.storageDefaultProvider, err)
		}
	} else {
		if err := s.SetDefaultProvider(baseName); err != nil {
			return fmt.Errorf("failed to set default storage provider to %q: %w", baseName, err)
		}
	}

	return nil
}

// registerStorageCryptoTransformer registers the crypto transformer with the
// given storage service.
//
// Takes s (storage_domain.Service) which is the storage service to register
// the transformer with.
func (c *Container) registerStorageCryptoTransformer(s storage_domain.Service) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	cryptoService, err := c.GetCryptoService()
	if err == nil && cryptoService != nil {
		cryptoTransformer := transformer_crypto.New(cryptoService, "crypto-service", cryptoTransformerPriority)
		if err := s.RegisterTransformer(c.GetAppContext(), cryptoTransformer); err != nil {
			l.Warn("Failed to register crypto service transformer for storage", logger_domain.Error(err))
		} else {
			l.Internal("Crypto service transformer registered for storage")
		}
	}
}
