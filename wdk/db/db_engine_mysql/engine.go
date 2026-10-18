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

package db_engine_mysql

import (
	"piko.sh/piko/wdk/db/db_provider_seagull"
)

const (
	// dialectName is the Seagull dialect this engine serves.
	dialectName = "mysql"
)

// Option configures the MySQL engine.
type Option func(*settings)

// settings holds the options the engine is built with.
type settings struct {
	// returning overrides RETURNING support; nil keeps the dialect default.
	returning *bool

	// name is the dialect name the engine reports; empty keeps the default.
	name string

	// maxParseDepth limits expression nesting; zero keeps the default.
	maxParseDepth int

	// maxTokensPerStatement limits the size of one statement; zero keeps the default.
	maxTokensPerStatement int
}

// MySQLEngine is Piko's MySQL engine. It reports its dialect name and supplies the
// toolchain that generates code, runs migrations and seeds, and checks database health.
type MySQLEngine struct {
	*db_provider_seagull.Engine
}

// WithReturningSupport sets whether the database supports RETURNING on INSERT, UPDATE and
// DELETE.
//
// Takes supported (bool) which enables or disables RETURNING.
//
// Returns Option which applies the setting.
func WithReturningSupport(supported bool) Option {
	return func(settings *settings) {
		settings.returning = &supported
	}
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

// NewMySQLEngine returns the MySQL engine.
//
// Takes options (...Option) which tune the engine.
//
// Returns *MySQLEngine which serves the dialect.
func NewMySQLEngine(options ...Option) *MySQLEngine {
	return newEngine(dialectName, options)
}

// NewEngineForDialect returns the engine for a MySQL-family dialect such as mariadb. The
// MariaDB engine module uses it; applications call their constructors instead.
//
// Takes dialect (string) which is the Seagull dialect name.
// Takes options (...Option) which tune the engine.
//
// Returns *MySQLEngine which serves the dialect.
func NewEngineForDialect(dialect string, options ...Option) *MySQLEngine {
	return newEngine(dialect, options)
}

// newEngine builds the engine for a dialect.
//
// Takes dialect (string) which is the Seagull dialect name.
// Takes options ([]Option) which tune the engine.
//
// Returns *MySQLEngine which serves the dialect.
func newEngine(dialect string, options []Option) *MySQLEngine {
	settings := &settings{
		returning:             nil,
		name:                  "",
		maxParseDepth:         0,
		maxTokensPerStatement: 0,
	}
	for _, option := range options {
		option(settings)
	}
	return &MySQLEngine{Engine: db_provider_seagull.NewBuiltinEngine(db_provider_seagull.DialectOptions{
		Dialect:               dialect,
		Returning:             settings.returning,
		JoinUseNulls:          nil,
		Name:                  settings.name,
		MaxParseDepth:         settings.maxParseDepth,
		MaxTokensPerStatement: settings.maxTokensPerStatement,
		MaxTypeParseDepth:     0,
	})}
}
