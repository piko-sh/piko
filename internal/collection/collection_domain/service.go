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

package collection_domain

import (
	"context"
	"fmt"
	"go/ast"
	"sync"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// logKeyProvider is the log key for the collection provider name.
	logKeyProvider = "provider"

	// logKeyItemCount is the log key for the number of items.
	logKeyItemCount = "item_count"

	// maxASTPreviewLength limits the AST string length for debug logging.
	maxASTPreviewLength = 200
)

// collectionService implements the CollectionService interface.
//
// This is the core domain service that manages provider interactions. It acts
// as a facade between other hexagons and the provider system.
type collectionService struct {
	// registry provides access to collection providers by name.
	registry ProviderRegistryPort

	// hybridRegistry stores snapshots of provider collections for runtime access.
	hybridRegistry HybridRegistryPort

	// encoder converts collection items to FlatBuffer format.
	encoder CollectionEncoderPort

	// clock provides time operations for health checks and timestamps.
	clock clock.Clock

	// defaultSandbox provides filesystem access for local content collections.
	defaultSandbox safedisk.Sandbox

	// resolver provides module resolution for external content sources
	// (p-collection-source directives).
	resolver resolver_domain.ResolverPort

	// cache stores content items by provider and collection name.
	cache map[string][]collection_dto.ContentItem

	// externalSandboxes tracks sandboxes created for external module content
	// sources so they can be closed when the service shuts down.
	externalSandboxes []safedisk.Sandbox

	// cacheMutex guards concurrent access to the cache map.
	cacheMutex sync.RWMutex

	// sandboxMutex guards concurrent access to externalSandboxes.
	sandboxMutex sync.Mutex
}

// CollectionServiceOption is a functional option for configuring
// CollectionService.
type CollectionServiceOption func(*collectionService)

// ProcessCollectionDirective expands a p-collection directive into entry
// points.
//
// This method is the public interface called by the Annotator. It uses DTO
// types for cross-hexagon communication.
//
// Takes directive (*collection_dto.CollectionDirectiveInfo) which specifies the
// collection to expand, including provider name, collection name, and layout.
//
// Returns []*collection_dto.CollectionEntryPoint which contains the expanded
// entry points for the collection.
// Returns error when the provider is unknown or has an unrecognised type.
func (s *collectionService) ProcessCollectionDirective(
	ctx context.Context,
	directive *collection_dto.CollectionDirectiveInfo,
) ([]*collection_dto.CollectionEntryPoint, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Processing collection directive",
		logger_domain.String(logKeyProvider, directive.ProviderName),
		logger_domain.String(logKeyCollection, directive.CollectionName),
		logger_domain.String("layout", directive.LayoutPath))

	provider, ok := s.registry.Get(directive.ProviderName)
	if !ok {
		availableProviders := s.registry.List()
		return nil, fmt.Errorf(
			"%w: unknown collection provider '%s'; available providers: %v",
			collection_dto.ErrProviderNotFound,
			directive.ProviderName,
			availableProviders,
		)
	}

	switch provider.Type() {
	case ProviderTypeStatic:
		return s.expandStaticCollection(ctx, provider, directive)

	case ProviderTypeDynamic:
		return s.expandDynamicCollection(ctx, provider, directive)

	case ProviderTypeHybrid:
		return s.expandHybridCollection(ctx, provider, directive)

	default:
		return nil, fmt.Errorf(
			"unknown provider type '%s' for provider '%s'",
			provider.Type(),
			provider.Name(),
		)
	}
}

// ProcessGetCollectionCall handles data.GetCollection() in user code.
//
// This is called by the Annotator when it encounters a GetCollection() call.
// It receives semantic information extracted from the Piko AST and generates
// the appropriate annotation for the Generator based on provider type.
//
// Takes collectionName (string) which identifies the collection to retrieve.
// Takes targetTypeName (string) which specifies the name of the target type.
// Takes targetTypeExpr (ast.Expr) which provides the AST expression for the
// target type.
// Takes optionsRaw (any) which contains provider-specific options.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the generated
// annotation for the code generator.
// Returns error when options parsing fails, the provider is not found, or the
// target type is invalid for the provider.
func (s *collectionService) ProcessGetCollectionCall(
	ctx context.Context,
	collectionName string,
	targetTypeName string,
	targetTypeExpr ast.Expr,
	optionsRaw any,
) (*ast_domain.GoGeneratorAnnotation, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Processing GetCollection() call",
		logger_domain.String(logKeyCollection, collectionName),
		logger_domain.String("targetType", targetTypeName))

	options, err := s.parseGetCollectionOptions(optionsRaw)
	if err != nil {
		return nil, fmt.Errorf("parsing GetCollection options for %q: %w", collectionName, err)
	}

	providerName := s.resolveProviderName(ctx, &options)
	provider, err := s.lookupProvider(providerName)
	if err != nil {
		return nil, fmt.Errorf("looking up provider %q for collection %q: %w", providerName, collectionName, err)
	}

	if err := provider.ValidateTargetType(targetTypeExpr); err != nil {
		return nil, fmt.Errorf("invalid target type for provider %s: %w", providerName, err)
	}

	return s.dispatchProviderAnnotation(ctx, provider, collectionName, targetTypeExpr, &options)
}

// ValidateConfiguration checks all provider configurations at startup.
//
// This method validates:
//  1. Default provider is registered
//  2. All explicitly configured providers are registered
//  3. All collection providers are registered
//
// Call this at application startup to fail fast on misconfiguration.
//
// Takes config (*Config) which specifies the providers and collections to
// validate.
//
// Returns error when any provider or collection configuration is invalid.
func (s *collectionService) ValidateConfiguration(
	ctx context.Context,
	config *Config,
) error {
	ctx, l := logger_domain.From(ctx, log)
	if config == nil {
		l.Internal("No configuration provided, skipping validation")
		return nil
	}

	l.Internal("Validating collection configuration",
		logger_domain.String("default_provider", config.DefaultProvider),
		logger_domain.Int("provider_count", len(config.Providers)),
		logger_domain.Int("collection_count", len(config.Collections)))

	var validationErrors []string
	validationErrors = s.validateDefaultProvider(config, validationErrors)
	validationErrors = s.validateProviders(config, validationErrors)
	validationErrors = s.validateCollections(config, validationErrors)

	if len(validationErrors) > 0 {
		return s.reportValidationErrors(ctx, validationErrors)
	}

	l.Internal("Collection configuration validation successful")
	return nil
}

// Close releases resources held by the service, including any sandboxes
// created for external module content sources.
//
// Returns error when one or more sandbox closes fail. The first error is
// returned, but all sandboxes are attempted.
//
// Concurrency: acquires sandboxMutex to drain the external sandboxes list.
func (s *collectionService) Close() error {
	s.sandboxMutex.Lock()
	sandboxes := s.externalSandboxes
	s.externalSandboxes = nil
	s.sandboxMutex.Unlock()

	var firstErr error
	for _, sb := range sandboxes {
		if err := sb.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// validateDefaultProvider checks that the default provider is registered.
//
// Takes config (*Config) which contains the default provider name to validate.
// Takes errors ([]string) which accumulates validation errors.
//
// Returns []string which is the updated slice with any validation error added.
func (s *collectionService) validateDefaultProvider(config *Config, errors []string) []string {
	if config.DefaultProvider != "" && !s.registry.Has(config.DefaultProvider) {
		errors = append(errors,
			fmt.Sprintf("default provider %q is not registered", config.DefaultProvider))
	}
	return errors
}

// validateProviders checks that all explicitly configured providers are
// registered.
//
// Takes config (*Config) which contains the provider configurations to check.
// Takes errors ([]string) which is the existing error list to append to.
//
// Returns []string which is the updated error list with any validation errors.
func (s *collectionService) validateProviders(config *Config, errors []string) []string {
	for providerName, providerConfig := range config.Providers {
		if !providerConfig.Enabled {
			continue
		}
		if !s.registry.Has(providerName) {
			errors = append(errors,
				fmt.Sprintf("configured provider %q is not registered", providerName))
		}
	}
	return errors
}

// validateCollections checks that all collection providers are registered.
//
// Takes config (*Config) which contains the collections to validate.
// Takes errors ([]string) which accumulates any validation errors found.
//
// Returns []string which contains all validation errors including any new ones.
func (s *collectionService) validateCollections(config *Config, errors []string) []string {
	for collectionName, collectionConfig := range config.Collections {
		if !collectionConfig.Enabled {
			continue
		}
		errors = s.validateCollectionProvider(config, collectionName, collectionConfig, errors)
	}
	return errors
}

// validateCollectionProvider validates a single collection's provider
// configuration.
//
// Takes config (*Config) which provides the default provider setting.
// Takes collectionName (string) which identifies the collection being checked.
// Takes collectionConfig (CollectionConfigEntry) which holds the collection's
// provider reference.
// Takes errors ([]string) which accumulates validation errors.
//
// Returns []string which contains the updated error list with any new
// validation failures appended.
func (s *collectionService) validateCollectionProvider(
	config *Config,
	collectionName string,
	collectionConfig CollectionConfigEntry,
	errors []string,
) []string {
	providerName := collectionConfig.Provider
	if providerName == "" {
		providerName = config.DefaultProvider
	}

	if providerName == "" {
		return append(errors,
			fmt.Sprintf("collection %q has no provider and no default provider is configured", collectionName))
	}

	if !s.registry.Has(providerName) {
		return append(errors,
			fmt.Sprintf("collection %q references unregistered provider %q", collectionName, providerName))
	}
	return errors
}

// reportValidationErrors logs validation errors and returns an error.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes errors ([]string) which contains the validation error messages to log.
//
// Returns error when called, with the error count in the message.
func (*collectionService) reportValidationErrors(ctx context.Context, errors []string) error {
	_, l := logger_domain.From(ctx, log)
	l.Error("Collection configuration validation failed",
		logger_domain.Int("error_count", len(errors)))
	for _, err := range errors {
		l.Error("Validation error", logger_domain.String("error", err))
	}
	return fmt.Errorf("collection configuration validation failed: %d error(s)", len(errors))
}

// getCacheKey generates a cache key for a provider and collection pair.
//
// Takes providerName (string) which identifies the data provider.
// Takes collectionName (string) which identifies the collection.
//
// Returns string which is the combined cache key in "provider:collection"
// format.
func (*collectionService) getCacheKey(providerName, collectionName string) string {
	return fmt.Sprintf("%s:%s", providerName, collectionName)
}

// getCachedContent retrieves cached content if available.
//
// Takes providerName (string) which identifies the content provider.
// Takes collectionName (string) which identifies the collection to retrieve.
//
// Returns []collection_dto.ContentItem which contains the cached content items.
// Returns bool which indicates whether the content was found in the cache.
//
// Safe for concurrent use; holds a read lock during retrieval.
func (s *collectionService) getCachedContent(providerName, collectionName string) ([]collection_dto.ContentItem, bool) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	items, ok := s.cache[s.getCacheKey(providerName, collectionName)]
	return items, ok
}

// setCachedContent stores content in the cache.
//
// Takes providerName (string) which identifies the content provider.
// Takes collectionName (string) which identifies the collection.
// Takes items ([]collection_dto.ContentItem) which contains the content to
// cache.
//
// Safe for concurrent use. Uses a mutex to protect the cache.
func (s *collectionService) setCachedContent(providerName, collectionName string, items []collection_dto.ContentItem) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cache[s.getCacheKey(providerName, collectionName)] = items
}

// parseGetCollectionOptions parses and checks the options parameter.
//
// Takes optionsRaw (any) which contains the raw options to parse.
//
// Returns collection_dto.FetchOptions which contains the parsed options.
// Returns error when optionsRaw is not nil and is not of type FetchOptions.
func (*collectionService) parseGetCollectionOptions(optionsRaw any) (collection_dto.FetchOptions, error) {
	var options collection_dto.FetchOptions
	if optionsRaw == nil {
		return options, nil
	}

	opts, ok := optionsRaw.(collection_dto.FetchOptions)
	if !ok {
		return options, fmt.Errorf("invalid options type: expected collection_dto.FetchOptions, got %T", optionsRaw)
	}

	return opts, nil
}

// resolveProviderName determines which provider to use, defaulting to
// "markdown".
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes options (*collection_dto.FetchOptions) which contains the requested
// provider name.
//
// Returns string which is the provider name from options, or "markdown" if
// none was specified.
func (*collectionService) resolveProviderName(ctx context.Context, options *collection_dto.FetchOptions) string {
	_, l := logger_domain.From(ctx, log)
	if options.ProviderName != "" {
		return options.ProviderName
	}

	l.Internal("No provider specified, using default", logger_domain.String("provider", "markdown"))
	return "markdown"
}

// lookupProvider retrieves a provider from the registry.
//
// Takes providerName (string) which identifies the provider to find.
//
// Returns CollectionProvider which is the requested provider.
// Returns error when the provider name is not found in the registry.
func (s *collectionService) lookupProvider(providerName string) (CollectionProvider, error) {
	provider, ok := s.registry.Get(providerName)
	if !ok {
		availableProviders := s.registry.List()
		return nil, fmt.Errorf(
			"%w: unknown collection provider '%s'; available providers: %v",
			collection_dto.ErrProviderNotFound,
			providerName,
			availableProviders,
		)
	}

	return provider, nil
}

// dispatchProviderAnnotation generates the appropriate annotation based on
// provider type.
//
// Takes provider (CollectionProvider) which specifies the source of items.
// Takes collectionName (string) which identifies the target collection.
// Takes targetTypeExpr (ast.Expr) which is the AST expression for the type.
// Takes options (*collection_dto.FetchOptions) which controls fetch behaviour.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the generated annotation.
// Returns error when the provider type is unknown.
func (s *collectionService) dispatchProviderAnnotation(
	ctx context.Context,
	provider CollectionProvider,
	collectionName string,
	targetTypeExpr ast.Expr,
	options *collection_dto.FetchOptions,
) (*ast_domain.GoGeneratorAnnotation, error) {
	switch provider.Type() {
	case ProviderTypeStatic:
		return s.generateStaticCollectionAnnotation(ctx, provider, collectionName, targetTypeExpr, options)

	case ProviderTypeDynamic:
		return s.generateDynamicAnnotation(ctx, provider, collectionName, targetTypeExpr, options)

	case ProviderTypeHybrid:
		return s.generateHybridAnnotation(ctx, provider, collectionName, targetTypeExpr, options)

	default:
		return nil, fmt.Errorf(
			"unknown provider type '%s' for provider '%s'",
			provider.Type(),
			provider.Name(),
		)
	}
}

// NewCollectionService creates a new CollectionService.
//
// Takes registry (ProviderRegistryPort) which provides lookup for content
// providers.
// Takes opts (...CollectionServiceOption) which provides optional functional
// options for customising the service.
//
// Returns CollectionService which is a fully initialised service ready for
// use.
func NewCollectionService(
	_ context.Context,
	registry ProviderRegistryPort,
	opts ...CollectionServiceOption,
) CollectionService {
	s := &collectionService{
		registry:       registry,
		hybridRegistry: newDefaultHybridRegistry(),
		encoder:        newDefaultCollectionEncoder(),
		clock:          clock.RealClock(),
		cache:          make(map[string][]collection_dto.ContentItem),
		cacheMutex:     sync.RWMutex{},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// withHybridRegistry sets a custom hybrid registry for the service.
//
// Takes registry (HybridRegistryPort) which provides the hybrid registry
// implementation.
//
// Returns CollectionServiceOption which configures the service to use the
// given registry.
func withHybridRegistry(registry HybridRegistryPort) CollectionServiceOption {
	return func(s *collectionService) {
		s.hybridRegistry = registry
	}
}

// withEncoder sets a custom encoder for the service.
//
// Takes encoder (CollectionEncoderPort) which provides custom
// encoding logic for collection data.
//
// Returns CollectionServiceOption which configures the service to use the
// given encoder.
func withEncoder(encoder CollectionEncoderPort) CollectionServiceOption {
	return func(s *collectionService) {
		s.encoder = encoder
	}
}

// withServiceClock sets a custom clock for time operations. This is mainly
// used for testing to make time-based logic deterministic.
//
// Takes c (clock.Clock) which provides the clock implementation to use.
//
// Returns CollectionServiceOption which configures the service with the clock.
func withServiceClock(c clock.Clock) CollectionServiceOption {
	return func(s *collectionService) {
		s.clock = c
	}
}

// WithDefaultSandbox sets the default sandbox for local content collections.
//
// Takes sandbox (safedisk.Sandbox) which provides filesystem access for the
// project's content/ directory.
//
// Returns CollectionServiceOption which configures the service sandbox.
func WithDefaultSandbox(sandbox safedisk.Sandbox) CollectionServiceOption {
	return func(s *collectionService) {
		s.defaultSandbox = sandbox
	}
}

// WithResolver sets the module resolver for external content sources.
//
// Takes resolver (resolver_domain.ResolverPort) which handles Go module path
// resolution for p-collection-source directives.
//
// Returns CollectionServiceOption which configures the service resolver.
func WithResolver(resolver resolver_domain.ResolverPort) CollectionServiceOption {
	return func(s *collectionService) {
		s.resolver = resolver
	}
}

// defaultContentSource returns a ContentSource for local project content.
//
// This is used by code paths that don't have a directive (e.g.
// GetAllCollectionItems calls in Go code) where the content is always local.
//
// Returns collection_dto.ContentSource which provides local sandbox access.
func (s *collectionService) defaultContentSource() collection_dto.ContentSource {
	return collection_dto.ContentSource{
		Sandbox:    s.defaultSandbox,
		IsExternal: false,
	}
}

// trackExternalSandbox records a sandbox for later cleanup by Close.
//
// Takes sandbox (safedisk.Sandbox) which is the sandbox to track.
//
// Concurrency: acquires sandboxMutex to append to the externalSandboxes slice.
func (s *collectionService) trackExternalSandbox(sandbox safedisk.Sandbox) {
	s.sandboxMutex.Lock()
	s.externalSandboxes = append(s.externalSandboxes, sandbox)
	s.sandboxMutex.Unlock()
}

// newDefaultCollectionEncoder returns the default FlatBuffer encoder for
// production use.
//
// Returns CollectionEncoderPort which provides the standard encoder.
func newDefaultCollectionEncoder() CollectionEncoderPort {
	return staticCollectionRegistry.encoder
}
