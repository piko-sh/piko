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

package annotator_dto

import "time"

// ActionSpec represents complete metadata for a parsed action. It implements
// ActionInfoProvider and is the primary output of action discovery, used by
// code generators, the annotator for template validation, and the LSP for
// completions and hover information.
type ActionSpec struct {
	// Call Signature
	// ReturnType describes the response type (first return value).
	// May be nil if the action returns nothing or only error.
	ReturnType *TypeSpec

	// ResourceLimits contains resource constraint configuration.
	ResourceLimits *ResourceLimitSpec

	// CacheConfig contains caching configuration if HasCacheConfig is true.
	CacheConfig *CacheConfigSpec

	// RateLimit contains rate limiting configuration if HasRateLimit is true.
	RateLimit *RateLimitSpec

	// Identity
	// PackagePath is the fully qualified Go package import path.
	PackagePath string

	// Go Metadata
	// PackageName is the Go package name, which is the last component of the
	// package path.
	PackageName string

	// StructName is the name of the action struct, such as "CreateAction".
	StructName string

	// Documentation
	// Description is the godoc comment for the action struct.
	Description string

	// FilePath is the path to the action file relative to the project root.
	FilePath string

	// Name is the action name in dot notation, derived from the file path.
	// For example, "customer.create" comes from actions/customer/create.go.
	Name string

	// TSFunctionName is the camelCase function name for TypeScript output.
	// It is derived from the action path, for example "customerCreate" from
	// "customer.create".
	TSFunctionName string

	// Configuration (from optional interfaces)
	// HTTPMethod is the HTTP method (default: "POST").
	// Set via MethodOverridable interface.
	HTTPMethod string

	// CallParams contains the parameters of the Call method.
	// May be empty if the action takes no parameters.
	CallParams []ParamSpec

	// Transport & Streaming
	// Transports lists the supported transport mechanisms.
	// Derived from interface implementations.
	Transports []Transport

	// HasSSE indicates if the action implements SSECapable.
	HasSSE bool

	// HasRateLimit indicates if the action implements RateLimitable.
	HasRateLimit bool

	// HasMiddlewares indicates if the action implements MiddlewareCapable.
	HasMiddlewares bool

	// HasResourceLimits indicates if the action implements ResourceLimitable.
	HasResourceLimits bool

	// HasCacheConfig indicates if the action implements Cacheable.
	HasCacheConfig bool

	// HasError indicates whether the Call method returns an error.
	// This is almost always true for actions.
	HasError bool
}

// Transport represents a supported transport mechanism.
type Transport string

const (
	// TransportHTTP represents the HTTP transport protocol.
	TransportHTTP Transport = "http"

	// TransportSSE is the Server-Sent Events transport type.
	TransportSSE Transport = "sse"
)

// ParamSpec describes a parameter in an action's Call method signature.
type ParamSpec struct {
	// Struct contains the struct definition if this param is a struct type.
	// nil for primitive types.
	Struct *TypeSpec

	// Name is the parameter name from the Go code.
	Name string

	// GoType is the Go type as a string (e.g., "int64", "string", "*Customer").
	GoType string

	// TSType is the equivalent TypeScript type
	// (e.g., "number", "string", "Customer | null").
	TSType string

	// JSONName is the JSON field name (from json tag or converted from Go name).
	JSONName string

	// Optional indicates if this is a pointer type (nullable in TypeScript).
	Optional bool

	// Special Type Flags
	// These indicate special handling required for this parameter.
	// IsFileUpload indicates this parameter is piko.FileUpload.
	//
	// Behaviour: The wrapper generator will generate multipart form
	// handling code.
	IsFileUpload bool

	// IsFileUploadSlice indicates this parameter is []piko.FileUpload.
	// The wrapper generator will generate multiple file handling code.
	IsFileUploadSlice bool

	// IsRawBody indicates this parameter is piko.RawBody.
	// The handler will pass the raw request body to this parameter.
	IsRawBody bool
}

// TypeSpec describes a struct type used in action parameters or return values.
type TypeSpec struct {
	// Name is the type name (e.g., "CreateInput", "CustomerResponse").
	Name string

	// PackagePath is the fully qualified Go package path where this type is defined.
	PackagePath string

	// PackageName is the declared Go package name (e.g., "arguments" for a
	// package in directory "args"). May differ from the last path segment.
	PackageName string

	// Description is the godoc comment for the type.
	Description string

	// Fields contains the struct fields.
	Fields []FieldSpec
}

// FieldSpec describes a field within a struct type.
type FieldSpec struct {
	// NestedType contains the type spec for nested struct fields.
	// nil for primitive types.
	NestedType *TypeSpec

	// Name is the Go field name.
	Name string

	// GoType is the Go type as a string.
	GoType string

	// TSType is the equivalent TypeScript type.
	TSType string

	// JSONName is the JSON field name (from json tag).
	JSONName string

	// Validation contains the validate tag value if present.
	Validation string

	// Description is the godoc comment for the field.
	Description string

	// Optional indicates if this field is optional (pointer or omitempty).
	Optional bool
}

// RateLimitSpec describes rate limiting configuration for an action.
type RateLimitSpec struct {
	// RequestsPerMinute is the maximum requests allowed per minute.
	RequestsPerMinute int

	// BurstSize is the maximum burst size for the rate limiter.
	BurstSize int

	// HasCustomKeyFunc indicates if the action uses a custom key function.
	HasCustomKeyFunc bool
}

// ResourceLimitSpec describes resource constraints for an action.
type ResourceLimitSpec struct {
	// MaxRequestBodySize is the maximum request body size in bytes.
	MaxRequestBodySize int64

	// MaxResponseSize is the maximum response size in bytes.
	MaxResponseSize int64

	// Timeout is the maximum execution time for the action.
	Timeout time.Duration

	// SlowThreshold is the duration after which a request is considered slow.
	SlowThreshold time.Duration

	// MaxConcurrent is the maximum number of concurrent executions.
	MaxConcurrent int

	// MaxMemoryUsage is the maximum memory usage in bytes.
	MaxMemoryUsage int64

	// SSE-specific limits
	// MaxSSEDuration is the maximum SSE connection duration.
	MaxSSEDuration time.Duration

	// SSEHeartbeatInterval is the interval between SSE heartbeat messages.
	SSEHeartbeatInterval time.Duration
}

// CacheConfigSpec describes caching configuration for an action.
type CacheConfigSpec struct {
	// VaryHeaders lists headers that affect cache key.
	VaryHeaders []string

	// TTL is the cache time-to-live duration.
	TTL time.Duration

	// HasCustomKeyFunc indicates if the action uses a custom cache key function.
	HasCustomKeyFunc bool
}

// Method returns the HTTP method for this action (implements
// ActionInfoProvider).
//
// Returns string which is the HTTP method, defaulting to "POST" if not set.
func (s *ActionSpec) Method() string {
	if s.HTTPMethod == "" {
		return "POST"
	}
	return s.HTTPMethod
}

// GetName returns the action name (for interface compatibility).
//
// Returns string which is the name of this action.
func (s *ActionSpec) GetName() string {
	return s.Name
}

// GetCallParams returns the call parameters (for interface compatibility).
//
// Returns []ParamSpec which contains the parameters for this action.
func (s *ActionSpec) GetCallParams() []ParamSpec {
	return s.CallParams
}

// GetReturnType returns the return type for interface compatibility.
//
// Returns *TypeSpec which is the return type of the action, or nil if none.
func (s *ActionSpec) GetReturnType() *TypeSpec {
	return s.ReturnType
}

// GetDescription returns the action description (for interface compatibility).
//
// Returns string which is the human-readable description of this action.
func (s *ActionSpec) GetDescription() string {
	return s.Description
}
