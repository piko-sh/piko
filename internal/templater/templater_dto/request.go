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

package templater_dto

import (
	"context"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/security/security_adapters"
)

const (
	// maxLocaleFallbackDepth is the maximum number of locales to check during
	// fallback resolution. This covers: current locale, language-only fallback,
	// default locale, and default language-only.
	maxLocaleFallbackDepth = 4

	// defaultPathParamsCapacity is the starting slice capacity for path
	// parameters.
	defaultPathParamsCapacity = 8

	// defaultQueryParamsCapacity is the initial size for query parameter maps.
	defaultQueryParamsCapacity = 8

	// defaultFormDataCapacity is the initial size for form data maps.
	defaultFormDataCapacity = 4

	// defaultCookiesCapacity is the initial slice capacity for HTTP cookies.
	defaultCookiesCapacity = 4
)

// KVPair holds a key and value pair for storing parameters without memory
// allocation. A slice of KVPair is used instead of a map to avoid the extra
// memory costs of map buckets and string copying.
type KVPair struct {
	// Key is the parameter name used for lookup.
	Key string

	// Value is the value linked to the key.
	Value string
}

var (
	// requestDataPool is a pool of RequestData structs for reuse.
	requestDataPool = sync.Pool{
		New: func() any {
			return &RequestData{
				pathParams:  make([]KVPair, 0, defaultPathParamsCapacity),
				queryParams: make(map[string][]string, defaultQueryParamsCapacity),
				formData:    make(map[string][]string, defaultFormDataCapacity),
				cookies:     make([]*http.Cookie, 0, defaultCookiesCapacity),
			}
		},
	}

	// builderPool is a pool of RequestDataBuilder structs for reuse.
	builderPool = sync.Pool{
		New: func() any {
			return &RequestDataBuilder{
				pathParams:  make([]KVPair, 0, defaultPathParamsCapacity),
				queryParams: make(map[string][]string, defaultQueryParamsCapacity),
				formData:    make(map[string][]string, defaultFormDataCapacity),
				cookies:     make([]*http.Cookie, 0, defaultCookiesCapacity),
			}
		},
	}

	// localeFallbackPool is a pool of locale fallback slices for reuse.
	localeFallbackPool = sync.Pool{
		New: func() any {
			return make([]string, 0, maxLocaleFallbackDepth)
		},
	}

	// localeSeenPool is a pool of "seen" maps for locale fallback computation.
	localeSeenPool = sync.Pool{
		New: func() any {
			return make(map[string]struct{}, maxLocaleFallbackDepth)
		},
	}
)

// RequestData encapsulates all information about an incoming HTTP request.
// It implements ActionInfoProvider and uses a builder pattern for immutable
// construction.
type RequestData struct {
	// collectionData holds extra data for collection pages; nil for regular pages.
	collectionData any

	// ctx holds the request context. It cannot be changed after the struct is
	// created.
	ctx context.Context

	// globalStore holds the shared translation store for i18n lookups.
	globalStore *i18n_domain.Store

	// formData holds parsed form data from the request body.
	formData map[string][]string

	// pathParams holds URL path parameters as key-value pairs.
	// Uses a slice instead of a map to avoid memory allocation during lookups.
	pathParams []KVPair

	// cookies holds HTTP cookies from the incoming request.
	cookies []*http.Cookie

	// url holds the parsed request URL. Use URL() to get a safe copy.
	url *url.URL

	// queryParams stores parsed URL query parameters, keyed by parameter name.
	queryParams map[string][]string

	// localStore holds translations specific to the current component.
	localStore *i18n_domain.Store

	// strBufPool holds shared string buffers for building translations.
	strBufPool *i18n_domain.StrBufPool

	// method is the HTTP request method (e.g. GET, POST).
	method string

	// host is the hostname from the HTTP request.
	host string

	// locale is the locale identifier for this request.
	locale string

	// defaultLocale is the locale to use when no specific locale is set.
	defaultLocale string

	// localeFallbackOrder caches the computed locale fallback order.
	// It is computed once per request and returned to the pool on reset.
	localeFallbackOrder []string

	// localeFallbackComputed indicates whether localeFallbackOrder has been
	// computed and stored.
	localeFallbackComputed bool
}

// Context returns the request context.
//
// Returns context.Context which is the context for this request.
func (r *RequestData) Context() context.Context { return r.ctx }

// ClientIP returns the real client IP address resolved by the trusted
// proxy chain. Returns empty string if the RealIP middleware has not
// run.
//
// Returns string which is the resolved client IP address.
func (r *RequestData) ClientIP() string {
	if pctx := daemon_dto.PikoRequestCtxFromContext(r.ctx); pctx != nil {
		return pctx.ClientIP
	}
	return ""
}

// RequestID returns the unique request identifier. Returns empty
// string if the RealIP middleware has not run.
//
// Returns string which is the formatted or forwarded request ID.
func (r *RequestData) RequestID() string {
	if pctx := daemon_dto.PikoRequestCtxFromContext(r.ctx); pctx != nil {
		return pctx.RequestID()
	}
	return ""
}

// Auth returns the authentication context for this request, or nil
// if no auth provider is configured or the request is
// unauthenticated.
//
// Returns daemon_dto.AuthContext which provides access to
// authentication state.
func (r *RequestData) Auth() daemon_dto.AuthContext {
	if pctx := daemon_dto.PikoRequestCtxFromContext(r.ctx); pctx != nil {
		if auth, ok := pctx.CachedAuth.(daemon_dto.AuthContext); ok {
			return auth
		}
	}
	return nil
}

// CSPTokenAttr returns the CSP token attribute for inline scripts and styles.
// Retrieves the per-request token from the context and returns it formatted as
// an HTML attribute that can be used directly in templates.
//
// When using CSPRequestToken in your CSP policy (via piko.WithCSP), add this
// attribute to inline script and style elements:
// <script {{ .CSPTokenAttr }}>console.log("Hello");</script>
// <style {{ .CSPTokenAttr }}>.my-class { color: red; }</style>
//
// Returns template.HTMLAttr which is safe for use in HTML templates, or an
// empty attribute if no token was generated for this request.
func (r *RequestData) CSPTokenAttr() template.HTMLAttr {
	if r.ctx == nil {
		return ""
	}
	token := security_adapters.GetRequestTokenFromContext(r.ctx)
	if token == "" {
		return ""
	}
	//nolint:gosec // token from crypto/rand
	return template.HTMLAttr(`nonce="` + token + `"`)
}

// Method returns the HTTP method (GET, POST, etc.).
//
// Returns string which is the HTTP method name.
func (r *RequestData) Method() string { return r.method }

// Host returns the host from the request.
//
// Returns string which is the request host.
func (r *RequestData) Host() string { return r.host }

// Locale returns the locale identifier for this request.
//
// Returns string which is the locale identifier.
func (r *RequestData) Locale() string { return r.locale }

// DefaultLocale returns the default fallback locale.
//
// Returns string which is the locale to use when no specific locale is set.
func (r *RequestData) DefaultLocale() string { return r.defaultLocale }

// CollectionData returns the pre-fetched collection data for p-collection
// pages.
//
// Returns any which is the collection data, or nil for regular pages.
func (r *RequestData) CollectionData() any { return r.collectionData }

// URL returns a defensive copy of the request URL.
// The copy prevents mutation of the original URL.
//
// Returns *url.URL which is a deep copy of the request URL, or nil if no URL
// is set.
func (r *RequestData) URL() *url.URL {
	if r.url == nil {
		return nil
	}
	urlCopy := *r.url
	if r.url.User != nil {
		urlCopy.User = new(*r.url.User)
	}
	return &urlCopy
}

// PathParam returns the value of a path parameter by key.
// Uses linear search which is optimal for typical path param counts (1-4).
//
// Takes key (string) which specifies the parameter name to look up.
//
// Returns string which is the parameter value, or empty if the key does not
// exist.
func (r *RequestData) PathParam(key string) string {
	for i := range r.pathParams {
		if r.pathParams[i].Key == key {
			return r.pathParams[i].Value
		}
	}
	return ""
}

// QueryParam returns the first value of a query parameter by key.
//
// Takes key (string) which specifies the query parameter name to look up.
//
// Returns string which is the parameter value, or empty string if the key
// does not exist.
func (r *RequestData) QueryParam(key string) string {
	if vals := r.queryParams[key]; len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// QueryParamValues returns all values for a query parameter key.
//
// Takes key (string) which specifies the query parameter name to look up.
//
// Returns []string which contains all values for the key, or nil if the key
// does not exist.
func (r *RequestData) QueryParamValues(key string) []string {
	return r.queryParams[key]
}

// FormValue returns the first value of a form field by key.
//
// Takes key (string) which specifies the form field name to look up.
//
// Returns string which is the field value, or empty if the key is not found.
func (r *RequestData) FormValue(key string) string {
	if vals := r.formData[key]; len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// FormValues returns all values for a form field key.
//
// Takes key (string) which specifies the form field name.
//
// Returns []string which contains all values for the key, or nil if the key
// does not exist.
func (r *RequestData) FormValues(key string) []string {
	return r.formData[key]
}

// Cookie returns the named cookie from the request. Returns
// http.ErrNoCookie if the cookie is not found.
//
// Takes name (string) which specifies the cookie name to look up.
//
// Returns *http.Cookie which is the cookie if found.
// Returns error which is http.ErrNoCookie if the cookie is not present.
func (r *RequestData) Cookie(name string) (*http.Cookie, error) {
	for _, c := range r.cookies {
		if c.Name == name {
			return c, nil
		}
	}
	return nil, http.ErrNoCookie
}

// Cookies returns a defensive copy of all cookies from the request.
//
// Returns []*http.Cookie which contains all request cookies.
func (r *RequestData) Cookies() []*http.Cookie {
	result := make([]*http.Cookie, len(r.cookies))
	copy(result, r.cookies)
	return result
}

// SetCookie adds a cookie to the HTTP response.
//
// Works in both pages and partials. Use piko.Cookie(),
// piko.SessionCookie(), or piko.ClearCookie() to create cookies
// with secure defaults. No-op if called outside a render context.
//
// Takes cookie (*http.Cookie) which specifies the cookie to set on the
// response.
func (r *RequestData) SetCookie(cookie *http.Cookie) {
	if acc := CookieAccumulatorFromContext(r.ctx); acc != nil {
		acc.Add(cookie)
	}
}

// PathParams returns a defensive copy of all path parameters.
// Prefer PathParam for single key lookup.
//
// Returns map[string]string which contains all path parameters.
func (r *RequestData) PathParams() map[string]string {
	result := make(map[string]string, len(r.pathParams))
	for i := range r.pathParams {
		result[r.pathParams[i].Key] = r.pathParams[i].Value
	}
	return result
}

// QueryParams returns a defensive copy of all query parameters.
// Prefer QueryParam(key) or QueryParamValues(key) for single key lookup.
//
// Returns map[string][]string which contains all query parameters.
func (r *RequestData) QueryParams() map[string][]string {
	result := make(map[string][]string, len(r.queryParams))
	for k, v := range r.queryParams {
		result[k] = append([]string(nil), v...)
	}
	return result
}

// FormData returns a defensive copy of all form data.
// Prefer FormValue(key) or FormValues(key) for single key lookup.
//
// Returns map[string][]string which contains all form key-value pairs.
func (r *RequestData) FormData() map[string][]string {
	result := make(map[string][]string, len(r.formData))
	for k, v := range r.formData {
		result[k] = append([]string(nil), v...)
	}
	return result
}

// RangePathParams iterates over all path parameters.
//
// Takes callback (func(key, value string) bool) which receives
// each key-value pair.
// Return false from the callback to stop iteration.
func (r *RequestData) RangePathParams(callback func(key, value string) bool) {
	for i := range r.pathParams {
		if !callback(r.pathParams[i].Key, r.pathParams[i].Value) {
			break
		}
	}
}

// RangeQueryParams iterates over all query parameters, calling the callback
// for each key and its values. Return false from the callback to stop early.
//
// Takes callback (func(key string, values []string) bool) which receives each
// parameter key and its values, returning false to stop iteration.
func (r *RequestData) RangeQueryParams(callback func(key string, values []string) bool) {
	for k, v := range r.queryParams {
		if !callback(k, v) {
			break
		}
	}
}

// RangeFormData iterates over all form data fields, calling the callback for
// each key and its values. Return false from the callback to stop iteration.
//
// Takes callback (func(key string, values []string) bool) which
// receives each field's
// key and values, returning false to stop iteration early.
func (r *RequestData) RangeFormData(callback func(key string, values []string) bool) {
	for k, v := range r.formData {
		if !callback(k, v) {
			break
		}
	}
}

// WithCollectionData returns a new RequestData with the collection data
// set. This is used by generated code for collection pages.
//
// Takes data (any) which is the collection data to set.
//
// Returns *RequestData which is a shallow copy with the collection data
// applied.
func (r *RequestData) WithCollectionData(data any) *RequestData {
	return &RequestData{
		ctx:            r.ctx,
		method:         r.method,
		host:           r.host,
		url:            r.url,
		pathParams:     r.pathParams,
		queryParams:    r.queryParams,
		formData:       r.formData,
		cookies:        r.cookies,
		locale:         r.locale,
		defaultLocale:  r.defaultLocale,
		collectionData: data,
		globalStore:    r.globalStore,
		localStore:     r.localStore,
		strBufPool:     r.strBufPool,
	}
}

// WithDefaultLocale returns a new RequestData with the default locale set.
// This is used by adapters to configure the fallback locale.
//
// Takes locale (string) which specifies the fallback locale to use.
//
// Returns *RequestData which is a shallow copy with the default locale
// applied.
func (r *RequestData) WithDefaultLocale(locale string) *RequestData {
	return &RequestData{
		ctx:            r.ctx,
		method:         r.method,
		host:           r.host,
		url:            r.url,
		pathParams:     r.pathParams,
		queryParams:    r.queryParams,
		formData:       r.formData,
		cookies:        r.cookies,
		locale:         r.locale,
		defaultLocale:  locale,
		collectionData: r.collectionData,
		globalStore:    r.globalStore,
		localStore:     r.localStore,
		strBufPool:     r.strBufPool,
	}
}

// RequestDataBuilder builds RequestData instances using a fluent interface.
// Field ordering is optimised for memory alignment.
type RequestDataBuilder struct {
	// ctx holds the request context for cancellation and timeout control.
	ctx context.Context

	// collectionData holds data for collection operations.
	collectionData any

	// url is the base URL for the request; nil uses the default URL.
	url *url.URL

	// queryParams stores URL query parameters as key-value pairs.
	queryParams map[string][]string

	// formData stores form field values as key-value pairs for the request body.
	formData map[string][]string

	// method is the HTTP method for the request (e.g. GET, POST).
	method string

	// host is the request host; empty uses the default.
	host string

	// locale is the request locale; empty means defaultLocale is used.
	locale string

	// defaultLocale is the fallback locale used when no locale is set; defaults
	// to "en_GB".
	defaultLocale string

	// pathParams holds URL path parameters that replace placeholders in the
	// request URL. Uses a slice instead of a map to avoid memory allocation.
	pathParams []KVPair

	// cookies holds HTTP cookies from the incoming request.
	cookies []*http.Cookie
}

// NewRequestDataBuilder creates a new builder from the pool with default
// values. The builder's slices and maps are reused from the pool to avoid
// allocations.
//
// Returns *RequestDataBuilder which is ready for use.
func NewRequestDataBuilder() *RequestDataBuilder {
	b, ok := builderPool.Get().(*RequestDataBuilder)
	if !ok {
		b = &RequestDataBuilder{
			pathParams:  make([]KVPair, 0, defaultPathParamsCapacity),
			queryParams: make(map[string][]string),
			formData:    make(map[string][]string),
			cookies:     make([]*http.Cookie, 0, defaultCookiesCapacity),
		}
	}
	b.resetBuilder()
	return b
}

// Release returns the builder to the pool for reuse.
//
// Do not use the builder after the call.
func (b *RequestDataBuilder) Release() {
	if b == nil {
		return
	}
	builderPool.Put(b)
}

// WithContext sets the request context.
//
// Takes ctx (context.Context) which provides the request context.
//
// Returns *RequestDataBuilder which enables method chaining.
func (b *RequestDataBuilder) WithContext(ctx context.Context) *RequestDataBuilder {
	b.ctx = ctx
	return b
}

// WithMethod sets the HTTP method.
//
// Takes m (string) which specifies the HTTP method to use.
//
// Returns *RequestDataBuilder which allows method chaining.
func (b *RequestDataBuilder) WithMethod(m string) *RequestDataBuilder {
	b.method = m
	return b
}

// WithHost sets the request host.
//
// Takes h (string) which specifies the host value for the request.
//
// Returns *RequestDataBuilder which allows method chaining.
func (b *RequestDataBuilder) WithHost(h string) *RequestDataBuilder {
	b.host = h
	return b
}

// WithURL sets the request URL.
//
// Takes u (*url.URL) which specifies the URL for the request.
//
// Returns *RequestDataBuilder which allows method chaining.
func (b *RequestDataBuilder) WithURL(u *url.URL) *RequestDataBuilder {
	b.url = u
	return b
}

// WithLocale sets the request locale.
//
// Takes l (string) which specifies the locale identifier.
//
// Returns *RequestDataBuilder which allows method chaining.
func (b *RequestDataBuilder) WithLocale(l string) *RequestDataBuilder {
	b.locale = l
	return b
}

// WithDefaultLocale sets the default/fallback locale.
//
// Takes l (string) which specifies the locale to use as the fallback.
//
// Returns *RequestDataBuilder which allows method chaining.
func (b *RequestDataBuilder) WithDefaultLocale(l string) *RequestDataBuilder {
	b.defaultLocale = l
	return b
}

// WithCollectionData sets the collection data.
//
// Takes d (any) which is the collection data to attach to the request.
//
// Returns *RequestDataBuilder which allows method chaining.
func (b *RequestDataBuilder) WithCollectionData(d any) *RequestDataBuilder {
	b.collectionData = d
	return b
}

// WithCookies sets the HTTP cookies from the incoming request.
//
// Takes cookies ([]*http.Cookie) which provides the request cookies.
//
// Returns *RequestDataBuilder which allows method chaining.
func (b *RequestDataBuilder) WithCookies(cookies []*http.Cookie) *RequestDataBuilder {
	b.cookies = append(b.cookies[:0], cookies...)
	return b
}

// AddPathParam adds a path parameter.
// Uses append which is zero-allocation when the slice has capacity.
//
// Takes k (string) which specifies the parameter name.
// Takes v (string) which specifies the parameter value.
//
// Returns *RequestDataBuilder which allows method chaining.
func (b *RequestDataBuilder) AddPathParam(k, v string) *RequestDataBuilder {
	b.pathParams = append(b.pathParams, KVPair{Key: k, Value: v})
	return b
}

// AddQueryParam adds a query parameter with multiple values.
//
// Takes k (string) which specifies the parameter name.
// Takes v ([]string) which provides the parameter values.
//
// Returns *RequestDataBuilder which allows method chaining.
func (b *RequestDataBuilder) AddQueryParam(k string, v []string) *RequestDataBuilder {
	b.queryParams[k] = v
	return b
}

// AddQueryParamValue adds a single query parameter value.
//
// Takes k (string) which specifies the query parameter name.
// Takes v (string) which specifies the query parameter value.
//
// Returns *RequestDataBuilder which allows method chaining.
func (b *RequestDataBuilder) AddQueryParamValue(k, v string) *RequestDataBuilder {
	b.queryParams[k] = append(b.queryParams[k], v)
	return b
}

// AddFormData adds form data with multiple values.
//
// Takes k (string) which specifies the form field name.
// Takes v ([]string) which provides the values for the field.
//
// Returns *RequestDataBuilder which allows method chaining.
func (b *RequestDataBuilder) AddFormData(k string, v []string) *RequestDataBuilder {
	b.formData[k] = v
	return b
}

// AddFormDataValue adds a single form data value.
//
// Takes k (string) which specifies the form field name.
// Takes v (string) which specifies the form field value.
//
// Returns *RequestDataBuilder which allows method chaining.
func (b *RequestDataBuilder) AddFormDataValue(k, v string) *RequestDataBuilder {
	b.formData[k] = append(b.formData[k], v)
	return b
}

// Build constructs the final RequestData instance from the pool.
//
// The builder's slices and maps are swapped with the RequestData's to avoid
// copying. After Build is called, the builder is returned to its pool and
// must not be used.
//
// Returns *RequestData which is the constructed request data ready for use.
func (b *RequestDataBuilder) Build() *RequestData {
	rd, ok := requestDataPool.Get().(*RequestData)
	if !ok {
		rd = &RequestData{
			pathParams:  make([]KVPair, 0, defaultPathParamsCapacity),
			queryParams: make(map[string][]string),
			formData:    make(map[string][]string),
			cookies:     make([]*http.Cookie, 0, defaultCookiesCapacity),
		}
	}

	rd.pathParams, b.pathParams = b.pathParams, rd.pathParams
	rd.queryParams, b.queryParams = b.queryParams, rd.queryParams
	rd.formData, b.formData = b.formData, rd.formData
	rd.cookies, b.cookies = b.cookies, rd.cookies

	rd.ctx = b.ctx
	rd.url = b.url
	rd.collectionData = b.collectionData
	rd.method = b.method
	rd.host = b.host
	rd.locale = b.locale
	rd.defaultLocale = b.defaultLocale

	rd.globalStore = nil
	rd.localStore = nil
	rd.strBufPool = nil

	rd.localeFallbackOrder = nil
	rd.localeFallbackComputed = false

	b.Release()

	return rd
}

// resetBuilder clears the builder's state so it can be reused from the pool.
func (b *RequestDataBuilder) resetBuilder() {
	b.ctx = context.Background()
	b.url = nil
	b.pathParams = b.pathParams[:0]
	clear(b.queryParams)
	clear(b.formData)
	b.cookies = b.cookies[:0]
	b.collectionData = nil
	b.method = ""
	b.host = ""
	b.locale = ""
	b.defaultLocale = "en_GB"
}

// T returns a Translation object for the given key from global translations.
// The Translation implements fmt.Stringer for automatic string conversion in
// templates.
//
// Usage in templates:
// {{ T("greeting") }}                             // Simple lookup
// {{ T("greeting", "Hello") }}                    // With fallback
// {{ T("welcome").StringVar("name", user.Name) }} // With variable substitution
// The returned Translation supports a fluent API for variable binding:
//   - StringVar(name, value) - bind a string variable
//   - IntVar(name, value) - bind an integer variable
//   - FloatVar(name, value) - bind a float variable
//   - Count(n) - set the count for pluralisation
//
// Takes keyAndFallbacks (...string) which contains the translation key and
// optional fallback values.
//
// Returns *i18n_domain.Translation which provides the resolved translation
// with support for variable substitution and pluralisation.
func (r *RequestData) T(keyAndFallbacks ...string) *i18n_domain.Translation {
	key, fallback := r.parseKeyAndFallback(keyAndFallbacks)
	if key == "" {
		return i18n_domain.NewTranslation("", nil, r.strBufPool)
	}

	if t := r.lookupInStore(r.globalStore, key); t != nil {
		return t
	}
	if t := r.lookupInStore(r.localStore, key); t != nil {
		return t
	}

	return i18n_domain.NewTranslationFromString(key, fallback, r.strBufPool)
}

// LT returns a Translation object for the given key from local
// (component-scoped) translations. Local translations are defined in the
// component's <i18n> block.
//
// Takes keyAndFallbacks (...string) which provides the translation key and
// optional fallback value.
//
// Returns *i18n_domain.Translation which contains the resolved translation
// or the fallback as a literal translation if not found.
//
// Usage in templates:
// {{ LT("button.save") }}                        // Simple lookup
// {{ LT("button.save", "Save") }}                // With fallback
// {{ LT("items").Count(count) }}                 // With pluralisation
func (r *RequestData) LT(keyAndFallbacks ...string) *i18n_domain.Translation {
	key, fallback := r.parseKeyAndFallback(keyAndFallbacks)
	if key == "" {
		return i18n_domain.NewTranslation("", nil, r.strBufPool)
	}

	if t := r.lookupInStore(r.localStore, key); t != nil {
		return t
	}

	return i18n_domain.NewTranslationFromString(key, fallback, r.strBufPool)
}

// LF returns a FormatBuilder with the current request locale pre-applied.
// Use this in templates for locale-aware formatting of numeric and temporal
// values with optional method chaining.
//
// Takes value (any) which is the value to format.
//
// Returns *i18n_domain.FormatBuilder which implements fmt.Stringer and
// supports fluent method chaining (Precision, Short, Long, DateOnly, etc.).
//
// Usage in templates:
// {{ LF(state.Price) }}                  // Locale-formatted number
// {{ LF(state.Price).Precision(2) }}     // With 2 decimal places
// {{ LF(state.Date).Short().DateOnly() }} // Short date only
func (r *RequestData) LF(value any) *i18n_domain.FormatBuilder {
	return i18n_domain.NewLF(value, r.locale)
}

// SetGlobalStore sets the v2 global translation store.
//
// Takes store (*i18n_domain.Store) which provides the translation data.
func (r *RequestData) SetGlobalStore(store *i18n_domain.Store) {
	r.globalStore = store
}

// SetLocalStore sets the v2 local (component-scoped) translation store.
//
// Takes store (*i18n_domain.Store) which provides the translations for this
// component.
func (r *RequestData) SetLocalStore(store *i18n_domain.Store) {
	r.localStore = store
}

// SetLocalStoreFromMap builds a local translation Store from a raw map and
// sets it on this request. This is used by generated BuildAST code where the
// internal i18n_domain package cannot be imported directly.
//
// Takes translations (map[string]map[string]string) which provides the
// translation strings keyed by locale and then by translation key.
func (r *RequestData) SetLocalStoreFromMap(translations map[string]map[string]string) {
	if len(translations) == 0 {
		return
	}
	r.localStore = i18n_domain.NewStoreFromTranslations(translations, r.defaultLocale)
}

// SetStrBufPool sets the string buffer pool for zero-allocation rendering.
//
// Takes pool (*i18n_domain.StrBufPool) which provides pooled buffers for
// string operations.
func (r *RequestData) SetStrBufPool(pool *i18n_domain.StrBufPool) {
	r.strBufPool = pool
}

// SetI18n configures the i18n system with stores and buffer pool.
//
// Takes globalStore (*i18n_domain.Store) which provides global translations.
// Takes localStore (*i18n_domain.Store) which provides local translations.
// Takes pool (*i18n_domain.StrBufPool) which provides reusable string buffers.
func (r *RequestData) SetI18n(globalStore, localStore *i18n_domain.Store, pool *i18n_domain.StrBufPool) {
	r.globalStore = globalStore
	r.localStore = localStore
	r.strBufPool = pool
}

// Release returns the RequestData to the pool for reuse, reducing GC
// pressure. The RequestData must not be used after this call.
func (r *RequestData) Release() {
	if r == nil {
		return
	}
	r.reset()
	requestDataPool.Put(r)
}

// reset clears all fields and returns the object to a clean state for reuse
// from the pool.
func (r *RequestData) reset() {
	r.ctx = nil
	r.url = nil
	r.pathParams = r.pathParams[:0]
	clear(r.queryParams)
	clear(r.formData)
	r.cookies = r.cookies[:0]
	r.collectionData = nil
	r.globalStore = nil
	r.localStore = nil
	r.strBufPool = nil
	r.method = ""
	r.host = ""
	r.locale = ""
	r.defaultLocale = ""

	if r.localeFallbackOrder != nil {
		localeFallbackPool.Put(r.localeFallbackOrder)
		r.localeFallbackOrder = nil
	}
	r.localeFallbackComputed = false
}

// parseKeyAndFallback extracts the key and fallback from variadic arguments.
//
// Takes keyAndFallbacks ([]string) which contains the key and optional fallback
// value.
//
// Returns key (string) which is the first element, or empty if the slice is
// empty.
// Returns fallback (string) which is the second element if present, otherwise
// defaults to the key value.
func (*RequestData) parseKeyAndFallback(keyAndFallbacks []string) (key, fallback string) {
	if len(keyAndFallbacks) == 0 {
		return "", ""
	}
	key = keyAndFallbacks[0]
	fallback = key
	if len(keyAndFallbacks) > 1 {
		fallback = keyAndFallbacks[1]
	}
	return key, fallback
}

// lookupInStore searches for a translation key in the given store.
//
// Takes store (*i18n_domain.Store) which is the translation store to search.
// Takes key (string) which is the translation key to find.
//
// Returns *i18n_domain.Translation which is the found translation, or nil if
// the store is nil or the key is not found in any fallback locale.
func (r *RequestData) lookupInStore(store *i18n_domain.Store, key string) *i18n_domain.Translation {
	if store == nil {
		return nil
	}
	for _, locale := range r.getLocaleFallbackOrder() {
		if entry, found := store.Get(locale, key); found {
			return i18n_domain.NewTranslation(key, entry, r.strBufPool)
		}
	}
	return nil
}

// getLocaleFallbackOrder returns the list of locales to try in order.
//
// Returns []string which contains locales from most specific to least
// specific. The result is cached after the first call.
func (r *RequestData) getLocaleFallbackOrder() []string {
	if r.localeFallbackComputed {
		return r.localeFallbackOrder
	}

	order, ok := localeFallbackPool.Get().([]string)
	if !ok {
		order = make([]string, 0, maxLocaleFallbackDepth)
	} else {
		order = order[:0]
	}

	seen, ok := localeSeenPool.Get().(map[string]struct{})
	if !ok {
		seen = make(map[string]struct{}, maxLocaleFallbackDepth)
	} else {
		clear(seen)
	}

	add := func(locale string) {
		if locale == "" {
			return
		}
		if _, exists := seen[locale]; exists {
			return
		}
		seen[locale] = struct{}{}
		order = append(order, locale)
	}

	add(r.locale)
	if index := strings.IndexByte(r.locale, '-'); index != -1 {
		add(r.locale[:index])
	}

	add(r.defaultLocale)
	if index := strings.IndexByte(r.defaultLocale, '-'); index != -1 {
		add(r.defaultLocale[:index])
	}

	localeSeenPool.Put(seen)

	r.localeFallbackOrder = order
	r.localeFallbackComputed = true

	return r.localeFallbackOrder
}
