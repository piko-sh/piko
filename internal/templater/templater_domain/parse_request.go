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

package templater_domain

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

const (
	// fallbackDefaultLocale is the locale used when no locale is provided.
	fallbackDefaultLocale = "en_GB"

	// DefaultMaxMultipartFormBytes is the fallback cap (32 MiB) used when no
	// caller-supplied limit is threaded through. Action handlers should pass
	// their configured maxMultipartFormBytes via ParseRequestDataWithLimit so
	// the cap matches the daemon configuration.
	DefaultMaxMultipartFormBytes int64 = 32 << 20
)

// DefaultMaxURLEncodedFormBytes caps url-encoded request bodies at 8 MiB.
//
// Applied when no caller-supplied limit is threaded through.
// http.Request.ParseForm has no internal cap, so a pathological client
// could otherwise stream arbitrarily large bodies into memory. It is a
// variable so tests can override it for regression coverage.
var DefaultMaxURLEncodedFormBytes int64 = 8 << 20

// emptyQueryParams is a package-level empty url.Values to avoid allocation
// when requests have no query string.
var emptyQueryParams = make(url.Values)

// ParseRequestData builds a RequestData object from an HTTP request using the
// package default multipart cap. Prefer ParseRequestDataWithLimit when the
// daemon configuration supplies an explicit cap so that the parser honours
// the operator's setting.
//
// It extracts path parameters, query strings, form data, and detects the
// locale from the request.
//
// When the request is nil (such as for background jobs), returns a minimal
// default RequestData using the fallback locale.
//
// Takes r (*http.Request) which is the HTTP request to parse.
// Takes defaultLocale (string) which sets the fallback locale for translation
// lookups.
//
// Returns *templater_dto.RequestData which contains the extracted request
// information.
// Returns error when form data cannot be parsed.
func ParseRequestData(r *http.Request, defaultLocale string) (*templater_dto.RequestData, error) {
	return ParseRequestDataWithLimit(r, defaultLocale, DefaultMaxMultipartFormBytes)
}

// ParseRequestDataWithLimit is the explicit form of ParseRequestData that
// accepts a caller-supplied cap on multipart form size. Action handlers
// should pass their configured maxMultipartFormBytes here so the templater
// honours daemon-level limits rather than the hard-coded default.
//
// Takes r (*http.Request) which is the HTTP request to parse.
// Takes defaultLocale (string) which sets the fallback locale.
// Takes maxMultipartBytes (int64) which is the maximum in-memory size for
// multipart form data; values <= 0 fall back to DefaultMaxMultipartFormBytes.
//
// Returns *templater_dto.RequestData which contains the extracted request
// information.
// Returns error when form data cannot be parsed.
func ParseRequestDataWithLimit(r *http.Request, defaultLocale string, maxMultipartBytes int64) (*templater_dto.RequestData, error) {
	if r == nil {
		return buildDefaultRequestData(defaultLocale), nil
	}

	var queryParams url.Values
	if r.URL.RawQuery != "" {
		queryParams = r.URL.Query()
	} else {
		queryParams = emptyQueryParams
	}

	b := buildBaseRequestDataBuilder(r, defaultLocale, queryParams)
	extractPathParams(r, b)
	extractQueryParamsFromParsed(queryParams, b)

	if err := parseAndExtractFormData(r, b, maxMultipartBytes); err != nil {
		return nil, fmt.Errorf("parsing form data: %w", err)
	}

	return b.Build(), nil
}

// NewRequestDataFromAction creates a RequestData object for server actions.
// Unlike ParseRequestData, it does not parse form bodies and assumes the
// request is already complete.
//
// Takes r (*http.Request) which provides the HTTP request to extract data from.
//
// Returns *templater_dto.RequestData which contains the extracted request
// metadata including path parameters, query parameters, and existing form data.
func NewRequestDataFromAction(r *http.Request) *templater_dto.RequestData {
	var queryParams url.Values
	if r.URL.RawQuery != "" {
		queryParams = r.URL.Query()
	} else {
		queryParams = emptyQueryParams
	}

	b := templater_dto.NewRequestDataBuilder().
		WithContext(r.Context()).
		WithMethod(r.Method).
		WithHost(r.Host).
		WithURL(r.URL).
		WithDefaultLocale(fallbackDefaultLocale).
		WithLocale(detectLocale(r, queryParams)).
		WithCookies(r.Cookies())

	routeCtx := chi.RouteContext(r.Context())
	for i, key := range routeCtx.URLParams.Keys {
		b.AddPathParam(key, routeCtx.URLParams.Values[i])
	}

	for key, values := range queryParams {
		b.AddQueryParam(key, values)
	}

	if r.PostForm != nil {
		for key, values := range r.PostForm {
			b.AddFormData(key, values)
		}
	}
	if r.MultipartForm != nil && r.MultipartForm.Value != nil {
		for key, values := range r.MultipartForm.Value {
			b.AddFormData(key, values)
		}
	}

	return b.Build()
}

// detectLocale finds the user's locale by checking several sources in order:
// route context, query parameter, cookie, Accept-Language header, or a default
// value.
//
// Takes r (*http.Request) which provides context, cookies, and headers.
// Takes queryParams (url.Values) which provides pre-parsed query parameters.
//
// Returns string which is the detected locale code.
func detectLocale(r *http.Request, queryParams url.Values) string {
	if pctx := daemon_dto.PikoRequestCtxFromContext(r.Context()); pctx != nil && pctx.Locale != "" {
		return pctx.Locale
	}

	if queryLocale := queryParams.Get("locale"); queryLocale != "" {
		return queryLocale
	}

	if cookie, err := r.Cookie("piko_locale"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	if acceptLang := r.Header.Get("Accept-Language"); acceptLang != "" {
		if primary, _, found := strings.Cut(acceptLang, ","); found {
			return primary
		}
		return acceptLang
	}

	return fallbackDefaultLocale
}

// buildDefaultRequestData creates a minimal RequestData for use outside HTTP
// contexts.
//
// Takes defaultLocale (string) which sets the locale. Uses a fallback value if
// empty.
//
// Returns *templater_dto.RequestData which contains default request data with
// GET method and the given locale.
func buildDefaultRequestData(defaultLocale string) *templater_dto.RequestData {
	if defaultLocale == "" {
		defaultLocale = fallbackDefaultLocale
	}
	return templater_dto.NewRequestDataBuilder().
		WithMethod("GET").
		WithURL(&url.URL{}).
		WithLocale(defaultLocale).
		WithDefaultLocale(defaultLocale).
		Build()
}

// buildBaseRequestDataBuilder creates a RequestDataBuilder with standard HTTP
// request fields filled in.
//
// Takes r (*http.Request) which provides the HTTP request to extract data from.
// Takes defaultLocale (string) which specifies the fallback locale when none is
// detected.
// Takes queryParams (url.Values) which contains pre-parsed query parameters to
// avoid parsing them again.
//
// Returns *templater_dto.RequestDataBuilder which is set up with the request
// context, method, host, URL, and locale settings.
func buildBaseRequestDataBuilder(r *http.Request, defaultLocale string, queryParams url.Values) *templater_dto.RequestDataBuilder {
	if defaultLocale == "" {
		defaultLocale = fallbackDefaultLocale
	}
	return templater_dto.NewRequestDataBuilder().
		WithContext(r.Context()).
		WithMethod(r.Method).
		WithHost(r.Host).
		WithURL(r.URL).
		WithDefaultLocale(defaultLocale).
		WithLocale(detectLocale(r, queryParams)).
		WithCookies(r.Cookies())
}

// extractPathParams gets path parameters from the chi router context and adds
// them to the request data builder.
//
// Takes r (*http.Request) which provides the request with its route context.
// Takes b (*templater_dto.RequestDataBuilder) which receives the path
// parameters.
func extractPathParams(r *http.Request, b *templater_dto.RequestDataBuilder) {
	routeCtx := chi.RouteContext(r.Context())
	for i, key := range routeCtx.URLParams.Keys {
		b.AddPathParam(key, routeCtx.URLParams.Values[i])
	}
}

// extractQueryParamsFromParsed gets query parameters from parsed url.Values
// and adds them to the request builder. It skips the special "params" key,
// which it parses separately and adds as path parameters.
//
// Takes queryParams (url.Values) which contains the parsed query string.
// Takes b (*templater_dto.RequestDataBuilder) which receives the parameters.
func extractQueryParamsFromParsed(queryParams url.Values, b *templater_dto.RequestDataBuilder) {
	for key, values := range queryParams {
		if key == "params" {
			continue
		}
		b.AddQueryParam(key, values)
	}

	if queryParams.Has("params") {
		rawParams := queryParams.Get("params")
		parseCustomParamsCallback(rawParams, func(k, v string) {
			b.AddPathParam(k, v)
		})
	}
}

// parseAndExtractFormData reads form data from a request and adds the values
// to the builder.
//
// The function checks the Content-Type header to find the correct parser. It
// handles multipart form data and URL-encoded form data.
//
// Takes r (*http.Request) which provides the request with form data.
// Takes b (*templater_dto.RequestDataBuilder) which receives the parsed data.
// Takes maxMultipartBytes (int64) which caps the in-memory size for
// multipart form data; values <= 0 fall back to DefaultMaxMultipartFormBytes.
//
// Returns error when the form data cannot be parsed.
func parseAndExtractFormData(r *http.Request, b *templater_dto.RequestDataBuilder, maxMultipartBytes int64) error {
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		return parseMultipartFormData(r, b, maxMultipartBytes)
	}
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		return parseURLEncodedFormData(r, b, 0)
	}
	return nil
}

// parseMultipartFormData parses multipart form data and adds it to the builder.
//
// Takes r (*http.Request) which contains the multipart form data to parse.
// Takes b (*templater_dto.RequestDataBuilder) which receives the parsed form
// values.
// Takes maxMultipartBytes (int64) which caps the in-memory size for
// multipart form data; values <= 0 fall back to DefaultMaxMultipartFormBytes.
//
// Returns error when the multipart form cannot be parsed.
func parseMultipartFormData(r *http.Request, b *templater_dto.RequestDataBuilder, maxMultipartBytes int64) error {
	if maxMultipartBytes <= 0 {
		maxMultipartBytes = DefaultMaxMultipartFormBytes
	}
	if err := r.ParseMultipartForm(maxMultipartBytes); err != nil {
		return fmt.Errorf("parsing multipart form data: %w", err)
	}
	if r.MultipartForm != nil && r.MultipartForm.Value != nil {
		for key, values := range r.MultipartForm.Value {
			b.AddFormData(key, values)
		}
	}
	return nil
}

// parseURLEncodedFormData parses URL-encoded form data and adds it to the
// builder. The request body is bounded by http.MaxBytesReader so a
// pathological client cannot stream an arbitrarily large body into memory.
//
// Takes r (*http.Request) which provides the form data to parse.
// Takes b (*templater_dto.RequestDataBuilder) which stores the parsed data.
// Takes maxBodyBytes (int64) which caps the request body; values <= 0 fall
// back to DefaultMaxURLEncodedFormBytes.
//
// Returns error when the form data cannot be parsed or the body exceeds the
// configured cap.
func parseURLEncodedFormData(r *http.Request, b *templater_dto.RequestDataBuilder, maxBodyBytes int64) error {
	if maxBodyBytes <= 0 {
		maxBodyBytes = DefaultMaxURLEncodedFormBytes
	}
	if r.Body != nil {
		r.Body = http.MaxBytesReader(nil, r.Body, maxBodyBytes)
	}
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("parsing URL-encoded form data: %w", err)
	}
	for key, values := range r.PostForm {
		b.AddFormData(key, values)
	}
	return nil
}

// parseCustomParamsCallback extracts key-value pairs from a bracketed
// parameter string using zero-allocation manual parsing. The expected format
// is [key=value][key2=value2].
//
// Takes raw (string) which contains the parameter string to parse.
// Takes callback (func(k, v string)) which receives each key-value pair.
func parseCustomParamsCallback(raw string, callback func(k, v string)) {
	if len(raw) == 0 {
		return
	}

	for len(raw) > 0 {
		start := strings.IndexByte(raw, '[')
		if start == -1 {
			break
		}

		end := strings.IndexByte(raw[start:], ']')
		if end == -1 {
			break
		}

		inner := raw[start+1 : start+end]

		eq := strings.IndexByte(inner, '=')
		if eq != -1 && eq > 0 && eq < len(inner)-1 {
			callback(inner[:eq], inner[eq+1:])
		}

		raw = raw[start+end+1:]
	}
}
