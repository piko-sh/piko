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
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// parseMaterializedViewSecurityClauses recognises the optional DEFINER = <user> and SQL
// SECURITY {DEFINER | INVOKER | NONE} clauses placed before the AS SELECT body.
//
// A recognised clause with a malformed body surfaces a precise diagnostic instead of
// being silently skipped.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the parsed security
// metadata.
//
// Returns error when a recognised clause has a malformed body.
func (p *parser) parseMaterializedViewSecurityClauses(mutation *querier_dto.CatalogueMutation) error {
	for {
		switch {
		case p.matchKeyword(keywordDefiner):
			if clauseErr := p.captureMVDefinerClause(mutation); clauseErr != nil {
				return clauseErr
			}
		case p.matchKeyword(keywordSQL):
			consumed, clauseErr := p.captureMVSQLSecurityClause(mutation)
			if clauseErr != nil {
				return clauseErr
			}
			if !consumed {
				return nil
			}
		default:
			return nil
		}
	}
}

// captureMVDefinerClause reads the `[=] user` body after the DEFINER keyword.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the parsed definer name.
//
// Returns error when the user identifier is malformed.
func (p *parser) captureMVDefinerClause(mutation *querier_dto.CatalogueMutation) error {
	if p.current().kind == tokenOperator && p.current().value == "=" {
		p.advance()
	}
	name, parseErr := p.parseIdentifierOrKeyword()
	if parseErr != nil {
		return parseErr
	}
	mutation.EngineSpecific[engineKeyMVDefiner] = name
	return nil
}

// captureMVSQLSecurityClause reads the `SECURITY {DEFINER | INVOKER | NONE}` body after
// the SQL keyword.
//
// The bool reports whether a valid SQL SECURITY form was consumed so the outer loop
// continues; false means SQL was not followed by SECURITY, ending the modifier loop.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the parsed security
// mode.
//
// Returns bool which is true when a valid SQL SECURITY form was consumed.
// Returns error when SECURITY is present but its body is malformed.
func (p *parser) captureMVSQLSecurityClause(mutation *querier_dto.CatalogueMutation) (bool, error) {
	if !p.matchKeyword("SECURITY") {
		return false, nil
	}
	security, parseErr := p.parseIdentifierOrKeyword()
	if parseErr != nil {
		return true, parseErr
	}
	mutation.EngineSpecific[engineKeyMVSQLSecurity] = strings.ToUpper(security)
	return true, nil
}
