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

package db_engine_postgres

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

type foobarExtension struct{}

const (
	statementKindFoobar           = StatementKindExtensionBase + 0
	statementKindOverridingSelect = StatementKindExtensionBase + 50
)

func (foobarExtension) Classify(tokens []Token) StatementKind {
	if len(tokens) < 2 {
		return 0
	}
	if !strings.EqualFold(tokens[0].value, "CREATE") {
		return 0
	}
	if !strings.EqualFold(tokens[1].value, "FOOBAR") {
		return 0
	}
	return statementKindFoobar
}

func (foobarExtension) Parse(p ParserContext, kind StatementKind) (*querier_dto.CatalogueMutation, error) {
	if kind != statementKindFoobar {
		return nil, nil
	}
	if err := p.ExpectKeyword("CREATE"); err != nil {
		return nil, err
	}
	if err := p.ExpectKeyword("FOOBAR"); err != nil {
		return nil, err
	}
	tok := p.Advance()
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateTable,
		SchemaName: "public",
		TableName:  tok.value,
		EngineSpecific: map[string]string{
			"FOOBAR": "true",
		},
	}, nil
}

func TestPostgresEngine_StatementExtension_RecognisesUnknownStatement(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine(WithStatementExtensions(foobarExtension{}))

	statements, err := engine.ParseStatements("CREATE FOOBAR widgets")
	require.NoError(t, err)
	require.Len(t, statements, 1)

	raw, ok := statements[0].Raw.(*parsedStatement)
	require.True(t, ok)
	assert.Equal(t, statementKindFoobar, raw.kind, "extension kind should survive ParseStatements")
}

func TestPostgresEngine_StatementExtension_DispatchProducesMutation(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine(WithStatementExtensions(foobarExtension{}))

	statements, err := engine.ParseStatements("CREATE FOOBAR widgets")
	require.NoError(t, err)
	require.Len(t, statements, 1)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
	assert.Equal(t, "widgets", mutation.TableName)
	assert.Equal(t, "true", mutation.EngineSpecific["FOOBAR"])
}

func TestPostgresEngine_StatementExtension_BuiltinPathUnchanged(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine(WithStatementExtensions(foobarExtension{}))

	statements, err := engine.ParseStatements("CREATE TABLE products (id BIGINT)")
	require.NoError(t, err)
	require.Len(t, statements, 1)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
	assert.Equal(t, "products", mutation.TableName)
	assert.Empty(t, mutation.EngineSpecific["FOOBAR"], "built-in DDL should not pick up the extension's marker")
}

func TestPostgresEngine_StatementExtension_DeclinedKindFallsThrough(t *testing.T) {
	t.Parallel()

	declining := decliningExtension{}
	engine := NewPostgresEngine(WithStatementExtensions(declining))

	statements, err := engine.ParseStatements("CREATE NONESUCH zzz")
	require.NoError(t, err)
	require.Len(t, statements, 1)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	assert.Nil(t, mutation)
}

type decliningExtension struct{}

func (decliningExtension) Classify(tokens []Token) StatementKind { return 0 }
func (decliningExtension) Parse(p ParserContext, kind StatementKind) (*querier_dto.CatalogueMutation, error) {
	return nil, nil
}

type misusingBuiltinKindExtension struct{}

func (misusingBuiltinKindExtension) Classify(tokens []Token) StatementKind {
	if len(tokens) < 2 {
		return 0
	}
	if !strings.EqualFold(tokens[0].value, "CREATE") {
		return 0
	}
	if !strings.EqualFold(tokens[1].value, "NONESUCH") {
		return 0
	}

	return statementKindDropTable
}

func (misusingBuiltinKindExtension) Parse(p ParserContext, kind StatementKind) (*querier_dto.CatalogueMutation, error) {
	return nil, nil
}

func TestPostgresEngine_StatementExtension_RejectsBuiltinKindMisuse(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine(WithStatementExtensions(misusingBuiltinKindExtension{}))

	statements, err := engine.ParseStatements("CREATE NONESUCH zzz")
	require.NoError(t, err)
	require.Len(t, statements, 1)

	raw, rawOk := statements[0].Raw.(*parsedStatement)
	require.True(t, rawOk, "extension misuse must still produce a *parsedStatement")
	assert.Equal(t, statementKindUnknown, raw.kind,
		"extension returning a built-in kind must be rejected and fall through to unknown")

	mutation, applyErr := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, applyErr)
	assert.Nil(t, mutation, "unknown statement must produce no mutation")
}

func TestPostgresEngine_PostParseHook_EnrichesExistingMutation(t *testing.T) {
	t.Parallel()

	tag := func(_ ParserContext, kind StatementKind, mutation *querier_dto.CatalogueMutation) error {
		if mutation == nil {
			return nil
		}
		if kind == statementKindCreateTable {
			if mutation.EngineSpecific == nil {
				mutation.EngineSpecific = map[string]string{}
			}
			mutation.EngineSpecific["TAGGED_BY_HOOK"] = "true"
		}
		return nil
	}
	engine := NewPostgresEngine(WithPostParseHook(tag))

	statements, err := engine.ParseStatements("CREATE TABLE widgets (id BIGINT)")
	require.NoError(t, err)
	require.Len(t, statements, 1)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "true", mutation.EngineSpecific["TAGGED_BY_HOOK"], "post-parse hook should run after built-in handler")
}

func TestPostgresEngine_PostParseHook_RunInRegistrationOrder(t *testing.T) {
	t.Parallel()

	order := []string{}
	first := func(_ ParserContext, _ StatementKind, _ *querier_dto.CatalogueMutation) error {
		order = append(order, "first")
		return nil
	}
	second := func(_ ParserContext, _ StatementKind, _ *querier_dto.CatalogueMutation) error {
		order = append(order, "second")
		return nil
	}
	engine := NewPostgresEngine(
		WithPostParseHook(first),
		WithPostParseHook(second),
	)

	statements, err := engine.ParseStatements("CREATE TABLE x (id BIGINT)")
	require.NoError(t, err)
	_, err = engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	assert.Equal(t, []string{"first", "second"}, order, "hooks should run in registration order")
}

func TestPostgresEngine_PostParseHook_RanOnExtensionStatement(t *testing.T) {
	t.Parallel()

	hookSawExtensionKind := false
	hook := func(_ ParserContext, kind StatementKind, _ *querier_dto.CatalogueMutation) error {
		if kind == statementKindFoobar {
			hookSawExtensionKind = true
		}
		return nil
	}
	engine := NewPostgresEngine(
		WithStatementExtensions(foobarExtension{}),
		WithPostParseHook(hook),
	)

	statements, err := engine.ParseStatements("CREATE FOOBAR x")
	require.NoError(t, err)
	_, err = engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	assert.True(t, hookSawExtensionKind, "post-parse hook must run for extension statements too")
}

type selectOverridingExtension struct{}

func (selectOverridingExtension) Classify(tokens []Token) StatementKind {
	if len(tokens) < 4 {
		return 0
	}
	if !strings.EqualFold(tokens[0].value, "SELECT") {
		return 0
	}
	if !strings.EqualFold(tokens[1].value, "magic_func") {
		return 0
	}
	if tokens[2].kind != tokenLeftParen {
		return 0
	}
	return statementKindOverridingSelect
}

func (selectOverridingExtension) Parse(p ParserContext, kind StatementKind) (*querier_dto.CatalogueMutation, error) {
	if kind != statementKindOverridingSelect {
		return nil, nil
	}
	p.ConsumeRemainder()
	return &querier_dto.CatalogueMutation{
		Kind:           querier_dto.MutationCreateTable,
		TableName:      "magic_target",
		EngineSpecific: map[string]string{"MAGIC": "true"},
	}, nil
}

func TestPostgresEngine_StatementExtension_OverridesBuiltinClassification(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine(WithStatementExtensions(selectOverridingExtension{}))

	statements, err := engine.ParseStatements("SELECT magic_func(1, 2)")
	require.NoError(t, err)
	require.Len(t, statements, 1)
	raw, rawOk := statements[0].Raw.(*parsedStatement)
	require.True(t, rawOk, "extension statement must carry a *parsedStatement payload")
	assert.Equal(t, statementKindOverridingSelect, raw.kind,
		"extension classification must override built-in SELECT classification")

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "magic_target", mutation.TableName)
	assert.Equal(t, "true", mutation.EngineSpecific["MAGIC"])
}

func TestPostgresEngine_PostParseHook_ErrorAbortsChain(t *testing.T) {
	t.Parallel()

	secondRan := false
	failing := func(_ ParserContext, _ StatementKind, _ *querier_dto.CatalogueMutation) error {
		return assert.AnError
	}
	tracking := func(_ ParserContext, _ StatementKind, _ *querier_dto.CatalogueMutation) error {
		secondRan = true
		return nil
	}
	engine := NewPostgresEngine(
		WithPostParseHook(failing),
		WithPostParseHook(tracking),
	)

	statements, err := engine.ParseStatements("CREATE TABLE x (id BIGINT)")
	require.NoError(t, err)
	_, applyErr := engine.ApplyDDL(context.Background(), statements[0])
	require.Error(t, applyErr, "failing hook must abort the chain")
	assert.False(t, secondRan, "subsequent hooks must not run after an error")
}

func TestPostgresEngine_NoExtensions_BehavesLikeBaseline(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()

	statements, err := engine.ParseStatements("CREATE FOOBAR widgets")
	require.NoError(t, err)
	require.Len(t, statements, 1)
	raw, rawOk := statements[0].Raw.(*parsedStatement)
	require.True(t, rawOk, "baseline statement must carry a *parsedStatement payload")
	assert.Equal(t, statementKindUnknown, raw.kind)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	assert.Nil(t, mutation)
}

func TestParserContext_MatchIfExists_PartialMatchRestoresCursor(t *testing.T) {
	t.Parallel()

	tokens, err := tokenise("IF NOT EXISTS table")
	require.NoError(t, err)
	parserInstance := newParser(tokens)
	parserContextInstance := newParserContext(parserInstance, NewPostgresEngine())

	startPosition := parserInstance.position
	matched := parserContextInstance.MatchIfExists()
	assert.False(t, matched, "IF NOT EXISTS must not satisfy MatchIfExists")
	assert.Equal(t, startPosition, parserInstance.position,
		"cursor must rewind to IF on a partial match")
	assert.True(t, parserInstance.isKeyword("IF"),
		"cursor should be back on the IF token so the next matcher can re-read it")
}

func TestParserContext_ParseReloptionList(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		want     map[string]string
		name     string
		sql      string
		wantErr  string
		consumed bool
	}{
		{
			name:     "key with a value",
			sql:      "(fillfactor = 70)",
			want:     map[string]string{"fillfactor": "70"},
			consumed: true,
		},
		{
			name:     "bare key means true",
			sql:      "(timescaledb.continuous)",
			want:     map[string]string{"timescaledb.continuous": "true"},
			consumed: true,
		},
		{
			name:     "bare key mixed with valued keys",
			sql:      "(timescaledb.continuous, timescaledb.materialized_only = false)",
			want:     map[string]string{"timescaledb.continuous": "true", "timescaledb.materialized_only": "false"},
			consumed: true,
		},
		{
			name:    "key followed by an unexpected token",
			sql:     "(fillfactor 70)",
			wantErr: "expected '=' after reloption key",
		},
		{
			name:    "missing open paren",
			sql:     "fillfactor = 70",
			wantErr: "expected '('",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			tokens, err := tokenise(testCase.sql)
			require.NoError(t, err)
			parserInstance := newParser(tokens)

			options, err := newParserContext(parserInstance, NewPostgresEngine()).ParseReloptionList()

			if testCase.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.want, options)
			assert.Equal(t, testCase.consumed, parserInstance.atEnd())
		})
	}
}

func TestParserContext_ExpectKeyword(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{name: "matching keyword is consumed", sql: "create table", wantErr: false},
		{name: "other keyword is an error", sql: "DROP table", wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			tokens, err := tokenise(testCase.sql)
			require.NoError(t, err)
			parserInstance := newParser(tokens)

			err = newParserContext(parserInstance, NewPostgresEngine()).ExpectKeyword("CREATE")

			if !testCase.wantErr {
				require.NoError(t, err)
				assert.Equal(t, 1, parserInstance.position)
				return
			}
			require.Error(t, err)
			assert.Same(t, err, parserInstance.syntax.err, "the mismatch fails the statement even if the extension drops it")
		})
	}
}

func TestPostgresEngine_StatementExtension_KeywordMismatchFailsApply(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine(WithStatementExtensions(foobarTableExtension{}))
	statements, err := engine.ParseStatements("CREATE FOOBAR widgets")
	require.NoError(t, err)

	_, err = engine.ApplyDDL(context.Background(), statements[0])

	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected keyword [TABLE]")
	assert.NotContains(t, err.Error(), "panic")
}

type foobarTableExtension struct {
	foobarExtension
}

func (foobarTableExtension) Parse(p ParserContext, kind StatementKind) (*querier_dto.CatalogueMutation, error) {
	if kind != statementKindFoobar {
		return nil, nil
	}
	if err := p.ExpectKeyword("CREATE"); err != nil {
		return nil, err
	}
	if err := p.ExpectKeyword("FOOBAR"); err != nil {
		return nil, err
	}
	if err := p.ExpectKeyword("TABLE"); err != nil {
		return nil, err
	}
	return nil, nil
}
