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
	"context"
	"sync"
	"time"

	"piko.sh/piko/internal/ast/ast_domain"
)

const (
	// defaultDateTimeStyle is the default style for formatting time.Time values.
	defaultDateTimeStyle = DateTimeStyleMedium

	// maxLinkedMessageDepth is the maximum recursion depth for linked message
	// resolution. This prevents infinite recursion from circular references.
	maxLinkedMessageDepth = 10

	// defaultBuildScopeStrBufCapacity is the default buffer capacity for scope
	// building.
	defaultBuildScopeStrBufCapacity = 64
)

// messageResolver defines the interface for looking up linked message
// references.
type messageResolver interface {
	// ResolveMessage looks up a message by its key and returns the rendered text.
	//
	// Takes key (string) which is the message identifier to look up.
	// Takes locale (string) which is the language code to use.
	// Takes scope (map[string]any) which provides values for message templates.
	// Takes depth (int) which tracks how deep the lookup has gone to prevent
	// endless loops when messages refer to each other.
	//
	// Returns string which is the rendered message text.
	// Returns bool which is true if the message was found.
	ResolveMessage(key, locale string, scope map[string]any, depth int) (string, bool)
}

// varLookup defines a way to look up variables by name.
type varLookup interface {
	// LookupVar looks up a variable by name and writes its value to the buffer.
	//
	// Takes name (string) which is the variable name to look up.
	// Takes buffer (*StrBuf) which receives the variable value if found.
	//
	// Returns bool which is true if the variable was found.
	LookupVar(name string, buffer *StrBuf) bool
}

// scopeProvider defines a type that can give its scope map directly.
// Types that have both varLookup and scopeProvider can skip the string
// conversion by giving the scope straight away.
type scopeProvider interface {
	// GetScope returns the current scope as a map of names to values.
	//
	// Returns map[string]any which holds the variable bindings in scope.
	GetScope() map[string]any
}

// renderContext holds the data needed when rendering i18n templates.
type renderContext struct {
	// resolver finds linked messages; nil turns off message linking.
	resolver messageResolver

	// scope holds variables used when working out expression values.
	scope map[string]any

	// buffer stores the output while building the rendered result.
	buffer *StrBuf

	// count specifies the number of items to display; nil means no limit.
	count *int

	// locale is the locale code for formatting dates and times.
	locale string

	// depth tracks how deep the recursion is when resolving linked messages.
	depth int
}

var (
	// expressionCache caches parsed expressions for templates.
	expressionCache sync.Map

	// emptyScope is a reusable empty scope map to avoid allocation when no
	// variables are present. This is safe because the map is never written to when
	// empty.
	emptyScope = make(map[string]any)
)

// ClearExpressionCache resets the i18n expression cache to an empty state.
// This is intended for test isolation between iterations.
func ClearExpressionCache() {
	expressionCache = sync.Map{}
}

// Render renders a translation entry with the given variables and count.
// This is a helper function for simple use cases with map-based variables.
//
// Takes entry (*Entry) which is the translation entry to render.
// Takes vars (map[string]any) which provides variable values for template
// substitution.
// Takes count (*int) which picks the plural form when not nil.
// Takes locale (string) which sets the locale for plural rules.
// Takes buffer (*StrBuf) which is a reusable buffer for faster rendering.
//
// Returns string which is the rendered translation, or empty if entry is nil.
func Render(entry *Entry, vars map[string]any, count *int, locale string, buffer *StrBuf) string {
	if entry == nil {
		return ""
	}

	buffer.Reset()

	var parts []TemplatePart
	if entry.HasPlurals && count != nil {
		if len(entry.PluralFormsParts) > 0 {
			index := selectPluralFormIndex(*count, locale, len(entry.PluralFormsParts))
			parts = entry.PluralFormsParts[index]
		} else if len(entry.PluralForms) > 0 {
			selectedForm := SelectPluralForm(*count, locale, entry.PluralForms)
			parts, _ = ParseTemplate(selectedForm)
		}
	}
	if len(parts) == 0 {
		if len(entry.Parts) > 0 {
			parts = entry.Parts
		} else {
			parts, _ = ParseTemplate(entry.Template)
		}
	}

	scope := vars
	if count != nil {
		if scope == nil {
			scope = make(map[string]any)
		}
		scope["count"] = *count
	}

	ctx := &renderContext{
		scope:    scope,
		resolver: nil,
		locale:   locale,
		count:    count,
		buffer:   buffer,
		depth:    0,
	}

	return renderTemplate(parts, ctx)
}

// getOrParseExpression returns a cached expression or parses and caches it.
//
// Takes source (string) which contains the expression text to parse.
//
// Returns ast_domain.Expression which is the parsed expression, or nil if
// parsing failed.
func getOrParseExpression(source string) ast_domain.Expression {
	if cached, ok := expressionCache.Load(source); ok {
		if expression, assertOk := cached.(ast_domain.Expression); assertOk {
			return expression
		}
	}

	ctx := context.Background()
	parser := ast_domain.NewExpressionParser(ctx, source, "i18n")
	expression, diagnostics := parser.ParseExpression(ctx)
	parser.Release()

	if !ast_domain.HasErrors(diagnostics) && expression != nil {
		expressionCache.Store(source, expression)
	}

	return expression
}

// renderTemplate builds a string from parsed template parts.
//
// When parts is empty or ctx.buffer is nil, returns an empty string.
//
// Takes parts ([]TemplatePart) which contains the parsed template parts to
// render.
// Takes ctx (*renderContext) which provides the rendering context including
// the output buffer and variable values.
//
// Returns string which is the rendered template output.
func renderTemplate(parts []TemplatePart, ctx *renderContext) string {
	if len(parts) == 0 || ctx.buffer == nil {
		return ""
	}

	ctx.buffer.Reset()

	for i := range parts {
		part := &parts[i]
		switch part.Kind {
		case PartLiteral:
			ctx.buffer.WriteString(part.Literal)

		case PartExpression:
			renderExpressionPart(part, ctx)

		case PartLinkedMessage:
			renderLinkedMessagePart(part, ctx)
		}
	}

	return ctx.buffer.String()
}

// renderExpressionPart renders a ${expression} part by evaluating it and
// writing the result to the output buffer.
//
// Takes part (*TemplatePart) which contains the expression to render.
// Takes ctx (*renderContext) which provides the output buffer and locale.
func renderExpressionPart(part *TemplatePart, ctx *renderContext) {
	expression := part.Expression
	if expression == nil {
		expression = getOrParseExpression(part.ExprSource)
		if expression == nil {
			ctx.buffer.WriteString("${")
			ctx.buffer.WriteString(part.ExprSource)
			_ = ctx.buffer.WriteByte('}')
			return
		}
		part.Expression = expression
	}

	result := ast_domain.EvaluateExpression(expression, ctx.scope)
	if result == nil {
		ctx.buffer.WriteString("${")
		ctx.buffer.WriteString(part.ExprSource)
		_ = ctx.buffer.WriteByte('}')
		return
	}

	switch v := result.(type) {
	case DateTime:
		ctx.buffer.WriteString(v.Format(ctx.locale))
	case time.Time:
		ctx.buffer.WriteString(FormatDateTime(v, ctx.locale, defaultDateTimeStyle, false, false))
	default:
		ctx.buffer.WriteAny(result)
	}
}

// renderLinkedMessagePart renders a @linked.message part by resolving the
// linked message key and writing the result to the buffer.
//
// When the depth limit is reached or no resolver is available, the original
// key is written with an @ prefix. When the key cannot be resolved, the
// original key is also written unchanged.
//
// Takes part (*TemplatePart) which contains the linked message key to resolve.
// Takes ctx (*renderContext) which provides the buffer, resolver, and depth.
func renderLinkedMessagePart(part *TemplatePart, ctx *renderContext) {
	if ctx.depth >= maxLinkedMessageDepth {
		_ = ctx.buffer.WriteByte('@')
		ctx.buffer.WriteString(part.LinkedKey)
		return
	}

	if ctx.resolver == nil {
		_ = ctx.buffer.WriteByte('@')
		ctx.buffer.WriteString(part.LinkedKey)
		return
	}

	resolved, found := ctx.resolver.ResolveMessage(part.LinkedKey, ctx.locale, ctx.scope, ctx.depth+1)
	if !found {
		_ = ctx.buffer.WriteByte('@')
		ctx.buffer.WriteString(part.LinkedKey)
		return
	}

	ctx.buffer.WriteString(resolved)
}

// renderWithVars renders a translation entry using a varLookup for variable
// resolution. This is the primary render function used by Translation.String().
//
// Takes entry (*Entry) which is the translation entry to render.
// Takes vars (varLookup) which provides variable resolution for template
// placeholders.
// Takes count (*int) which specifies the count for plural form selection, or
// nil if not applicable.
// Takes locale (string) which determines the plural form rules to apply.
// Takes buffer (*StrBuf) which is a reusable buffer for building the output.
//
// Returns string which is the rendered translation with variables substituted.
func renderWithVars(entry *Entry, vars varLookup, count *int, locale string, buffer *StrBuf) string {
	if entry == nil {
		return ""
	}

	buffer.Reset()

	var parts []TemplatePart
	if entry.HasPlurals && count != nil {
		if len(entry.PluralFormsParts) > 0 {
			index := selectPluralFormIndex(*count, locale, len(entry.PluralFormsParts))
			parts = entry.PluralFormsParts[index]
		} else if len(entry.PluralForms) > 0 {
			selectedForm := SelectPluralForm(*count, locale, entry.PluralForms)
			parts, _ = ParseTemplate(selectedForm)
		}
	}
	if len(parts) == 0 {
		if len(entry.Parts) > 0 {
			parts = entry.Parts
		} else {
			parts, _ = ParseTemplate(entry.Template)
		}
	}

	var scope map[string]any
	if sp, ok := vars.(scopeProvider); ok {
		scope = sp.GetScope()
	} else {
		scope = buildScopeFromVars(vars, parts)
	}

	if count != nil {
		if scope == nil {
			scope = make(map[string]any, 1)
		}
		scope["count"] = *count
	} else if scope == nil {
		scope = emptyScope
	}

	ctx := &renderContext{
		scope:    scope,
		resolver: nil,
		locale:   locale,
		count:    count,
		buffer:   buffer,
		depth:    0,
	}

	return renderTemplate(parts, ctx)
}

// buildScopeFromVars extracts variable values from a lookup function into a
// scope map.
//
// Takes vars (varLookup) which provides the function to look up variable
// values.
// Takes parts ([]TemplatePart) which contains the template parts to scan for
// variable names.
//
// Returns map[string]any which contains the variable names and their values.
func buildScopeFromVars(vars varLookup, parts []TemplatePart) map[string]any {
	scope := make(map[string]any)
	if vars == nil {
		return scope
	}

	tmpBuf := NewStrBuf(defaultBuildScopeStrBufCapacity)
	for _, part := range parts {
		if part.Kind == PartExpression {
			name := part.ExprSource
			tmpBuf.Reset()
			if vars.LookupVar(name, tmpBuf) {
				scope[name] = tmpBuf.String()
			}
		}
	}
	return scope
}

// renderSimple renders a template string with variables.
// This is a helper function for templates that have not been parsed beforehand.
//
// Takes template (string) which is the template string to render.
// Takes vars (map[string]any) which holds the values for substitution.
// Takes buffer (*StrBuf) which is a reusable buffer for building the output.
//
// Returns string which is the rendered template, or the original template
// if parsing fails.
func renderSimple(template string, vars map[string]any, buffer *StrBuf) string {
	parts, errs := ParseTemplate(template)
	if len(errs) > 0 || len(parts) == 0 {
		return template
	}

	ctx := &renderContext{
		scope:    vars,
		resolver: nil,
		locale:   "",
		count:    nil,
		buffer:   buffer,
		depth:    0,
	}

	return renderTemplate(parts, ctx)
}
