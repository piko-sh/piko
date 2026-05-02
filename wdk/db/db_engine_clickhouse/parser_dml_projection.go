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

// tryProjectionExpressionTree builds a querier_dto.Expression tree for the computed
// projection item at the cursor, so the type resolver can infer its type. Without it a
// function call such as toInt64(count()) resolves to `any` (no expression is attached).
//
// It is a side-effect-free trial: it never registers parameters and restores the cursor,
// so the caller still drives the real placeholder-registering consume afterwards. The
// tree is returned only when the expression parser consumes exactly up to the projection
// boundary the consume stops at; a shorter or longer span (an operator the grammar does
// not model, a placeholder, an unrepresentable cast) yields nil so the column falls back
// to an untyped projection rather than carrying a partial tree.
//
// Returns querier_dto.Expression which is the parsed tree, or nil when none fits.
func (p *parser) tryProjectionExpressionTree() querier_dto.Expression {
	start := p.position
	boundary := p.projectionBoundaryPosition()
	expression, err := p.parseLambdaBodyExpression()
	end := p.position
	p.position = start
	if err != nil || expression == nil || end != boundary {
		return nil
	}
	return expression
}

// projectionBoundaryPosition returns the offset where the current projection item ends,
// scanning over parentheses and stopping at a top-level comma or projection keyword. It
// registers nothing and restores the cursor, so it validates an expression-tree trial
// without disturbing parameter numbering.
//
// Returns int which is the offset of the projection boundary.
func (p *parser) projectionBoundaryPosition() int {
	start := p.position
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if depth == 0 && tokenIsTopLevelStop(tok, projectionStopKeywords) {
			break
		}
		newDepth, halt := advanceParenDepth(tok, depth)
		if halt {
			break
		}
		depth = newDepth
		p.advance()
	}
	boundary := p.position
	p.position = start
	return boundary
}
