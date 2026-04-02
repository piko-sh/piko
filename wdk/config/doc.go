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

// Package config provides a reflection-based configuration loading
// framework that populates structs from multiple sources with a clear
// precedence order.
//
// Sources are applied in the following order (highest precedence last):
// programmatic defaults, struct tag defaults, config files (JSON/YAML),
// .env files, environment variables, command-line flags, and secret
// resolvers.
//
// The framework also provides a shared global flag coordinator for
// conflict-free flag parsing across independent config structs, a
// global resolver registry for reusable secret providers, pluggable
// struct validation, and detailed source tracking for debugging.
//
// # Usage
//
// Define a config struct with field tags, then call [Load]:
//
//	type AppConfig struct {
//	    DatabaseURL string `json:"databaseUrl" env:"APP_DB_URL" flag:"dbUrl" validate:"required"`
//	    APIKey      string `json:"apiKey" env:"APP_API_KEY"`
//	}
//
//	appConfig := &AppConfig{}
//	_, err := config.Load(ctx, appConfig, config.LoaderOptions{
//	    FilePaths:          []string{"config.json"},
//	    FlagPrefix:         "app",
//	    UseGlobalResolvers: true,
//	})
//
// Flags are coordinated globally via [RegisterFlags], allowing
// multiple independent config structs to register flags without
// conflicts. In most cases, flag registration happens automatically
// when calling [Load] with a FlagPrefix.
//
// # Secret resolvers
//
// Cloud and infrastructure secret resolvers are available in the
// config_resolver_* sub-packages. Built-in resolvers ([EnvResolver],
// [Base64Resolver], [FileResolver]) are included automatically when
// using [Load] or [WithDefaultResolvers].
//
// # Thread safety
//
// The global flag coordinator and resolver registry are safe for
// concurrent use. Individual [Loader] instances should not be
// shared between goroutines.
package config
