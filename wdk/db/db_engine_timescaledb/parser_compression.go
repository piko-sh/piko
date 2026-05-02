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

package db_engine_timescaledb

import (
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_postgres"
)

// parseAlterTableCompression parses an ALTER TABLE or ALTER MATERIALIZED VIEW statement
// that sets timescaledb compression reloptions.
//
// The recognised forms are:
//
//	ALTER TABLE [IF EXISTS] [schema.]name SET (timescaledb.compress = ..., ...)
//	ALTER MATERIALIZED VIEW [IF EXISTS] [schema.]name SET (timescaledb.compress = ..., ...)
//
// Captures recognised timescaledb.* keys into mutation.EngineSpecific with
// TIMESCALE_COMPRESSION_* prefixes. Other reloptions are preserved verbatim in
// TIMESCALE_RELOPTION_<key> entries so nothing is lost. The optional `IF EXISTS`
// qualifier is consumed transparently so callers can use the standard postgres dialect.
//
// Takes p (db_engine_postgres.ParserContext) which is the parser positioned at the ALTER
// keyword.
//
// Returns *querier_dto.CatalogueMutation which describes the compression metadata; the
// catalogue schema is not otherwise modified.
// Returns error when the header or reloption body fails to parse.
func parseAlterTableCompression(p db_engine_postgres.ParserContext) (*querier_dto.CatalogueMutation, error) {
	isMaterialized, schema, name, headerErr := parseAlterCompressionHeader(p)
	if headerErr != nil {
		return nil, headerErr
	}

	reloptions, reloptionErr := parseAlterCompressionReloptionBody(p, name)
	if reloptionErr != nil {
		return nil, reloptionErr
	}

	engineSpecific := buildCompressionEngineSpecific(isMaterialized, reloptions)

	return &querier_dto.CatalogueMutation{
		Kind:           querier_dto.MutationAlterTableAlterColumn,
		SchemaName:     schema,
		TableName:      name,
		EngineSpecific: engineSpecific,
	}, nil
}

// parseAlterCompressionHeader consumes the ALTER TABLE / ALTER MATERIALIZED VIEW header,
// the optional IF EXISTS clause, and the qualified target name.
//
// Takes p (db_engine_postgres.ParserContext) which is the parser positioned at the ALTER
// keyword.
//
// Returns isMaterialized (bool) which is true for an ALTER MATERIALIZED VIEW header.
// Returns schema (string) which is the target schema name.
// Returns name (string) which is the target table or view name.
// Returns err (error) when the qualified name fails to parse.
func parseAlterCompressionHeader(p db_engine_postgres.ParserContext) (isMaterialized bool, schema string, name string, err error) {
	p.MustKeyword("ALTER")
	if p.MatchKeyword("MATERIALIZED") {
		p.MustKeyword("VIEW")
		isMaterialized = true
	} else {
		p.MustKeyword("TABLE")
	}

	p.MatchIfExists()

	parsedSchema, parsedName, parseErr := p.ParseQualifiedName()
	if parseErr != nil {
		return false, "", "", fmt.Errorf("compression alter: %w", parseErr)
	}
	return isMaterialized, parsedSchema, parsedName, nil
}

// parseAlterCompressionReloptionBody walks the rest of the statement looking for the SET
// (...) reloption body.
//
// Takes p (db_engine_postgres.ParserContext) which is the parser positioned after the
// target name.
// Takes name (string) which is the target name used in error messages.
//
// Returns map[string]string which is the parsed reloption map.
// Returns error when the SET (timescaledb...) clause is missing or the body fails to
// parse.
func parseAlterCompressionReloptionBody(p db_engine_postgres.ParserContext, name string) (map[string]string, error) {
	foundCompressionSet := false
	for !p.AtEnd() {
		if !p.MatchKeyword("SET") {
			p.Advance()
			continue
		}

		if p.CurrentToken().Kind() != db_engine_postgres.TokenLeftParen {
			continue
		}

		if peekIsKeyword(p, "timescaledb") {
			foundCompressionSet = true
			break
		}

		if skipErr := skipParenGroup(p); skipErr != nil {
			return nil, fmt.Errorf("compression alter %q: %w", name, skipErr)
		}
	}
	if !foundCompressionSet {
		return nil, fmt.Errorf("compression alter %q: missing SET (timescaledb...) clause", name)
	}

	reloptions, reloptionErr := p.ParseReloptionList()
	if reloptionErr != nil {
		return nil, fmt.Errorf("compression alter %q: reloption body: %w", name, reloptionErr)
	}
	return reloptions, nil
}

// buildCompressionEngineSpecific projects the raw reloption map into the EngineSpecific
// map shape expected by downstream consumers.
//
// Recognised timescaledb.* keys map to TIMESCALE_* names; unrecognised keys are preserved
// as TIMESCALE_RELOPTION_<UPPER_KEY> entries so nothing is lost.
//
// Takes isMaterialized (bool) which is true for an ALTER MATERIALIZED VIEW target.
// Takes reloptions (map[string]string) which is the raw reloption key/value map.
//
// Returns map[string]string which is the projected EngineSpecific map.
func buildCompressionEngineSpecific(isMaterialized bool, reloptions map[string]string) map[string]string {
	engineSpecific := map[string]string{}
	if isMaterialized {
		engineSpecific["TIMESCALE_ALTER_MATERIALIZED"] = literalTrue
	}
	for key, value := range reloptions {
		if specific := timescaleReloptionToEngineSpecific(key); specific != "" {
			engineSpecific[specific] = value
		} else {
			engineSpecific["TIMESCALE_RELOPTION_"+strings.ToUpper(key)] = value
		}
	}
	return engineSpecific
}
