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
	"net/http"

	cache_provider_mock "piko.sh/piko/internal/cache/cache_adapters/provider_mock"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/internal/image/image_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_adapters"
	"piko.sh/piko/internal/premailer"
	"piko.sh/piko/internal/registry/registry_adapters"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/shutdown"
	storage_mock "piko.sh/piko/internal/storage/storage_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/wdk/events/events_provider_gochannel"
	"piko.sh/piko/wdk/media/image_provider_mock"
)

// mockProviderName is the name used to register mock providers in tests.
const mockProviderName = "mock"

// mockEmailTemplateService implements EmailTemplateService for testing.
// It returns empty content for tests that need a working email system without
// rendering real templates.
type mockEmailTemplateService struct{}

var _ templater_domain.EmailTemplateService = (*mockEmailTemplateService)(nil)

// Render implements templater_domain.EmailTemplateService.
//
// Returns *templater_dto.RenderedEmailContent which contains simple mock
// content with basic HTML, suitable for testing.
// Returns error which is always nil in this mock.
func (*mockEmailTemplateService) Render(
	_ context.Context,
	_ *http.Request,
	_ string,
	_ any,
	_ *premailer.Options,
	_ bool,
) (*templater_dto.RenderedEmailContent, error) {
	return &templater_dto.RenderedEmailContent{
		HTML:               "<html><body>Mock Email</body></html>",
		PlainText:          "Mock Email",
		CSS:                "",
		AttachmentRequests: nil,
	}, nil
}

// InitialiseForTesting initialises Piko's global services with minimal
// dependencies suitable for unit and integration tests.
//
// This function creates a fully mocked Piko environment with:
//   - In-memory cache provider (no Redis/external cache)
//   - In-memory storage provider (no S3/disk writes)
//   - In-memory registry (no metadata.db SQLite file)
//   - Mock email provider
//   - Mock email template service
//   - Mock image transformer
//   - In-memory event bus (Watermill GoChannel)
//
// This keeps tests fast, isolated, and free of any persistent state.
//
// Takes mockEmailProvider (EmailProviderPort) which provides a mock email
// sending implementation.
// Takes emailProviderName (string) which identifies the email provider for
// logging and debugging.
//
// Returns *Container which provides access to services and cleanup methods.
func InitialiseForTesting(mockEmailProvider email_domain.EmailProviderPort, emailProviderName string) *Container {
	configProvider := config.NewConfigProvider()
	container := NewContainer(configProvider)

	initialiseTestingRegistryService(container)
	initialiseTestingEmailService(container, mockEmailProvider, emailProviderName)
	initialiseTestingCacheService(container)
	initialiseTestingStorageService(container)
	initialiseTestingImageService(container)
	initialiseGlobalServices(container)

	return container
}

// initialiseTestingRegistryService sets up the registry service
// with in-memory mocks for testing.
//
// Takes container (*Container) which receives the mock registry service.
//
// Panics if the GoChannel provider cannot be created or started.
func initialiseTestingRegistryService(container *Container) {
	testCtx, l := logger_domain.From(context.Background(), log)

	mockMetadataStore := registry_adapters.NewMockMetadataStore()
	mockBlobStore := registry_adapters.NewMockBlobStore()
	mockMetadataCache := registry_adapters.NewNoOpMetadataCache()

	goChannelConfig := events_provider_gochannel.DefaultConfig()
	goChannelProvider, err := events_provider_gochannel.NewGoChannelProvider(goChannelConfig)
	if err != nil {
		l.Error("Failed to create GoChannel provider for testing", logger_domain.Error(err))
		panic("piko: failed to create events provider for testing: " + err.Error())
	}

	if err := goChannelProvider.Start(testCtx); err != nil {
		l.Error("Failed to start GoChannel provider for testing", logger_domain.Error(err))
		panic("piko: failed to start events provider for testing: " + err.Error())
	}

	mockEventBus := orchestrator_adapters.NewWatermillEventBus(
		goChannelProvider.Publisher(),
		goChannelProvider.Subscriber(),
		goChannelProvider.Router(),
	)

	shutdown.Register(testCtx, "TestingEventsProvider", func(_ context.Context) error {
		return goChannelProvider.Close()
	})

	blobStores := map[string]registry_domain.BlobStore{
		mockProviderName: mockBlobStore,
	}

	mockRegistryService := registry_domain.NewRegistryService(
		mockMetadataStore,
		blobStores,
		mockEventBus,
		mockMetadataCache,
	)
	container.registryServiceOverride = mockRegistryService
}

// initialiseTestingEmailService sets up the email service with a mock provider for
// testing.
//
// Takes container (*Container) which holds the service dependencies.
// Takes mockEmailProvider (EmailProviderPort) which provides the mock email
// sender.
// Takes emailProviderName (string) which names the provider.
func initialiseTestingEmailService(container *Container, mockEmailProvider email_domain.EmailProviderPort, emailProviderName string) {
	if mockEmailProvider != nil {
		container.AddEmailProvider(emailProviderName, mockEmailProvider)
		container.SetEmailDefaultProvider(emailProviderName)
	}
	container.SetEmailTemplateService(&mockEmailTemplateService{})
}

// initialiseTestingCacheService sets up the cache service with a mock provider for
// testing.
//
// Takes container (*Container) which holds the application services.
func initialiseTestingCacheService(container *Container) {
	ctx, l := logger_domain.From(context.Background(), log)

	cacheService := cache_domain.NewService(mockProviderName)
	mockCacheProvider := cache_provider_mock.NewMockProvider()
	if err := cacheService.RegisterProvider(ctx, mockProviderName, mockCacheProvider); err != nil {
		l.Error("Failed to register mock cache provider during testing initialisation", logger_domain.Error(err))
		_ = mockCacheProvider.Close()
		_ = cacheService.Close(ctx)
		return
	}

	container.SetCacheService(cacheService)
	shutdown.Register(ctx, "TestingCacheService", func(ctx context.Context) error {
		return cacheService.Close(ctx)
	})
}

// initialiseTestingStorageService sets up the storage service with a mock provider
// that stores data in memory.
//
// Takes container (*Container) which holds the service dependencies.
func initialiseTestingStorageService(container *Container) {
	mockStorageProvider := storage_mock.NewMockStorageProvider()
	container.AddStorageProvider(storage_dto.StorageProviderDefault, mockStorageProvider)
	container.SetStorageDefaultProvider(storage_dto.StorageProviderDefault)
}

// initialiseTestingImageService sets up the image service with a mock transformer
// for use in tests.
//
// Takes container (*Container) which holds the dependencies to set up.
func initialiseTestingImageService(container *Container) {
	_, l := logger_domain.From(context.Background(), log)

	mockImageTransformer := image_provider_mock.NewProvider()
	transformers := map[string]image_domain.TransformerPort{
		image_dto.ImageNameDefault: mockImageTransformer,
		mockProviderName:           mockImageTransformer,
	}
	mockImageService, err := image_domain.NewService(transformers, image_dto.ImageNameDefault, image_domain.DefaultServiceConfig())
	if err != nil {
		l.Error("Failed to create mock image service during testing initialisation", logger_domain.Error(err))
		return
	}
	container.SetImageService(mockImageService)
}
