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

// Package config defines the application's configuration structures and
// serves as the main entry point for loading configuration.
//
// It aggregates all configuration types used throughout the Piko framework.
// Structs are populated by the config_domain loader, which reads values
// from struct tags, files, environment variables, and command-line flags
// in a prioritised order.
//
// # Configuration sources
//
// Configuration values are loaded in the following order of precedence
// (highest to lowest):
//
//  1. Programmatic overrides (WithXxx functional options)
//  2. Command-line flags (--flag=value)
//  3. Environment variables (PIKO_*)
//  4. Local config file (piko.local.yaml)
//  5. Environment-specific config (piko-{env}.yaml)
//  6. Base config file (piko.yaml)
//  7. Struct tag defaults
//  8. Programmatic defaults
//
// # Usage
//
//	provider := config.NewConfigProvider()
//	ctx, err := provider.LoadConfig(&config.ServerConfig{}, nil)
//	if err != nil {
//	    return err
//	}
//	// Access loaded configuration
//	port := provider.ServerConfig.Network.Port
//
// # Integration
//
// The config package integrates with [config_domain] for the loading mechanics
// and [config_adapters] for secret resolution from external providers such as
// AWS Secrets Manager, HashiCorp Vault, and Kubernetes secrets.
package config
