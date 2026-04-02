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

package highlight_chroma

import (
	"context"

	chromalib "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"piko.sh/piko/wdk/logger"
)

// PikoLexer is the Chroma lexer for Piko Single File Components (.pk, .pkc).
//
// It supports SFC blocks (<template>, <script>, <style>, <i18n>), script
// variations (Go and TypeScript), Piko directives (p-if, p-for, p-bind, etc.)
// with modifiers, shorthand syntax (:attr for binding, @event for events),
// interpolation ({{ expression }}), and embedded language highlighting for
// Go, TypeScript, CSS, and JSON.
var PikoLexer = chromalib.MustNewLexer(
	&chromalib.Config{
		Name:            "Piko",
		Aliases:         []string{"piko", "pk", "pkc"},
		Filenames:       []string{"*.pk", "*.pkc"},
		AliasFilenames:  nil,
		MimeTypes:       []string{"text/x-piko"},
		CaseInsensitive: false,
		DotAll:          false,
		NotMultiline:    false,
		EnsureNL:        false,
		Priority:        0,
		Analyse:         nil,
	},
	func() chromalib.Rules {
		return chromalib.Rules{
			"root": {
				{
					Pattern: `(<)(script)\b`,
					Type: chromalib.ByGroups(
						chromalib.Punctuation,
						chromalib.NameTag,
					),
					Mutator: chromalib.Push("script-tag"),
				},
				{
					Pattern: `(<)(template)\b`,
					Type: chromalib.ByGroups(
						chromalib.Punctuation,
						chromalib.NameTag,
					),
					Mutator: chromalib.Push("template-tag"),
				},
				{
					Pattern: `(<)(style)\b`,
					Type: chromalib.ByGroups(
						chromalib.Punctuation,
						chromalib.NameTag,
					),
					Mutator: chromalib.Push("style-tag"),
				},
				{
					Pattern: `(<)(i18n)\b`,
					Type: chromalib.ByGroups(
						chromalib.Punctuation,
						chromalib.NameTag,
					),
					Mutator: chromalib.Push("i18n-tag"),
				},
				{Pattern: `<!--[\s\S]*?-->`, Type: chromalib.Comment, Mutator: nil},
				{Pattern: `\s+`, Type: chromalib.Text, Mutator: nil},
				{Pattern: `.`, Type: chromalib.Text, Mutator: nil},
			},
			"script-tag": {
				{Pattern: `>`, Type: chromalib.Punctuation, Mutator: chromalib.Push("script-content-go")},
				{
					Pattern: `(type|lang|name|enable)(=)`,
					Type: chromalib.ByGroups(
						chromalib.NameAttribute,
						chromalib.Punctuation,
					),
					Mutator: chromalib.Push("sfc-attr-value"),
				},
				{
					Pattern: `([a-zA-Z_][a-zA-Z0-9_-]*)(=)`,
					Type: chromalib.ByGroups(
						chromalib.NameAttribute,
						chromalib.Punctuation,
					),
					Mutator: chromalib.Push("sfc-attr-value"),
				},
				{Pattern: `\s+`, Type: chromalib.Text, Mutator: nil},
			},

			"sfc-attr-value": {
				{Pattern: `"[^"]*"`, Type: chromalib.LiteralString, Mutator: chromalib.Pop(1)},
				{Pattern: `'[^']*'`, Type: chromalib.LiteralString, Mutator: chromalib.Pop(1)},
			},
			"template-tag": {
				{Pattern: `>`, Type: chromalib.Punctuation, Mutator: chromalib.Push("template-content")},
				{
					Pattern: `([a-zA-Z_][a-zA-Z0-9_-]*)(=)`,
					Type: chromalib.ByGroups(
						chromalib.NameAttribute,
						chromalib.Punctuation,
					),
					Mutator: chromalib.Push("sfc-attr-value"),
				},
				{Pattern: `\s+`, Type: chromalib.Text, Mutator: nil},
			},
			"style-tag": {
				{Pattern: `>`, Type: chromalib.Punctuation, Mutator: chromalib.Push("style-content")},
				{
					Pattern: `([a-zA-Z_][a-zA-Z0-9_-]*)(=)`,
					Type: chromalib.ByGroups(
						chromalib.NameAttribute,
						chromalib.Punctuation,
					),
					Mutator: chromalib.Push("sfc-attr-value"),
				},
				{Pattern: `\s+`, Type: chromalib.Text, Mutator: nil},
			},
			"i18n-tag": {
				{Pattern: `>`, Type: chromalib.Punctuation, Mutator: chromalib.Push("i18n-content")},
				{
					Pattern: `([a-zA-Z_][a-zA-Z0-9_-]*)(=)`,
					Type: chromalib.ByGroups(
						chromalib.NameAttribute,
						chromalib.Punctuation,
					),
					Mutator: chromalib.Push("sfc-attr-value"),
				},
				{Pattern: `\s+`, Type: chromalib.Text, Mutator: nil},
			},
			"script-content-go": {
				{
					Pattern: `(</)(script)(>)`,
					Type: chromalib.ByGroups(
						chromalib.Punctuation,
						chromalib.NameTag,
						chromalib.Punctuation,
					),
					Mutator: chromalib.Pop(2),
				},
				{Pattern: `[^<]+`, Type: chromalib.Using("go"), Mutator: nil},
				{Pattern: `<(?!/script)`, Type: chromalib.Using("go"), Mutator: nil},
			},
			"template-content": {
				{
					Pattern: `(</)(template)(>)`,
					Type: chromalib.ByGroups(
						chromalib.Punctuation,
						chromalib.NameTag,
						chromalib.Punctuation,
					),
					Mutator: chromalib.Pop(2),
				},
				{Pattern: `\{\{`, Type: chromalib.Punctuation, Mutator: chromalib.Push("interpolation")},
				{Pattern: `<!--[\s\S]*?-->`, Type: chromalib.Comment, Mutator: nil},
				{
					Pattern: `(<)([\w:-]+)`,
					Type: chromalib.ByGroups(
						chromalib.Punctuation,
						chromalib.NameTag,
					),
					Mutator: chromalib.Push("tag-attributes"),
				},
				{
					Pattern: `(</)([\w:-]+)(>)`,
					Type: chromalib.ByGroups(
						chromalib.Punctuation,
						chromalib.NameTag,
						chromalib.Punctuation,
					),
					Mutator: nil,
				},
				{Pattern: `[^<{]+`, Type: chromalib.Text, Mutator: nil},
				{Pattern: `.`, Type: chromalib.Text, Mutator: nil},
			},
			"tag-attributes": {
				{Pattern: `/>`, Type: chromalib.Punctuation, Mutator: chromalib.Pop(1)},
				{Pattern: `>`, Type: chromalib.Punctuation, Mutator: chromalib.Pop(1)},
				{
					Pattern: `(p-(?:if|else-if|else|for|show|class|style|text|html|bind|model|ref|key|context|format|scaffold|on|event|modal)(?::[a-zA-Z0-9-]+)?(?:\.[a-zA-Z0-9]+)*)(=)`,
					Type: chromalib.ByGroups(
						chromalib.NameDecorator,
						chromalib.Punctuation,
					),
					Mutator: chromalib.Push("attr-value"),
				},
				{Pattern: `p-(?:else|scaffold)\b`, Type: chromalib.NameDecorator, Mutator: nil},
				{
					Pattern: `(:[a-zA-Z_][a-zA-Z0-9_-]*)(=)`,
					Type: chromalib.ByGroups(
						chromalib.NameDecorator,
						chromalib.Punctuation,
					),
					Mutator: chromalib.Push("attr-value"),
				},
				{
					Pattern: `(@[a-zA-Z_][a-zA-Z0-9_-]*)(=)`,
					Type: chromalib.ByGroups(
						chromalib.NameDecorator,
						chromalib.Punctuation,
					),
					Mutator: chromalib.Push("attr-value"),
				},
				{
					Pattern: `([a-zA-Z_][a-zA-Z0-9_:-]*)(=)`,
					Type: chromalib.ByGroups(
						chromalib.NameAttribute,
						chromalib.Punctuation,
					),
					Mutator: chromalib.Push("attr-value"),
				},
				{Pattern: `[a-zA-Z_][a-zA-Z0-9_:-]*`, Type: chromalib.NameAttribute, Mutator: nil},
				{Pattern: `\s+`, Type: chromalib.Text, Mutator: nil},
			},
			"attr-value": {
				{Pattern: `"`, Type: chromalib.LiteralStringDouble, Mutator: chromalib.Push("attr-value-double")},
				{Pattern: `'`, Type: chromalib.LiteralStringSingle, Mutator: chromalib.Push("attr-value-single")},
			},

			"attr-value-double": {
				{Pattern: `"`, Type: chromalib.LiteralStringDouble, Mutator: chromalib.Pop(2)},
				chromalib.Include("piko-expression"),
			},

			"attr-value-single": {
				{Pattern: `'`, Type: chromalib.LiteralStringSingle, Mutator: chromalib.Pop(2)},
				chromalib.Include("piko-expression"),
			},
			"interpolation": {
				{Pattern: `\}\}`, Type: chromalib.Punctuation, Mutator: chromalib.Pop(1)},
				chromalib.Include("piko-expression"),
			},
			"piko-expression": {
				{Pattern: `\b(true|false|nil|null|in)\b`, Type: chromalib.KeywordConstant, Mutator: nil},
				{Pattern: `dt'[^']*'`, Type: chromalib.LiteralDate, Mutator: nil},
				{Pattern: `d'[^']*'`, Type: chromalib.LiteralDate, Mutator: nil},
				{Pattern: `t'[^']*'`, Type: chromalib.LiteralDate, Mutator: nil},
				{Pattern: `du'[^']*'`, Type: chromalib.LiteralDate, Mutator: nil},
				{Pattern: `r'[^']*'`, Type: chromalib.LiteralStringChar, Mutator: nil},
				{Pattern: `\d+\.?\d*[dn]`, Type: chromalib.LiteralNumber, Mutator: nil},
				{Pattern: `\d+\.\d+`, Type: chromalib.LiteralNumberFloat, Mutator: nil},
				{Pattern: `\d+`, Type: chromalib.LiteralNumberInteger, Mutator: nil},
				{Pattern: "`", Type: chromalib.LiteralStringBacktick, Mutator: chromalib.Push("template-literal")},
				{Pattern: `"(?:[^"\\]|\\.)*"`, Type: chromalib.LiteralStringDouble, Mutator: nil},
				{Pattern: `'(?:[^'\\]|\\.)*'`, Type: chromalib.LiteralStringSingle, Mutator: nil},
				{Pattern: `!~=|~=|==|!=|<=|>=|&&|\|\||\?\?|\?\.`, Type: chromalib.Operator, Mutator: nil},
				{Pattern: `[+\-*/%<>!?:~]`, Type: chromalib.Operator, Mutator: nil},
				{Pattern: `@[a-zA-Z_][a-zA-Z0-9_.]*`, Type: chromalib.NameVariable, Mutator: nil},
				{Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`, Type: chromalib.NameVariable, Mutator: nil},
				{Pattern: `[()[\]{},.]`, Type: chromalib.Punctuation, Mutator: nil},
				{Pattern: `\s+`, Type: chromalib.Text, Mutator: nil},
			},
			"template-literal": {
				{Pattern: "`", Type: chromalib.LiteralStringBacktick, Mutator: chromalib.Pop(1)},
				{Pattern: `\$\{`, Type: chromalib.Punctuation, Mutator: chromalib.Push("template-literal-expr")},
				{Pattern: "[^`$]+", Type: chromalib.LiteralStringBacktick, Mutator: nil},
				{Pattern: `\$`, Type: chromalib.LiteralStringBacktick, Mutator: nil},
			},

			"template-literal-expr": {
				{Pattern: `\}`, Type: chromalib.Punctuation, Mutator: chromalib.Pop(1)},
				chromalib.Include("piko-expression"),
			},
			"style-content": {
				{
					Pattern: `(</)(style)(>)`,
					Type: chromalib.ByGroups(
						chromalib.Punctuation,
						chromalib.NameTag,
						chromalib.Punctuation,
					),
					Mutator: chromalib.Pop(2),
				},
				{Pattern: `[^<]+`, Type: chromalib.Using("css"), Mutator: nil},
				{Pattern: `<(?!/style)`, Type: chromalib.Using("css"), Mutator: nil},
			},
			"i18n-content": {
				{
					Pattern: `(</)(i18n)(>)`,
					Type: chromalib.ByGroups(
						chromalib.Punctuation,
						chromalib.NameTag,
						chromalib.Punctuation,
					),
					Mutator: chromalib.Pop(2),
				},
				{Pattern: `[^<]+`, Type: chromalib.Using("json"), Mutator: nil},
				{Pattern: `<(?!/i18n)`, Type: chromalib.Using("json"), Mutator: nil},
			},
		}
	},
)

func init() {
	lexers.Register(PikoLexer)

	_, l := logger.From(context.Background(), log)
	l.Internal("Registered Piko lexer with Chroma")
}
