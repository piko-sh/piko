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
	"maps"
	"net/http"
	"slices"
	"sync"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

// ASTFunc is the signature for a compiled component's BuildAST function.
// It takes request data and optional props, and returns the final abstract
// syntax tree along with metadata and any runtime diagnostics.
type ASTFunc func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic)

// CachePolicyFunc is a function type that sets the cache policy for a
// component. It receives request data to allow dynamic policies based on the
// request.
type CachePolicyFunc func(r *templater_dto.RequestData) templater_dto.CachePolicy

// MiddlewareFunc is a function type that returns a slice of HTTP middleware
// handlers for a component.
type MiddlewareFunc func() []func(http.Handler) http.Handler

// SupportedLocalesFunc is the signature for a component's SupportedLocales
// function.
type SupportedLocalesFunc func() []string

// AuthPolicyFunc is a function type that returns the authentication
// requirements for a page. It receives request data for consistency
// with the CachePolicyFunc signature pattern.
type AuthPolicyFunc func(r *templater_dto.RequestData) daemon_dto.AuthPolicy

// PreviewFunc is the signature for a component's Preview convention function.
// It returns preview scenarios with sample props for dev-mode rendering.
type PreviewFunc func() []templater_dto.PreviewScenario

// FunctionRegistry provides access to registered template functions.
// It supports both global and isolated instances: the global registry is used
// in production where templates register via init(), while isolated instances
// enable parallel testing without shared state.
type FunctionRegistry interface {
	// RegisterASTFunc registers a BuildAST function for a component.
	//
	// Takes packagePath (string) which identifies the package to associate with the
	// function.
	// Takes registryFunction (ASTFunc) which builds the AST for the specified package.
	RegisterASTFunc(packagePath string, registryFunction ASTFunc)

	// GetASTFunc retrieves the BuildAST function for a component.
	//
	// Takes packagePath (string) which identifies the package to look up.
	//
	// Returns ASTFunc which is the function used to build the AST.
	// Returns bool which indicates whether the function was found.
	GetASTFunc(packagePath string) (ASTFunc, bool)

	// RegisterCachePolicyFunc registers a cache policy
	// function for a package.
	//
	// Takes packagePath (string) which specifies the package
	// path to apply the policy to.
	// Takes registryFunction (CachePolicyFunc) which
	// determines caching behaviour for the package.
	RegisterCachePolicyFunc(packagePath string, registryFunction CachePolicyFunc)

	// GetCachePolicyFunc retrieves the cache policy function for the given
	// package.
	//
	// Takes packagePath (string) which specifies the package path to look up.
	//
	// Returns CachePolicyFunc which determines caching behaviour for the package.
	GetCachePolicyFunc(packagePath string) CachePolicyFunc

	// RegisterMiddlewareFunc registers a middleware function for a package.
	//
	// Takes packagePath (string) which identifies the package to apply middleware to.
	// Takes registryFunction (MiddlewareFunc) which is the middleware to register.
	RegisterMiddlewareFunc(packagePath string, registryFunction MiddlewareFunc)

	// GetMiddlewareFunc retrieves the middleware function for the given package.
	//
	// Takes packagePath (string) which identifies the package to get middleware for.
	//
	// Returns MiddlewareFunc which is the middleware for the specified package.
	GetMiddlewareFunc(packagePath string) MiddlewareFunc

	// RegisterSupportedLocalesFunc registers a function
	// that provides the list of supported locales for a
	// given package.
	//
	// Takes packagePath (string) which identifies the
	// package.
	// Takes registryFunction (SupportedLocalesFunc) which
	// returns the supported locales.
	RegisterSupportedLocalesFunc(packagePath string, registryFunction SupportedLocalesFunc)

	// GetSupportedLocalesFunc retrieves the function that provides supported
	// locales for the given package.
	//
	// Takes packagePath (string) which specifies the package to get locale support
	// for.
	//
	// Returns SupportedLocalesFunc which provides the locale validation for that
	// package.
	GetSupportedLocalesFunc(packagePath string) SupportedLocalesFunc

	// RegisterAuthPolicyFunc registers an auth policy function for a package.
	//
	// Takes packagePath (string) which identifies the package.
	// Takes registryFunction (AuthPolicyFunc) which returns the auth
	// requirements.
	RegisterAuthPolicyFunc(packagePath string, registryFunction AuthPolicyFunc)

	// GetAuthPolicyFunc retrieves the auth policy function for the given
	// package.
	//
	// Takes packagePath (string) which identifies the package.
	//
	// Returns AuthPolicyFunc which provides the auth requirements for that
	// package.
	GetAuthPolicyFunc(packagePath string) AuthPolicyFunc

	// RegisterPreviewFunc registers a Preview function for a package.
	// Preview functions are dev-mode only and provide sample props for
	// component previewing.
	//
	// Takes packagePath (string) which identifies the package.
	// Takes registryFunction (PreviewFunc) which returns preview scenarios.
	RegisterPreviewFunc(packagePath string, registryFunction PreviewFunc)

	// GetPreviewFunc retrieves the Preview function for the given package.
	//
	// Takes packagePath (string) which identifies the package.
	//
	// Returns PreviewFunc which provides preview scenarios.
	// Returns bool which indicates whether the function was found.
	GetPreviewFunc(packagePath string) (PreviewFunc, bool)

	// Unregister removes all functions for a package.
	//
	// Use it for hot-reloading and tests.
	//
	// Takes packagePath (string) which identifies the package to unregister.
	Unregister(packagePath string)

	// Clear removes all registrations. Use it for full rebuilds and test
	// cleanup.
	Clear()

	// List returns all registered package paths.
	List() []string
}

// globalFunctionRegistry implements FunctionRegistry and provides thread-safe
// storage for compiled template functions.
type globalFunctionRegistry struct {
	// astFuncs maps package paths to their AST analysis functions.
	astFuncs map[string]ASTFunc

	// cachePolicyFuncs maps package paths to their cache policy functions.
	cachePolicyFuncs map[string]CachePolicyFunc

	// middlewareFuncs maps package paths to their middleware functions.
	middlewareFuncs map[string]MiddlewareFunc

	// supportedLocalesFuncs maps package paths to their locale provider functions.
	supportedLocalesFuncs map[string]SupportedLocalesFunc

	// authPolicyFuncs maps package paths to their auth policy functions.
	authPolicyFuncs map[string]AuthPolicyFunc

	// previewFuncs maps package paths to their preview functions.
	previewFuncs map[string]PreviewFunc

	// mu guards access to the registry maps for safe concurrent use.
	mu sync.RWMutex
}

var _ FunctionRegistry = (*globalFunctionRegistry)(nil)

// RegisterASTFunc adds an AST function to the registry for the given package
// path.
//
// Takes packagePath (string) which identifies the package this
// function belongs to.
// Takes registryFunction (ASTFunc) which is the AST function to register.
//
// Safe for concurrent use; protected by mu.
func (r *globalFunctionRegistry) RegisterASTFunc(packagePath string, registryFunction ASTFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.astFuncs[packagePath] = registryFunction
}

// GetASTFunc retrieves a registered AST function by its package path.
//
// Takes packagePath (string) which identifies the package to look up.
//
// Returns ASTFunc which is the registered function for the package.
// Returns bool which indicates whether the function was found.
//
// Safe for concurrent use; protected by mu.
func (r *globalFunctionRegistry) GetASTFunc(packagePath string) (ASTFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	registryFunction, ok := r.astFuncs[packagePath]
	return registryFunction, ok
}

// RegisterCachePolicyFunc registers a cache policy function for the given
// package path.
//
// Takes packagePath (string) which identifies the package to associate with the
// function.
// Takes registryFunction (CachePolicyFunc) which determines whether
// results should be cached.
//
// Safe for concurrent use; protected by mu.
func (r *globalFunctionRegistry) RegisterCachePolicyFunc(packagePath string, registryFunction CachePolicyFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cachePolicyFuncs[packagePath] = registryFunction
}

// GetCachePolicyFunc returns the cache policy function for the given package.
//
// Takes packagePath (string) which specifies the package path to look up.
//
// Returns CachePolicyFunc which provides the cache policy for the package, or
// a default function returning a disabled cache policy if none is registered.
//
// Safe for concurrent use; protected by mu.
func (r *globalFunctionRegistry) GetCachePolicyFunc(packagePath string) CachePolicyFunc {
	r.mu.RLock()
	registryFunction, ok := r.cachePolicyFuncs[packagePath]
	r.mu.RUnlock()

	if ok {
		return registryFunction
	}
	return func(_ *templater_dto.RequestData) templater_dto.CachePolicy {
		return templater_dto.CachePolicy{
			MaxAgeSeconds:  0,
			Enabled:        false,
			OnRender:       false,
			Static:         false,
			MustRevalidate: false,
			NoStore:        false,
			Key:            "",
		}
	}
}

// RegisterMiddlewareFunc adds a middleware function to the global registry.
//
// Takes packagePath (string) which identifies the package for this middleware.
// Takes registryFunction (MiddlewareFunc) which is the middleware
// function to register.
//
// Safe for concurrent use; protected by mu.
func (r *globalFunctionRegistry) RegisterMiddlewareFunc(packagePath string, registryFunction MiddlewareFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middlewareFuncs[packagePath] = registryFunction
}

// GetMiddlewareFunc retrieves the middleware function for a package path.
//
// Takes packagePath (string) which specifies the package path to look up.
//
// Returns MiddlewareFunc which is the registered middleware function, or a
// no-op function if none is registered for the given path.
//
// Safe for concurrent use; protected by mu.
func (r *globalFunctionRegistry) GetMiddlewareFunc(packagePath string) MiddlewareFunc {
	r.mu.RLock()
	registryFunction, ok := r.middlewareFuncs[packagePath]
	r.mu.RUnlock()

	if ok {
		return registryFunction
	}
	return func() []func(http.Handler) http.Handler {
		return nil
	}
}

// RegisterSupportedLocalesFunc registers a function that returns supported
// locales for the given package path.
//
// Takes packagePath (string) which identifies the package to register.
// Takes registryFunction (SupportedLocalesFunc) which provides the
// locale lookup function.
//
// Safe for concurrent use; protected by mu.
func (r *globalFunctionRegistry) RegisterSupportedLocalesFunc(packagePath string, registryFunction SupportedLocalesFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.supportedLocalesFuncs[packagePath] = registryFunction
}

// GetSupportedLocalesFunc returns the supported locales function for a package.
//
// Takes packagePath (string) which specifies the package path to look up.
//
// Returns SupportedLocalesFunc which provides the locales for the package, or
// a fallback function returning nil if none is registered.
//
// Safe for concurrent use; protected by mu.
func (r *globalFunctionRegistry) GetSupportedLocalesFunc(packagePath string) SupportedLocalesFunc {
	r.mu.RLock()
	registryFunction, ok := r.supportedLocalesFuncs[packagePath]
	r.mu.RUnlock()

	if ok {
		return registryFunction
	}
	return func() []string {
		return nil
	}
}

// RegisterAuthPolicyFunc registers an auth policy function for a
// package.
//
// Takes packagePath (string) which identifies the package.
// Takes registryFunction (AuthPolicyFunc) which returns auth
// requirements.
//
// Safe for concurrent use; protected by mu.
func (r *globalFunctionRegistry) RegisterAuthPolicyFunc(packagePath string, registryFunction AuthPolicyFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.authPolicyFuncs[packagePath] = registryFunction
}

// GetAuthPolicyFunc retrieves the auth policy function for the given
// package. Returns a no-op function when no auth policy is
// registered.
//
// Takes packagePath (string) which identifies the package.
//
// Returns AuthPolicyFunc which provides the auth requirements.
//
// Safe for concurrent use; protected by mu.
func (r *globalFunctionRegistry) GetAuthPolicyFunc(packagePath string) AuthPolicyFunc {
	r.mu.RLock()
	registryFunction, ok := r.authPolicyFuncs[packagePath]
	r.mu.RUnlock()

	if ok {
		return registryFunction
	}
	return func(_ *templater_dto.RequestData) daemon_dto.AuthPolicy {
		return daemon_dto.AuthPolicy{}
	}
}

// RegisterPreviewFunc registers a Preview function for the given package path.
//
// Takes packagePath (string) which identifies the package.
// Takes registryFunction (PreviewFunc) which returns preview scenarios.
//
// Safe for concurrent use; protected by mu.
func (r *globalFunctionRegistry) RegisterPreviewFunc(packagePath string, registryFunction PreviewFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.previewFuncs[packagePath] = registryFunction
}

// GetPreviewFunc retrieves the Preview function for a package path.
//
// Takes packagePath (string) which specifies the package path to look up.
//
// Returns PreviewFunc which provides preview scenarios for the package.
// Returns bool which indicates whether a preview function was found.
//
// Safe for concurrent use; protected by mu.
func (r *globalFunctionRegistry) GetPreviewFunc(packagePath string) (PreviewFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	registryFunction, ok := r.previewFuncs[packagePath]
	return registryFunction, ok
}

// Unregister removes all registered functions for the given package path.
//
// Takes packagePath (string) which identifies the package to unregister.
//
// Safe for concurrent use; protected by mu.
func (r *globalFunctionRegistry) Unregister(packagePath string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.astFuncs, packagePath)
	delete(r.cachePolicyFuncs, packagePath)
	delete(r.middlewareFuncs, packagePath)
	delete(r.supportedLocalesFuncs, packagePath)
	delete(r.authPolicyFuncs, packagePath)
	delete(r.previewFuncs, packagePath)
}

// Clear removes all registered functions from the registry.
//
// Safe for concurrent use; protected by mu.
func (r *globalFunctionRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	clear(r.astFuncs)
	clear(r.cachePolicyFuncs)
	clear(r.middlewareFuncs)
	clear(r.supportedLocalesFuncs)
	clear(r.authPolicyFuncs)
	clear(r.previewFuncs)
}

// List returns all registered package paths from the function registry.
//
// Returns []string which contains the unique package paths across all
// registered function types.
//
// Safe for concurrent use; protected by mu.
func (r *globalFunctionRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pkgMap := make(map[string]struct{})
	for packagePath := range r.astFuncs {
		pkgMap[packagePath] = struct{}{}
	}
	for packagePath := range r.cachePolicyFuncs {
		pkgMap[packagePath] = struct{}{}
	}
	for packagePath := range r.middlewareFuncs {
		pkgMap[packagePath] = struct{}{}
	}
	for packagePath := range r.supportedLocalesFuncs {
		pkgMap[packagePath] = struct{}{}
	}

	return slices.Collect(maps.Keys(pkgMap))
}

// defaultRegistry is the global instance used in production.
// Compiled templates register themselves here via init() functions.
var defaultRegistry = &globalFunctionRegistry{
	astFuncs:              make(map[string]ASTFunc),
	cachePolicyFuncs:      make(map[string]CachePolicyFunc),
	middlewareFuncs:       make(map[string]MiddlewareFunc),
	supportedLocalesFuncs: make(map[string]SupportedLocalesFunc),
	authPolicyFuncs:       make(map[string]AuthPolicyFunc),
	previewFuncs:          make(map[string]PreviewFunc),
	mu:                    sync.RWMutex{},
}

// RegisterASTFunc registers an AST building function for a component package.
//
// Takes packagePath (string) which identifies the component package path.
// Takes registryFunction (ASTFunc) which provides the AST building
// function to register.
func RegisterASTFunc(packagePath string, registryFunction ASTFunc) {
	defaultRegistry.RegisterASTFunc(packagePath, registryFunction)
}

// RegisterCachePolicyFunc registers a cache policy function for a given component
// package. Safe for use from multiple goroutines.
//
// Takes packagePath (string) which specifies the component package path.
// Takes registryFunction (CachePolicyFunc) which provides the cache policy logic.
func RegisterCachePolicyFunc(packagePath string, registryFunction CachePolicyFunc) {
	defaultRegistry.RegisterCachePolicyFunc(packagePath, registryFunction)
}

// RegisterMiddlewareFunc registers a middleware function for a given component
// package. Safe for use by multiple goroutines.
//
// Takes packagePath (string) which specifies the component package path.
// Takes registryFunction (MiddlewareFunc) which provides the
// middleware to register.
func RegisterMiddlewareFunc(packagePath string, registryFunction MiddlewareFunc) {
	defaultRegistry.RegisterMiddlewareFunc(packagePath, registryFunction)
}

// RegisterSupportedLocalesFunc registers the SupportedLocales function for a
// given component package. Safe for use by multiple goroutines.
//
// Takes packagePath (string) which identifies the component package.
// Takes registryFunction (SupportedLocalesFunc) which provides the
// locale support check.
func RegisterSupportedLocalesFunc(packagePath string, registryFunction SupportedLocalesFunc) {
	defaultRegistry.RegisterSupportedLocalesFunc(packagePath, registryFunction)
}

// RegisterAuthPolicyFunc registers an auth policy function for a component
// package. Called from generated init() functions.
//
// Takes packagePath (string) which identifies the component package path.
// Takes registryFunction (AuthPolicyFunc) which returns auth requirements.
func RegisterAuthPolicyFunc(packagePath string, registryFunction AuthPolicyFunc) {
	defaultRegistry.RegisterAuthPolicyFunc(packagePath, registryFunction)
}

// RegisterPreviewFunc registers a Preview function for a component package.
// Called from generated init() functions in dev mode.
//
// Takes packagePath (string) which identifies the component package path.
// Takes registryFunction (PreviewFunc) which returns preview scenarios.
func RegisterPreviewFunc(packagePath string, registryFunction PreviewFunc) {
	defaultRegistry.RegisterPreviewFunc(packagePath, registryFunction)
}

// Unregister removes all function pointers for the given package path.
// This is needed for hot-reloading to clear old functions before new ones
// are added.
//
// Takes packagePath (string) which specifies the package path to unregister.
func Unregister(packagePath string) {
	defaultRegistry.Unregister(packagePath)
}

// Clear resets all registry maps to their initial empty state. Use it for
// full project rebuilds in watch mode.
func Clear() {
	defaultRegistry.Clear()
}

// GetASTFunc retrieves the AST function for a given package path.
//
// Takes packagePath (string) which specifies the package path to look up.
//
// Returns ASTFunc which is the function linked to the package path.
// Returns bool which indicates whether the function was found.
func GetASTFunc(packagePath string) (ASTFunc, bool) {
	return defaultRegistry.GetASTFunc(packagePath)
}

// GetCachePolicyFunc retrieves the cache policy function for a given package
// path.
//
// Takes packagePath (string) which specifies the package path to look up.
//
// Returns CachePolicyFunc which is the cache policy for the specified package,
// or a safe no-op default if not found.
func GetCachePolicyFunc(packagePath string) CachePolicyFunc {
	return defaultRegistry.GetCachePolicyFunc(packagePath)
}

// GetMiddlewareFunc retrieves the MiddlewareFunc for a given package path.
// If not found, it returns a safe no-op default.
//
// Takes packagePath (string) which specifies the package path to look up.
//
// Returns MiddlewareFunc which is the registered middleware or a no-op default.
func GetMiddlewareFunc(packagePath string) MiddlewareFunc {
	return defaultRegistry.GetMiddlewareFunc(packagePath)
}

// GetSupportedLocalesFunc retrieves the SupportedLocalesFunc for a given
// package path. If not found, it returns a safe, no-op default.
//
// Takes packagePath (string) which specifies the package path to look up.
//
// Returns SupportedLocalesFunc which provides the locale function for the
// given package, or a no-op default if not found.
func GetSupportedLocalesFunc(packagePath string) SupportedLocalesFunc {
	return defaultRegistry.GetSupportedLocalesFunc(packagePath)
}

// List returns all registered package paths.
//
// Returns []string which contains the paths of all registered packages.
func List() []string {
	return defaultRegistry.List()
}

// GetDefaultRegistry returns the global registry instance. Adapters use it to
// access the registry directly, such as when using dependency injection.
//
// Returns FunctionRegistry which provides access to template functions.
func GetDefaultRegistry() FunctionRegistry {
	return defaultRegistry
}

// NewIsolatedRegistry creates a new isolated function registry for testing.
// Each test can use its own registry to prevent test pollution and allow
// parallel test execution with t.Parallel().
//
// Returns FunctionRegistry which is an empty registry ready for use.
func NewIsolatedRegistry() FunctionRegistry {
	return &globalFunctionRegistry{
		astFuncs:              make(map[string]ASTFunc),
		cachePolicyFuncs:      make(map[string]CachePolicyFunc),
		middlewareFuncs:       make(map[string]MiddlewareFunc),
		supportedLocalesFuncs: make(map[string]SupportedLocalesFunc),
		authPolicyFuncs:       make(map[string]AuthPolicyFunc),
		previewFuncs:          make(map[string]PreviewFunc),
		mu:                    sync.RWMutex{},
	}
}
