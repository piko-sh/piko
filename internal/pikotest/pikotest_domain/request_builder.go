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

package pikotest_domain

import (
	"context"
	"maps"
	"net/http"
	"net/url"

	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

// RequestBuilder provides a fluent API for constructing RequestData for tests.
// It simplifies the creation of test requests with proper context injection
// for mocks.
type RequestBuilder struct {
	// collectionData holds extra data passed to the request for use in
	// templates.
	collectionData any

	// globalTranslations holds translation strings mapped by language code and
	// then by translation key.
	globalTranslations map[string]map[string]string

	// queryParams holds URL query parameters mapped by name.
	queryParams map[string][]string

	// formData holds form field values, keyed by field name.
	formData map[string][]string

	// pathParams holds URL path parameters to replace in the request path.
	pathParams map[string]string

	// localTranslations holds translations for specific components, grouped by
	// locale.
	localTranslations map[string]map[string]string

	// headers stores HTTP headers to add to the request.
	headers map[string]string

	// locale is the locale code for the request (e.g. "es").
	locale string

	// defaultLocale is the fallback locale used when the primary locale is not
	// set.
	defaultLocale string

	// host is the value for the HTTP Host header.
	host string

	// method is the HTTP method for the request (e.g. GET, POST).
	method string

	// path is the URL path for the HTTP request.
	path string
}

// NewRequest creates a new RequestBuilder for the given HTTP method and path.
// This is the main starting point for building test requests.
//
// Takes method (string) which specifies the HTTP method (e.g. GET, POST).
// Takes path (string) which specifies the request path.
//
// Returns *RequestBuilder which provides a fluent interface for setting up
// the test request.
func NewRequest(method, path string) *RequestBuilder {
	return &RequestBuilder{
		collectionData:     nil,
		globalTranslations: nil,
		queryParams:        make(map[string][]string),
		formData:           make(map[string][]string),
		pathParams:         make(map[string]string),
		localTranslations:  nil,
		headers:            make(map[string]string),
		locale:             "en",
		defaultLocale:      "en",
		host:               "localhost",
		method:             method,
		path:               path,
	}
}

// WithQueryParam adds a single query parameter to the request.
// Call multiple times to add more than one value for the same key.
//
// Takes key (string) which is the query parameter name.
// Takes value (string) which is the query parameter value.
//
// Returns *RequestBuilder which allows method chaining.
func (b *RequestBuilder) WithQueryParam(key, value string) *RequestBuilder {
	b.queryParams[key] = append(b.queryParams[key], value)
	return b
}

// WithQueryParams adds multiple query parameters at once using a map.
//
// Takes params (map[string][]string) which maps parameter names to their
// values.
//
// Returns *RequestBuilder which allows method chaining.
func (b *RequestBuilder) WithQueryParams(params map[string][]string) *RequestBuilder {
	for key, values := range params {
		b.queryParams[key] = append(b.queryParams[key], values...)
	}
	return b
}

// WithFormData adds a single form field to the request.
// Call multiple times to add multiple values for the same key.
//
// Takes key (string) which specifies the form field name.
// Takes value (string) which specifies the form field value.
//
// Returns *RequestBuilder which allows method chaining.
func (b *RequestBuilder) WithFormData(key, value string) *RequestBuilder {
	b.formData[key] = append(b.formData[key], value)
	return b
}

// WithFormDataMap adds multiple form fields at once using a map.
//
// Takes data (map[string][]string) which contains the form field names and
// their values to add.
//
// Returns *RequestBuilder which allows method chaining.
func (b *RequestBuilder) WithFormDataMap(data map[string][]string) *RequestBuilder {
	for key, values := range data {
		b.formData[key] = append(b.formData[key], values...)
	}
	return b
}

// WithPathParam adds a path parameter to the request URL.
//
// Takes key (string) which specifies the parameter name without the colon.
// Takes value (string) which specifies the value to use in its place.
//
// Returns *RequestBuilder which allows method chaining.
func (b *RequestBuilder) WithPathParam(key, value string) *RequestBuilder {
	b.pathParams[key] = value
	return b
}

// WithPathParams adds multiple path parameters at once.
//
// Takes params (map[string]string) which contains the path parameters to add.
//
// Returns *RequestBuilder which allows method chaining.
func (b *RequestBuilder) WithPathParams(params map[string]string) *RequestBuilder {
	maps.Copy(b.pathParams, params)
	return b
}

// WithLocale sets the locale for the request. This affects translation
// lookups.
//
// Takes locale (string) which specifies the locale code for translations.
//
// Returns *RequestBuilder which allows method chaining.
func (b *RequestBuilder) WithLocale(locale string) *RequestBuilder {
	b.locale = locale
	return b
}

// WithDefaultLocale sets the default/fallback locale for the request.
//
// Takes locale (string) which specifies the fallback locale code.
//
// Returns *RequestBuilder which allows method chaining.
func (b *RequestBuilder) WithDefaultLocale(locale string) *RequestBuilder {
	b.defaultLocale = locale
	return b
}

// WithHost sets the host header for the request.
//
// Takes host (string) which specifies the host header value.
//
// Returns *RequestBuilder which allows method chaining.
func (b *RequestBuilder) WithHost(host string) *RequestBuilder {
	b.host = host
	return b
}

// WithHeader adds an HTTP header to the request.
//
// Takes key (string) which specifies the header name.
// Takes value (string) which specifies the header value.
//
// Returns *RequestBuilder which allows method chaining.
func (b *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	b.headers[key] = value
	return b
}

// WithGlobalTranslations sets the global translation map for the request.
// The format is map[locale]map[key]translation.
//
// Takes translations (map[string]map[string]string) which maps locales to
// their key-value translation pairs.
//
// Returns *RequestBuilder which allows method chaining.
func (b *RequestBuilder) WithGlobalTranslations(translations map[string]map[string]string) *RequestBuilder {
	b.globalTranslations = translations
	return b
}

// WithLocalTranslations sets the local translation map for the request.
// The format is map[locale]map[key]translation for component-specific
// translations.
//
// Takes translations (map[string]map[string]string) which provides the
// locale-to-key-to-translation mapping.
//
// Returns *RequestBuilder which allows method chaining.
func (b *RequestBuilder) WithLocalTranslations(translations map[string]map[string]string) *RequestBuilder {
	b.localTranslations = translations
	return b
}

// WithCollectionData sets the collection data for p-collection virtual pages.
// Only needed when testing p-collection components.
//
// Takes data (any) which specifies the collection items for the virtual page.
//
// Returns *RequestBuilder which allows method chaining.
func (b *RequestBuilder) WithCollectionData(data any) *RequestBuilder {
	b.collectionData = data
	return b
}

// Build constructs the final RequestData from the builder's configuration.
//
// Takes ctx (context.Context) which is the request context for cancellation,
// deadlines, and injecting mock dependencies.
//
// Returns *templater_dto.RequestData which contains the fully configured
// request data ready for use.
func (b *RequestBuilder) Build(ctx context.Context) *templater_dto.RequestData {
	parsedURL, _ := url.Parse(b.path)

	builder := templater_dto.NewRequestDataBuilder().
		WithContext(ctx).
		WithMethod(b.method).
		WithHost(b.host).
		WithURL(parsedURL).
		WithLocale(b.locale).
		WithDefaultLocale(b.defaultLocale).
		WithCollectionData(b.collectionData)

	for key, value := range b.pathParams {
		builder.AddPathParam(key, value)
	}
	for key, values := range b.queryParams {
		builder.AddQueryParam(key, values)
	}
	for key, values := range b.formData {
		builder.AddFormData(key, values)
	}

	reqData := builder.Build()

	if b.globalTranslations != nil {
		reqData.SetGlobalStore(i18n_domain.NewStoreFromTranslations(b.globalTranslations, b.defaultLocale))
	}
	if b.localTranslations != nil {
		reqData.SetLocalStore(i18n_domain.NewStoreFromTranslations(b.localTranslations, b.defaultLocale))
	}

	return reqData
}

// BuildHTTPRequest creates an actual *http.Request in addition to RequestData.
// Use it in tests that need to simulate the full HTTP layer.
//
// Takes ctx (context.Context) which is the request context for cancellation,
// deadlines, and injecting mock dependencies.
//
// Returns *http.Request which is the constructed HTTP request ready for use.
// Returns *templater_dto.RequestData which contains the parsed request data.
func (b *RequestBuilder) BuildHTTPRequest(ctx context.Context) (*http.Request, *templater_dto.RequestData) {
	parsedURL, _ := url.Parse(b.path)

	if len(b.queryParams) > 0 {
		q := parsedURL.Query()
		for key, values := range b.queryParams {
			for _, value := range values {
				q.Add(key, value)
			}
		}
		parsedURL.RawQuery = q.Encode()
	}

	httpReq, _ := http.NewRequestWithContext(ctx, b.method, parsedURL.String(), nil)
	httpReq.Host = b.host

	for key, value := range b.headers {
		httpReq.Header.Set(key, value)
	}

	builder := templater_dto.NewRequestDataBuilder().
		WithContext(httpReq.Context()).
		WithMethod(httpReq.Method).
		WithHost(httpReq.Host).
		WithURL(httpReq.URL).
		WithLocale(b.locale).
		WithDefaultLocale(b.defaultLocale).
		WithCollectionData(b.collectionData)

	for key, value := range b.pathParams {
		builder.AddPathParam(key, value)
	}
	for key, values := range b.queryParams {
		builder.AddQueryParam(key, values)
	}
	for key, values := range b.formData {
		builder.AddFormData(key, values)
	}

	reqData := builder.Build()

	if b.globalTranslations != nil {
		reqData.SetGlobalStore(i18n_domain.NewStoreFromTranslations(b.globalTranslations, b.defaultLocale))
	}
	if b.localTranslations != nil {
		reqData.SetLocalStore(i18n_domain.NewStoreFromTranslations(b.localTranslations, b.defaultLocale))
	}

	return httpReq, reqData
}
