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
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// engineKeyPartitionOp is the EngineSpecific key recording the partition operation name.
	engineKeyPartitionOp = "PARTITION_OP"

	// engineKeyPartitionTarget is the EngineSpecific key recording PART versus PARTITION.
	engineKeyPartitionTarget = "PARTITION_TARGET"

	// engineKeyPartitionExpr is the EngineSpecific key recording the partition expression.
	engineKeyPartitionExpr = "PARTITION_EXPR"

	// engineKeyPartitionDest is the EngineSpecific key recording the move destination spec.
	engineKeyPartitionDest = "PARTITION_DEST"

	// engineKeyPartitionFrom is the EngineSpecific key recording the source table copied
	// from.
	engineKeyPartitionFrom = "PARTITION_FROM_TABLE"

	// engineKeyPartitionBackup is the EngineSpecific key recording the WITH NAME backup.
	engineKeyPartitionBackup = "PARTITION_BACKUP_NAME"
)

// parseAlterPartitionOperation handles the partition family of ALTER TABLE actions.
//
// The recognised forms are FREEZE / UNFREEZE PARTITION expr [WITH NAME 'name'], ATTACH
// PARTITION expr [FROM source] and ATTACH PARTITION ALL FROM source, DETACH PARTITION /
// PART expr, DROP PARTITION / PART expr, MOVE PARTITION expr TO {TABLE name | DISK 'name'
// | VOLUME 'name'}, REPLACE PARTITION expr FROM source, and FETCH PARTITION expr FROM
// 'path'. The helper captures PARTITION_OP, PARTITION_TARGET (PART versus PARTITION),
// PARTITION_EXPR, and the optional PARTITION_DEST, PARTITION_BACKUP_NAME and
// PARTITION_FROM_TABLE metadata.
//
// Takes database (string) which is the target database name.
// Takes table (string) which is the target table name.
// Takes operation (string) which is the partition operation keyword.
// Takes presetIsPart (bool) which, when true, signals that the caller has already
// consumed the PART or PARTITION keyword upstream (the `DROP DETACHED PART` / `DROP
// DETACHED PARTITION` form goes through parseAlterDropDetached, which consumes the
// keyword before dispatching) so the helper does not re-consume it; when false the helper
// expects PART or PARTITION at the cursor and consumes whichever appears, defaulting to
// PARTITION when neither is present.
//
// Returns *querier_dto.CatalogueMutation which describes the captured partition op.
// Returns error when a malformed tail clause is encountered.
func (p *parser) parseAlterPartitionOperation(database, table, operation string, presetIsPart bool) (*querier_dto.CatalogueMutation, error) {
	isPart := presetIsPart
	if !presetIsPart {
		switch {
		case p.matchKeyword(keywordPart):
			isPart = true
		case p.matchKeyword(keywordPartition):
			isPart = false
		default:
			isPart = false
		}
	}
	target := keywordPartition
	if isPart {
		target = keywordPart
	}
	mutation := &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTablePartition,
		SchemaName: database,
		TableName:  table,
		EngineSpecific: map[string]string{
			engineKeyPartitionOp:     operation,
			engineKeyPartitionTarget: target,
		},
	}
	if operation == keywordAttach && p.matchKeyword(kwAll) {
		mutation.EngineSpecific[engineKeyPartitionExpr] = kwAll
		if p.matchKeyword(keywordFrom) {
			_, source, sourceErr := p.parseDatabaseQualifiedName()
			if sourceErr != nil {
				return nil, sourceErr
			}
			mutation.EngineSpecific[engineKeyPartitionFrom] = source
		}
		return mutation, nil
	}
	mutation.EngineSpecific[engineKeyPartitionExpr] = p.captureAlterPartitionExpression()
	if err := p.captureAlterPartitionTail(mutation); err != nil {
		return nil, err
	}
	return mutation, nil
}

// captureAlterPartitionExpression reads the partition expression following PARTITION /
// PART. Stops at one of the recognised tail keywords (FROM, TO, WITH, USING).
//
// Returns string which is the trimmed partition expression text.
func (p *parser) captureAlterPartitionExpression() string {
	var builder strings.Builder
	depth := 0
	for !p.atEnd() {
		tok := p.current()
		if depth == 0 && tok.kind == tokenIdentifier && isPartitionTailKeyword(tok.value) {
			break
		}
		if depth == 0 && tok.kind == tokenComma {
			break
		}
		switch tok.kind {
		case tokenLeftParen:
			depth++
		case tokenRightParen:
			if depth == 0 {
				return strings.TrimSpace(builder.String())
			}
			depth--
		default:
		}
		writeTokenAsSourceText(&builder, tok)
		builder.WriteByte(' ')
		p.advance()
	}
	return strings.TrimSpace(builder.String())
}

// isPartitionTailKeyword reports whether the identifier marks the end of a partition
// expression and the start of a destination or backup-name clause.
//
// Takes text (string) which is the identifier value to test.
//
// Returns bool which is true when the identifier is a partition tail keyword.
func isPartitionTailKeyword(text string) bool {
	switch strings.ToUpper(text) {
	case keywordFrom, keywordTo, keywordWith, "USING":
		return true
	default:
		return false
	}
}

// captureAlterPartitionTail reads the optional destination and backup-name clauses that
// follow the partition expression. A malformed `WITH` tail (missing NAME, or NAME without
// a string literal) returns an error so the caller surfaces a diagnostic rather than
// silently dropping the backup-name capture.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the captured tail data.
//
// Returns error when the WITH tail is malformed.
func (p *parser) captureAlterPartitionTail(mutation *querier_dto.CatalogueMutation) error {
	for !p.atEnd() {
		switch {
		case p.matchKeyword(keywordFrom):
			mutation.EngineSpecific[engineKeyPartitionDest] = p.captureAlterPartitionExpression()
		case p.matchKeyword(keywordTo):
			p.captureAlterPartitionMoveDestination(mutation)
		case p.matchKeyword(keywordWith):
			if !p.matchKeyword(keywordName) {
				return fmt.Errorf("expected NAME after WITH at position %d", p.current().position)
			}
			if p.current().kind != tokenString {
				return fmt.Errorf("expected string literal after WITH NAME at position %d", p.current().position)
			}
			mutation.EngineSpecific[engineKeyPartitionBackup] = p.current().value
			p.advance()
		default:
			return nil
		}
	}
	return nil
}

// captureAlterPartitionMoveDestination reads the destination spec after a `TO` keyword.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the destination data.
func (p *parser) captureAlterPartitionMoveDestination(mutation *querier_dto.CatalogueMutation) {
	if p.isAnyKeyword("TABLE", "DISK", "VOLUME") {
		destinationKind := strings.ToUpper(p.current().value)
		p.advance()
		value := p.captureAlterPartitionExpression()
		mutation.EngineSpecific[engineKeyPartitionDest] = destinationKind + " " + value
		return
	}
	mutation.EngineSpecific[engineKeyPartitionDest] = p.captureAlterPartitionExpression()
}
