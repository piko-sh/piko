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
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ThreeDotsLabs/watermill/message"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/events/events_domain"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/internal/image/image_dto"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/storage/storage_domain"
)

var (
	// errNotInitialised is returned when a service is accessed before the framework
	// has been initialised via piko.New().
	errNotInitialised = errors.New("piko: service accessed before framework initialisation; " +
		"ensure piko.New() has been called and the server is running")

	// globalContainer stores a reference to the initialised container.
	//
	// It is set by initialiseGlobalServices after piko.New() and subsequent
	// initialisation. Access is coordinated via atomic operations to ensure
	// memory visibility across goroutines.
	globalContainer atomic.Pointer[Container]

	// initialiseOnce guards single execution of initialiseGlobalServices,
	// even if called concurrently from multiple goroutines.
	initialiseOnce sync.Once
)

// GetEmailService returns the global email service instance.
//
// Returns email_domain.Service which is the configured email service.
// Returns error when the framework is not initialised or the service
// cannot be created.
func GetEmailService() (email_domain.Service, error) {
	container, err := getContainer()
	if err != nil {
		return nil, fmt.Errorf("get email service: %w", err)
	}
	return container.GetEmailService()
}

// GetI18nService returns the global internationalisation service instance. This
// function is concurrency-safe and can be called from multiple goroutines.
//
// Returns i18n_domain.Service which is the configured i18n service.
// Returns error when the framework is not initialised or the service cannot be
// created.
func GetI18nService() (i18n_domain.Service, error) {
	container, err := getContainer()
	if err != nil {
		return nil, fmt.Errorf("get i18n service: %w", err)
	}
	return container.GetI18nService()
}

// GetStorageService returns the global storage service instance.
//
// Returns storage_domain.Service which is the configured storage service.
// Returns error when the framework is not initialised or the service
// cannot be created.
func GetStorageService() (storage_domain.Service, error) {
	container, err := getContainer()
	if err != nil {
		return nil, fmt.Errorf("get storage service: %w", err)
	}
	return container.GetStorageService()
}

// GetCacheService returns the global cache service instance.
//
// Returns cache_domain.Service which is the configured cache service.
// Returns error when the framework is not initialised or the service
// cannot be created.
func GetCacheService() (cache_domain.Service, error) {
	container, err := getContainer()
	if err != nil {
		return nil, fmt.Errorf("get cache service: %w", err)
	}
	return container.GetCacheService()
}

// GetCryptoService returns the global crypto service instance.
// It is safe to call from multiple goroutines.
//
// Returns crypto_domain.CryptoServicePort which is the configured crypto
// service.
// Returns error when the framework is not initialised or the service
// cannot be created.
func GetCryptoService() (crypto_domain.CryptoServicePort, error) {
	container, err := getContainer()
	if err != nil {
		return nil, fmt.Errorf("get crypto service: %w", err)
	}
	return container.GetCryptoService()
}

// GetCaptchaService returns the global captcha service instance.
// It is safe to call from multiple goroutines.
//
// Returns captcha_domain.CaptchaServicePort which is the configured captcha
// service.
// Returns error when the framework is not initialised or the service
// cannot be created.
func GetCaptchaService() (captcha_domain.CaptchaServicePort, error) {
	container, err := getContainer()
	if err != nil {
		return nil, fmt.Errorf("get captcha service: %w", err)
	}
	return container.GetCaptchaService()
}

// GetLLMService returns the global LLM service instance.
//
// Returns llm_domain.Service which is the configured LLM service.
// Returns error when the framework is not initialised or the service
// cannot be created.
func GetLLMService() (llm_domain.Service, error) {
	container, err := getContainer()
	if err != nil {
		return nil, fmt.Errorf("get LLM service: %w", err)
	}
	return container.GetLLMService()
}

// GetSearchService returns the global search service instance.
//
// Returns collection_domain.SearchServicePort which is the configured
// search service.
// Returns error when the framework is not initialised or the service
// cannot be created.
func GetSearchService() (collection_domain.SearchServicePort, error) {
	container, err := getContainer()
	if err != nil {
		return nil, fmt.Errorf("get search service: %w", err)
	}
	return container.GetSearchService()
}

// GetImageService returns the global image service instance.
//
// Returns image_domain.Service which is the configured image service.
// Returns error when the framework is not initialised or the service
// cannot be created.
func GetImageService() (image_domain.Service, error) {
	container, err := getContainer()
	if err != nil {
		return nil, fmt.Errorf("get image service: %w", err)
	}
	return container.GetImageService()
}

// GetImagePredefinedVariants returns the map of predefined image variants.
// This function is safe to call from multiple goroutines.
//
// Returns map[string]image_dto.TransformationSpec which maps variant names
// to their transformation settings. Returns nil if the framework is not
// initialised or no variants are configured.
func GetImagePredefinedVariants() map[string]image_dto.TransformationSpec {
	container := globalContainer.Load()
	if container == nil {
		return nil
	}
	return container.GetImagePredefinedVariants()
}

// GetEventsProvider returns the global events provider instance.
//
// Returns events_domain.Provider which is the configured events provider.
// Returns error when the framework is not initialised or the provider
// cannot be created.
func GetEventsProvider() (events_domain.Provider, error) {
	container, err := getContainer()
	if err != nil {
		return nil, fmt.Errorf("get events provider: %w", err)
	}
	return container.GetEventsProvider()
}

// GetEventsRouter returns the Watermill Router configured by Piko.
// Users can add their own handlers to this router for custom event
// processing.
//
// Returns *message.Router which is the configured events router.
// Returns error when the framework is not initialised or the provider
// cannot be created.
func GetEventsRouter() (*message.Router, error) {
	provider, err := GetEventsProvider()
	if err != nil {
		return nil, fmt.Errorf("get events router: %w", err)
	}
	return provider.Router(), nil
}

// GetEventsPublisher returns the Watermill Publisher configured by Piko.
// Users can use this publisher to send their own messages.
//
// Returns message.Publisher which is the configured events publisher.
// Returns error when the framework is not initialised or the provider
// cannot be created.
func GetEventsPublisher() (message.Publisher, error) {
	provider, err := GetEventsProvider()
	if err != nil {
		return nil, fmt.Errorf("get events publisher: %w", err)
	}
	return provider.Publisher(), nil
}

// GetEventsSubscriber returns the Watermill Subscriber set up by Piko.
// Users can use this to create their own subscriptions.
//
// Returns message.Subscriber which is the configured events subscriber.
// Returns error when the framework is not set up or the provider cannot be
// created.
func GetEventsSubscriber() (message.Subscriber, error) {
	provider, err := GetEventsProvider()
	if err != nil {
		return nil, fmt.Errorf("get events subscriber: %w", err)
	}
	return provider.Subscriber(), nil
}

// IsEventsRunning reports whether the events router has been started and is
// running. Use this to check if the events system is ready before adding
// handlers.
//
// Returns bool which is true if the events router is running, false otherwise.
func IsEventsRunning() bool {
	container, err := getContainer()
	if err != nil {
		return false
	}

	provider, err := container.GetEventsProvider()
	if err != nil {
		return false
	}

	return provider.Running()
}

// GetPdfWriterService returns the global PDF writer service instance.
//
// Returns pdfwriter_domain.PdfWriterService which is the configured PDF writer
// service.
// Returns error when the framework is not initialised or the service
// is not available.
func GetPdfWriterService() (pdfwriter_domain.PdfWriterService, error) {
	container, err := getContainer()
	if err != nil {
		return nil, fmt.Errorf("get pdf writer service: %w", err)
	}
	svc := container.GetPdfWriterService()
	if svc == nil {
		return nil, errors.New("pdf writer service not initialised")
	}
	return svc, nil
}

// GetNotificationService returns the global notification service instance. This
// function is concurrency-safe and can be called from multiple goroutines.
//
// Returns notification_domain.Service which is the configured notification
// service.
// Returns error when the framework is not initialised or the service cannot be
// created.
func GetNotificationService() (notification_domain.Service, error) {
	container, err := getContainer()
	if err != nil {
		return nil, fmt.Errorf("get notification service: %w", err)
	}
	return container.GetNotificationService()
}

// GetDatabaseConnection returns the *sql.DB for a named database registered
// via AddDatabase during bootstrap. This function is concurrency-safe.
//
// Takes name (string) which identifies the database.
//
// Returns *sql.DB which is the open database connection.
// Returns error when the framework is not initialised or the database is not
// registered.
func GetDatabaseConnection(name string) (*sql.DB, error) {
	container, err := getContainer()
	if err != nil {
		return nil, fmt.Errorf("get database connection: %w", err)
	}
	return container.GetDatabaseConnection(name)
}

// GetDatabaseReader returns the DBTX for reading from a named database
// registered via AddDatabase during bootstrap, where replicas are balanced via
// round-robin when configured or the primary connection is returned otherwise.
//
// Takes name (string) which identifies the database.
//
// Returns DBTX which can execute read queries.
// Returns error when the framework is not initialised or the database is not
// registered.
func GetDatabaseReader(name string) (DBTX, error) {
	container, containerError := getContainer()
	if containerError != nil {
		return nil, fmt.Errorf("get database reader: %w", containerError)
	}
	return container.GetDatabaseReader(name)
}

// GetDatabaseWriter returns the DBTX for writing to a named database
// registered via AddDatabase during bootstrap. When EnableOTel is set on the
// registration, the returned DBTX is wrapped with OpenTelemetry spans and
// metrics.
//
// Takes name (string) which identifies the database.
//
// Returns DBTX which can execute write queries.
// Returns error when the framework is not initialised or the database is not
// registered.
func GetDatabaseWriter(name string) (DBTX, error) {
	container, containerError := getContainer()
	if containerError != nil {
		return nil, fmt.Errorf("get database writer: %w", containerError)
	}
	return container.GetDatabaseWriter(name)
}

// GetMigrationService returns the migration service for a named database
// registered via AddDatabase during bootstrap. This function is
// concurrency-safe.
//
// Takes name (string) which identifies the database.
//
// Returns querier_domain.MigrationServicePort which can apply and roll back
// migrations.
// Returns error when the framework is not initialised, the database is not
// registered, or no migration filesystem was configured.
func GetMigrationService(name string) (querier_domain.MigrationServicePort, error) {
	container, err := getContainer()
	if err != nil {
		return nil, fmt.Errorf("get migration service: %w", err)
	}
	return container.GetQuerierMigrationService(name)
}

// GetSeedService returns the seed service for a named database registered
// during bootstrap. This function is concurrency-safe and can be called from
// multiple goroutines.
//
// Takes name (string) which identifies the database.
//
// Returns querier_domain.SeedServicePort which can apply and inspect seeds.
// Returns error when the framework is not initialised, the database is not
// registered, or no seed filesystem was configured.
func GetSeedService(name string) (querier_domain.SeedServicePort, error) {
	container, err := getContainer()
	if err != nil {
		return nil, fmt.Errorf("get seed service: %w", err)
	}
	return container.GetQuerierSeedService(name)
}

// initialiseGlobalServices stores the container reference for global service
// access.
//
// This is called after the container is fully set up. It is safe to call this
// function more than once; only the first call will have any effect.
//
// Takes container (*Container) which is the fully set up service container to
// store for global access.
func initialiseGlobalServices(container *Container) {
	initialiseOnce.Do(func() {
		globalContainer.Store(container)
	})
}

// getContainer returns the global container.
//
// Returns *Container which holds the initialised service dependencies.
// Returns error when the container has not been initialised.
func getContainer() (*Container, error) {
	container := globalContainer.Load()
	if container == nil {
		return nil, errNotInitialised
	}
	return container, nil
}
