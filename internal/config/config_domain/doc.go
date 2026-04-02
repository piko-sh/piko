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

// Package config_domain orchestrates reflection-based configuration loading
// from multiple sources.
//
// It merges defaults, files (JSON, YAML), environment variables,
// command-line flags, and pluggable resolvers for secrets, with struct
// tag-based validation and per-field source tracking.
//
// # Usage
//
// Basic loading with default settings:
//
//	type Config struct {
//	    Port    int    `default:"8080" env:"PORT" validate:"min=1,max=65535"`
//	    APIKey  string `env:"API_KEY" validate:"required"`
//	}
//
//	config := &Config{}
//	ctx, err := config_domain.Load(context.Background(), config, config_domain.LoaderOptions{
//	    FilePaths: []string{"config.yaml"},
//	})
//
// For secrets that should be resolved lazily:
//
//	type Config struct {
//	    DBPassword config_domain.Secret[string] `env:"DB_PASSWORD"`
//	}
//
//	// Later, when the secret is needed:
//	handle, err := config.DBPassword.Acquire(ctx)
//	if err != nil { return err }
//	defer handle.Release()
//	password := handle.Value()
//
// # Pass order
//
// Configuration is loaded in passes, with later passes overriding earlier:
//
//  1. Programmatic defaults (from LoaderOptions.ProgrammaticDefaults)
//  2. Struct tag defaults (`default:"value"`)
//  3. File loading (JSON/YAML, merged in order)
//  4. DotEnv files (.env)
//  5. Environment variables
//  6. Command-line flags
//  7. Resolver placeholders (e.g., "env:SECRET_KEY")
//  8. Validation (via pluggable StructValidator interface)
//
// # Thread safety
//
// [Loader] instances are not safe for concurrent use during Load, but multiple
// loaders can operate independently. [SecretManager], [FlagCoordinator], and
// [ResolverRegistry] are safe for concurrent use.
package config_domain
