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
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/events/events_domain"
	"piko.sh/piko/internal/image/image_dto"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
)

type servicesStubEventsProvider struct {
	events_domain.Provider
	running bool
}

type servicesStubPdfWriter struct {
	pdfwriter_domain.PdfWriterService
}

func (*servicesStubEventsProvider) Router() *message.Router { return nil }

func (*servicesStubEventsProvider) Publisher() message.Publisher { return nil }

func (*servicesStubEventsProvider) Subscriber() message.Subscriber { return nil }

func (p *servicesStubEventsProvider) Running() bool { return p.running }

func installGlobalContainer(t *testing.T, container *Container) {
	t.Helper()

	previous := globalContainer.Swap(container)
	t.Cleanup(func() { globalContainer.Store(previous) })
}

func TestGlobalServiceAccessors_FailBeforeInitialisation(t *testing.T) {
	installGlobalContainer(t, nil)

	accessors := map[string]func() error{
		"email":        func() error { _, err := GetEmailService(); return err },
		"i18n":         func() error { _, err := GetI18nService(); return err },
		"storage":      func() error { _, err := GetStorageService(); return err },
		"cache":        func() error { _, err := GetCacheService(); return err },
		"crypto":       func() error { _, err := GetCryptoService(); return err },
		"captcha":      func() error { _, err := GetCaptchaService(); return err },
		"spam":         func() error { _, err := GetSpamDetectService(); return err },
		"llm":          func() error { _, err := GetLLMService(); return err },
		"search":       func() error { _, err := GetSearchService(); return err },
		"image":        func() error { _, err := GetImageService(); return err },
		"events":       func() error { _, err := GetEventsProvider(); return err },
		"router":       func() error { _, err := GetEventsRouter(); return err },
		"publisher":    func() error { _, err := GetEventsPublisher(); return err },
		"subscriber":   func() error { _, err := GetEventsSubscriber(); return err },
		"pdf":          func() error { _, err := GetPdfWriterService(); return err },
		"notification": func() error { _, err := GetNotificationService(); return err },
		"connection":   func() error { _, err := GetDatabaseConnection("primary"); return err },
		"reader":       func() error { _, err := GetDatabaseReader("primary"); return err },
		"writer":       func() error { _, err := GetDatabaseWriter("primary"); return err },
		"migration":    func() error { _, err := GetMigrationService("primary"); return err },
		"seed":         func() error { _, err := GetSeedService("primary"); return err },
	}

	for name, accessor := range accessors {
		assert.ErrorIs(t, accessor(), errNotInitialised, name)
	}

	assert.False(t, IsEventsRunning())
	assert.Nil(t, GetImagePredefinedVariants())
}

func TestGlobalServiceAccessors_DelegateToTheInstalledContainer(t *testing.T) {
	email := &stubEmailService{}
	i18n := &stubI18nService{}
	storage := &stubStorageService{}
	cacheService := &optionsStubCacheService{}
	cryptoService := crypto_domain.NewDisabledCryptoService()
	llm := &stubLLMService{}
	search := &stubSearchServicePort{}
	image := &stubImageService{}
	events := &servicesStubEventsProvider{running: true}
	pdfWriter := &servicesStubPdfWriter{}
	variants := map[string]image_dto.TransformationSpec{"thumbnail": {Width: 64}}

	c := NewContainer(
		WithEmailService(email),
		WithI18nService(i18n),
		WithCacheService(cacheService),
		WithCryptoService(cryptoService),
		WithSearchService(search),
		WithImageService(image),
	)
	c.storageServiceOverride = storage
	c.llmServiceOverride = llm
	c.eventsProviderOverride = events
	c.pdfWriterService = pdfWriter
	c.imagePredefinedVariants = variants
	installGlobalContainer(t, c)

	gotEmail, err := GetEmailService()
	require.NoError(t, err)
	assert.Same(t, email, gotEmail)

	gotI18n, err := GetI18nService()
	require.NoError(t, err)
	assert.Same(t, i18n, gotI18n)

	gotStorage, err := GetStorageService()
	require.NoError(t, err)
	assert.Same(t, storage, gotStorage)

	gotCache, err := GetCacheService()
	require.NoError(t, err)
	assert.Same(t, cacheService, gotCache)

	gotCrypto, err := GetCryptoService()
	require.NoError(t, err)
	assert.Same(t, cryptoService, gotCrypto)

	gotCaptcha, err := GetCaptchaService()
	require.NoError(t, err)
	assert.NotNil(t, gotCaptcha)

	gotSpam, err := GetSpamDetectService()
	require.NoError(t, err)
	assert.NotNil(t, gotSpam)

	gotLLM, err := GetLLMService()
	require.NoError(t, err)
	assert.Same(t, llm, gotLLM)

	gotSearch, err := GetSearchService()
	require.NoError(t, err)
	assert.Same(t, search, gotSearch)

	gotImage, err := GetImageService()
	require.NoError(t, err)
	assert.Same(t, image, gotImage)

	gotEvents, err := GetEventsProvider()
	require.NoError(t, err)
	assert.Same(t, events, gotEvents)

	_, err = GetEventsRouter()
	require.NoError(t, err)
	_, err = GetEventsPublisher()
	require.NoError(t, err)
	_, err = GetEventsSubscriber()
	require.NoError(t, err)
	assert.True(t, IsEventsRunning())

	gotPdf, err := GetPdfWriterService()
	require.NoError(t, err)
	assert.Same(t, pdfWriter, gotPdf)

	assert.Equal(t, variants, GetImagePredefinedVariants())

	_, err = GetDatabaseConnection("missing")
	require.ErrorIs(t, err, ErrDatabaseNotRegistered)
	_, err = GetDatabaseReader("missing")
	require.ErrorIs(t, err, ErrDatabaseNotRegistered)
	_, err = GetDatabaseWriter("missing")
	require.ErrorIs(t, err, ErrDatabaseNotRegistered)
	_, err = GetMigrationService("missing")
	require.ErrorIs(t, err, ErrDatabaseNotRegistered)
	_, err = GetSeedService("missing")
	require.ErrorIs(t, err, ErrDatabaseNotRegistered)
}

func TestGlobalServiceAccessors_ReportMissingPdfWriterAndStoppedEvents(t *testing.T) {
	c := NewContainer()
	c.eventsProviderOverride = &servicesStubEventsProvider{running: false}
	installGlobalContainer(t, c)

	_, err := GetPdfWriterService()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pdf writer service not initialised")
	assert.False(t, IsEventsRunning())
}
