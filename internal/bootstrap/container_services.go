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

package bootstrap

// This file contains various service accessors: SEO, I18n, Image,
// CSRF, PML, Render, Validator, HealthProbe, and Frontend modules.

import (
	"fmt"
	"maps"

	"piko.sh/piko/internal/component/component_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/healthprobe/healthprobe_domain"
	"piko.sh/piko/internal/highlight/highlight_domain"
	"piko.sh/piko/internal/i18n/i18n_adapters"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/internal/image/image_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/markdown/markdown_domain"
	"piko.sh/piko/internal/pml/pml_adapters"
	"piko.sh/piko/internal/pml/pml_components"
	"piko.sh/piko/internal/pml/pml_domain"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/security/security_adapters"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/seo/seo_adapters"
	"piko.sh/piko/internal/seo/seo_domain"
	"piko.sh/piko/internal/video/video_domain"
	"piko.sh/piko/wdk/safedisk"
)

// GetCSRFService returns the CSRF token service, creating it if needed.
//
// Returns security_domain.CSRFTokenService which provides CSRF token
// creation and validation.
func (c *Container) GetCSRFService() security_domain.CSRFTokenService {
	c.csrfOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.csrfServiceOverride != nil {
			l.Internal("Using provided CSRFService override.")
			c.csrfService = c.csrfServiceOverride
			return
		}
		c.createDefaultCSRFService()
	})
	return c.csrfService
}

// createDefaultCSRFService sets up the CSRF token service with default
// settings.
func (c *Container) createDefaultCSRFService() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating default CSRFService with cookie source...")
	secret := c.csrfSecretKeyProvider()
	serverConfig := c.serverConfig

	cookieSource := c.csrfCookieSourceOverride
	if cookieSource == nil {
		isHTTPS := deref(serverConfig.Network.ForceHTTPS, false)

		secureCookieWriter := security_adapters.NewSecureCookieWriter(NewCookieSecurityValues(&serverConfig.Security.Cookies), isHTTPS)

		cookieSource = security_adapters.NewCookieCSRFSourceAdapter(secureCookieWriter, 0)
	}
	c.csrfCookieSource = cookieSource

	var err error
	c.csrfService, err = security_domain.NewCSRFTokenService(
		security_domain.SecurityConfig{
			HMACSecretKey:   secret,
			CSRFTokenMaxAge: c.csrfTokenMaxAge,
		},
		security_adapters.NewIPBinderAdapter(),
		cookieSource,
	)
	if err != nil {
		l.Error("Failed to create default CSRF service", logger_domain.Error(err))
	}
}

// SetCSRFCookieSource sets a custom CSRF cookie source adapter. Use it with
// custom session management systems.
//
// Takes source (CSRFCookieSourceAdapter) which provides CSRF cookie handling.
func (c *Container) SetCSRFCookieSource(source security_domain.CSRFCookieSourceAdapter) {
	c.csrfCookieSourceOverride = source
	c.csrfCookieSource = source
}

// GetCSRFCookieSource returns the CSRF cookie source adapter. This is
// typically used internally but may be useful for custom CSRF handling.
//
// Returns security_domain.CSRFCookieSourceAdapter which provides access to
// CSRF cookie operations.
func (c *Container) GetCSRFCookieSource() security_domain.CSRFCookieSourceAdapter {
	_ = c.GetCSRFService()
	return c.csrfCookieSource
}

// SetCSPConfig sets the Content-Security-Policy builder configuration. This
// allows programmatic CSP configuration via piko.WithCSP().
//
// Takes builder (*security_domain.CSPBuilder) which is the configured CSP
// builder.
func (c *Container) SetCSPConfig(builder *security_domain.CSPBuilder) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	c.cspBuilder = builder
	l.Internal("CSP builder configured",
		logger_domain.Bool("report_only", builder.IsReportOnly()),
		logger_domain.Bool("uses_request_tokens", builder.UsesRequestTokens()))
}

// SetCSPPolicyString sets a raw CSP policy string directly, bypassing the
// builder pattern. An empty string turns off the CSP header.
//
// Takes policy (string) which is the raw CSP header value.
func (c *Container) SetCSPPolicyString(policy string) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	c.cspPolicyString = policy
	c.cspPolicyStringSet = true
	if policy == "" {
		l.Internal("CSP disabled via empty policy string")
	} else {
		l.Internal("CSP configured via raw policy string")
	}
}

// SetCrossOriginResourcePolicy sets the Cross-Origin-Resource-Policy header
// value. This controls which origins can load resources from this server.
//
// Takes policy (string) which specifies the CORP policy value:
//   - "same-origin" (default): Only same-origin requests can load resources
//   - "same-site": Same-site requests can load resources
//   - "cross-origin": Any origin can load resources (required for headless CMS)
func (c *Container) SetCrossOriginResourcePolicy(policy string) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	c.crossOriginResourcePolicy = policy
	l.Internal("Cross-Origin-Resource-Policy configured",
		logger_domain.String("policy", policy))
}

// GetCrossOriginResourcePolicy returns the CORP policy override, if set.
// Returns an empty string if no override was configured.
//
// Returns string which is the configured CORP policy override.
func (c *Container) GetCrossOriginResourcePolicy() string {
	return c.crossOriginResourcePolicy
}

// GetCSPConfig returns the CSP builder configuration, if set.
// Returns nil if no CSP builder has been configured.
//
// Returns *security_domain.CSPBuilder which is the configured builder, or nil.
func (c *Container) GetCSPConfig() *security_domain.CSPBuilder {
	return c.cspBuilder
}

// GetCSPPolicyString returns the raw CSP policy string, if set.
// This takes precedence over the builder if both are set.
//
// Returns (string, bool) where the string is the policy and the bool indicates
// whether a raw policy string was explicitly set via SetCSPPolicyString.
func (c *Container) GetCSPPolicyString() (string, bool) {
	return c.cspPolicyString, c.cspPolicyStringSet
}

// GetCSPPolicy returns the effective Content Security Policy string to use.
// It checks sources in priority order: raw policy string set via
// WithCSPString (which allows disabling with an empty string), then builder
// via WithCSP, then config file value, and finally Piko defaults.
//
// Returns string which is the CSP header value to use.
func (c *Container) GetCSPPolicy() string {
	if rawPolicy, wasSet := c.GetCSPPolicyString(); wasSet {
		return rawPolicy
	}

	if c.cspBuilder != nil {
		return c.cspBuilder.Build()
	}

	if csp := deref(c.serverConfig.Security.Headers.ContentSecurityPolicy, ""); csp != "" {
		return csp
	}

	return security_domain.NewCSPBuilder().WithPikoDefaults().Build()
}

// SetReportingEndpoints sets the reporting endpoints for the
// Reporting-Endpoints HTTP header.
//
// Takes endpoints ([]config.ReportingEndpoint) which lists the named endpoints
// to include in the header.
func (c *Container) SetReportingEndpoints(endpoints []config.ReportingEndpoint) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	c.reportingEndpoints = endpoints
	l.Internal("Reporting endpoints configured",
		logger_domain.Int("endpoint_count", len(endpoints)))
}

// GetReportingConfig returns the effective reporting configuration for the
// middleware. Endpoints set via WithReportingEndpoints take priority over
// config file values.
//
// Returns config.ReportingConfig which contains the enabled state and
// endpoints.
func (c *Container) GetReportingConfig() config.ReportingConfig {
	if len(c.reportingEndpoints) > 0 {
		return config.ReportingConfig{
			Enabled:   new(true),
			Endpoints: c.reportingEndpoints,
		}
	}

	return c.serverConfig.Security.Reporting
}

// GetPMLTransformer returns the PikoMarkupLanguage transformer, creating it
// if needed.
//
// Returns pml_domain.Transformer which is the configured transformer instance.
func (c *Container) GetPMLTransformer() pml_domain.Transformer {
	c.pmlTransformerOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.pmlTransformerOverride != nil {
			l.Internal("Using provided PML Transformer override.")
			c.pmlTransformer = c.pmlTransformerOverride
			return
		}
		c.createDefaultPMLTransformer()
	})
	return c.pmlTransformer
}

// createDefaultPMLTransformer sets up the default PML transformer with
// built-in components and collectors.
func (c *Container) createDefaultPMLTransformer() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating default PML Transformer...")

	pmlRegistry, err := pml_components.RegisterBuiltIns(c.GetAppContext())
	if err != nil {
		l.Error("Failed to register PML built-in components", logger_domain.Error(err))
		c.pmlTransformer = nil
		return
	}

	mediaQueryCollector := pml_adapters.NewMediaQueryCollector()
	msoConditionalCollector := pml_adapters.NewMSOConditionalCollector()

	c.pmlTransformer = pml_domain.NewTransformer(
		pmlRegistry,
		mediaQueryCollector,
		msoConditionalCollector,
	)
}

// SetPMLTransformer sets a custom PML transformer for the container.
//
// Takes transformer (pml_domain.Transformer) which provides the custom
// transformation logic.
func (c *Container) SetPMLTransformer(transformer pml_domain.Transformer) {
	c.pmlTransformerOverride = transformer
	c.pmlTransformer = transformer
}

// GetRenderer returns the component rendering service, creating it if needed.
//
// Returns render_domain.RenderService which provides component rendering.
func (c *Container) GetRenderer() render_domain.RenderService {
	c.rendererOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.rendererOverride != nil {
			l.Internal("Using provided Renderer override.")
			c.renderer = c.rendererOverride
			return
		}
		c.createDefaultRenderer()
	})
	return c.renderer
}

// createDefaultRenderer sets up the default renderer component.
func (c *Container) createDefaultRenderer() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating default Renderer (RenderOrchestrator)...")

	renderOpts := []render_domain.RenderOrchestratorOption{
		render_domain.WithStripHTMLComments(c.experimentalCommentStripping),
	}

	if c.cssResetCSS != "" {
		renderOpts = append(renderOpts, render_domain.WithCSSResetCSS(c.cssResetCSS))
	}

	if captchaService, captchaErr := c.GetCaptchaService(); captchaErr == nil {
		renderOpts = append(renderOpts, render_domain.WithCaptchaService(captchaService))
	} else {
		_, l := logger_domain.From(c.GetAppContext(), log)
		l.Internal("Captcha service not available for renderer", logger_domain.Error(captchaErr))
	}

	c.renderer = render_domain.NewRenderOrchestrator(
		c.GetPMLTransformer(),
		[]render_domain.TransformationPort{},
		c.GetRenderRegistry(),
		c.GetCSRFService(),
		renderOpts...,
	)
}

// GetI18nService returns the internationalisation service, creating it if
// necessary.
//
// Returns i18n_domain.Service which provides translation and localisation.
// Returns error when the service cannot be created.
func (c *Container) GetI18nService() (i18n_domain.Service, error) {
	c.i18nOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.i18nServiceOverride != nil {
			l.Internal("Using provided I18nService override.")
			c.i18nService = c.i18nServiceOverride
			return
		}
		c.createDefaultI18nService()
	})
	return c.i18nService, c.i18nErr
}

// createDefaultI18nService sets up the default translation service.
// It creates a read-only sandbox for translation file operations and loads
// the translations.
func (c *Container) createDefaultI18nService() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating default I18nService...")
	serverConfig := c.serverConfig

	i18nSandbox, err := c.createSandbox("i18n-source", deref(serverConfig.Paths.BaseDir, "."), safedisk.ModeReadOnly)
	if err != nil {
		c.i18nErr = err
		return
	}

	c.i18nService, c.i18nErr = i18n_adapters.NewService(
		c.GetAppContext(), i18nSandbox,
		deref(serverConfig.I18nDefaultLocale, "en"), deref(serverConfig.Paths.I18nSourceDir, "locales"),
	)
}

// GetImageService returns the image service, creating a default one if none
// was set.
//
// Returns image_domain.Service which is the image service instance.
// Returns error when creating the default service fails.
func (c *Container) GetImageService() (image_domain.Service, error) {
	c.imageOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.imageServiceOverride != nil {
			l.Internal("Using provided ImageService override.")
			c.imageService = c.imageServiceOverride
			return
		}
		c.createDefaultImageService()
	})
	return c.imageService, c.imageErr
}

// SetImageService sets a custom image service implementation.
//
// If the service has a shutdown method (Close, Shutdown, or Stop), it will be
// registered for graceful shutdown.
//
// Takes service (image_domain.Service) which provides the image service to use.
func (c *Container) SetImageService(service image_domain.Service) {
	c.imageServiceOverride = service
	c.imageService = service
	registerCloseableForShutdown(c.GetAppContext(), "ImageService", service)
}

// createDefaultImageService sets up the default image service with the
// registered transformers.
func (c *Container) createDefaultImageService() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	if len(c.imageTransformers) == 0 {
		l.Internal("No image transformers configured; image service disabled")
		c.imageService = nil
		return
	}

	l.Internal("Creating default ImageService with registered transformers...",
		logger_domain.Int("transformer_count", len(c.imageTransformers)),
		logger_domain.String("default_transformer", c.defaultImageTransformer))

	var imageConfig image_domain.ServiceConfig
	if c.imageServiceConfigOverride != nil {
		imageConfig = *c.imageServiceConfigOverride
	} else {
		imageConfig = image_domain.DefaultServiceConfig()
	}

	transformers := make(map[string]image_domain.TransformerPort, len(c.imageTransformers)+1)
	maps.Copy(transformers, c.imageTransformers)
	if _, hasDefault := transformers[image_dto.ImageNameDefault]; !hasDefault && c.defaultImageTransformer != "" {
		transformers[image_dto.ImageNameDefault] = c.imageTransformers[c.defaultImageTransformer]
	}

	c.imageService, c.imageErr = image_domain.NewService(transformers, c.defaultImageTransformer, imageConfig)
}

// GetVideoService returns the video service, creating a default one if none
// was provided.
//
// Returns video_domain.Service which is the video service instance.
// Returns error when the default service creation fails.
func (c *Container) GetVideoService() (video_domain.Service, error) {
	c.videoOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.videoServiceOverride != nil {
			l.Internal("Using provided VideoService override.")
			c.videoService = c.videoServiceOverride
			return
		}
		c.createDefaultVideoService()
	})
	return c.videoService, c.videoErr
}

// SetVideoService sets a custom video service implementation.
//
// If the service implements a shutdown interface (Close, Shutdown, or Stop),
// it will be registered for graceful shutdown.
//
// Takes service (video_domain.Service) which provides the video service to use.
func (c *Container) SetVideoService(service video_domain.Service) {
	c.videoServiceOverride = service
	c.videoService = service
	registerCloseableForShutdown(c.GetAppContext(), "VideoService", service)
}

// AddImageTransformer registers a named image transformer for image processing.
//
// If the transformer implements a shutdown interface (Close, Shutdown, or
// Stop), it will be automatically registered for graceful shutdown.
//
// The first transformer registered becomes the default unless
// SetDefaultImageTransformer is called.
//
// Takes name (string) which identifies the transformer for later retrieval.
// Takes transformer (TransformerPort) which handles image transformations.
func (c *Container) AddImageTransformer(name string, transformer image_domain.TransformerPort) {
	if c.imageTransformers == nil {
		c.imageTransformers = make(map[string]image_domain.TransformerPort)
	}
	c.imageTransformers[name] = transformer
	if c.defaultImageTransformer == "" {
		c.defaultImageTransformer = name
	}
	registerCloseableForShutdown(c.GetAppContext(), "ImageTransformer-"+name, transformer)
}

// SetDefaultImageTransformer sets the name of the transformer to use as
// default.
//
// Takes name (string) which is the transformer name to use as the default.
func (c *Container) SetDefaultImageTransformer(name string) {
	c.defaultImageTransformer = name
}

// SetImageConfig applies the given image configuration to the container.
// This is the recommended way to configure image processing with full control
// using the builder pattern.
//
// Takes imageConfig (*image_domain.ImageConfig) which contains the complete
// settings including providers, predefined variants, and service options.
// Returns without making changes when imageConfig is nil.
func (c *Container) SetImageConfig(imageConfig *image_domain.ImageConfig) {
	if imageConfig == nil {
		return
	}

	for name, transformer := range imageConfig.Providers {
		c.AddImageTransformer(name, transformer)
	}

	if imageConfig.DefaultProvider != "" {
		c.SetDefaultImageTransformer(imageConfig.DefaultProvider)
	}

	c.imagePredefinedVariants = imageConfig.PredefinedVariants

	c.imageServiceConfigOverride = &imageConfig.ServiceConfig

	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Image config applied via builder",
		logger_domain.Int("provider_count", len(imageConfig.Providers)),
		logger_domain.Int("variant_count", len(imageConfig.PredefinedVariants)),
		logger_domain.String("default_provider", imageConfig.DefaultProvider))
}

// GetImagePredefinedVariants returns the map of predefined variant specs.
//
// Returns map[string]image_dto.TransformationSpec which maps variant names
// to their transformation specifications.
func (c *Container) GetImagePredefinedVariants() map[string]image_dto.TransformationSpec {
	return c.imagePredefinedVariants
}

// AddVideoTranscoder registers a named video transcoder for video processing.
//
// If the transcoder implements a shutdown interface (Close, Shutdown, or Stop),
// it will be automatically registered for graceful shutdown.
//
// The first transcoder registered becomes the default unless
// SetDefaultVideoTranscoder is called.
//
// Takes name (string) which identifies the transcoder for later retrieval.
// Takes transcoder (TranscoderPort) which handles video transcoding.
func (c *Container) AddVideoTranscoder(name string, transcoder video_domain.TranscoderPort) {
	if c.videoTranscoders == nil {
		c.videoTranscoders = make(map[string]video_domain.TranscoderPort)
	}
	c.videoTranscoders[name] = transcoder
	if c.defaultVideoTranscoder == "" {
		c.defaultVideoTranscoder = name
	}
	registerCloseableForShutdown(c.GetAppContext(), "VideoTranscoder-"+name, transcoder)
}

// SetDefaultVideoTranscoder sets the name of the transcoder to use as default.
//
// Takes name (string) which is the transcoder name to use as the default.
func (c *Container) SetDefaultVideoTranscoder(name string) {
	c.defaultVideoTranscoder = name
}

// createDefaultVideoService sets up the video service with registered
// transcoders.
//
// Sets videoService to nil if no transcoders are configured. Otherwise creates the
// service with all registered transcoders and adds a "default" alias.
func (c *Container) createDefaultVideoService() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	if len(c.videoTranscoders) == 0 {
		l.Internal("No video transcoders configured; video service disabled")
		c.videoService = nil
		return
	}

	l.Internal("Creating default VideoService with registered transcoders...",
		logger_domain.Int("transcoder_count", len(c.videoTranscoders)),
		logger_domain.String("default_transcoder", c.defaultVideoTranscoder))

	videoConfig := video_domain.DefaultServiceConfig()

	transcoders := make(map[string]video_domain.TranscoderPort, len(c.videoTranscoders)+1)
	maps.Copy(transcoders, c.videoTranscoders)
	if _, hasDefault := transcoders["default"]; !hasDefault && c.defaultVideoTranscoder != "" {
		transcoders["default"] = c.videoTranscoders[c.defaultVideoTranscoder]
	}

	c.videoService, c.videoErr = video_domain.NewService(transcoders, c.defaultVideoTranscoder, videoConfig)
}

// GetSEOService returns the SEO service, initialising a default one if none
// was provided. The SEO service is optional - if configuration is disabled or
// dependencies fail, it returns nil.
//
// Returns seo_domain.SEOService which provides SEO functionality, or nil if
// disabled.
// Returns error when the default SEO service could not be created.
func (c *Container) GetSEOService() (seo_domain.SEOService, error) {
	c.seoOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.seoServiceOverride != nil {
			l.Internal("Using provided SEOService override.")
			c.seoService = c.seoServiceOverride
			return
		}
		c.createDefaultSEOService()
	})
	return c.seoService, c.seoErr
}

// createDefaultSEOService sets up the default SEO service for the container.
// Sets c.seoService to nil if no SEO config was provided, or sets c.seoErr if
// setup fails unexpectedly.
func (c *Container) createDefaultSEOService() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating default SEOService...")

	if c.seoConfigOverride == nil {
		l.Internal("No SEO config provided, skipping SEO service creation")
		c.seoService = nil
		return
	}

	seoConfig := *c.seoConfigOverride

	if !seoConfig.Enabled {
		l.Internal("SEO is disabled in configuration, skipping SEO service creation")
		c.seoService = nil
		return
	}

	if seoConfig.Sitemap.Hostname == "" {
		l.Internal("SEO is enabled but sitemap hostname is not configured, skipping SEO service creation")
		c.seoService = nil
		return
	}

	registryService, err := c.GetRegistryService()
	if err != nil {
		c.seoErr = fmt.Errorf("failed to get registry service for SEO: %w", err)
		return
	}

	storageAdapter := seo_adapters.NewRegistryStorageAdapter(registryService)
	httpSourceAdapter := seo_adapters.NewHTTPSourceAdapter()

	var seoOpts []seo_domain.SEOServiceOption
	if factory, factoryErr := c.GetSandboxFactory(); factoryErr == nil {
		seoOpts = append(seoOpts, seo_domain.WithSEOSandboxFactory(factory))
	}

	seoService, err := seo_domain.NewSEOService(
		seoConfig,
		deref(c.serverConfig.I18nDefaultLocale, "en"),
		storageAdapter,
		httpSourceAdapter,
		seoOpts...,
	)
	if err != nil {
		c.seoErr = fmt.Errorf("failed to create SEO service: %w", err)
		return
	}

	c.seoService = seoService

	l.Internal("SEO service created successfully")
}

// SetSEOConfig stores the SEO configuration for use when creating the default
// SEO service. Must be called before GetSEOService.
//
// Takes seoConfig (config.SEOConfig) which specifies the SEO settings.
func (c *Container) SetSEOConfig(seoConfig config.SEOConfig) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	c.seoConfigOverride = &seoConfig
	l.Internal("SEO config set via programmatic API",
		logger_domain.String("hostname", seoConfig.Sitemap.Hostname),
		logger_domain.Bool("enabled", seoConfig.Enabled))
}

// SetAssetsConfig stores the assets configuration for use when creating the
// annotator service. Must be called before build/annotation starts.
//
// Takes assetsConfig (config.AssetsConfig) which specifies the asset profiles
// and responsive image settings.
func (c *Container) SetAssetsConfig(assetsConfig config.AssetsConfig) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	c.assetsConfigOverride = &assetsConfig
	l.Internal("Assets config set via programmatic API",
		logger_domain.Int("image_profiles", len(assetsConfig.Image.Profiles)),
		logger_domain.Int("video_profiles", len(assetsConfig.Video.Profiles)))
}

// GetAssetsConfig returns the assets configuration, or nil if none was
// provided.
//
// Returns *config.AssetsConfig which contains the asset profiles, or nil.
func (c *Container) GetAssetsConfig() *config.AssetsConfig {
	return c.assetsConfigOverride
}

// SetSEOService allows builders to provide a pre-configured SEO service to
// the container.
//
// If the service implements a shutdown interface (Close, Shutdown, or Stop),
// it will be automatically registered for graceful shutdown.
//
// Takes service (seo_domain.SEOService) which is the SEO service to use.
func (c *Container) SetSEOService(service seo_domain.SEOService) {
	c.seoServiceOverride = service
	c.seoService = service
	registerCloseableForShutdown(c.GetAppContext(), "SEOService", service)
}

// GetValidator returns the validator instance, or nil if no validator was
// provided via [WithValidator] in which case the caller must handle a nil
// validator by skipping validation.
//
// To supply a validator, use the validation_provider_playground WDK module
// or any type that implements [StructValidator].
//
// Returns StructValidator which is the configured validator, or nil.
func (c *Container) GetValidator() StructValidator {
	c.validatorOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.validatorOverride != nil {
			l.Internal("Using provided Validator override.")
			c.validator = c.validatorOverride
			return
		}
		l.Internal("No validator configured; validation will be skipped.")
	})
	return c.validator
}

// SetValidator sets a custom validator implementation.
// This completely replaces any previously configured validator.
//
// Takes v (StructValidator) which is the custom validator to use.
func (c *Container) SetValidator(v StructValidator) {
	c.validatorOverride = v
	c.validator = v
}

// GetHealthProbeService returns the singleton HealthProbe service, creating
// it if needed.
//
// Returns healthprobe_domain.Service which provides health probe operations.
// Returns error when service creation fails.
func (c *Container) GetHealthProbeService() (healthprobe_domain.Service, error) {
	c.healthProbeOnce.Do(func() {
		c.createDefaultHealthProbeService()
	})
	return c.healthProbeService, c.healthProbeErr
}

// createDefaultHealthProbeService creates and assigns the default health probe
// service to the container.
func (c *Container) createDefaultHealthProbeService() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating default HealthProbeService...")
	service, err := createHealthProbeService(c)
	if err != nil {
		c.healthProbeErr = err
		l.Error("Failed to create healthprobe service", logger_domain.Error(err))
		return
	}
	c.healthProbeService = service
}

// AddCustomHealthProbe registers a custom health probe with the health check
// system. Applications can add their own health checks that will be included
// in the /live and /ready endpoints.
//
// Takes probe (healthprobe_domain.Probe) which is the health probe to
// register.
//
// The probe is registered when the HealthProbeService starts.
func (c *Container) AddCustomHealthProbe(probe healthprobe_domain.Probe) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	c.customHealthProbes = append(c.customHealthProbes, probe)
	l.Internal("Custom health probe registered", logger_domain.String("probe_name", probe.Name()))
}

// SetMarkdownParser sets the markdown parser implementation used by the
// collection service for processing markdown content.
//
// Takes parser (markdown_domain.MarkdownParserPort) which provides the
// markdown parsing implementation.
func (c *Container) SetMarkdownParser(parser markdown_domain.MarkdownParserPort) {
	c.markdownParser = parser
}

// GetMarkdownParser returns the configured markdown parser, or nil if none
// has been set.
//
// Returns markdown_domain.MarkdownParserPort which is the active parser.
func (c *Container) GetMarkdownParser() markdown_domain.MarkdownParserPort {
	return c.markdownParser
}

// SetHighlighter sets the syntax highlighter for Markdown code blocks.
//
// Takes h (Highlighter) which provides syntax highlighting for code blocks.
func (c *Container) SetHighlighter(h highlight_domain.Highlighter) {
	c.highlighter = h
}

// GetHighlighter returns the configured syntax highlighter, or nil if none is
// set.
//
// Returns highlight_domain.Highlighter which provides syntax highlighting.
func (c *Container) GetHighlighter() highlight_domain.Highlighter {
	return c.highlighter
}

// AddFrontendModule enables a built-in frontend module to be loaded across
// all sites. Duplicate modules are silently ignored.
//
// Takes module (FrontendModule) which specifies the module to enable.
// Takes moduleConfig (any) which provides optional configuration; use a typed
// config struct (AnalyticsConfig, ModalsConfig, ToastsConfig) for built-in
// modules, or nil if not needed.
func (c *Container) AddFrontendModule(module daemon_frontend.FrontendModule, moduleConfig any) {
	for _, existing := range c.frontendModules {
		if existing.Module == module {
			return
		}
	}
	c.frontendModules = append(c.frontendModules, daemon_frontend.ModuleEntry{
		Module: module,
		Config: moduleConfig,
	})
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Frontend module enabled", logger_domain.String("module", module.String()))
}

// AddExternalComponents appends additional external component definitions to
// the container, which must be called before the lifecycle service is created
// (i.e. before the daemon builder runs) for the components to be discovered.
//
// Takes defs (...component_dto.ComponentDefinition) which are the component
// definitions to register.
func (c *Container) AddExternalComponents(defs ...component_dto.ComponentDefinition) {
	c.externalComponents = append(c.externalComponents, defs...)
}

// AddCustomFrontendModule registers a custom frontend JavaScript module.
// The module will be served at /_piko/dist/ppframework.{name}.min.js.
//
// Takes name (string) which specifies the module name.
// Takes content ([]byte) which provides the JavaScript source code.
// Takes moduleConfig (map[string]any) which provides optional settings for the
// module (can be nil).
func (c *Container) AddCustomFrontendModule(name string, content []byte, moduleConfig map[string]any) {
	if c.customFrontendModules == nil {
		c.customFrontendModules = make(map[string]*daemon_frontend.CustomFrontendModule)
	}
	c.customFrontendModules[name] = daemon_frontend.NewCustomFrontendModule(name, content, moduleConfig)
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Custom frontend module registered",
		logger_domain.String("name", name),
		logger_domain.Int("size", len(content)))
}

// GetFrontendModules returns the list of enabled built-in frontend modules
// with their configs.
//
// Returns []daemon_frontend.ModuleEntry which contains the module entries.
func (c *Container) GetFrontendModules() []daemon_frontend.ModuleEntry {
	return c.frontendModules
}

// GetCustomFrontendModules returns the map of custom frontend modules.
//
// Returns map[string]*daemon_frontend.CustomFrontendModule which maps module
// names to their custom frontend module definitions.
func (c *Container) GetCustomFrontendModules() map[string]*daemon_frontend.CustomFrontendModule {
	return c.customFrontendModules
}

// createSandbox creates a sandboxed filesystem for safe file operations.
// This extracts the common sandbox creation pattern used across multiple
// services.
//
// If a custom SandboxFactory has been injected via WithSandboxFactory, it is
// used instead of the default factory, enabling testing with mock sandboxes.
//
// Takes name (string) which identifies the sandbox instance.
// Takes baseDir (string) which specifies the root directory for the sandbox.
// Takes mode (safedisk.Mode) which defines the access permissions.
//
// Returns safedisk.Sandbox which provides controlled filesystem access.
// Returns error when the sandbox factory or sandbox creation fails.
func (c *Container) createSandbox(name, baseDir string, mode safedisk.Mode) (safedisk.Sandbox, error) {
	if c.sandboxFactory != nil {
		sandbox, err := c.sandboxFactory(name, baseDir, mode)
		if err != nil {
			return nil, fmt.Errorf(errCreateSourceSandbox, err)
		}
		return sandbox, nil
	}

	factory, err := c.GetSandboxFactory()
	if err != nil {
		return nil, fmt.Errorf(errCreateSandboxFactory, err)
	}
	sandbox, err := factory.Create(name, baseDir, mode)
	if err != nil {
		return nil, fmt.Errorf(errCreateSourceSandbox, err)
	}
	return sandbox, nil
}

// SetRegistryMetadataCacheConfig configures the Registry service's metadata
// cache.
//
// Takes cacheConfig (RegistryMetadataCacheConfig) which specifies
// the cache settings.
func (c *Container) SetRegistryMetadataCacheConfig(cacheConfig RegistryMetadataCacheConfig) {
	c.registryMetadataCacheConfig = &cacheConfig
}
