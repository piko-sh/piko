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
	"strings"

	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
)

// parenthesisScanIndex returns the statement's parenthesis scan index, building it on
// first use so statements without parameters never pay for it.
//
// Returns *engine_shared.ParenthesisScanIndex which answers enclosing-group and
// enclosing-LIKE lookups in constant time.
func (p *parser) parenthesisScanIndex() *engine_shared.ParenthesisScanIndex {
	if p.parenthesisIndex == nil {
		p.parenthesisIndex = engine_shared.NewParenthesisScanIndex(len(p.tokens), p.classifyScanToken)
	}
	return p.parenthesisIndex
}

// classifyScanToken reports the role the token at index plays in the parenthesis-aware
// parameter-context scans.
//
// Takes index (int) which is the token's position.
//
// Returns engine_shared.TokenClass which is the token's role; clause boundaries take
// precedence over pattern operators.
func (p *parser) classifyScanToken(index int) engine_shared.TokenClass {
	tok := p.tokens[index]
	if tok.kind == tokenLeftParen {
		return engine_shared.TokenClassLeftParen
	}
	if tok.kind == tokenRightParen {
		return engine_shared.TokenClassRightParen
	}
	if tok.kind != tokenIdentifier {
		return engine_shared.TokenClassOther
	}
	keyword := strings.ToUpper(tok.value)
	if isLikeBoundaryKeyword(keyword) {
		return engine_shared.TokenClassBoundary
	}
	if isLikePatternKeyword(keyword) {
		return engine_shared.TokenClassPattern
	}
	return engine_shared.TokenClassOther
}
