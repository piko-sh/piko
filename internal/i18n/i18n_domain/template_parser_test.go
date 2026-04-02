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

package i18n_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTemplate_Empty(t *testing.T) {
	parts, errs := ParseTemplate("")
	assert.Nil(t, parts)
	assert.Nil(t, errs)
}

func TestParseTemplate_LiteralOnly(t *testing.T) {
	parts, errs := ParseTemplate("Hello, World!")
	require.NoError(t, errorsToError(errs))
	require.Len(t, parts, 1)
	assert.Equal(t, PartLiteral, parts[0].Kind)
	assert.Equal(t, "Hello, World!", parts[0].Literal)
}

func TestParseTemplate_SingleExpression(t *testing.T) {
	parts, errs := ParseTemplate("${name}")
	require.NoError(t, errorsToError(errs))
	require.Len(t, parts, 1)
	assert.Equal(t, PartExpression, parts[0].Kind)
	assert.Equal(t, "name", parts[0].ExprSource)
}

func TestParseTemplate_ExpressionWithLiterals(t *testing.T) {
	parts, errs := ParseTemplate("Hello, ${name}!")
	require.NoError(t, errorsToError(errs))
	require.Len(t, parts, 3)

	assert.Equal(t, PartLiteral, parts[0].Kind)
	assert.Equal(t, "Hello, ", parts[0].Literal)

	assert.Equal(t, PartExpression, parts[1].Kind)
	assert.Equal(t, "name", parts[1].ExprSource)

	assert.Equal(t, PartLiteral, parts[2].Kind)
	assert.Equal(t, "!", parts[2].Literal)
}

func TestParseTemplate_MultipleExpressions(t *testing.T) {
	parts, errs := ParseTemplate("${greeting}, ${name}! You have ${count} messages.")
	require.NoError(t, errorsToError(errs))
	require.Len(t, parts, 6)

	assert.Equal(t, PartExpression, parts[0].Kind)
	assert.Equal(t, "greeting", parts[0].ExprSource)

	assert.Equal(t, PartLiteral, parts[1].Kind)
	assert.Equal(t, ", ", parts[1].Literal)

	assert.Equal(t, PartExpression, parts[2].Kind)
	assert.Equal(t, "name", parts[2].ExprSource)

	assert.Equal(t, PartLiteral, parts[3].Kind)
	assert.Equal(t, "! You have ", parts[3].Literal)

	assert.Equal(t, PartExpression, parts[4].Kind)
	assert.Equal(t, "count", parts[4].ExprSource)

	assert.Equal(t, PartLiteral, parts[5].Kind)
	assert.Equal(t, " messages.", parts[5].Literal)
}

func TestParseTemplate_AdjacentExpressions(t *testing.T) {
	parts, errs := ParseTemplate("${first}${second}${third}")
	require.NoError(t, errorsToError(errs))
	require.Len(t, parts, 3)

	assert.Equal(t, PartExpression, parts[0].Kind)
	assert.Equal(t, "first", parts[0].ExprSource)
	assert.Equal(t, PartExpression, parts[1].Kind)
	assert.Equal(t, "second", parts[1].ExprSource)
	assert.Equal(t, PartExpression, parts[2].Kind)
	assert.Equal(t, "third", parts[2].ExprSource)
}

func TestParseTemplate_MemberAccess(t *testing.T) {
	parts, errs := ParseTemplate("Hello, ${user.name}!")
	require.NoError(t, errorsToError(errs))
	require.Len(t, parts, 3)

	assert.Equal(t, PartExpression, parts[1].Kind)
	assert.Equal(t, "user.name", parts[1].ExprSource)
}

func TestParseTemplate_NestedBraces(t *testing.T) {
	parts, errs := ParseTemplate("Data: ${obj.items[0]}")
	require.NoError(t, errorsToError(errs))
	require.Len(t, parts, 2)

	assert.Equal(t, PartExpression, parts[1].Kind)
	assert.Equal(t, "obj.items[0]", parts[1].ExprSource)
}

func TestParseTemplate_EscapedDollar(t *testing.T) {
	parts, errs := ParseTemplate("Price: \\$100")
	require.NoError(t, errorsToError(errs))
	require.Len(t, parts, 1)

	assert.Equal(t, PartLiteral, parts[0].Kind)
	assert.Equal(t, "Price: $100", parts[0].Literal)
}

func TestParseTemplate_EscapedAt(t *testing.T) {
	parts, errs := ParseTemplate("Email: user\\@example.com")
	require.NoError(t, errorsToError(errs))
	require.Len(t, parts, 1)

	assert.Equal(t, PartLiteral, parts[0].Kind)
	assert.Equal(t, "Email: user@example.com", parts[0].Literal)
}

func TestParseTemplate_LinkedMessage(t *testing.T) {
	parts, errs := ParseTemplate("Welcome to @app_name!")
	require.NoError(t, errorsToError(errs))
	require.Len(t, parts, 3)

	assert.Equal(t, PartLiteral, parts[0].Kind)
	assert.Equal(t, "Welcome to ", parts[0].Literal)

	assert.Equal(t, PartLinkedMessage, parts[1].Kind)
	assert.Equal(t, "app_name", parts[1].LinkedKey)

	assert.Equal(t, PartLiteral, parts[2].Kind)
	assert.Equal(t, "!", parts[2].Literal)
}

func TestParseTemplate_LinkedMessageWithPath(t *testing.T) {
	parts, errs := ParseTemplate("See @common.messages.greeting")
	require.NoError(t, errorsToError(errs))
	require.Len(t, parts, 2)

	assert.Equal(t, PartLiteral, parts[0].Kind)
	assert.Equal(t, "See ", parts[0].Literal)

	assert.Equal(t, PartLinkedMessage, parts[1].Kind)
	assert.Equal(t, "common.messages.greeting", parts[1].LinkedKey)
}

func TestParseTemplate_LinkedMessageAndExpression(t *testing.T) {
	parts, errs := ParseTemplate("@greeting, ${name}!")
	require.NoError(t, errorsToError(errs))
	require.Len(t, parts, 4)

	assert.Equal(t, PartLinkedMessage, parts[0].Kind)
	assert.Equal(t, "greeting", parts[0].LinkedKey)

	assert.Equal(t, PartLiteral, parts[1].Kind)
	assert.Equal(t, ", ", parts[1].Literal)

	assert.Equal(t, PartExpression, parts[2].Kind)
	assert.Equal(t, "name", parts[2].ExprSource)

	assert.Equal(t, PartLiteral, parts[3].Kind)
	assert.Equal(t, "!", parts[3].Literal)
}

func TestParseTemplate_UnterminatedExpression(t *testing.T) {
	_, errs := ParseTemplate("Hello ${name")
	require.NotNil(t, errs)
	assert.Contains(t, errs[0], "Unterminated expression")
}

func TestParseTemplate_EmptyLinkedMessage(t *testing.T) {
	_, errs := ParseTemplate("Hello @!")
	require.NotNil(t, errs)
	assert.Contains(t, errs[0], "Expected identifier after '@'")
}

func TestExtractExpressions_Empty(t *testing.T) {
	exprs := ExtractExpressions("")
	assert.Nil(t, exprs)
}

func TestExtractExpressions_NoExpressions(t *testing.T) {
	exprs := ExtractExpressions("Hello, World!")
	assert.Nil(t, exprs)
}

func TestExtractExpressions_SingleExpression(t *testing.T) {
	exprs := ExtractExpressions("Hello, ${name}!")
	assert.Equal(t, []string{"name"}, exprs)
}

func TestExtractExpressions_MultipleExpressions(t *testing.T) {
	exprs := ExtractExpressions("${greeting}, ${name}! You have ${count} messages.")
	assert.Equal(t, []string{"greeting", "name", "count"}, exprs)
}

func TestExtractLinkedKeys_Empty(t *testing.T) {
	keys := extractLinkedKeys("")
	assert.Nil(t, keys)
}

func TestExtractLinkedKeys_NoLinks(t *testing.T) {
	keys := extractLinkedKeys("Hello, ${name}!")
	assert.Nil(t, keys)
}

func TestExtractLinkedKeys_SingleLink(t *testing.T) {
	keys := extractLinkedKeys("Welcome to @app_name!")
	assert.Equal(t, []string{"app_name"}, keys)
}

func TestExtractLinkedKeys_MultipleLinks(t *testing.T) {
	keys := extractLinkedKeys("@greeting from @sender")
	assert.Equal(t, []string{"greeting", "sender"}, keys)
}

func errorsToError(errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	return &parseErrors{errs}
}

type parseErrors struct {
	errs []string
}

func (e *parseErrors) Error() string {
	if len(e.errs) == 0 {
		return ""
	}
	return e.errs[0]
}

func BenchmarkParseTemplate_Simple(b *testing.B) {
	template := "Hello, ${name}!"
	b.ResetTimer()

	for b.Loop() {
		_, _ = ParseTemplate(template)
	}
}

func BenchmarkParseTemplate_Complex(b *testing.B) {
	template := "${greeting}, ${name}! You have ${count} new messages from ${sender}."
	b.ResetTimer()

	for b.Loop() {
		_, _ = ParseTemplate(template)
	}
}

func BenchmarkParseTemplate_LinkedMessages(b *testing.B) {
	template := "@greeting from @app_name - ${message}"
	b.ResetTimer()

	for b.Loop() {
		_, _ = ParseTemplate(template)
	}
}

func BenchmarkExtractExpressions(b *testing.B) {
	template := "${greeting}, ${name}! You have ${count} new messages."
	b.ResetTimer()

	for b.Loop() {
		_ = ExtractExpressions(template)
	}
}
