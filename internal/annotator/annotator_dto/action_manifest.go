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

import (
	"time"

	"piko.sh/piko/internal/ast/ast_domain"
)

// ActionManifest contains all discovered actions from the actions/ directory.
// It is populated during Phase 1 of the annotator pipeline and stored in
// VirtualModule for downstream consumers (generator, typegen, LSP).
type ActionManifest struct {
	// Actions is the list of all discovered actions.
	Actions []ActionDefinition

	// ByName provides fast lookup by dot-notation name (e.g., "email.contact").
	ByName map[string]*ActionDefinition

	// Diagnostics contains any errors or warnings from action discovery.
	Diagnostics []*ast_domain.Diagnostic
}

// NewActionManifest creates an empty ActionManifest.
//
// Returns *ActionManifest which is ready for use with no actions defined.
func NewActionManifest() *ActionManifest {
	return &ActionManifest{
		Actions: make([]ActionDefinition, 0),
		ByName:  make(map[string]*ActionDefinition),
	}
}

// AddAction adds an action to the manifest and updates the lookup map.
//
// Takes action (ActionDefinition) which specifies the action to add.
func (m *ActionManifest) AddAction(action ActionDefinition) {
	m.Actions = append(m.Actions, action)
	m.ByName[action.Name] = &m.Actions[len(m.Actions)-1]
}

// GetAction returns an action by name, or nil if not found.
//
// Takes name (string) which specifies the action to retrieve.
//
// Returns *ActionDefinition which is the matching action, or nil if not found.
func (m *ActionManifest) GetAction(name string) *ActionDefinition {
	return m.ByName[name]
}

// ActionDefinition represents a discovered action from the actions/ directory.
// This is the primary data structure for action metadata throughout the
// pipeline.
type ActionDefinition struct {
	// OutputType describes the Call method return type.
	// May be nil if the action returns nothing or only error.
	OutputType *ActionTypeInfo

	// Capabilities holds optional interface implementations.
	Capabilities ActionCapabilities

	// Go Metadata
	// StructName is the name of the Go struct that implements this action, for
	// example "ContactAction".
	StructName string

	// Identity
	// Name is the dot-notation action name derived from the file path, such as
	// "email.contact" from actions/email/contact.go.
	Name string

	// TSFunctionName is the camelCase function name used in TypeScript code.
	// It is derived from the action name; for example, "email.contact" becomes
	// "emailContact".
	TSFunctionName string

	// FilePath is the path to the action file relative to the project root.
	FilePath string

	// PackagePath is the fully qualified Go package path.
	PackagePath string

	// PackageName is the Go package name, which is the last component of the
	// package path.
	PackageName string

	// Configuration
	// HTTPMethod is the HTTP method (default: "POST").
	// Set via MethodOverridable interface.
	HTTPMethod string

	// Documentation
	// Description is the godoc comment for the action struct.
	Description string

	// Call Signature
	// CallParams describes each parameter of the Call method, in order.
	// May be empty if the action takes no parameters.
	CallParams []ActionTypeInfo

	// StructLine is the 1-based line number of the action struct declaration
	// in the source file. Used for go-to-definition navigation.
	StructLine int

	// HasError indicates whether the Call method returns an error.
	// This is almost always true for actions.
	HasError bool
}

// Method returns the HTTP method for this action.
// Implements the ActionInfoProvider interface for annotator validation.
//
// Returns string which is the HTTP method, defaulting to POST if not set.
func (a *ActionDefinition) Method() string {
	if a.HTTPMethod == "" {
		return "POST"
	}
	return a.HTTPMethod
}

// GetCallParamTypes returns the parameter types of the action's Call method.
// Implements the ActionParamProvider interface for argument validation.
//
// Returns []ActionTypeInfo which describes each parameter, or nil if the
// action takes no parameters.
func (a *ActionDefinition) GetCallParamTypes() []ActionTypeInfo {
	return a.CallParams
}

// ActionCapabilities tracks which optional interfaces an action implements.
// These are detected by looking for specific method signatures on the action
// struct.
type ActionCapabilities struct {
	// RateLimit contains rate limiting configuration if HasRateLimit is true.
	RateLimit *RateLimitConfig

	// ResourceLimits contains resource constraint configuration if
	// HasResourceLimits is true.
	ResourceLimits *ResourceLimitConfig

	// CacheConfig contains caching configuration if HasCacheConfig is true.
	CacheConfig *CacheConfig

	// HasSSE indicates if the action implements SSECapable (StreamProgress
	// method).
	HasSSE bool

	// HasMiddlewares indicates if the action implements MiddlewareCapable
	// (Middlewares method).
	HasMiddlewares bool

	// HasRateLimit indicates whether the action implements RateLimitable
	// (RateLimit method).
	HasRateLimit bool

	// HasResourceLimits indicates if the action implements ResourceLimitable
	// (has a ResourceLimits method).
	HasResourceLimits bool

	// HasCacheConfig indicates if the action implements Cacheable via the
	// CacheConfig method.
	HasCacheConfig bool
}

// ActionTypeInfo describes a type used in action signatures (input or output).
type ActionTypeInfo struct {
	// Name is the type name (e.g., "ContactInput", "ContactResponse").
	Name string

	// ParamName is the original parameter name from the Go method signature
	// (e.g., "input" from "func (a Action) Call(input ContactInput)").
	// Empty for output types.
	ParamName string

	// PackagePath is the fully qualified Go package path where the type is
	// defined.
	PackagePath string

	// PackageName is the declared Go package name (e.g., "arguments" for a
	// package in directory "args"). May differ from the last path segment.
	PackageName string

	// TSType is the equivalent TypeScript type.
	TSType string

	// Description is the godoc comment for the type.
	Description string

	// Fields contains the struct fields (for struct types).
	// Empty for primitive types.
	Fields []ActionFieldInfo

	// IsPointer indicates if this is a pointer type.
	IsPointer bool
}

// ActionFieldInfo describes a field within a struct type.
type ActionFieldInfo struct {
	// NestedType contains the type info for nested struct fields.
	// nil for primitive types.
	NestedType *ActionTypeInfo

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

	// Optional indicates if the field is optional (pointer or omitempty).
	Optional bool
}

// RateLimitConfig describes rate limiting configuration for an action.
type RateLimitConfig struct {
	// RequestsPerMinute is the maximum requests allowed per minute.
	RequestsPerMinute int

	// BurstSize is the maximum burst size for the rate limiter.
	BurstSize int

	// HasCustomKeyFunc indicates if the action uses a custom key function.
	HasCustomKeyFunc bool
}

// ResourceLimitConfig describes resource constraints for an action.
type ResourceLimitConfig struct {
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

// CacheConfig describes caching configuration for an action.
type CacheConfig struct {
	// VaryHeaders lists headers that affect cache key.
	VaryHeaders []string

	// TTL is the cache time-to-live duration.
	TTL time.Duration

	// HasCustomKeyFunc indicates if the action uses a custom cache key function.
	HasCustomKeyFunc bool
}

// ActionCandidate holds preliminary action information before type resolution.
// This is populated during the AST scanning phase (Stage 1.6) and later
// enriched with full type information during Stage 3.5.
type ActionCandidate struct {
	// FilePath is the absolute path to the action file.
	FilePath string

	// RelativePath is the path relative to the project root.
	RelativePath string

	// PackagePath is the fully qualified Go package path.
	PackagePath string

	// PackageName is the Go package name.
	PackageName string

	// StructName is the name of the action struct.
	StructName string

	// ActionName is the derived dot-notation name (e.g., "email.contact").
	ActionName string

	// TSFunctionName is the camelCase TypeScript function name.
	TSFunctionName string

	// DocComment is the godoc comment for the struct.
	DocComment string

	// StructLine is the 1-based line number of the struct declaration in the
	// source file. Captured from the Go AST during discovery.
	StructLine int
}
