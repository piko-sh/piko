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

package config

import (
	"context"

	"piko.sh/piko/internal/config/config_domain"
	"piko.sh/piko/wdk/safedisk"
)

// Loader handles loading configuration from multiple sources into a struct.
// It runs a series of passes to fill in values step by step.
type Loader = config_domain.Loader

// LoaderOptions sets how a Loader behaves when loading configuration.
type LoaderOptions = config_domain.LoaderOptions

// LoaderOption is a functional option for setting up a Loader.
type LoaderOption = config_domain.LoaderOption

// LoadContext holds the results of a loading operation, including debugging
// information.
type LoadContext = config_domain.LoadContext

// Pass represents a single stage in the configuration loading process.
type Pass = config_domain.Pass

// Resolver is the interface for custom secret resolvers. Resolvers process
// placeholder strings (e.g., "aws-ssm:my-secret") and replace them with actual
// values.
type Resolver = config_domain.Resolver

// BatchResolver is an optimised resolver that can resolve multiple secrets in
// one operation.
type BatchResolver = config_domain.BatchResolver

const (
	// PassProgrammatic applies default values set in code.
	PassProgrammatic = config_domain.PassProgrammatic

	// PassDefaults applies default values from struct tags.
	PassDefaults = config_domain.PassDefaults

	// PassFiles loads and merges configuration from files.
	PassFiles = config_domain.PassFiles

	// PassDotEnv applies settings from .env files.
	PassDotEnv = config_domain.PassDotEnv

	// PassEnv applies settings from environment variables.
	PassEnv = config_domain.PassEnv

	// PassFlags applies settings from command-line flags.
	PassFlags = config_domain.PassFlags

	// PassResolvers is the resolver pass that processes placeholder strings.
	PassResolvers = config_domain.PassResolvers

	// PassValidation validates the final populated struct.
	PassValidation = config_domain.PassValidation
)

// EnvResolver resolves placeholders like "env:MY_VAR" to environment variable
// values.
type EnvResolver = config_domain.EnvResolver

// Base64Resolver resolves placeholders like "base64:SGVsbG8=" by decoding
// base64 strings.
type Base64Resolver = config_domain.Base64Resolver

// FileResolver resolves placeholders like "file:/path/to/secret.txt" by reading
// file contents.
type FileResolver = config_domain.FileResolver

// RegisterFlags registers a config struct's flags with the global flag
// coordinator.
//
// In most cases, manual registration is unnecessary. Flag registration happens
// automatically when you call Load with a FlagPrefix. Manual registration is
// only needed in advanced scenarios where you need to register flags before
// calling Load, such as when implementing custom flag parsing logic.
//
// The prefix is prepended to all flag names. For example, if prefix is "app"
// and a field has `flag:"dbUrl"`, the resulting flag will be "--app.dbUrl".
// Pass an empty string for prefix to register flags without a prefix.
//
// Takes ptr (any) which is a pointer to the config struct to register.
// Takes prefix (string) which is prepended to all flag names.
//
// Returns error when registration fails.
func RegisterFlags(ptr any, prefix string) error {
	coordinator := config_domain.GetGlobalFlagCoordinator()
	tempLoader := config_domain.NewLoader(config_domain.LoaderOptions{})
	return coordinator.RegisterStruct(ptr, prefix, tempLoader)
}

// ParseFlags parses os.Args using the global flag coordinator.
// This is called automatically during Load() if the PassFlags pass is enabled,
// but can be called manually if you need to parse flags before loading.
//
// Subsequent calls are idempotent (the flags are only parsed once).
//
// Returns error when flag parsing fails.
func ParseFlags() error {
	coordinator := config_domain.GetGlobalFlagCoordinator()
	return coordinator.Parse()
}

// RegisterResolver registers a resolver in the global resolver registry.
// Loaders with UseGlobalResolvers=true will automatically inherit all
// global resolvers.
//
// Takes resolver (Resolver) which is the resolver to register globally.
//
// Returns error when registration fails.
//
// Example:
//
//	awsResolver := config_resolver_aws.NewResolver()
//	config.RegisterResolver(awsResolver)
func RegisterResolver(resolver Resolver) error {
	registry := config_domain.GetGlobalResolverRegistry()
	return registry.Register(resolver)
}

// NewLoader creates a new loader with the given options.
// For most use cases, use the Load convenience function instead.
//
// Takes opts (LoaderOptions) which specifies the loader configuration.
// Takes options (...LoaderOption) which provides optional behaviour controls.
//
// Returns *Loader which is the configured loader ready for use.
//
// Example:
//
//	loader := config.NewLoader(config.LoaderOptions{
//	    FilePaths:          []string{"config.json"},
//	    FlagPrefix:         "app",
//	    UseGlobalResolvers: true,
//	}, config.WithDefaultResolvers())
//
//	ctx := context.Background()
//	loadCtx, err := loader.Load(ctx, &myConfig)
func NewLoader(opts LoaderOptions, options ...LoaderOption) *Loader {
	return config_domain.NewLoader(opts, options...)
}

// WithDefaultResolvers is a functional option that registers the built-in
// dependency-free resolvers (env:, base64:, file:).
//
// Returns LoaderOption which configures the loader with default resolvers.
//
// Example:
//
//	loader := config.NewLoader(opts, config.WithDefaultResolvers())
func WithDefaultResolvers() LoaderOption {
	return config_domain.WithDefaultResolvers()
}

// Load creates a new Loader and loads configuration into the provided struct
// pointer. It automatically includes default resolvers.
//
// The ptr parameter must be a pointer to a struct with appropriate tags:
//   - `json` / `yaml`: field names in config files
//   - `default`: default value for the field
//   - `env`: environment variable name
//   - `flag`: command-line flag name (will be prefixed with opts.FlagPrefix)
//   - `validate`: validation rules (requires a StructValidator to be configured)
//
// Example:
//
//	type Config struct {
//	    Port int `json:"port" env:"APP_PORT" flag:"port" default:"8080" validate:"required"`
//	}
//
//	appConfig := &Config{}
//	_, err := config.Load(context.Background(), appConfig, config.LoaderOptions{
//	    FilePaths:  []string{"config.json"},
//	    FlagPrefix: "app",
//	})
//
// Takes ptr (any) which must be a pointer to a struct to populate.
// Takes opts (LoaderOptions) which specifies file paths, prefixes, and other
// loading options.
//
// Returns *LoadContext which contains metadata about the loaded configuration.
// Returns error when the pointer is invalid or configuration loading fails.
func Load(ctx context.Context, ptr any, opts LoaderOptions) (*LoadContext, error) {
	return config_domain.Load(ctx, ptr, opts)
}

// NewFileResolver creates a new file resolver for reading configuration files.
//
// When sandbox is nil, the resolver creates per-file sandboxes for each file
// read. For improved security, provide a sandbox to restrict file access to a
// specific directory.
//
// Takes sandbox (safedisk.Sandbox) which restricts file access to a directory.
//
// Returns *FileResolver which resolves and reads configuration files.
func NewFileResolver(sandbox safedisk.Sandbox) *FileResolver {
	return config_domain.NewFileResolver(sandbox)
}
