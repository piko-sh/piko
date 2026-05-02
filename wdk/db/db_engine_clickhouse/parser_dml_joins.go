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
	"piko.sh/piko/internal/querier/querier_dto"
)

// matchJoinKeyword recognises a JOIN-introducing keyword sequence and returns the
// corresponding JoinKind.
//
// ClickHouse extensions are mapped to their specific JoinKind enum values so the
// downstream analyser can distinguish the row-semantics implications. ASOF, SEMI, and
// ANTI are the classic ClickHouse join shapes, and the LEFT and RIGHT-biased variants
// preserve outer-row semantics. The GLOBAL prefix is a distributed-join broadcast hint,
// recorded as JoinGlobal when no other strictness modifier precedes it. The ANY and ALL
// strictness keywords are a ClickHouse multiplicity hint, recorded as JoinAny or JoinAll
// respectively when no body-specific keyword follows. When ANY or ALL is paired with a
// body keyword (INNER, LEFT, and similar) the body keyword wins; the strictness modifier
// is consumed transparently and the body kind is returned.
//
// Returns querier_dto.JoinKind which is the matched join kind.
// Returns bool which is true when a join keyword sequence was consumed.
func (p *parser) matchJoinKeyword() (kind querier_dto.JoinKind, matched bool) {
	saved := p.position
	hasGlobal := p.matchKeyword(kwGlobal)
	hasStrictness, strictnessKind := p.matchStrictnessPrefix()

	if (hasStrictness || hasGlobal) && p.isKeyword(kwJoin) && !p.isAnyKeyword(kwInner, kwLeft, kwRight, kwFull, kwCross, kwAsof, kwSemi, kwAnti) {
		p.advance()
		if hasStrictness {
			return strictnessKind, true
		}
		return querier_dto.JoinGlobal, true
	}
	resolved, isResolved := p.matchJoinBody(saved)
	if isResolved {
		return resolved, true
	}
	if hasStrictness {
		if p.matchKeyword(kwJoin) {
			return strictnessKind, true
		}
	}
	if hasGlobal {
		if p.matchKeyword(kwJoin) {
			return querier_dto.JoinGlobal, true
		}
	}
	p.position = saved
	return 0, false
}

// matchStrictnessPrefix consumes an optional ClickHouse strictness keyword (ANY or ALL)
// that precedes the join body.
//
// Returns bool which is true when a strictness keyword was consumed, leaving the cursor
// unchanged otherwise.
// Returns querier_dto.JoinKind which is the matched strictness join kind.
func (p *parser) matchStrictnessPrefix() (matched bool, kind querier_dto.JoinKind) {
	switch {
	case p.matchKeyword(kwAny):
		return true, querier_dto.JoinAny
	case p.matchKeyword(kwAll):
		return true, querier_dto.JoinAll
	}
	return false, 0
}

// matchJoinBody attempts to consume the body of a join.
//
// The recognised bodies are INNER, LEFT, RIGHT, FULL, CROSS, ASOF, SEMI, ANTI, and plain
// JOIN. When no body keyword is consumed (and only strictness or GLOBAL prefixes have
// run), the caller falls back to its own handling.
//
// Takes saved (int) which is the cursor position to rewind to on a non-match.
//
// Returns querier_dto.JoinKind which is the resolved join kind.
// Returns bool which is true when a body keyword sequence completed a JOIN match.
func (p *parser) matchJoinBody(saved int) (kind querier_dto.JoinKind, resolved bool) {
	switch {
	case p.matchKeyword(kwInner):
		p.matchKeyword(kwJoin)
		return querier_dto.JoinInner, true
	case p.matchKeyword(kwLeft):
		return p.matchSidedJoin(saved, querier_dto.JoinLeftSemi, querier_dto.JoinLeftAnti, querier_dto.JoinLeft)
	case p.matchKeyword(kwRight):
		return p.matchSidedJoin(saved, querier_dto.JoinRightSemi, querier_dto.JoinRightAnti, querier_dto.JoinRight)
	case p.matchKeyword(kwFull):
		return p.matchOptionalOuterJoin(saved, querier_dto.JoinFull)
	case p.matchKeyword(kwCross):
		return p.matchPlainJoinKeyword(saved, querier_dto.JoinCross)
	case p.matchKeyword(kwAsof):
		p.matchKeyword(kwInner)
		return p.matchPlainJoinKeyword(saved, querier_dto.JoinAsof)
	case p.matchKeyword(kwSemi):
		return p.matchPlainJoinKeyword(saved, querier_dto.JoinSemi)
	case p.matchKeyword(kwAnti):
		return p.matchPlainJoinKeyword(saved, querier_dto.JoinAnti)
	case p.matchKeyword(kwJoin):
		return querier_dto.JoinInner, true
	}
	return 0, false
}

// matchSidedJoin handles the LEFT and RIGHT join variants, which share the same optional
// OUTER, optional SEMI or ANTI, then JOIN shape with side-specific JoinKind mappings.
//
// Takes saved (int) which is the cursor position to rewind to on a non-match.
// Takes semiKind (querier_dto.JoinKind) which is the JoinKind for the SEMI form.
// Takes antiKind (querier_dto.JoinKind) which is the JoinKind for the ANTI form.
// Takes plainKind (querier_dto.JoinKind) which is the JoinKind for the plain JOIN form.
//
// Returns querier_dto.JoinKind which is the matched join kind, or zero on a non-match.
// Returns bool which is true when a JOIN token followed, rewinding the cursor to saved
// otherwise.
func (p *parser) matchSidedJoin(saved int, semiKind querier_dto.JoinKind, antiKind querier_dto.JoinKind, plainKind querier_dto.JoinKind) (querier_dto.JoinKind, bool) {
	p.matchKeyword("OUTER")
	if p.matchKeyword(kwSemi) {
		return p.matchPlainJoinKeyword(saved, semiKind)
	}
	if p.matchKeyword(kwAnti) {
		return p.matchPlainJoinKeyword(saved, antiKind)
	}
	if p.matchKeyword(kwJoin) {
		return plainKind, true
	}
	p.position = saved
	return 0, false
}

// matchOptionalOuterJoin handles the FULL [OUTER] JOIN form.
//
// OUTER is consumed when present; JOIN is required, otherwise the cursor rewinds to
// saved.
//
// Takes saved (int) which is the cursor position to rewind to on a non-match.
// Takes joinKind (querier_dto.JoinKind) which is the JoinKind to return on a match.
//
// Returns querier_dto.JoinKind which is joinKind on a match, or zero on a non-match.
// Returns bool which is true when a JOIN token followed.
func (p *parser) matchOptionalOuterJoin(saved int, joinKind querier_dto.JoinKind) (querier_dto.JoinKind, bool) {
	p.matchKeyword("OUTER")
	return p.matchPlainJoinKeyword(saved, joinKind)
}

// matchPlainJoinKeyword finalises a join match that has already consumed its preceding
// modifier such as SEMI, ANTI, or OUTER.
//
// JOIN must follow; otherwise the cursor is rewound to saved.
//
// Takes saved (int) which is the cursor position to rewind to on a non-match.
// Takes joinKind (querier_dto.JoinKind) which is the JoinKind to return on a match.
//
// Returns querier_dto.JoinKind which is joinKind on a match, or zero on a non-match.
// Returns bool which is true when a JOIN token followed.
func (p *parser) matchPlainJoinKeyword(saved int, joinKind querier_dto.JoinKind) (querier_dto.JoinKind, bool) {
	if p.matchKeyword(kwJoin) {
		return joinKind, true
	}
	p.position = saved
	return 0, false
}
