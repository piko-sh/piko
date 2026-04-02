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

package config_provider_piko

import (
	"piko.sh/piko/internal/config"
)

// ServerConfig is the root configuration struct for the Piko framework. It
// contains all framework settings including paths, cache, compiler, network,
// and observability.
type ServerConfig = config.ServerConfig

// Provider manages loading the framework's ServerConfig and WebsiteConfig.
// It handles file discovery, precedence, and validation.
type Provider = config.Provider

// WebsiteConfig holds website settings such as translations, fonts, and
// favicons.
type WebsiteConfig = config.WebsiteConfig

// PathsConfig defines file system paths used by the framework.
type PathsConfig = config.PathsConfig

// BuildModeConfig configures build-time optimisations and behaviours.
type BuildModeConfig = config.BuildModeConfig

// NetworkConfig configures network settings such as host, port, and TLS.
type NetworkConfig = config.NetworkConfig

// OtlpConfig is an alias for config.OtlpConfig which holds settings for
// OpenTelemetry Protocol export.
type OtlpConfig = config.OtlpConfig

// OtlpTLSConfig is an alias for config.OtlpTLSConfig which configures TLS
// settings for OTLP connections.
type OtlpTLSConfig = config.OtlpTLSConfig

// I18nConfig provides settings for language and locale support.
type I18nConfig = config.I18nConfig

// FaviconDefinition defines a favicon resource.
type FaviconDefinition = config.FaviconDefinition

// FontDefinition defines a web font resource.
type FontDefinition = config.FontDefinition

// SEOConfig is an alias for config.SEOConfig containing SEO settings.
type SEOConfig = config.SEOConfig

// SitemapConfig sets options for XML sitemap generation.
type SitemapConfig = config.SitemapConfig

// SitemapEntryDefaults defines the default values for sitemap entries.
type SitemapEntryDefaults = config.SitemapEntryDefaults

// SitemapChunkConfig sets how sitemaps are split into smaller parts.
type SitemapChunkConfig = config.SitemapChunkConfig

// RobotsConfig sets the options for robots.txt file creation.
type RobotsConfig = config.RobotsConfig

// RobotsRuleGroup defines a group of rules for specific user agents.
type RobotsRuleGroup = config.RobotsRuleGroup

// AssetsConfig sets up how assets are processed and served.
type AssetsConfig = config.AssetsConfig

// ImageAssetsConfig sets options for image asset processing.
type ImageAssetsConfig = config.ImageAssetsConfig

// AssetTransformationStep defines a single step in an asset transformation.
type AssetTransformationStep = config.AssetTransformationStep

// NewConfigProvider creates a new framework config provider.
//
// This is typically only used internally by the framework's bootstrap
// process. Most applications should create their own config structs instead
// of extending ServerConfig.
//
// Returns *Provider which provides access to framework configuration.
func NewConfigProvider() *Provider {
	return config.NewConfigProvider()
}
