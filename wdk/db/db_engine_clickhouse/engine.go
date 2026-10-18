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

package db_engine_clickhouse

import (
	"piko.sh/piko/wdk/db/db_provider_seagull"
)

const (
	// dialectName is the Seagull dialect this engine serves.
	dialectName = "clickhouse"
)

// Option configures the ClickHouse engine.
type Option func(*settings)

// settings holds the options the engine is built with.
type settings struct {
	// joinUseNulls is the join_use_nulls server setting the analysis assumes; nil keeps the
	// server's default, off.
	joinUseNulls *bool

	// name is the dialect name the engine reports; empty keeps the default.
	name string

	// maxParseDepth limits expression nesting; zero keeps the default.
	maxParseDepth int

	// maxTokensPerStatement limits the size of one statement; zero keeps the default.
	maxTokensPerStatement int

	// maxTypeParseDepth limits type nesting; zero keeps the default.
	maxTypeParseDepth int
}

// ClickHouseEngine is Piko's ClickHouse engine. It reports its dialect name and supplies
// the toolchain that generates code, runs migrations and seeds, and checks database
// health.
type ClickHouseEngine struct {
	*db_provider_seagull.Engine
}

// WithDialectName overrides the dialect name the engine reports, for compatible databases
// that need their own name in logs and diagnostics.
//
// Takes name (string) which is the reported dialect name.
//
// Returns Option which applies the setting.
func WithDialectName(name string) Option {
	return func(settings *settings) {
		settings.name = name
	}
}

// WithMaxParseDepth limits how deeply expressions may nest before the parser reports an
// error.
//
// Takes depth (int) which is the maximum nesting depth; zero keeps the default.
//
// Returns Option which applies the setting.
func WithMaxParseDepth(depth int) Option {
	return func(settings *settings) {
		settings.maxParseDepth = depth
	}
}

// WithMaxTokensPerStatement limits the number of tokens in one statement.
//
// Takes limit (int) which is the maximum token count; zero keeps the default.
//
// Returns Option which applies the setting.
func WithMaxTokensPerStatement(limit int) Option {
	return func(settings *settings) {
		settings.maxTokensPerStatement = limit
	}
}

// WithMaxTypeParseDepth limits how deeply ClickHouse types such as Array(Map(String,
// Tuple(...))) may nest.
//
// Takes depth (int) which is the maximum type nesting depth; zero keeps the default.
//
// Returns Option which applies the setting.
func WithMaxTypeParseDepth(depth int) Option {
	return func(settings *settings) {
		settings.maxTypeParseDepth = depth
	}
}

// WithJoinUseNulls states the join_use_nulls setting of the servers queries run against.
//
// By default the analysis assumes ClickHouse's default, off, under which a LEFT, RIGHT or
// FULL JOIN fills an unmatched row with the column type's default value, so a joined
// column is a pointer only when its type is Nullable. Pass true when a server or settings
// profile turns join_use_nulls on, so the outer side's columns become pointers that can
// hold the NULLs it then returns.
//
// Takes enabled (bool) which is the server's join_use_nulls setting.
//
// Returns Option which applies the setting.
func WithJoinUseNulls(enabled bool) Option {
	return func(settings *settings) {
		settings.joinUseNulls = &enabled
	}
}

// NewClickHouseEngine returns the ClickHouse engine.
//
// Takes options (...Option) which tune the engine.
//
// Returns *ClickHouseEngine which serves the dialect.
func NewClickHouseEngine(options ...Option) *ClickHouseEngine {
	return newEngine(dialectName, options)
}

// newEngine builds the engine for a dialect.
//
// Takes dialect (string) which is the Seagull dialect name.
// Takes options ([]Option) which tune the engine.
//
// Returns *ClickHouseEngine which serves the dialect.
func newEngine(dialect string, options []Option) *ClickHouseEngine {
	settings := &settings{
		joinUseNulls:          nil,
		name:                  "",
		maxParseDepth:         0,
		maxTokensPerStatement: 0,
		maxTypeParseDepth:     0,
	}
	for _, option := range options {
		option(settings)
	}
	return &ClickHouseEngine{Engine: db_provider_seagull.NewBuiltinEngine(db_provider_seagull.DialectOptions{
		Dialect:               dialect,
		Returning:             nil,
		JoinUseNulls:          settings.joinUseNulls,
		Name:                  settings.name,
		MaxParseDepth:         settings.maxParseDepth,
		MaxTokensPerStatement: settings.maxTokensPerStatement,
		MaxTypeParseDepth:     settings.maxTypeParseDepth,
	})}
}
