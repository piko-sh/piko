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

package seo_domain

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"time"

	"piko.sh/piko/internal/capabilities/capabilities_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/seo/seo_dto"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// sourceVariant is the registry variant ID of the original uploaded artefact; the
	// sitemap compression profiles consume it as their input.
	sourceVariant = "source"

	// sitemapAssetType labels the resulting compressed sitemap variants.
	sitemapAssetType = "sitemap"

	// sitemapStorageBackendID is the blob backend SEO artefacts are written to; it must
	// match the backend used by RegistryStorageAdapter ("local_disk_cache").
	sitemapStorageBackendID = "local_disk_cache"

	// sitemapMimeType is the content type served for sitemap variants.
	sitemapMimeType = "application/xml"

	// logFieldHostname is the log field key for the configured sitemap hostname.
	logFieldHostname = "hostname"
)

// seoService generates SEO files such as sitemaps and robots.txt. It implements
// SEOService, SEOServicePort, and Probe interfaces.
type seoService struct {
	// storagePort handles writing sitemaps and robots.txt files to storage.
	storagePort SEOStoragePort

	// sitemapBuilder builds XML sitemaps for the site.
	sitemapBuilder *sitemapBuilder

	// robotsBuilder builds the content for robots.txt files.
	robotsBuilder *robotsBuilder

	// config holds the SEO settings for sitemap and robots.txt generation.
	config config.SEOConfig
}

// GenerateArtefacts creates sitemap.xml and robots.txt files and stores them in the
// registry. It implements the SEOService interface.
//
// robots.txt is attempted even when the sitemap fails, and both errors are reported
// together. Returning early on the sitemap would lose robots.txt to an unrelated storage
// fault, and a build that ships neither artefact serves 404 for /robots.txt, which
// crawlers read as permission to crawl everything.
//
// Takes view (*seo_dto.ProjectView) which provides the project data for sitemap
// generation.
//
// Returns error when sitemap generation fails, robots.txt cannot be stored, or both.
func (s *seoService) GenerateArtefacts(ctx context.Context, view *seo_dto.ProjectView) error {
	ctx, l := logger_domain.From(ctx, log)
	return l.RunInSpan(ctx, "GenerateArtefacts", func(spanCtx context.Context, _ logger_domain.Logger) error {
		startTime := time.Now()

		totalURLs, sitemapErr := s.generateAndStoreSitemaps(spanCtx, view)
		if sitemapErr != nil {
			sitemapErr = fmt.Errorf("generating and storing sitemaps: %w", sitemapErr)
		}

		robotsErr := s.generateAndStoreRobotsTxt(spanCtx)
		if robotsErr != nil {
			robotsErr = fmt.Errorf("generating and storing robots.txt: %w", robotsErr)
		}

		if err := errors.Join(sitemapErr, robotsErr); err != nil {
			return err
		}

		s.recordMetrics(spanCtx, startTime, totalURLs)

		return nil
	})
}

// Name returns the probe's identifier for health check reporting. It implements the
// healthprobe_domain.Probe interface.
//
// Returns string which is the probe name "SEOService".
func (*seoService) Name() string {
	return "SEOService"
}

// Check implements the healthprobe_domain.Probe interface. It checks whether the SEO
// service is working.
//
// Returns healthprobe_dto.Status which indicates healthy state when fully operational, or
// degraded state when the storage port is not initialised.
func (s *seoService) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	state := healthprobe_dto.StateHealthy
	message := "SEO service operational"

	if s.storagePort == nil {
		state = healthprobe_dto.StateDegraded
		message = "SEO service storage not initialised (sitemap generation may be affected)"
	}

	return healthprobe_dto.Status{
		Name:         s.Name(),
		State:        state,
		Message:      message,
		Timestamp:    time.Now(),
		Duration:     time.Since(startTime).String(),
		Dependencies: []*healthprobe_dto.Status{},
	}
}

// generateAndStoreSitemaps builds and stores all sitemap files.
//
// Takes view (*seo_dto.ProjectView) which provides the project data for sitemap creation.
//
// Returns int which is the total number of URLs in the sitemaps.
// Returns error when building or storing the sitemap fails.
func (s *seoService) generateAndStoreSitemaps(
	ctx context.Context,
	view *seo_dto.ProjectView,
) (int, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Generating sitemap...")
	result, err := s.sitemapBuilder.Build(ctx, view)
	if err != nil {
		return 0, fmt.Errorf("building sitemap: %w", err)
	}

	compressionProfiles := s.buildCompressionProfiles()

	if result.Index != nil {
		return s.storeSitemapWithIndex(ctx, result, compressionProfiles)
	}

	return s.storeSingleSitemap(ctx, result.Sitemaps[0], compressionProfiles)
}

// storeSitemapWithIndex stores several sitemap chunks and a sitemap index.
//
// Takes result (*seo_dto.SitemapBuildResult) which contains the sitemaps and index to
// store.
// Takes compressionProfiles ([]registry_dto.NamedProfile) which specifies the compression
// formats to use.
//
// Returns int which is the total number of URLs stored across all chunks.
// Returns error when storing a sitemap chunk fails or the index cannot be saved.
func (s *seoService) storeSitemapWithIndex(
	ctx context.Context,
	result *seo_dto.SitemapBuildResult,
	compressionProfiles []registry_dto.NamedProfile,
) (int, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Storing sitemap chunks", logger_domain.Int("chunk_count", len(result.Sitemaps)))

	totalURLs := 0
	for i := range result.Sitemaps {
		sitemap := &result.Sitemaps[i]
		artefactID := fmt.Sprintf("sitemap-%d.xml", i+1)

		if err := s.marshalAndStoreSitemap(ctx, sitemap, artefactID, compressionProfiles); err != nil {
			return 0, fmt.Errorf("storing sitemap chunk %d: %w", i+1, err)
		}

		totalURLs += len(sitemap.URLs)
		l.Trace("Stored sitemap chunk",
			logger_domain.String("artefact_id", artefactID),
			logger_domain.Int("url_count", len(sitemap.URLs)))
	}

	if err := s.storeSitemapIndexFile(ctx, result.Index, compressionProfiles); err != nil {
		return 0, err
	}

	l.Trace("Sitemap index generated successfully",
		logger_domain.Int("total_urls", totalURLs),
		logger_domain.Int("chunk_count", len(result.Sitemaps)))

	return totalURLs, nil
}

// storeSingleSitemap saves a single sitemap file as sitemap.xml.
//
// Takes sitemap (seo_dto.Sitemap) which contains the URLs to store.
// Takes compressionProfiles ([]registry_dto.NamedProfile) which specifies which
// compression formats to use.
//
// Returns int which is the number of URLs stored.
// Returns error when XML conversion fails or storage fails.
func (s *seoService) storeSingleSitemap(
	ctx context.Context,
	sitemap seo_dto.Sitemap,
	compressionProfiles []registry_dto.NamedProfile,
) (int, error) {
	xmlData, err := xml.MarshalIndent(sitemap, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("marshalling sitemap to XML: %w", err)
	}

	xmlContent := []byte(xml.Header + string(xmlData))
	if err := s.storagePort.StoreSitemap(ctx, "sitemap.xml", xmlContent, compressionProfiles); err != nil {
		return 0, fmt.Errorf("storing sitemap: %w", err)
	}

	_, l := logger_domain.From(ctx, log)
	totalURLs := len(sitemap.URLs)
	l.Trace("Sitemap generated successfully",
		logger_domain.Int("url_count", totalURLs),
		logger_domain.Int("size_bytes", len(xmlContent)))

	return totalURLs, nil
}

// marshalAndStoreSitemap marshals a sitemap to XML and stores it.
//
// Takes sitemap (*seo_dto.Sitemap) which provides the sitemap data to marshal.
// Takes artefactID (string) which identifies where to store the result.
// Takes compressionProfiles ([]registry_dto.NamedProfile) which specifies which
// compression formats to apply.
//
// Returns error when XML marshalling fails or storage fails.
func (s *seoService) marshalAndStoreSitemap(
	ctx context.Context,
	sitemap *seo_dto.Sitemap,
	artefactID string,
	compressionProfiles []registry_dto.NamedProfile,
) error {
	xmlData, err := xml.MarshalIndent(sitemap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling sitemap to XML: %w", err)
	}

	xmlContent := []byte(xml.Header + string(xmlData))
	if err := s.storagePort.StoreSitemap(ctx, artefactID, xmlContent, compressionProfiles); err != nil {
		return fmt.Errorf("storing sitemap %s: %w", artefactID, err)
	}

	return nil
}

// storeSitemapIndexFile marshals and stores a sitemap index as sitemap.xml.
//
// Takes index (*seo_dto.SitemapIndex) which specifies the sitemap index to marshal.
// Takes compressionProfiles ([]registry_dto.NamedProfile) which defines the compression
// formats to apply when storing.
//
// Returns error when XML marshalling fails or storage fails.
func (s *seoService) storeSitemapIndexFile(
	ctx context.Context,
	index *seo_dto.SitemapIndex,
	compressionProfiles []registry_dto.NamedProfile,
) error {
	indexXML, err := xml.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling sitemap index to XML: %w", err)
	}

	indexContent := []byte(xml.Header + string(indexXML))
	if err := s.storagePort.StoreSitemap(ctx, "sitemap.xml", indexContent, compressionProfiles); err != nil {
		return fmt.Errorf("storing sitemap index: %w", err)
	}

	return nil
}

// generateAndStoreRobotsTxt builds and stores the robots.txt file.
//
// Returns error when building or storing the robots.txt content fails.
func (s *seoService) generateAndStoreRobotsTxt(
	ctx context.Context,
) error {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Generating robots.txt...")
	robotsTxt, err := s.robotsBuilder.Build(ctx, sitemapIndexURL(s.config.Sitemap.Hostname))
	if err != nil {
		return fmt.Errorf("building robots.txt: %w", err)
	}

	if err := s.storagePort.StoreRobotsTxt(ctx, robotsTxt); err != nil {
		return fmt.Errorf("storing robots.txt: %w", err)
	}

	l.Trace("Robots.txt generated successfully", logger_domain.Int("size_bytes", len(robotsTxt)))
	return nil
}

// recordMetrics logs and records metrics for the generation process.
//
// Takes startTime (time.Time) which marks when generation began.
// Takes totalURLs (int) which is the count of URLs processed.
func (*seoService) recordMetrics(
	ctx context.Context,
	startTime time.Time,
	totalURLs int,
) {
	ctx, l := logger_domain.From(ctx, log)
	duration := time.Since(startTime).Milliseconds()
	sitemapGenerationDuration.Record(ctx, float64(duration))
	sitemapURLCount.Add(ctx, int64(totalURLs))
	robotsTxtGenerationCount.Add(ctx, 1)

	l.Trace("SEO artefacts generated successfully",
		logger_domain.Int64("duration_ms", duration),
		logger_domain.Int("total_urls", totalURLs))
}

// buildCompressionProfiles creates compression profiles for sitemap storage.
//
// Returns []registry_dto.NamedProfile which contains profiles for gzip and brotli
// compression.
func (*seoService) buildCompressionProfiles() []registry_dto.NamedProfile {
	makeCompressProfile := func(name, capability, encoding, ext string) registry_dto.NamedProfile {
		var tags registry_dto.Tags
		tags.SetByName("type", sitemapAssetType)
		tags.SetByName("contentEncoding", encoding)
		tags.SetByName("storageBackendId", sitemapStorageBackendID)
		tags.SetByName("mimeType", sitemapMimeType)
		tags.SetByName("fileExtension", ext)

		var deps registry_dto.Dependencies
		deps.Add(sourceVariant)

		return registry_dto.NamedProfile{
			Name: name,
			Profile: registry_dto.DesiredProfile{
				Priority:       registry_dto.PriorityWant,
				CapabilityName: capability,
				DependsOn:      deps,
				ResultingTags:  tags,
			},
		}
	}

	return []registry_dto.NamedProfile{
		makeCompressProfile("gzip", capabilities_dto.CapabilityCompressGzip.String(), "gzip", ".xml.gz"),
		makeCompressProfile("br", capabilities_dto.CapabilityCompressBrotli.String(), "br", ".xml.br"),
	}
}

// SEOServiceOption configures the SEO service during creation.
type SEOServiceOption func(*seoServiceOptions)

// seoServiceOptions holds optional configuration for the SEO service.
type seoServiceOptions struct {
	// sandboxFactory holds the sandbox factory for file operations such as checking sitemap
	// file modification times.
	sandboxFactory safedisk.Factory

	// urlProvider supplies additional sitemap URLs at build time, in-process. Optional.
	urlProvider SitemapURLProvider

	// productionMode reports whether this is a production build.
	//
	// Nil means unknown, which is treated as production so a plumbing gap fails open (a live
	// site keeps indexing) rather than accidentally de-indexing production. When explicitly
	// false, robots.txt blocks all crawlers.
	productionMode *bool

	// i18nStrategy is the locale-routing strategy (one of i18n_domain.Strategy*) used when
	// building localised <loc> values and hreflang alternates. Empty behaves as bare
	// patterns (query-only / disabled).
	i18nStrategy string

	// i18nLocales is the full configured locale set, supplied to route sources via
	// RouteContext.
	i18nLocales []string

	// routeSources enumerate URLs for pages bound to a p-route-source directive.
	routeSources []RouteSource
}

// WithSEOSandboxFactory sets a sandbox factory for file operations such as checking
// sitemap file modification times.
//
// Takes factory (safedisk.Factory) which provides sandboxed filesystem access.
//
// Returns SEOServiceOption which configures the sandbox factory on the service.
func WithSEOSandboxFactory(factory safedisk.Factory) SEOServiceOption {
	return func(o *seoServiceOptions) {
		o.sandboxFactory = factory
	}
}

// WithSEOURLProvider sets a build-time, in-process provider of additional sitemap URLs.
// Use it for dynamic routes whose URLs come from application data rather than content
// collections (auto-expanded) or runtime HTTP Sources.
//
// Takes provider (SitemapURLProvider) which enumerates the extra URLs during generation.
//
// Returns SEOServiceOption which configures the URL provider on the service.
func WithSEOURLProvider(provider SitemapURLProvider) SEOServiceOption {
	return func(o *seoServiceOptions) {
		o.urlProvider = provider
	}
}

// WithI18nStrategy sets the locale-routing strategy (one of i18n_domain.Strategy*) used
// when building localised sitemap URLs and hreflang alternates. Passing the same strategy
// the runtime router uses keeps the sitemap and the served routes in lockstep.
//
// Takes strategy (string) which is the i18n routing strategy.
//
// Returns SEOServiceOption which configures the strategy on the service.
func WithI18nStrategy(strategy string) SEOServiceOption {
	return func(o *seoServiceOptions) {
		o.i18nStrategy = strategy
	}
}

// WithProductionMode declares whether SEO artefacts are generated for a production build.
//
// In non-production builds robots.txt blocks all crawlers, so a development rebuild
// cannot leave a permissive file behind in the local registry. Only a run mode supplies
// this; when the option is not supplied the service assumes production, so a missing
// wiring path fails open rather than de-indexing a live site. To withhold a real deploy,
// use RobotsConfig.NeverIndex or RobotsConfig.PreviewDeployment.
//
// Takes isProduction (bool) which is true for a production build.
//
// Returns SEOServiceOption which configures the production flag on the service.
func WithProductionMode(isProduction bool) SEOServiceOption {
	return func(o *seoServiceOptions) {
		o.productionMode = &isProduction
	}
}

// WithI18nLocales sets the full configured locale set, supplied to route sources via
// RouteContext when a RouteURL does not name its own locales.
//
// Takes locales ([]string) which is the configured locale set.
//
// Returns SEOServiceOption which configures the locale set on the service.
func WithI18nLocales(locales []string) SEOServiceOption {
	return func(o *seoServiceOptions) {
		o.i18nLocales = locales
	}
}

// WithRouteSources registers build-time route sources that enumerate the concrete URLs
// for pages bound to a p-route-source directive. It is additive across calls-worth of
// sources supplied here.
//
// Takes sources ([]RouteSource) which are the route sources.
//
// Returns SEOServiceOption which configures the sources on the service.
func WithRouteSources(sources []RouteSource) SEOServiceOption {
	return func(o *seoServiceOptions) {
		o.routeSources = append(o.routeSources, sources...)
	}
}

// resolveBlockAllIndexing decides whether the generated robots.txt replaces its
// permissive base rule with a site-wide "Disallow: /", and logs the outcome either way so
// the indexing posture of a build is never silent.
//
// Takes seoConfig (config.SEOConfig) which carries the robots settings and the hostname
// used to identify the site in the log message.
// Takes productionMode (*bool) which is the caller's production signal, or nil when none
// was supplied.
//
// Returns bool which is true when every crawler should be blocked.
func resolveBlockAllIndexing(seoConfig config.SEOConfig, productionMode *bool) bool {
	isProduction := productionMode == nil || *productionMode
	blockAllIndexing := seoConfig.Robots.NeverIndex || !isProduction

	switch {
	case seoConfig.Robots.NeverIndex:
		log.Warn("SEO: robots.neverIndex is set - robots.txt will block all crawlers",
			logger_domain.String(logFieldHostname, seoConfig.Sitemap.Hostname))
	case blockAllIndexing:
		log.Warn("SEO: development build - robots.txt will block all crawlers. A production build is unaffected",
			logger_domain.String(logFieldHostname, seoConfig.Sitemap.Hostname))
	default:
		log.Notice("SEO: robots.txt allows all crawlers. Set robots.neverIndex for a project that must never be indexed, or robots.previewDeployment on a deploy that is not the live one",
			logger_domain.String(logFieldHostname, seoConfig.Sitemap.Hostname))
	}

	return blockAllIndexing
}

// NewSEOService creates a new SEO service with all required dependencies.
//
// Takes seoConfig (config.SEOConfig) which specifies the SEO settings.
// Takes i18nDefaultLocale (string) which specifies the default locale for sitemap
// generation.
// Takes storagePort (SEOStoragePort) which provides the storage backend for SEO data.
// Takes dynamicURLPort (DynamicURLSourcePort) which supplies dynamic URL sources for
// sitemap generation.
//
// Returns SEOService which is the configured service ready for use.
// Returns error when SEO is disabled in the configuration or when the sitemap hostname is
// not set.
func NewSEOService(
	seoConfig config.SEOConfig,
	i18nDefaultLocale string,
	storagePort SEOStoragePort,
	dynamicURLPort DynamicURLSourcePort,
	opts ...SEOServiceOption,
) (SEOService, error) {
	var options seoServiceOptions
	for _, opt := range opts {
		opt(&options)
	}

	if !seoConfig.Enabled {
		return nil, errors.New("SEO service is disabled in configuration")
	}

	if seoConfig.Sitemap.Hostname == "" {
		return nil, errors.New("sitemap hostname is required but not configured")
	}

	var sitemapOpts []sitemapBuilderOption
	if options.sandboxFactory != nil {
		sitemapOpts = append(sitemapOpts, withSitemapSandboxFactory(options.sandboxFactory))
	}
	if options.urlProvider != nil {
		sitemapOpts = append(sitemapOpts, withSitemapURLProvider(options.urlProvider))
	}
	if options.i18nStrategy != "" {
		sitemapOpts = append(sitemapOpts, withSitemapI18nStrategy(options.i18nStrategy))
	}
	if len(options.i18nLocales) > 0 {
		sitemapOpts = append(sitemapOpts, withSitemapI18nLocales(options.i18nLocales))
	}
	if len(options.routeSources) > 0 {
		sitemapOpts = append(sitemapOpts, withSitemapRouteSources(options.routeSources))
	}

	sitemapBuilder := newSitemapBuilder(
		seoConfig.Sitemap,
		i18nDefaultLocale,
		dynamicURLPort,
		sitemapOpts...,
	)

	robotsBuilder := newRobotsBuilder(seoConfig.Robots, resolveBlockAllIndexing(seoConfig, options.productionMode))

	return &seoService{
		storagePort:    storagePort,
		sitemapBuilder: sitemapBuilder,
		robotsBuilder:  robotsBuilder,
		config:         seoConfig,
	}, nil
}
