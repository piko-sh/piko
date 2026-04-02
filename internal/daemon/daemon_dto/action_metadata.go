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

package daemon_dto

import (
	"context"
	"net/http"
	"net/url"
	"sync"
)

// ActionMetadata is embedded in action structs to provide access to request
// metadata and response building capabilities.
//
// Actions embed this type to gain access to:
//   - HTTP request metadata via Request()
//   - Response building via Response()
//
// The framework injects values before calling the action's Call method.
// Request context is passed as the first parameter to Call.
type ActionMetadata struct {
	// request holds the HTTP request metadata; nil until initialised by the
	// framework.
	request *RequestMetadata

	// response holds the HTTP response writer; nil until initialised by the
	// framework.
	response *ResponseWriter
}

// Request returns the HTTP request metadata.
// Returns nil if the action has not been initialised by the framework.
//
// Returns *RequestMetadata which contains the HTTP request details.
func (m *ActionMetadata) Request() *RequestMetadata {
	return m.request
}

// Response returns the response writer for setting cookies and headers.
//
// Returns *ResponseWriter which provides access to the underlying response.
// Returns nil when the action has not been initialised by the framework.
func (m *ActionMetadata) Response() *ResponseWriter {
	return m.response
}

// SetRequest injects request metadata into the action.
//
// Takes request (*RequestMetadata) which provides the request context.
//
// Called by the framework to inject request metadata. This method is
// intentionally unexported in the public API to prevent user modification.
func (m *ActionMetadata) SetRequest(request *RequestMetadata) {
	m.request = request
}

// SetResponse injects the response writer into the metadata.
//
// Takes response (*ResponseWriter) which provides the response writer instance.
//
// This method is intentionally unexported in the public API to prevent
// user modification.
func (m *ActionMetadata) SetResponse(response *ResponseWriter) {
	m.response = response
}

// InheritMetadata copies request and response metadata from another action,
// typically when one action calls another internally.
//
// Takes from (*ActionMetadata) which provides the metadata to copy.
func (m *ActionMetadata) InheritMetadata(from *ActionMetadata) {
	m.request = from.request
	m.response = from.response
}

// Ctx returns the request context for this action.
//
// Returns context.Context which is the request context, or
// context.Background if the action has not been initialised.
func (m *ActionMetadata) Ctx() context.Context {
	if m.request != nil && m.request.RawRequest != nil {
		return m.request.RawRequest.Context()
	}
	return context.Background()
}

// ClientIP returns the real client IP address resolved by the trusted
// proxy chain. Returns empty string if the action has not been
// initialised or the RealIP middleware has not run.
//
// Returns string which is the resolved client IP address.
func (m *ActionMetadata) ClientIP() string {
	if pctx := PikoRequestCtxFromContext(m.Ctx()); pctx != nil {
		return pctx.ClientIP
	}
	return ""
}

// RequestID returns the unique request identifier for this action's
// request. Returns empty string if the action has not been initialised.
//
// Returns string which is the formatted or forwarded request ID.
func (m *ActionMetadata) RequestID() string {
	if pctx := PikoRequestCtxFromContext(m.Ctx()); pctx != nil {
		return pctx.RequestID()
	}
	return ""
}

// Auth returns the authentication context for this action's request,
// or nil if no auth provider is configured or the request is
// unauthenticated.
//
// Returns AuthContext which provides access to authentication state.
func (m *ActionMetadata) Auth() AuthContext {
	if pctx := PikoRequestCtxFromContext(m.Ctx()); pctx != nil {
		if auth, ok := pctx.CachedAuth.(AuthContext); ok {
			return auth
		}
	}
	return nil
}

// RequestMetadata contains HTTP request information available to actions.
type RequestMetadata struct {
	// Headers contains the HTTP request headers.
	Headers http.Header

	// QueryParams contains the URL query parameters.
	QueryParams url.Values

	// Session contains session data if available.
	Session *Session

	// RawRequest provides access to the underlying http.Request.
	// Use this as an escape hatch when you need features not exposed
	// by the higher-level API.
	RawRequest *http.Request

	// CSRFToken is the ephemeral CSRF token if provided.
	CSRFToken *string

	// Method is the HTTP method (GET, POST, etc.).
	Method string

	// Path is the request URL path.
	Path string

	// RemoteAddr is the client's remote address.
	RemoteAddr string
}

// Session contains session data for the current request.
type Session struct {
	// Data contains arbitrary session data.
	Data map[string]any

	// ID is the session identifier.
	ID string

	// UserID is the authenticated user's ID, empty if not authenticated.
	UserID string
}

// ResponseWriter allows actions to set response metadata like cookies,
// headers, and client-side helper calls.
type ResponseWriter struct {
	// headers stores HTTP response headers to be sent with the response.
	headers http.Header

	// cookies holds HTTP cookies to be sent with the response.
	cookies []*http.Cookie

	// helpers stores the helper function calls recorded by AddHelper.
	helpers []HelperCall

	// mu guards concurrent access to cookies and headers.
	mu sync.Mutex
}

// NewResponseWriter creates a new ResponseWriter.
//
// Returns *ResponseWriter which is ready for capturing HTTP responses.
func NewResponseWriter() *ResponseWriter {
	return &ResponseWriter{
		headers: make(http.Header),
	}
}

// SetCookie adds a cookie to the response.
//
// Takes cookie (*http.Cookie) which specifies the cookie to add.
//
// Safe for concurrent use.
func (w *ResponseWriter) SetCookie(cookie *http.Cookie) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.cookies = append(w.cookies, cookie)
}

// AddHeader adds a header to the response.
// Multiple values can be added for the same key.
//
// Takes key (string) which specifies the header name.
// Takes value (string) which specifies the header value.
//
// Safe for concurrent use.
func (w *ResponseWriter) AddHeader(key, value string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.headers.Add(key, value)
}

// SetHeader sets a header value, replacing any existing values.
//
// Takes key (string) which specifies the header name.
// Takes value (string) which specifies the header value.
//
// Safe for concurrent use.
func (w *ResponseWriter) SetHeader(key, value string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.headers.Set(key, value)
}

// AddHelper queues a client-side helper call to be executed after the action.
// Helpers are functions registered on the client that can perform UI updates.
//
// Takes name (string) which is the name of the helper function to call.
// Takes arguments (...any) which are the arguments to pass to the helper.
//
// Safe for concurrent use.
func (w *ResponseWriter) AddHelper(name string, arguments ...any) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.helpers = append(w.helpers, HelperCall{
		Name: name,
		Args: arguments,
	})
}

// GetCookies returns all cookies set on this response.
//
// Returns []*http.Cookie which contains a copy of all cookies.
//
// Safe for concurrent use.
func (w *ResponseWriter) GetCookies() []*http.Cookie {
	w.mu.Lock()
	defer w.mu.Unlock()
	result := make([]*http.Cookie, len(w.cookies))
	copy(result, w.cookies)
	return result
}

// GetHeaders returns all headers set on this response.
//
// Returns http.Header which is a copy of the response headers.
//
// Safe for concurrent use.
func (w *ResponseWriter) GetHeaders() http.Header {
	w.mu.Lock()
	defer w.mu.Unlock()
	result := make(http.Header, len(w.headers))
	for k, v := range w.headers {
		result[k] = append([]string(nil), v...)
	}
	return result
}

// GetHelpers returns all helper calls queued for this response.
//
// Returns []HelperCall which is a copy of the queued helper calls.
//
// Safe for concurrent use. Returns a copy to prevent data races.
func (w *ResponseWriter) GetHelpers() []HelperCall {
	w.mu.Lock()
	defer w.mu.Unlock()
	result := make([]HelperCall, len(w.helpers))
	copy(result, w.helpers)
	return result
}

// HelperCall represents a client-side helper function call.
type HelperCall struct {
	// Name is the helper function name registered on the client.
	Name string `json:"name"`

	// Args are the arguments to pass to the helper function.
	Args []any `json:"args,omitempty"`
}

// ActionFullResponse is the wire format returned by actions. It wraps the
// action's return data with optional helpers for client-side execution.
type ActionFullResponse struct {
	// Data is the action's return value.
	Data any `json:"data,omitempty"`

	// Helpers are client-side helper calls to be executed after the action.
	Helpers []HelperCall `json:"_helpers,omitempty"`
}
