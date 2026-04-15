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

package piko

import (
	"piko.sh/piko/internal/daemon/daemon_adapters"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

// RequestData encapsulates all information about an incoming HTTP request.
// It is made available within a component's Render function and includes the
// request context, URL, headers, and parsed form/query data.
type RequestData = templater_dto.RequestData

// Metadata allows a component's Render function to return SEO information,
// caching policies, and other page-level metadata that influences the final
// HTTP response.
type Metadata = templater_dto.Metadata

// OGTag represents an Open Graph protocol <meta> tag for social media sharing
// (e.g., <meta property="og:title" content="...">).
type OGTag = templater_dto.OGTag

// MetaTag represents an HTML <meta> tag used for page metadata
// (e.g., <meta name="author" content="...">).
type MetaTag = templater_dto.MetaTag

// NoProps is a placeholder type for components that do not accept any
// properties. It is used in the Render function signature when a component is
// self-contained.
type NoProps = templater_dto.NoProps

// NoResponse is a placeholder type for Render functions that do not return any
// data for the template. The template can still access global state such as the
// request object.
type NoResponse = templater_dto.NoResponse

// CachePolicy defines the caching behaviour for a component's output.
type CachePolicy = templater_dto.CachePolicy

// HTTPMethod defines the valid HTTP methods for an action.
type HTTPMethod = daemon_domain.HTTPMethod

// CookieOption is a functional option for configuring cookies created with
// Cookie().
type CookieOption = daemon_dto.CookieOption

// AuthContext represents the resolved authentication state for a request.
// Users implement this interface to expose their auth system's data to
// Piko pages and actions.
type AuthContext = daemon_dto.AuthContext

// AuthProvider resolves authentication state from an HTTP request.
// Piko calls Authenticate on every request when a provider is registered.
type AuthProvider = daemon_dto.AuthProvider

// AuthPolicy declares authentication requirements for a page.
// Returned by the optional AuthPolicy() function in PK files.
type AuthPolicy = daemon_dto.AuthPolicy

// PreviewScenario describes a single named preview scenario for a component.
// The Preview() convention function returns a slice of these, providing sample
// data for rendering the component in the dev tools preview panel.
type PreviewScenario = templater_dto.PreviewScenario

// AuthGuardConfig controls prefix-level and page-level authentication
// enforcement.
type AuthGuardConfig = daemon_dto.AuthGuardConfig

// ActionMetadata is embedded in action structs to provide access to
// request context, metadata, and response building capabilities.
//
// Example:
//
//	type DeleteAction struct {
//	    piko.ActionMetadata
//	}
//
//	func (a DeleteAction) Call(id int64) (DeleteResponse, error) {
//	    ctx := a.Ctx()
//	    userID := a.Request().Session.UserID
//	    a.Response().AddHelper("showToast", "Deleted", "success")
//	    return DeleteResponse{Success: true}, nil
//	}
type ActionMetadata = daemon_dto.ActionMetadata

// RequestMetadata contains HTTP request information available to actions.
type RequestMetadata = daemon_dto.RequestMetadata

// ResponseWriter allows actions to set response metadata.
type ResponseWriter = daemon_dto.ResponseWriter

// Session contains session data for the current request.
type Session = daemon_dto.Session

// HelperCall represents a client-side helper function call.
type HelperCall = daemon_dto.HelperCall

// SSECapable is an interface that actions can implement to support
// Server-Sent Events (SSE) streaming for progressive updates.
//
// Example:
//
//	func (a UploadAction) StreamProgress(stream *piko.SSEStream) error {
//	    for progress := range uploadWithProgress(a.File) {
//	        stream.Send("progress", progress)
//	    }
//	    return stream.SendComplete(UploadResponse{URL: finalURL})
//	}
type SSECapable = daemon_domain.SSECapable

// SSEStream provides methods for sending Server-Sent Events.
type SSEStream = daemon_domain.SSEStream

// Transport represents a supported transport mechanism for actions.
type Transport = daemon_domain.Transport

const (
	// TransportHTTP is the HTTP transport protocol identifier.
	TransportHTTP = daemon_domain.TransportHTTP

	// TransportSSE is the Server-Sent Events transport type.
	TransportSSE = daemon_domain.TransportSSE

	// MethodGet is the HTTP GET method identifier.
	MethodGet = daemon_domain.MethodGet

	// MethodPost is the HTTP POST method constant.
	MethodPost = daemon_domain.MethodPost

	// MethodPut is the HTTP PUT method for updating resources.
	MethodPut = daemon_domain.MethodPut

	// MethodDelete is the HTTP DELETE method identifier.
	MethodDelete = daemon_domain.MethodDelete

	// MethodPatch is the HTTP PATCH method for partial resource updates.
	MethodPatch = daemon_domain.MethodPatch

	// MethodHead is the HTTP HEAD method identifier.
	MethodHead = daemon_domain.MethodHead

	// MethodOptions is an alias for daemon_domain.MethodOptions.
	MethodOptions = daemon_domain.MethodOptions
)

// ResourceLimitable is an interface that actions can implement to configure
// resource limits for protection against resource exhaustion.
type ResourceLimitable = daemon_domain.ResourceLimitable

// ResourceLimits defines resource constraints for an action.
type ResourceLimits = daemon_domain.ResourceLimits

// Cacheable is an interface for configuring response caching.
type Cacheable = daemon_domain.Cacheable

// CacheConfig defines caching behaviour for action responses.
type CacheConfig = daemon_domain.CacheConfig

// CaptchaProtected is an interface for requiring captcha verification.
type CaptchaProtected = daemon_domain.CaptchaProtected

// CaptchaConfig defines captcha verification configuration.
type CaptchaConfig = daemon_domain.CaptchaConfig

// RateLimitable is an interface for configuring rate limiting.
type RateLimitable = daemon_domain.RateLimitable

// RateLimit defines rate limiting configuration.
type RateLimit = daemon_domain.RateLimit

// RateLimitKeyFunc extracts a rate limit key from a request.
type RateLimitKeyFunc = daemon_domain.RateLimitKeyFunc

var (
	// RateLimitByIP is a rate limiting strategy that limits requests by IP
	// address.
	RateLimitByIP = daemon_domain.RateLimitByIP

	// RateLimitByUser is a rate limiting strategy that tracks limits per user.
	RateLimitByUser = daemon_domain.RateLimitByUser

	// RateLimitBySession is the rate limiting strategy that tracks requests per
	// session.
	RateLimitBySession = daemon_domain.RateLimitBySession

	// NewFileUpload creates a FileUpload from a multipart.FileHeader.
	// This is called by generated wrapper code; users typically don't need this.
	NewFileUpload = daemon_dto.NewFileUpload

	// NewRawBody creates a RawBody from raw data.
	// This is called by the action handler; users typically don't need this.
	NewRawBody = daemon_dto.NewRawBody

	// NewValidationError creates a validation error with
	// field-specific messages.
	//
	// Example:
	// return nil, piko.NewValidationError(map[string]string{
	//     "email": "invalid email format",
	//     "age":   "must be at least 18",
	// })
	NewValidationError = daemon_dto.NewValidationError

	// ValidationField creates a single-field validation error. This is a helper
	// for the common case of one field failing validation.
	//
	// Example:
	// return nil, piko.ValidationField("email", "invalid email format")
	ValidationField = daemon_dto.ValidationField

	// NotFound creates a not found error for a resource.
	//
	// Example:
	// return nil, piko.NotFound("user", userID)
	NotFound = daemon_dto.NotFound

	// NotFoundResource creates a not found error without a specific ID.
	// Use this when the resource type is known but there's no specific identifier.
	//
	// Example:
	// return nil, piko.NotFoundResource("configuration")
	NotFoundResource = daemon_dto.NotFoundResource

	// Conflict creates a conflict error with the default CONFLICT code.
	//
	// Example:
	// return nil, piko.Conflict("email already registered")
	Conflict = daemon_dto.Conflict

	// ConflictWithCode creates a conflict error with a custom error code. Use this
	// when the client needs to discriminate between different conflict types.
	//
	// Example:
	// return nil, piko.ConflictWithCode("email already registered", "EMAIL_EXISTS")
	ConflictWithCode = daemon_dto.ConflictWithCode

	// Forbidden creates a forbidden error.
	//
	// Example:
	// return nil, piko.Forbidden("you do not have permission to delete this resource")
	Forbidden = daemon_dto.Forbidden

	// Unauthorised creates an unauthorised error.
	//
	// Example:
	// return nil, piko.Unauthorised("session expired")
	Unauthorised = daemon_dto.Unauthorised

	// BadRequest creates a bad request error.
	//
	// Example:
	// return nil, piko.BadRequest("missing required header: X-Request-ID")
	BadRequest = daemon_dto.BadRequest

	// PageError creates a generic page error with an arbitrary HTTP status
	// code. Use this when none of the specific error helpers match.
	//
	// Example:
	// return Response{}, piko.Metadata{}, piko.PageError(429, "too many requests")
	PageError = daemon_dto.PageError

	// Teapot creates an HTTP 418 I'm a Teapot error. Short and stout.
	//
	// Example:
	//	return Response{}, piko.Metadata{},
	//		piko.Teapot("this server is a teapot, not a coffee machine")
	Teapot = daemon_dto.Teapot

	// SessionCookie creates a secure session cookie with sensible defaults.
	//
	// The cookie is set up with:
	//   - HttpOnly: true (stops JavaScript access, protects against XSS)
	//   - Secure: true (HTTPS only - use SessionCookieInsecure for local work)
	//   - SameSite: Lax (protects against CSRF while allowing normal browsing)
	//   - Path: "/" (works for the whole site)
	//
	// Example usage in a login action:
	// return piko.ActionResponse{
	//     Status:  200,
	//     Message: "Login successful",
	//     Cookies: []*http.Cookie{
	//         piko.SessionCookie("pp_session", session.ID.String(), session.ExpiresAt),
	//     },
	// }
	SessionCookie = daemon_dto.SessionCookie

	// SessionCookieInsecure is a session cookie option that allows HTTP
	// connections. Use this only for local development where HTTPS is not
	// available.
	//
	// Warning: Do not use in production. Cookies will be sent as plain text.
	SessionCookieInsecure = daemon_dto.SessionCookieInsecure

	// ClearCookie creates a cookie that instructs the browser to delete an
	// existing cookie.
	//
	// Example usage in a logout action:
	// return piko.ActionResponse{
	//     Status:  200,
	//     Message: "Logged out",
	//     Cookies: []*http.Cookie{
	//         piko.ClearCookie("pp_session"),
	//     },
	// }
	ClearCookie = daemon_dto.ClearCookie

	// ClearCookieInsecure creates a cookie deletion instruction that works over
	// HTTP. Use this only for local development where HTTPS is not available.
	ClearCookieInsecure = daemon_dto.ClearCookieInsecure

	// SmartSessionCookie creates a session cookie that automatically sets
	// the Secure flag based on the runtime environment. In production
	// Secure is true; in development Secure is false.
	SmartSessionCookie = daemon_dto.SmartSessionCookie

	// SmartClearCookie creates a clear-cookie that automatically adapts
	// the Secure flag to match the runtime environment.
	SmartClearCookie = daemon_dto.SmartClearCookie

	// Cookie is a helper that creates a cookie with custom settings.
	// Use this when you need more control than SessionCookie gives.
	//
	// Example:
	// return piko.ActionResponse{
	//     Status: 200,
	//     Cookies: []*http.Cookie{
	//         piko.Cookie("preferences", prefs, time.Hour*24*365,
	//             piko.WithPath("/settings"),
	//             piko.WithSameSiteStrict(),
	//         ),
	//     },
	// }
	Cookie = daemon_dto.Cookie

	// WithPath sets the path attribute for a cookie.
	WithPath = daemon_dto.WithPath

	// WithDomain sets the cookie domain.
	WithDomain = daemon_dto.WithDomain

	// WithInsecure allows the cookie to be sent over HTTP.
	// WARNING: Only use for local development.
	WithInsecure = daemon_dto.WithInsecure

	// WithJavaScriptAccess allows JavaScript to access the cookie (removes
	// HttpOnly). WARNING: Only use if the cookie does not contain sensitive data.
	WithJavaScriptAccess = daemon_dto.WithJavaScriptAccess

	// WithSameSiteStrict sets the cookie to SameSite=Strict mode. This gives the
	// strongest CSRF protection but may stop some normal website features from
	// working correctly.
	WithSameSiteStrict = daemon_dto.WithSameSiteStrict

	// WithSameSiteNone sets the cookie to SameSite=None mode, permitting
	// cross-site requests but requiring Secure to be true.
	WithSameSiteNone = daemon_dto.WithSameSiteNone
)

// ActionHandlerEntry describes a registered action for the runtime.
// Generated code creates entries for each action and registers them at startup.
type ActionHandlerEntry = daemon_adapters.ActionHandlerEntry

// FileUpload represents an uploaded file from a multipart form request.
// Use this type in action Call parameters to receive file uploads.
//
// Example:
//
//	func (a *UploadAction) Call(document piko.FileUpload) (Response, error) {
//	    content, err := document.ReadAll()
//	    if err != nil {
//	        return Response{}, err
//	    }
//	    return Response{Size: document.Size}, nil
//	}
//
// For multiple files, use []piko.FileUpload:
//
//	func (a *UploadAction) Call(documents []piko.FileUpload) (Response, error) {
//	    for _, doc := range documents {
//	        // Process each file...
//	    }
//	    return Response{Count: len(documents)}, nil
//	}
type FileUpload = daemon_dto.FileUpload

// RawBody provides access to the unparsed request body.
// Use this when you need to verify signatures or parse custom formats.
//
// Example:
//
//	func (a *WebhookAction) Call(body piko.RawBody) (Response, error) {
//	    signature := a.Request().RawRequest.Header.Get("X-Signature")
//	    if !verifySignature(body.Bytes(), signature) {
//	        return Response{}, piko.Unauthorised("invalid signature")
//	    }
//	    return Response{OK: true}, nil
//	}
type RawBody = daemon_dto.RawBody

// ActionError is the base interface for action errors with HTTP semantics.
// Errors implementing this interface are automatically discriminated by the
// action handler to return appropriate HTTP status codes and structured
// responses.
type ActionError = daemon_dto.ActionError

// ValidationError represents validation failures (HTTP 422).
// Use this when user input fails validation rules.
type ValidationError = daemon_dto.ValidationError

// NotFoundError represents resource not found (HTTP 404).
// Use this when a requested resource does not exist.
type NotFoundError = daemon_dto.NotFoundError

// ConflictError represents a conflict (HTTP 409). Use this when an operation
// cannot complete due to a conflict with current state.
type ConflictError = daemon_dto.ConflictError

// ForbiddenError represents authorisation failure (HTTP 403).
// Use this when the user is authenticated but lacks permission.
type ForbiddenError = daemon_dto.ForbiddenError

// UnauthorisedError represents authentication failure (HTTP 401).
// Use this when the user needs to authenticate.
type UnauthorisedError = daemon_dto.UnauthorisedError

// BadRequestError represents a malformed request (HTTP 400).
// Use this when the request itself is invalid, not just the data within it.
type BadRequestError = daemon_dto.BadRequestError

// GenericPageError represents an error with an arbitrary HTTP status code.
// Use this when none of the specific error types match the status code you
// need.
type GenericPageError = daemon_dto.GenericPageError

// TeapotError represents HTTP 418 I'm a Teapot (RFC 2324).
type TeapotError = daemon_dto.TeapotError

// ErrorPageContext carries error details into a custom error page's Render
// function. When a page returns an error (or a route is not found), the
// runtime injects this context before rendering the matching error page.
//
// Access it via GetErrorContext in your error page's Render function:
//
//	func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
//	    if errCtx := piko.GetErrorContext(r); errCtx != nil {
//	        // errCtx.StatusCode, errCtx.Message, errCtx.OriginalPath
//	    }
//	}
type ErrorPageContext = daemon_dto.ErrorPageContext

// RegisterActions registers multiple actions with the global action registry.
// This is called by generated code in init() functions to auto-register
// actions.
//
// Takes entries (map[string]ActionHandlerEntry) which maps action names
// (e.g., "email.contact") to their handler entries.
func RegisterActions(entries map[string]ActionHandlerEntry) {
	daemon_adapters.RegisterActions(entries)
}

// GetErrorContext returns the error page context from the request, if
// present, providing the status code, message, and original request
// path for use inside error page Render functions.
//
// Takes r (*RequestData) which is the current request to check for error
// context.
//
// Returns nil when the request is not being rendered as an error page.
//
// Example:
//
//	func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
//	    errCtx := piko.GetErrorContext(r)
//	    if errCtx != nil {
//	        return Response{Code: errCtx.StatusCode, Message: errCtx.Message}, piko.Metadata{}, nil
//	    }
//	    return Response{Code: 500, Message: "Unknown error"}, piko.Metadata{}, nil
//	}
func GetErrorContext(r *RequestData) *ErrorPageContext {
	if r == nil {
		return nil
	}
	ctx := r.Context()
	if ctx == nil {
		return nil
	}
	epc, ok := daemon_dto.GetErrorPageContext(ctx)
	if !ok {
		return nil
	}
	return &epc
}
