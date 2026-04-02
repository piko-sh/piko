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

package resolver_adapters

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

var _ resolver_domain.ResolverPort = (*ChainedResolver)(nil)

const (
	// logKeyIndex is the log field key for the resolver index in the chain.
	logKeyIndex = "index"

	// logKeyResolver is the log field key for the resolver type being used.
	logKeyResolver = "resolver"

	// logKeyImportPath is the log key for the import path being resolved.
	logKeyImportPath = "importPath"

	// logFormatResolverType is the format string for logging resolver type names.
	logFormatResolverType = "%T"

	// errorItemFormat is the format string used to list errors from resolvers.
	errorItemFormat = "  %d. %T: %v"

	// errNoResolversConfigured is the error message returned when ChainedResolver
	// methods are called but no resolvers have been added to the chain.
	errNoResolversConfigured = "no resolvers configured in chain"
)

// stringResolverFunc is a callback that tries to resolve a path using a
// single resolver.
type stringResolverFunc func(resolver resolver_domain.ResolverPort) (string, error)

// ChainedResolver implements ResolverPort using the Chain of Responsibility
// pattern. It delegates to multiple resolvers in priority order, letting local
// module paths take precedence over external Go module paths.
type ChainedResolver struct {
	// resolvers holds the chain of resolvers to try in order; the first is
	// the primary resolver and must succeed, while others are optional.
	resolvers []resolver_domain.ResolverPort
}

// NewChainedResolver creates a chained resolver from the given
// resolvers. Resolvers are tried in order, so the first resolver
// has the highest priority.
//
// Typical usage:
// localResolver := NewLocalModuleResolver(baseDir)
// cacheResolver := NewGoModuleCacheResolver()
// chainedResolver := NewChainedResolver(localResolver, cacheResolver)
//
// Takes resolvers (...ResolverPort) which are the resolvers to
// chain together.
//
// Returns *ChainedResolver which wraps the given resolvers in
// priority order.
func NewChainedResolver(resolvers ...resolver_domain.ResolverPort) *ChainedResolver {
	return &ChainedResolver{resolvers: resolvers}
}

// DetectLocalModule initialises all resolvers in the chain by calling their
// DetectLocalModule methods.
//
// The first resolver is the primary and its error is fatal. Secondary
// resolvers are also initialised to populate their internal state, but their
// errors are ignored.
//
// Returns error when no resolvers are configured or when the primary resolver
// fails to detect.
func (cr *ChainedResolver) DetectLocalModule(ctx context.Context) error {
	if len(cr.resolvers) == 0 {
		return errors.New(errNoResolversConfigured)
	}

	if err := cr.resolvers[0].DetectLocalModule(ctx); err != nil {
		return fmt.Errorf("detecting local module in primary resolver: %w", err)
	}

	for i := 1; i < len(cr.resolvers); i++ {
		_ = cr.resolvers[i].DetectLocalModule(ctx)
	}

	return nil
}

// GetModuleName delegates to the first resolver in the chain.
// This returns the local project's module name.
//
// Returns string which is the module name, or empty if no resolvers exist.
func (cr *ChainedResolver) GetModuleName() string {
	if len(cr.resolvers) == 0 {
		return ""
	}
	return cr.resolvers[0].GetModuleName()
}

// GetBaseDir delegates to the first resolver in the chain.
// This returns the local project's base directory.
//
// Returns string which is the base directory path, or empty if no resolvers
// exist.
func (cr *ChainedResolver) GetBaseDir() string {
	if len(cr.resolvers) == 0 {
		return ""
	}
	return cr.resolvers[0].GetBaseDir()
}

// ConvertEntryPointPathToManifestKey delegates to the first resolver in the
// chain to generate manifest keys using the local module's naming convention.
//
// Takes entryPointPath (string) which is the path to convert.
//
// Returns string which is the manifest key, or the original path if the chain
// is empty.
func (cr *ChainedResolver) ConvertEntryPointPathToManifestKey(entryPointPath string) string {
	if len(cr.resolvers) == 0 {
		return entryPointPath
	}
	return cr.resolvers[0].ConvertEntryPointPathToManifestKey(entryPointPath)
}

// ResolvePKPath resolves a Piko component path using the chain of
// resolvers.
//
// The first resolver to successfully resolve the path wins. If all resolvers
// fail, an error is returned with details from all attempted resolvers.
//
// This implements the core Chain of Responsibility pattern:
//  1. Try LocalModuleResolver - resolves local project components
//  2. If that fails, try GoModuleCacheResolver - resolves external module
//     components
//  3. If all fail, return a combined error
//
// Takes importPath (string) which is the Piko component path to resolve.
// Takes containingFilePath (string) which is the absolute path of the file
// containing the import statement, used to resolve the @ alias to the correct
// module.
//
// Returns string which is the resolved file system path to the component.
// Returns error when no resolvers are configured or all resolvers fail.
func (cr *ChainedResolver) ResolvePKPath(ctx context.Context, importPath string, containingFilePath string) (string, error) {
	ctx, span, _ := log.Span(ctx, "ChainedResolver.ResolvePKPath",
		logger_domain.String("importPath", importPath),
		logger_domain.String("containingFilePath", containingFilePath),
		logger_domain.Int("resolverCount", len(cr.resolvers)),
	)
	defer span.End()

	if len(cr.resolvers) == 0 {
		return "", errors.New(errNoResolversConfigured)
	}

	return cr.tryResolvers(ctx, "component", importPath, func(r resolver_domain.ResolverPort) (string, error) {
		return r.ResolvePKPath(ctx, importPath, containingFilePath)
	})
}

// ResolveCSSPath tries to resolve a CSS import path using each resolver in the
// chain. The first resolver to find the path is used.
//
// Takes importPath (string) which specifies the CSS import path to resolve.
// Takes containingDir (string) which specifies the folder of the importing
// file.
//
// Returns string which is the resolved absolute path to the CSS file.
// Returns error when no resolvers are set up or all resolvers fail.
func (cr *ChainedResolver) ResolveCSSPath(ctx context.Context, importPath string, containingDir string) (string, error) {
	ctx, span, _ := log.Span(ctx, "ChainedResolver.ResolveCSSPath",
		logger_domain.String("importPath", importPath),
		logger_domain.String("containingDir", containingDir),
	)
	defer span.End()

	if len(cr.resolvers) == 0 {
		return "", errors.New(errNoResolversConfigured)
	}

	return cr.tryResolvers(ctx, "CSS", importPath, func(r resolver_domain.ResolverPort) (string, error) {
		return r.ResolveCSSPath(ctx, importPath, containingDir)
	})
}

// ResolveAssetPath attempts to resolve an asset path using each resolver in
// the chain. The first resolver to successfully resolve the path wins.
//
// Takes importPath (string) which is the path to the asset to resolve.
// Takes containingFilePath (string) which is the absolute path of the
// component file containing the asset reference, used to resolve the @ alias
// to the correct module.
//
// Returns string which is the resolved absolute path to the asset.
// Returns error when no resolvers are configured or all resolvers fail.
func (cr *ChainedResolver) ResolveAssetPath(ctx context.Context, importPath string, containingFilePath string) (string, error) {
	ctx, span, _ := log.Span(ctx, "ChainedResolver.ResolveAssetPath",
		logger_domain.String("importPath", importPath),
		logger_domain.String("containingFilePath", containingFilePath),
	)
	defer span.End()

	if len(cr.resolvers) == 0 {
		return "", errors.New(errNoResolversConfigured)
	}

	return cr.tryResolvers(ctx, "asset", importPath, func(r resolver_domain.ResolverPort) (string, error) {
		return r.ResolveAssetPath(ctx, importPath, containingFilePath)
	})
}

// GetModuleDir resolves a Go module path to its filesystem directory by
// attempting each resolver in the chain. This is used for accessing content
// from external Go modules (e.g., for p-collection-source).
//
// Takes modulePath (string) which is the Go module path to resolve.
//
// Returns string which is the absolute path to the module directory.
// Returns error when no resolvers are configured or all resolvers fail.
func (cr *ChainedResolver) GetModuleDir(ctx context.Context, modulePath string) (string, error) {
	ctx, span, _ := log.Span(ctx, "ChainedResolver.GetModuleDir",
		logger_domain.String("modulePath", modulePath),
	)
	defer span.End()

	if len(cr.resolvers) == 0 {
		return "", errors.New(errNoResolversConfigured)
	}

	return cr.tryResolvers(ctx, "module directory", modulePath, func(r resolver_domain.ResolverPort) (string, error) {
		return r.GetModuleDir(ctx, modulePath)
	})
}

// FindModuleBoundary splits an import path into the module path and subpath
// by attempting each resolver in the chain. This uses the known modules from
// go.mod for accurate boundary detection.
//
// Takes importPath (string) which is the full import path to split.
//
// Returns modulePath (string) which is the Go module portion.
// Returns subpath (string) which is the path within the module.
// Returns error when no resolvers are configured or all resolvers fail.
func (cr *ChainedResolver) FindModuleBoundary(ctx context.Context, importPath string) (modulePath string, subpath string, err error) {
	ctx, span, l := log.Span(ctx, "ChainedResolver.FindModuleBoundary",
		logger_domain.String(logKeyImportPath, importPath),
	)
	defer span.End()

	if len(cr.resolvers) == 0 {
		return "", "", errors.New(errNoResolversConfigured)
	}

	allErrors := make([]string, 0, len(cr.resolvers))

	for i, resolver := range cr.resolvers {
		l.Trace("Trying resolver for module boundary",
			logger_domain.Int(logKeyIndex, i),
			logger_domain.String(logKeyResolver, fmt.Sprintf(logFormatResolverType, resolver)),
		)

		modulePath, subpath, err := resolver.FindModuleBoundary(ctx, importPath)
		if err == nil {
			l.Trace("Module boundary resolver succeeded",
				logger_domain.Int(logKeyIndex, i),
				logger_domain.String(logKeyResolver, fmt.Sprintf(logFormatResolverType, resolver)),
				logger_domain.String("modulePath", modulePath),
				logger_domain.String("subpath", subpath),
			)
			return modulePath, subpath, nil
		}

		allErrors = append(allErrors, fmt.Sprintf(errorItemFormat, i+1, resolver, err))
		l.Trace("Module boundary resolver failed, trying next",
			logger_domain.Int(logKeyIndex, i),
			logger_domain.String(logKeyResolver, fmt.Sprintf(logFormatResolverType, resolver)),
			logger_domain.Error(err),
		)
	}

	errorMessage := fmt.Sprintf(
		"failed to find module boundary for '%s' using %d resolvers:\n%s",
		importPath,
		len(cr.resolvers),
		strings.Join(allErrors, "\n"),
	)

	l.Error("All module boundary resolvers failed", logger_domain.String(logKeyImportPath, importPath))
	return "", "", errors.New(errorMessage)
}

// tryResolvers tries each resolver in the chain and returns the result from
// the first one that succeeds.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes resourceType (string) which describes what is being resolved (e.g.
// "component", "CSS", "asset").
// Takes resourcePath (string) which is the path to resolve.
// Takes resolve (stringResolverFunc) which is called for each resolver.
//
// Returns string which is the resolved path from the first successful resolver.
// Returns error when all resolvers fail, with details of each failure.
func (cr *ChainedResolver) tryResolvers(
	ctx context.Context,
	resourceType string,
	resourcePath string,
	resolve stringResolverFunc,
) (string, error) {
	ctx, l := logger_domain.From(ctx, log)
	allErrors := make([]string, 0, len(cr.resolvers))

	for i, resolver := range cr.resolvers {
		l.Trace("Trying resolver",
			logger_domain.Int(logKeyIndex, i),
			logger_domain.String(logKeyResolver, fmt.Sprintf(logFormatResolverType, resolver)),
			logger_domain.String("resourceType", resourceType),
		)

		resolvedPath, err := resolve(resolver)
		if err == nil {
			l.Trace("Resolver succeeded",
				logger_domain.Int(logKeyIndex, i),
				logger_domain.String(logKeyResolver, fmt.Sprintf(logFormatResolverType, resolver)),
				logger_domain.String("resolvedPath", resolvedPath),
			)
			return resolvedPath, nil
		}

		allErrors = append(allErrors, fmt.Sprintf(errorItemFormat, i+1, resolver, err))
		l.Trace("Resolver failed, trying next",
			logger_domain.Int(logKeyIndex, i),
			logger_domain.String(logKeyResolver, fmt.Sprintf(logFormatResolverType, resolver)),
			logger_domain.Error(err),
		)
	}

	errorMessage := fmt.Sprintf(
		"failed to resolve %s '%s' using %d resolvers:\n%s",
		resourceType,
		resourcePath,
		len(cr.resolvers),
		strings.Join(allErrors, "\n"),
	)

	l.Error("All resolvers failed",
		logger_domain.String("resourceType", resourceType),
		logger_domain.String(logKeyImportPath, resourcePath),
	)
	return "", errors.New(errorMessage)
}
