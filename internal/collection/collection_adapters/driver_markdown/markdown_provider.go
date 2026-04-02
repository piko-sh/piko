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

package driver_markdown

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"io/fs"
	"maps"
	"path/filepath"
	"strings"
	"time"

	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/markdown/markdown_domain"
	"piko.sh/piko/internal/markdown/markdown_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// keyPath is the structured logging key for file and directory paths.
	keyPath = "path"

	// keyContent is the directory name for markdown content files.
	keyContent = "content"

	// keyCollection is the logging key for collection names.
	keyCollection = "collection"

	// defaultNavOrder is the default sort order for navigation items.
	defaultNavOrder = 999

	// fnv1aOffset64 is the FNV-1a 64-bit hash offset basis.
	fnv1aOffset64 = 0xcbf29ce484222325

	// fnv1aPrime64 is the 64-bit prime number used in FNV-1a hash calculations.
	fnv1aPrime64 = 0x100000001b3
)

// MarkdownProvider implements CollectionProvider for static markdown files.
// All file operations are sandboxed for security.
//
// This is the canonical implementation of a static provider. It demonstrates
// the full lifecycle of a collection provider:
//   - Discovery: Recursively scans directories for .md files
//   - Analysis: Extracts locale, slug, and metadata from paths and frontmatter
//   - Processing: Parses markdown to AST using the markdown service
//   - Linking: Groups translations by translation key
//
// Design Philosophy:
//   - Convention over configuration: Infers structure from file paths
//   - Build-time only: All content is resolved during build
//   - Locale-aware: First-class i18n support
//   - AST-native: Returns Piko AST, not HTML strings
//   - Secure: All file operations are sandboxed
//
// Performance Characteristics:
//   - Discovery: O(n) where n = total files
//   - Processing: O(m) where m = markdown files
//   - Memory: ~10KB per markdown file + AST size
type MarkdownProvider struct {
	// sandbox provides restricted file system access for reading content files.
	sandbox safedisk.Sandbox

	// markdownService turns markdown content into structured output.
	markdownService markdown_domain.MarkdownService

	// renderService extracts plain text from AST for search indexing.
	renderService render_domain.RenderService

	// resolver provides module resolution for external content sources.
	// Optional; only needed when using p-collection-source.
	resolver resolver_domain.ResolverPort

	// scanner finds content files in collection directories.
	scanner *fileScanner

	// name is the unique identifier for this provider.
	name string

	// basePath is the root path used to find content files.
	basePath string

	// isExternalModule indicates the provider is reading from an external module.
	// When true, the sandbox root is the content root directly, so no
	// content/{collection} prefix is applied.
	isExternalModule bool
}

// MarkdownProviderOption is a function that sets up a MarkdownProvider.
type MarkdownProviderOption func(*MarkdownProvider)

// NewMarkdownProvider creates a new markdown collection provider.
//
// The sandbox is used for all file operations to ensure security.
//
// Takes name (string) which is the unique provider name (typically
// "markdown").
// Takes sandbox (safedisk.Sandbox) which provides secure file system access.
// Takes markdownService (markdown_domain.MarkdownService) which parses
// markdown files.
// Takes renderService (render_domain.RenderService) which extracts plain text
// from AST for search indexing. Can be nil if plain text extraction is not
// needed.
// Takes opts (...MarkdownProviderOption) which provides optional functional
// options for configuring features like module resolution.
//
// Returns *MarkdownProvider which is fully initialised and ready for use.
func NewMarkdownProvider(
	name string,
	sandbox safedisk.Sandbox,
	markdownService markdown_domain.MarkdownService,
	renderService render_domain.RenderService,
	opts ...MarkdownProviderOption,
) *MarkdownProvider {
	m := &MarkdownProvider{
		sandbox:         sandbox,
		markdownService: markdownService,
		renderService:   renderService,

		resolver:         nil,
		scanner:          newFileScanner(sandbox),
		name:             name,
		basePath:         "",
		isExternalModule: false,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// SetBasePath sets the project base path for resolving content directories.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes basePath (string) which specifies the directory path.
//
// This must be called before FetchStaticContent if the provider is used
// outside of its default working directory context.
func (m *MarkdownProvider) SetBasePath(ctx context.Context, basePath string) {
	m.basePath = basePath
	_, l := logger_domain.From(ctx, log)
	l.Internal("Base path set for markdown provider",
		logger_domain.String("base_path", basePath))
}

// SetContentModulePath configures the provider to read content from an
// external Go module via GOMODCACHE or a local replace directive.
//
// This implements the ContentModuleConfigurable interface, enabling the
// p-collection-source feature.
//
// The modulePath can include a subpath (e.g., "github.com/org/repo/docs").
// The resolver splits this into the module path and subpath, resolves the
// module's filesystem location, and creates a new sandbox for reading content.
//
// Security model: The safedisk factory is created on-demand with the resolved
// module path as its only allowed path. This is safe because:
//   - The path originates from go.mod (developer-controlled)
//   - Module resolution is performed by Go tooling
//   - The trust boundary is established by the module system, not filesystem paths
//
// Takes modulePath (string) which is the full import path including any
// subpath.
//
// Returns error when:
//   - The resolver is not configured
//   - The module cannot be found in GOMODCACHE or via replace directive
//   - The sandbox cannot be created for the module content
func (m *MarkdownProvider) SetContentModulePath(ctx context.Context, modulePath string) error {
	ctx, l := logger_domain.From(ctx, log)
	if m.resolver == nil {
		return errors.New("resolver not configured for module content sourcing; use WithModuleResolver option")
	}

	contentRoot, moduleBase, err := m.resolveModuleContentRoot(ctx, modulePath)
	if err != nil {
		return fmt.Errorf("resolving module content root for %q: %w", modulePath, err)
	}

	moduleSandbox, err := createModuleSandbox(moduleBase, contentRoot)
	if err != nil {
		return fmt.Errorf("creating module sandbox for %q: %w", modulePath, err)
	}

	m.sandbox = moduleSandbox
	m.scanner = newFileScanner(moduleSandbox)
	m.basePath = contentRoot
	m.isExternalModule = true

	l.Internal("Configured provider to read from external module",
		logger_domain.String("module_path", modulePath),
		logger_domain.String("content_root", contentRoot))

	return nil
}

// Name returns the unique identifier for this provider.
//
// Returns string which is the provider's unique name.
func (m *MarkdownProvider) Name() string {
	return m.name
}

// Type returns the provider type for this markdown provider.
//
// Returns collection_domain.ProviderType which is always ProviderTypeStatic.
func (*MarkdownProvider) Type() collection_domain.ProviderType {
	return collection_domain.ProviderTypeStatic
}

// DiscoverCollections scans the configured paths and returns available
// collections.
//
// For markdown, each subdirectory under the base path is treated as a
// collection.
//
// Takes config (collection_dto.ProviderConfig) which provides the base path
// and locale settings.
//
// Returns []collection_dto.CollectionInfo which describes each discovered
// collection.
// Returns error when the base path cannot be accessed.
func (m *MarkdownProvider) DiscoverCollections(
	ctx context.Context,
	config collection_dto.ProviderConfig,
) ([]collection_dto.CollectionInfo, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Discovering markdown collections",
		logger_domain.String("base_path", config.BasePath))

	contentPath := keyContent

	if _, err := m.sandbox.Stat(contentPath); err != nil {
		l.Warn("Content directory does not exist",
			logger_domain.String(keyPath, contentPath))
		return []collection_dto.CollectionInfo{}, nil
	}

	entries, err := m.sandbox.ReadDir(contentPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read content directory: %w", err)
	}

	collections := make([]collection_dto.CollectionInfo, 0, len(entries))
	for _, entry := range entries {
		info, ok := m.processCollectionEntry(ctx, entry, contentPath, config)
		if ok {
			collections = append(collections, info)
		}
	}

	l.Internal("Collection discovery complete",
		logger_domain.Int("collection_count", len(collections)))

	return collections, nil
}

// ValidateTargetType checks that a target struct type works with markdown.
//
// This is called at build time when the user writes:
// posts, err := data.GetCollection[BlogPost]("blog")
// The markdown provider accepts any struct type. Validation happens at compile
// time instead. If the generated struct literal has fields that do not match,
// the Go compiler will report the error. This gives clearer error messages and
// avoids repeating type checks that the compiler already does well.
//
// Returns error when the type is not valid or not compatible.
func (*MarkdownProvider) ValidateTargetType(_ ast.Expr) error {
	return nil
}

// FetchStaticContent loads and processes all markdown files in a collection.
//
// This is the core method for static providers. It scans the collection
// directory for .md files, parses each file's frontmatter and content,
// converts markdown to Piko AST, analyses paths for locale and slug, and
// links translations by translation key.
//
// Takes collectionName (string) which specifies the collection to fetch
// (e.g. "blog").
//
// Returns []collection_dto.ContentItem which contains the processed markdown
// files ready to be transformed into virtual entry points by CollectionService.
// Returns error when the collection directory cannot be scanned.
func (m *MarkdownProvider) FetchStaticContent(
	ctx context.Context,
	collectionName string,
) ([]collection_dto.ContentItem, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Fetching static markdown content",
		logger_domain.String(keyCollection, collectionName),
		logger_domain.Bool("is_external_module", m.isExternalModule))

	config := m.getDefaultConfig()

	var collectionPath string
	if m.isExternalModule {
		collectionPath = "."
	} else {
		collectionPath = filepath.Join(keyContent, collectionName)
	}

	files, err := m.scanner.scanDirectory(ctx, collectionPath)
	if err != nil {
		return nil, fmt.Errorf("scanning collection directory: %w", err)
	}

	if len(files) == 0 {
		l.Warn("No markdown files found in collection",
			logger_domain.String(keyCollection, collectionName),
			logger_domain.String(keyPath, collectionPath))
		return []collection_dto.ContentItem{}, nil
	}

	l.Internal("Markdown files discovered",
		logger_domain.Int("count", len(files)))

	analyser := newPathAnalyser(config.Locales, config.DefaultLocale)
	items := make([]collection_dto.ContentItem, 0, len(files))
	translationGroups := make(map[string][]int)

	for _, file := range files {
		item, err := m.processMarkdownFile(ctx, file, collectionName, analyser)
		if err != nil {
			l.Warn("Failed to process markdown file",
				logger_domain.String(keyPath, file.relativePath),
				logger_domain.Error(err))
			continue
		}

		index := len(items)
		items = append(items, item)
		translationGroups[item.TranslationKey] = append(translationGroups[item.TranslationKey], index)
	}

	l.Internal("Markdown processing complete",
		logger_domain.Int("items_processed", len(items)),
		logger_domain.Int("translation_groups", len(translationGroups)))

	m.linkTranslations(ctx, items, translationGroups)

	return items, nil
}

// GenerateRuntimeFetcher generates code for runtime data fetching.
//
// Markdown is a static provider, so this method returns an error.
// Dynamic fetching is not supported for markdown files.
//
// If hybrid (ISR) support is added in the future, this method would generate
// code to re-parse the markdown file at runtime for revalidation.
//
// Returns *collection_dto.RuntimeFetcherCode which is always nil for this
// provider.
// Returns error when called, as markdown does not support runtime fetching.
func (*MarkdownProvider) GenerateRuntimeFetcher(
	_ context.Context,
	_ string,
	_ ast.Expr,
	_ collection_dto.FetchOptions,
) (*collection_dto.RuntimeFetcherCode, error) {
	return nil, errors.New("markdown provider does not support runtime fetching (static only)")
}

// Check implements the healthprobe_domain.Probe interface.
// It verifies that the markdown content directory is accessible.
//
// Returns healthprobe_dto.Status which indicates whether the content directory
// exists and is accessible.
func (m *MarkdownProvider) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	contentPath := "content"
	fullContentPath := filepath.Join(m.sandbox.Root(), contentPath)

	_, err := m.sandbox.Stat(contentPath)

	state := healthprobe_dto.StateHealthy
	message := fmt.Sprintf("Markdown content directory accessible at %s", fullContentPath)

	if err != nil {
		message = fmt.Sprintf("Markdown content directory does not exist at %s (will be created on first use)", fullContentPath)
	}

	return healthprobe_dto.Status{
		Name:         fmt.Sprintf("%s (MarkdownProvider)", m.name),
		State:        state,
		Message:      message,
		Timestamp:    time.Now(),
		Duration:     time.Since(startTime).String(),
		Dependencies: nil,
	}
}

// ComputeETag computes a content fingerprint for hybrid mode staleness
// detection.
//
// For markdown files, the ETag is computed from the aggregate hash of all file
// modification times in the collection. This is efficient as it avoids reading
// file contents while still detecting changes.
//
// ETag format: "md-{xxhash64 hex}" (e.g., "md-a1b2c3d4e5f67890")
//
// Takes collectionName (string) which specifies the collection to compute the
// ETag for.
//
// Returns string which is the computed ETag, or "md-empty" if no files exist.
// Returns error when scanning the collection directory fails.
func (m *MarkdownProvider) ComputeETag(
	ctx context.Context,
	collectionName string,
) (string, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Computing ETag for markdown collection",
		logger_domain.String(keyCollection, collectionName))

	var collectionPath string
	if m.isExternalModule {
		collectionPath = "."
	} else {
		collectionPath = filepath.Join(keyContent, collectionName)
	}

	files, err := m.scanner.scanDirectory(ctx, collectionPath)
	if err != nil {
		return "", fmt.Errorf("scanning for ETag computation: %w", err)
	}

	if len(files) == 0 {
		return "md-empty", nil
	}

	modTimes := make([]string, 0, len(files))
	for _, file := range files {
		var relativePath string
		if m.isExternalModule {
			relativePath = file.relativePath
		} else {
			relativePath = filepath.Join(keyContent, collectionName, file.relativePath)
		}
		info, err := m.sandbox.Stat(relativePath)
		if err != nil {
			continue
		}
		modTimes = append(modTimes, fmt.Sprintf("%s:%d", file.relativePath, info.ModTime().UnixNano()))
	}

	sortStrings(modTimes)

	h := newXXHash64()
	for _, mt := range modTimes {
		h.WriteString(mt)
		h.WriteString("|")
	}

	etag := fmt.Sprintf("md-%016x", h.Sum64())

	l.Internal("Computed ETag",
		logger_domain.String(keyCollection, collectionName),
		logger_domain.String("etag", etag),
		logger_domain.Int("file_count", len(files)))

	return etag, nil
}

// ValidateETag checks if the current content matches an expected ETag.
//
// This method efficiently detects changes by recomputing the ETag and comparing
// it against the expected value. It avoids reading file contents.
//
// Takes collectionName (string) which specifies the collection to validate.
// Takes expectedETag (string) which is the previously computed ETag to
// compare against.
//
// Returns currentETag (string) which is the freshly computed ETag.
// Returns changed (bool) which is true when the content has changed.
// Returns err (error) when the ETag computation fails.
func (m *MarkdownProvider) ValidateETag(
	ctx context.Context,
	collectionName string,
	expectedETag string,
) (currentETag string, changed bool, err error) {
	ctx, l := logger_domain.From(ctx, log)
	currentETag, err = m.ComputeETag(ctx, collectionName)
	if err != nil {
		return "", false, fmt.Errorf("computing ETag for collection %q: %w", collectionName, err)
	}

	changed = currentETag != expectedETag

	l.Internal("Validated ETag",
		logger_domain.String(keyCollection, collectionName),
		logger_domain.String("expected", expectedETag),
		logger_domain.String("current", currentETag),
		logger_domain.Bool("changed", changed))

	return currentETag, changed, nil
}

// GenerateRevalidator returns nil to indicate no generated code is needed.
//
// For the markdown provider, revalidation is handled directly by the hybrid
// registry at runtime, not through generated code. The hybrid registry calls:
//  1. ValidateETag() to check for file changes
//  2. FetchStaticContent() to re-scan and re-parse markdown files if changed
//  3. Serialises new content to FlatBuffer and updates the cache
//
// This approach is preferred over code generation because:
//   - Runtime calls to provider methods are more maintainable
//   - No duplicate logic between generated and provider code
//   - Easier to test and debug
//
// Returns *collection_dto.RuntimeFetcherCode which is always nil.
// Returns error which is always nil.
func (*MarkdownProvider) GenerateRevalidator(
	_ context.Context,
	_ string,
	_ ast.Expr,
	_ collection_dto.HybridConfig,
) (*collection_dto.RuntimeFetcherCode, error) {
	return nil, nil
}

// resolveModuleContentRoot resolves a module import path to its filesystem
// content root directory.
//
// Takes modulePath (string) which is the full import path (e.g.,
// "github.com/org/repo/docs").
//
// Returns contentRoot (string) which is the filesystem path to the content.
// Returns moduleBase (string) which is the Go module path without subpath.
// Returns error when module boundary detection or directory resolution fails.
func (m *MarkdownProvider) resolveModuleContentRoot(
	ctx context.Context,
	modulePath string,
) (contentRoot string, moduleBase string, err error) {
	ctx, l := logger_domain.From(ctx, log)
	moduleBase, subpath, err := m.resolver.FindModuleBoundary(ctx, modulePath)
	if err != nil {
		return "", "", fmt.Errorf("finding module boundary for %q: %w", modulePath, err)
	}

	l.Internal("Resolved module boundary",
		logger_domain.String("module_path", modulePath),
		logger_domain.String("module_base", moduleBase),
		logger_domain.String("subpath", subpath))

	moduleDir, err := m.resolver.GetModuleDir(ctx, moduleBase)
	if err != nil {
		return "", "", fmt.Errorf("resolving module directory for %q: %w", moduleBase, err)
	}

	contentRoot = filepath.Join(moduleDir, subpath)

	l.Internal("Resolved content root for external module",
		logger_domain.String("module_base", moduleBase),
		logger_domain.String("content_root", contentRoot))

	return contentRoot, moduleBase, nil
}

// processCollectionEntry processes a single directory entry during collection
// discovery.
//
// Takes entry (fs.DirEntry) which is the directory entry to process.
// Takes contentPath (string) which is the path to the content directory.
// Takes config (collection_dto.ProviderConfig) which provides locale settings.
//
// Returns collection_dto.CollectionInfo which contains the discovered
// collection metadata.
// Returns bool which indicates whether the entry was a valid collection.
func (m *MarkdownProvider) processCollectionEntry(
	ctx context.Context,
	entry fs.DirEntry,
	contentPath string,
	config collection_dto.ProviderConfig,
) (collection_dto.CollectionInfo, bool) {
	ctx, l := logger_domain.From(ctx, log)
	if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
		return collection_dto.CollectionInfo{}, false
	}

	collectionName := entry.Name()
	collectionPath := filepath.Join(contentPath, collectionName)

	files, err := m.scanner.scanDirectory(ctx, collectionPath)
	if err != nil {
		l.Warn("Cannot scan collection directory",
			logger_domain.String(keyCollection, collectionName),
			logger_domain.Error(err))
		return collection_dto.CollectionInfo{}, false
	}

	localesFound := m.detectLocalesInFiles(files, config.Locales, config.DefaultLocale)
	info := buildCollectionInfo(collectionName, m.sandbox.Root(), collectionPath, files, localesFound)

	l.Internal("Discovered markdown collection",
		logger_domain.String("name", collectionName),
		logger_domain.Int("item_count", len(files)),
		logger_domain.Strings("locales", localesFound))

	return info, true
}

// detectLocalesInFiles analyses file paths to determine which locales are
// present.
//
// Takes files ([]*discoveredFile) which contains the discovered files to
// analyse.
// Takes configuredLocales ([]string) which lists the known locale codes.
// Takes defaultLocale (string) which specifies the fallback locale.
//
// Returns []string which contains the unique locales found in the file paths.
func (*MarkdownProvider) detectLocalesInFiles(
	files []*discoveredFile,
	configuredLocales []string,
	defaultLocale string,
) []string {
	analyser := newPathAnalyser(configuredLocales, defaultLocale)
	localeSet := make(map[string]bool)

	for _, file := range files {
		pathInfo := analyser.Analyse(file.relativePath, "")
		localeSet[pathInfo.locale] = true
	}

	locales := make([]string, 0, len(localeSet))
	for locale := range localeSet {
		locales = append(locales, locale)
	}

	return locales
}

// processMarkdownFile loads and processes a single markdown file.
//
// Takes file (*discoveredFile) which identifies the markdown file to process.
// Takes collectionName (string) which specifies the content collection.
// Takes analyser (*pathAnalyser) which extracts path metadata.
//
// Returns collection_dto.ContentItem which contains the processed content.
// Returns error when the file cannot be read or markdown processing fails.
func (m *MarkdownProvider) processMarkdownFile(
	ctx context.Context,
	file *discoveredFile,
	collectionName string,
	analyser *pathAnalyser,
) (collection_dto.ContentItem, error) {
	ctx, l := logger_domain.From(ctx, log)
	pathInfo := analyser.Analyse(file.relativePath, collectionName)

	l.Trace("Processing markdown file",
		logger_domain.String(keyPath, file.relativePath),
		logger_domain.String("locale", pathInfo.locale),
		logger_domain.String("slug", pathInfo.slug))

	relativePath := m.resolveContentPath(file.relativePath, collectionName)

	content, err := m.sandbox.ReadFile(relativePath)
	if err != nil {
		return collection_dto.ContentItem{}, fmt.Errorf("reading file: %w", err)
	}

	processed, err := m.markdownService.Process(ctx, content, file.absolutePath)
	if err != nil {
		return collection_dto.ContentItem{}, fmt.Errorf("processing markdown: %w", err)
	}

	return m.buildContentItem(ctx, processed, pathInfo, collectionName, content, file.relativePath)
}

// resolveContentPath returns the sandbox-relative path for a content file.
//
// For external modules, file paths are relative to the sandbox root.
// For local content, the path is prefixed with content/{collection}/.
//
// Takes filePath (string) which is the path to the content file.
// Takes collectionName (string) which identifies the content collection.
//
// Returns string which is the resolved path within the sandbox.
func (m *MarkdownProvider) resolveContentPath(filePath, collectionName string) string {
	if m.isExternalModule {
		return filePath
	}
	return filepath.Join(keyContent, collectionName, filePath)
}

// buildContentItem constructs a ContentItem from processed markdown.
//
// Takes processed (*markdown_dto.ProcessedMarkdown) which contains the parsed
// markdown with metadata and AST.
// Takes pathInfo (*pathInfo) which provides URL, slug, and locale information.
// Takes collectionName (string) which identifies the content collection.
// Takes content ([]byte) which holds the raw markdown content.
// Takes relativePath (string) which specifies the file path for logging.
//
// Returns collection_dto.ContentItem which is the fully populated content item.
// Returns error when content item construction fails.
func (m *MarkdownProvider) buildContentItem(
	ctx context.Context,
	processed *markdown_dto.ProcessedMarkdown,
	pathInfo *pathInfo,
	collectionName string,
	content []byte,
	relativePath string,
) (collection_dto.ContentItem, error) {
	fm := processed.Metadata.Frontmatter
	isDraft := processed.Metadata.Draft
	dates := extractDates(fm)
	if !processed.Metadata.PublishDate.IsZero() {
		dates.publishedAt = processed.Metadata.PublishDate.Format(time.RFC3339)
	}
	override := extractSlugOverride(fm, collectionName, pathInfo.url)

	finalSlug := pathInfo.slug
	if override.slug != "" {
		finalSlug = override.slug
	}

	navMeta := m.extractNavigationMetadata(ctx, processed, pathInfo)
	metadata := buildMetadata(processed, isDraft)
	plainContent := m.extractPlainContent(ctx, processed, relativePath)

	item := collection_dto.ContentItem{
		ID:             finalSlug,
		Slug:           finalSlug,
		Locale:         pathInfo.locale,
		TranslationKey: pathInfo.translationKey,
		Metadata:       metadata,
		RawContent:     string(content),
		PlainContent:   plainContent,
		ContentAST:     processed.PageAST,
		ExcerptAST:     processed.ExcerptAST,
		URL:            override.url,
		ReadingTime:    processed.Metadata.ReadingTime,
		CreatedAt:      dates.createdAt,
		UpdatedAt:      dates.updatedAt,
		PublishedAt:    dates.publishedAt,
	}
	if navMeta != nil {
		item.Metadata[collection_dto.MetaKeyNavigation] = navMeta
	}

	return item, nil
}

// extractPlainContent extracts plain text from AST for search indexing.
//
// Takes processed (*markdown_dto.ProcessedMarkdown) which contains the parsed
// markdown AST.
// Takes relativePath (string) which identifies the file for logging.
//
// Returns string which contains the plain text content, or empty if extraction
// fails.
func (m *MarkdownProvider) extractPlainContent(
	ctx context.Context,
	processed *markdown_dto.ProcessedMarkdown,
	relativePath string,
) string {
	ctx, l := logger_domain.From(ctx, log)
	if m.renderService == nil || processed.PageAST == nil {
		return ""
	}
	plainContent, err := m.renderService.RenderASTToPlainText(ctx, processed.PageAST)
	if err != nil {
		l.Warn("Failed to extract plain text from AST",
			logger_domain.String(keyPath, relativePath),
			logger_domain.Error(err))
		return ""
	}
	return plainContent
}

// linkTranslations populates alternate locale information for translation
// linking.
//
// Takes items ([]collection_dto.ContentItem) which receives alternate locale
// metadata for each content item.
// Takes translationGroups (map[string][]int) which maps translation keys to
// indices of items that are translations of each other.
func (*MarkdownProvider) linkTranslations(
	ctx context.Context,
	items []collection_dto.ContentItem,
	translationGroups map[string][]int,
) {
	_, l := logger_domain.From(ctx, log)
	linkedCount := 0

	for translationKey, indices := range translationGroups {
		if len(indices) < 2 {
			continue
		}

		l.Trace("Linking translations",
			logger_domain.String("translation_key", translationKey),
			logger_domain.Int("locale_count", len(indices)))

		for _, index := range indices {
			item := &items[index]

			if item.Metadata == nil {
				item.Metadata = make(map[string]any)
			}

			alternates := make(map[string]string)
			for _, otherIndex := range indices {
				if otherIndex == index {
					continue
				}
				otherItem := items[otherIndex]
				alternates[otherItem.Locale] = otherItem.URL
			}

			item.Metadata["Alternates"] = alternates
			linkedCount++
		}
	}

	l.Trace("Translation linking complete",
		logger_domain.Int("items_linked", linkedCount))
}

// extractNavigationMetadata derives or extracts navigation metadata for a
// content item.
//
// Priority order:
//  1. Explicit frontmatter nav metadata (if provided)
//  2. Path-based derivation from PathSegments for default "sidebar" group
//  3. Apply path-based defaults to fill in missing fields
//
// This preserves backward compatibility: files without nav frontmatter
// still get organised hierarchically based on their directory structure.
//
// Takes processed (*markdown_dto.ProcessedMarkdown) which provides the parsed
// content with any existing navigation metadata.
// Takes pathInfo (*pathInfo) which contains the path segments for derivation.
//
// Returns *markdown_dto.NavigationMetadata which contains the navigation
// settings, either from frontmatter or derived from the path.
func (*MarkdownProvider) extractNavigationMetadata(
	ctx context.Context,
	processed *markdown_dto.ProcessedMarkdown,
	pathInfo *pathInfo,
) *markdown_dto.NavigationMetadata {
	_, l := logger_domain.From(ctx, log)
	nav := processed.Metadata.Navigation

	if nav == nil || len(nav.Groups) == 0 {
		nav = &markdown_dto.NavigationMetadata{
			Groups: map[string]*markdown_dto.NavGroupMetadata{
				"sidebar": deriveNavFromPath(pathInfo),
			},
		}
		l.Trace("Derived navigation from path",
			logger_domain.Strings("path_segments", pathInfo.pathSegments))
	}

	for groupName, group := range nav.Groups {
		nav.Groups[groupName] = applyNavDefaults(group, pathInfo)
	}

	return nav
}

// getDefaultConfig returns a default configuration.
//
// Uses the basePath field if set (via SetBasePath), otherwise falls back
// to the sandbox root directory.
//
// Returns collection_dto.ProviderConfig which contains sensible defaults
// including common locales and English as the default locale.
func (m *MarkdownProvider) getDefaultConfig() collection_dto.ProviderConfig {
	basePath := m.basePath
	if basePath == "" {
		basePath = m.sandbox.Root()
	}

	return collection_dto.ProviderConfig{
		BasePath:      basePath,
		Locales:       []string{"en", "fr", "de", "es", "it", "pt", "ja", "zh", "ko"},
		DefaultLocale: "en",
		Custom:        make(map[string]any),
	}
}

// contentDates holds date values taken from frontmatter.
type contentDates struct {
	// createdAt is the creation time in RFC3339 format.
	createdAt string

	// updatedAt is when the content was last changed, in RFC3339 format.
	updatedAt string

	// publishedAt is the date the content was published, in RFC3339 format.
	publishedAt string
}

// slugOverride holds a custom slug and URL from frontmatter.
type slugOverride struct {
	// slug overrides the path-based slug when set from frontmatter.
	slug string

	// url is the full URL path for the content item.
	url string
}

// xxHash64 provides FNV-1a hashing despite the misleading name.
// A local version avoids adding an external dependency.
type xxHash64 struct {
	// h is the current hash value.
	h uint64
}

// WriteString adds the bytes of s to the hash state.
//
// Takes s (string) which provides the bytes to add to the hash.
func (x *xxHash64) WriteString(s string) {
	for i := range len(s) {
		x.h ^= uint64(s[i])
		x.h *= fnv1aPrime64
	}
}

// Sum64 returns the current hash value.
//
// Returns uint64 which is the computed hash.
func (x *xxHash64) Sum64() uint64 {
	return x.h
}

// WithModuleResolver sets a module resolver for the provider.
//
// This enables the p-collection-source feature, which lets the provider read
// content from external Go modules.
//
// Takes resolver (resolver_domain.ResolverPort) which handles module path
// lookup.
//
// Returns MarkdownProviderOption which adds module resolution support.
func WithModuleResolver(resolver resolver_domain.ResolverPort) MarkdownProviderOption {
	return func(m *MarkdownProvider) {
		m.resolver = resolver
	}
}

// createModuleSandbox creates a read-only sandbox for accessing content from
// an external Go module.
//
// The sandbox factory is created on-demand with only the content root as an
// allowed path. This is safe because module paths come from go.mod resolution.
//
// Takes moduleBase (string) which is the Go module path for naming.
// Takes contentRoot (string) which is the filesystem path to sandbox.
//
// Returns safedisk.Sandbox which is the configured sandbox.
// Returns error when factory or sandbox creation fails.
func createModuleSandbox(moduleBase, contentRoot string) (safedisk.Sandbox, error) {
	moduleFactory, err := safedisk.NewFactory(safedisk.FactoryConfig{
		Enabled:      true,
		AllowedPaths: []string{contentRoot},
		CWD:          contentRoot,
	})
	if err != nil {
		return nil, fmt.Errorf("creating factory for module content at %q: %w", contentRoot, err)
	}

	moduleSandbox, err := moduleFactory.Create(
		fmt.Sprintf("module-content-%s", moduleBase),
		contentRoot,
		safedisk.ModeReadOnly,
	)
	if err != nil {
		return nil, fmt.Errorf("creating sandbox for module content at %q: %w", contentRoot, err)
	}

	return moduleSandbox, nil
}

// extractDates gets date fields from frontmatter data.
//
// Takes fm (map[string]any) which contains the frontmatter key-value pairs.
//
// Returns contentDates which holds the date strings for published, created,
// and updated times in RFC3339 format.
func extractDates(fm map[string]any) contentDates {
	var dates contentDates
	if fm == nil {
		return dates
	}
	if date, ok := fm["date"].(time.Time); ok && !date.IsZero() {
		dates.publishedAt = date.Format(time.RFC3339)
	}
	if created, ok := fm["created"].(time.Time); ok && !created.IsZero() {
		dates.createdAt = created.Format(time.RFC3339)
	}
	if updated, ok := fm["updated"].(time.Time); ok && !updated.IsZero() {
		dates.updatedAt = updated.Format(time.RFC3339)
	}
	return dates
}

// extractSlugOverride extracts a custom slug from frontmatter if present.
//
// Takes fm (map[string]any) which contains the frontmatter data to search.
// Takes collectionName (string) which specifies the collection path segment.
// Takes defaultURL (string) which provides the fallback URL when no slug is
// found.
//
// Returns slugOverride which contains the extracted slug and the computed URL.
func extractSlugOverride(fm map[string]any, collectionName, defaultURL string) slugOverride {
	override := slugOverride{slug: "", url: defaultURL}
	if fm == nil {
		return override
	}
	if slugVal, ok := fm["slug"].(string); ok && slugVal != "" {
		override.slug = slugVal
		override.url = "/" + collectionName + "/" + slugVal
	}
	return override
}

// buildMetadata creates a metadata map from processed markdown.
//
// Takes processed (*markdown_dto.ProcessedMarkdown) which provides the parsed
// markdown content and its extracted metadata.
// Takes isDraft (bool) which shows whether the content is a draft.
//
// Returns map[string]any which contains the frontmatter and standard metadata
// fields ready for use in templates.
func buildMetadata(processed *markdown_dto.ProcessedMarkdown, isDraft bool) map[string]any {
	metadata := make(map[string]any)
	if processed.Metadata.Frontmatter != nil {
		maps.Copy(metadata, processed.Metadata.Frontmatter)
	}
	metadata[collection_dto.MetaKeyTitle] = processed.Metadata.Title
	metadata[collection_dto.MetaKeyDraft] = isDraft
	metadata[collection_dto.MetaKeyWordCount] = processed.Metadata.WordCount
	metadata[collection_dto.MetaKeySections] = processed.Metadata.Sections
	if processed.Metadata.Description != "" {
		metadata[collection_dto.MetaKeyDescription] = processed.Metadata.Description
	}
	if len(processed.Metadata.Tags) > 0 {
		metadata[collection_dto.MetaKeyTags] = processed.Metadata.Tags
	}
	return metadata
}

// buildCollectionInfo creates a CollectionInfo struct for a markdown
// collection.
//
// Takes name (string) which specifies the collection name.
// Takes sandboxRoot (string) which is the root path of the sandbox.
// Takes collectionPath (string) which is the path to the collection.
// Takes files ([]*discoveredFile) which contains the discovered files.
// Takes locales ([]string) which lists the available locales.
//
// Returns collection_dto.CollectionInfo which contains the collection
// metadata.
func buildCollectionInfo(
	name, sandboxRoot, collectionPath string,
	files []*discoveredFile,
	locales []string,
) collection_dto.CollectionInfo {
	return collection_dto.CollectionInfo{
		Name:      name,
		Path:      filepath.Join(sandboxRoot, collectionPath),
		ItemCount: len(files),
		Locales:   locales,
		Schema: map[string]string{
			"Title":       "string",
			"Description": "string",
			"Date":        "string",
			"Tags":        "[]string",
			"Draft":       "bool",
		},
		Metadata: map[string]any{
			"type":   "markdown",
			"format": "commonmark+frontmatter",
		},
	}
}

// deriveNavFromPath creates navigation metadata from file path structure.
//
// Takes pathInfo (*pathInfo) which contains the parsed path segments.
//
// Returns *markdown_dto.NavGroupMetadata which contains the derived
// navigation metadata with section and subsection populated from the path.
func deriveNavFromPath(pathInfo *pathInfo) *markdown_dto.NavGroupMetadata {
	nav := &markdown_dto.NavGroupMetadata{
		Order:      defaultNavOrder,
		Hidden:     false,
		Section:    "",
		Subsection: "",
		Icon:       "",
		Parent:     "",
		Label:      "",
	}

	segments := pathInfo.pathSegments

	if len(segments) > 1 {
		nav.Section = segments[1]
	}

	if len(segments) > 2 {
		nav.Subsection = segments[2]
	}

	return nav
}

// applyNavDefaults fills in missing navigation fields with path-based defaults.
//
// Frontmatter can partly override navigation while keeping path-based
// derivation for fields that are not set.
//
// Takes group (*markdown_dto.NavGroupMetadata) which contains the navigation
// metadata to fill with defaults.
// Takes pathInfo (*pathInfo) which provides path context for deriving defaults.
//
// Returns *markdown_dto.NavGroupMetadata which is the same group with any empty
// fields set from path-based derivation.
func applyNavDefaults(
	group *markdown_dto.NavGroupMetadata,
	pathInfo *pathInfo,
) *markdown_dto.NavGroupMetadata {
	pathNav := deriveNavFromPath(pathInfo)

	if group.Section == "" {
		group.Section = pathNav.Section
	}
	if group.Subsection == "" {
		group.Subsection = pathNav.Subsection
	}
	if group.Order == 0 {
		group.Order = defaultNavOrder
	}

	return group
}

// sortStrings sorts a slice of strings in place in ascending order.
//
// Takes s ([]string) which is the slice to sort.
func sortStrings(s []string) {
	for i := range len(s) - 1 {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

// newXXHash64 creates a new xxHash64 hash instance.
//
// Returns *xxHash64 which is set to the FNV-1a offset basis.
func newXXHash64() *xxHash64 {
	return &xxHash64{h: fnv1aOffset64}
}
