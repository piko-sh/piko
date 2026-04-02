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
	"testing"

	chromalib "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPikoLexerRegistration(t *testing.T) {
	t.Run("registered by name", func(t *testing.T) {
		lexer := lexers.Get("piko")
		require.NotNil(t, lexer, "Piko lexer should be registered")
		assert.Equal(t, "Piko", lexer.Config().Name)
	})

	t.Run("registered by pk alias", func(t *testing.T) {
		lexer := lexers.Get("pk")
		require.NotNil(t, lexer, "pk alias should be registered")
	})

	t.Run("registered by pkc alias", func(t *testing.T) {
		lexer := lexers.Get("pkc")
		require.NotNil(t, lexer, "pkc alias should be registered")
	})
}

func TestPikoLexerFilenameMatch(t *testing.T) {
	t.Run("matches .pk files", func(t *testing.T) {
		lexer := lexers.Match("test.pk")
		require.NotNil(t, lexer)
		assert.Contains(t, lexer.Config().Aliases, "piko")
	})

	t.Run("matches .pkc files", func(t *testing.T) {
		lexer := lexers.Match("component.pkc")
		require.NotNil(t, lexer)
		assert.Contains(t, lexer.Config().Aliases, "piko")
	})
}

func TestPikoLexerTemplateBlock(t *testing.T) {
	input := `<template>
  <div p-if="visible">{{ message }}</div>
</template>`

	tokens := tokenise(t, input)

	t.Run("recognises template tag", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameTag, "template"),
			"should recognise template tag")
	})

	t.Run("recognises p-if directive", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameDecorator, "p-if"),
			"should recognise p-if directive")
	})

	t.Run("recognises interpolation delimiters", func(t *testing.T) {
		assert.True(t, hasTokenValue(tokens, "{{"),
			"should recognise opening interpolation")
		assert.True(t, hasTokenValue(tokens, "}}"),
			"should recognise closing interpolation")
	})

	t.Run("recognises variable in interpolation", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameVariable, "message"),
			"should recognise variable in interpolation")
	})
}

func TestPikoLexerDirectives(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		directive string
	}{
		{
			name:      "p-if directive",
			input:     `<template><div p-if="cond"></div></template>`,
			directive: "p-if",
		},
		{
			name:      "p-for directive",
			input:     `<template><div p-for="item in items"></div></template>`,
			directive: "p-for",
		},
		{
			name:      "p-else directive",
			input:     `<template><div p-else></div></template>`,
			directive: "p-else",
		},
		{
			name:      "p-show directive",
			input:     `<template><div p-show="visible"></div></template>`,
			directive: "p-show",
		},
		{
			name:      "p-bind directive",
			input:     `<template><div p-bind:class="cls"></div></template>`,
			directive: "p-bind:class",
		},
		{
			name:      "p-on directive",
			input:     `<template><button p-on:click="handle"></button></template>`,
			directive: "p-on:click",
		},
		{
			name:      "p-model directive",
			input:     `<template><input p-model="value"></template>`,
			directive: "p-model",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tokens := tokenise(t, tc.input)
			assert.True(t, hasToken(tokens, chromalib.NameDecorator, tc.directive),
				"should recognise %s directive", tc.directive)
		})
	}
}

func TestPikoLexerShorthandSyntax(t *testing.T) {
	t.Run("shorthand binding :attr", func(t *testing.T) {
		input := `<template><img :src="imagePath"></template>`
		tokens := tokenise(t, input)
		assert.True(t, hasToken(tokens, chromalib.NameDecorator, ":src"),
			"should recognise :src binding shorthand")
	})

	t.Run("shorthand event @event", func(t *testing.T) {
		input := `<template><button @click="handleClick"></button></template>`
		tokens := tokenise(t, input)
		assert.True(t, hasToken(tokens, chromalib.NameDecorator, "@click"),
			"should recognise @click event shorthand")
	})
}

func TestPikoLexerOperators(t *testing.T) {
	input := `<template><div p-if="a && b || c?.d ?? e ~= f !~= g"></div></template>`
	tokens := tokenise(t, input)

	operators := []string{"&&", "||", "?.", "??", "~=", "!~="}
	for _, op := range operators {
		t.Run(op, func(t *testing.T) {
			assert.True(t, hasToken(tokens, chromalib.Operator, op),
				"should recognise %s operator", op)
		})
	}
}

func TestPikoLexerPrefixedLiterals(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		literal string
	}{
		{
			name:    "date literal",
			input:   `<template><span p-text="d'2024-01-15'"></span></template>`,
			literal: "d'2024-01-15'",
		},
		{
			name:    "time literal",
			input:   `<template><span p-text="t'14:30:00'"></span></template>`,
			literal: "t'14:30:00'",
		},
		{
			name:    "datetime literal",
			input:   `<template><span p-text="dt'2024-01-15T14:30:00Z'"></span></template>`,
			literal: "dt'2024-01-15T14:30:00Z'",
		},
		{
			name:    "duration literal",
			input:   `<template><span p-text="du'5m30s'"></span></template>`,
			literal: "du'5m30s'",
		},
		{
			name:    "rune literal",
			input:   `<template><span p-text="r'A'"></span></template>`,
			literal: "r'A'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tokens := tokenise(t, tc.input)
			assert.True(t, hasTokenValue(tokens, tc.literal),
				"should recognise %s", tc.literal)
		})
	}
}

func TestPikoLexerKeywords(t *testing.T) {
	input := `<template><div p-if="true && false || nil ?? null"></div></template>`
	tokens := tokenise(t, input)

	keywords := []string{"true", "false", "nil", "null"}
	for _, kw := range keywords {
		t.Run(kw, func(t *testing.T) {
			assert.True(t, hasToken(tokens, chromalib.KeywordConstant, kw),
				"should recognise %s keyword", kw)
		})
	}
}

func TestPikoLexerNumbers(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		value     string
		tokenType chromalib.TokenType
	}{
		{
			name:      "integer",
			input:     `<template><span p-text="123"></span></template>`,
			value:     "123",
			tokenType: chromalib.LiteralNumberInteger,
		},
		{
			name:      "float",
			input:     `<template><span p-text="123.45"></span></template>`,
			value:     "123.45",
			tokenType: chromalib.LiteralNumberFloat,
		},
		{
			name:      "decimal suffix",
			input:     `<template><span p-text="123.45d"></span></template>`,
			value:     "123.45d",
			tokenType: chromalib.LiteralNumber,
		},
		{
			name:      "bigint suffix",
			input:     `<template><span p-text="123n"></span></template>`,
			value:     "123n",
			tokenType: chromalib.LiteralNumber,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tokens := tokenise(t, tc.input)
			assert.True(t, hasToken(tokens, tc.tokenType, tc.value),
				"should recognise %s as %s", tc.value, tc.tokenType)
		})
	}
}

func TestPikoLexerScriptBlock(t *testing.T) {
	input := `<script type="application/x-go">
type Response struct {
    Title string
}
</script>`

	tokens := tokenise(t, input)

	t.Run("recognises script tag", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameTag, "script"),
			"should recognise script tag")
	})

	t.Run("recognises Go keywords", func(t *testing.T) {
		assert.True(t, hasTokenValue(tokens, "type") || hasTokenValue(tokens, "struct"),
			"should contain Go code tokens")
	})
}

func TestPikoLexerStyleBlock(t *testing.T) {
	input := `<style>
.container {
    color: red;
}
</style>`

	tokens := tokenise(t, input)

	t.Run("recognises style tag", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameTag, "style"),
			"should recognise style tag")
	})
}

func TestPikoLexerI18nBlock(t *testing.T) {
	input := `<i18n>
{
    "greeting": "Hello"
}
</i18n>`

	tokens := tokenise(t, input)

	t.Run("recognises i18n tag", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameTag, "i18n"),
			"should recognise i18n tag")
	})
}

func TestPikoLexerHTMLComments(t *testing.T) {
	input := `<template>
  <!-- This is a comment -->
  <div>Content</div>
</template>`

	tokens := tokenise(t, input)

	t.Run("recognises HTML comment", func(t *testing.T) {
		found := false
		for _, token := range tokens {
			if token.Type == chromalib.Comment {
				found = true
				break
			}
		}
		assert.True(t, found, "should recognise HTML comment")
	})
}

func TestPikoLexerForLoopExpression(t *testing.T) {
	input := `<template><div p-for="(index, item) in items"></div></template>`
	tokens := tokenise(t, input)

	t.Run("recognises in keyword", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.KeywordConstant, "in"),
			"should recognise 'in' keyword in for loop")
	})

	t.Run("recognises loop variables", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameVariable, "index"),
			"should recognise index variable")
		assert.True(t, hasToken(tokens, chromalib.NameVariable, "item"),
			"should recognise item variable")
		assert.True(t, hasToken(tokens, chromalib.NameVariable, "items"),
			"should recognise items variable")
	})
}

func TestPikoLexerScriptWithLangAttribute(t *testing.T) {
	input := `<template name="pp-button" enable="form"></template>
<script lang="ts">
const state = { value: '' as string };
</script>`

	tokens := tokenise(t, input)

	t.Run("recognises script tag", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameTag, "script"),
			"should recognise script tag")
	})

	t.Run("recognises lang attribute", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameAttribute, "lang"),
			"should recognise lang attribute")
	})

	t.Run("recognises name attribute", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameAttribute, "name"),
			"should recognise name attribute")
	})

	t.Run("recognises enable attribute", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameAttribute, "enable"),
			"should recognise enable attribute")
	})
}

func TestPikoLexerStyleWithAesthetic(t *testing.T) {
	input := `<style aesthetic="dark">
:host { color: white; }
</style>`

	tokens := tokenise(t, input)

	t.Run("recognises style tag", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameTag, "style"),
			"should recognise style tag")
	})

	t.Run("recognises aesthetic attribute", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameAttribute, "aesthetic"),
			"should recognise aesthetic attribute")
	})
}

func TestPikoLexerEventModifiers(t *testing.T) {
	input := `<template><button p-on:click.prevent="handleClick"></button></template>`
	tokens := tokenise(t, input)

	t.Run("recognises p-on directive with modifier", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameDecorator, "p-on:click.prevent"),
			"should recognise p-on:click.prevent directive")
	})
}

func TestPikoLexerModalDirective(t *testing.T) {
	input := `<template><pp-button p-modal:selector="[modal=quote]" p-modal:title="Create quote"></pp-button></template>`
	tokens := tokenise(t, input)

	t.Run("recognises p-modal:selector", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameDecorator, "p-modal:selector"),
			"should recognise p-modal:selector directive")
	})

	t.Run("recognises p-modal:title", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameDecorator, "p-modal:title"),
			"should recognise p-modal:title directive")
	})
}

func TestPikoLexerUnderscoreRefAttribute(t *testing.T) {
	input := `<template><div _ref="container"></div></template>`
	tokens := tokenise(t, input)

	t.Run("recognises _ref attribute", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameAttribute, "_ref"),
			"should recognise _ref attribute")
	})
}

func TestPikoLexerSlotElement(t *testing.T) {
	input := `<template><slot name="actions"></slot></template>`
	tokens := tokenise(t, input)

	t.Run("recognises slot tag", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameTag, "slot"),
			"should recognise slot element")
	})

	t.Run("recognises name attribute on slot", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameAttribute, "name"),
			"should recognise name attribute on slot")
	})
}

func TestPikoLexerPEventDirective(t *testing.T) {
	input := `<template><slot p-event:pp-button-click-cancel="onCancel"></slot></template>`
	tokens := tokenise(t, input)

	t.Run("recognises p-event directive", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameDecorator, "p-event:pp-button-click-cancel"),
			"should recognise p-event directive with event name")
	})
}

func TestPikoLexerPEventDirectiveWithModifier(t *testing.T) {
	input := `<template><div p-event:modal-confirmed.once="handleConfirm()"></div></template>`
	tokens := tokenise(t, input)

	t.Run("recognises p-event with modifier", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameDecorator, "p-event:modal-confirmed.once"),
			"should recognise p-event:modal-confirmed.once directive")
	})
}

func TestPikoLexerRealWorldComponent(t *testing.T) {
	input := `<template>
  <button
    class="button-base"
    :disabled="state.disabled || state.loading"
    p-on:click="handleButtonClick"
  >
  <span class="content-wrapper">
    <span class="spinner" p-if="state.loading"></span>
    <span class="content" p-class="{ hidden: state.loading }">
      <slot></slot>
    </span>
  </span>
</button>
</template>

<script lang="ts" name="pp-button">
const state = {
    variant: 'primary' as string,
    disabled: false as boolean,
};
</script>

<style>
:host {
  display: inline-block;
}
</style>

<style aesthetic="dark">
:host {
  --pp-button-color: white;
}
</style>`

	tokens := tokenise(t, input)

	t.Run("handles template block", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameTag, "template"))
		assert.True(t, hasToken(tokens, chromalib.NameTag, "button"))
		assert.True(t, hasToken(tokens, chromalib.NameTag, "span"))
		assert.True(t, hasToken(tokens, chromalib.NameTag, "slot"))
	})

	t.Run("handles directives", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameDecorator, ":disabled"))
		assert.True(t, hasToken(tokens, chromalib.NameDecorator, "p-on:click"))
		assert.True(t, hasToken(tokens, chromalib.NameDecorator, "p-if"))
		assert.True(t, hasToken(tokens, chromalib.NameDecorator, "p-class"))
	})

	t.Run("handles expression operators", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.Operator, "||"))
	})

	t.Run("handles script attributes", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameTag, "script"))
		assert.True(t, hasToken(tokens, chromalib.NameAttribute, "lang"))
		assert.True(t, hasToken(tokens, chromalib.NameAttribute, "name"))
	})

	t.Run("handles style with aesthetic", func(t *testing.T) {
		assert.True(t, hasToken(tokens, chromalib.NameTag, "style"))
		assert.True(t, hasToken(tokens, chromalib.NameAttribute, "aesthetic"))
	})
}

func tokenise(t *testing.T, input string) []chromalib.Token {
	t.Helper()
	lexer := lexers.Get("piko")
	require.NotNil(t, lexer, "Piko lexer should be available")

	iter, err := lexer.Tokenise(nil, input)
	require.NoError(t, err, "Tokenisation should not error")

	var tokens []chromalib.Token
	for tok := iter(); tok != chromalib.EOF; tok = iter() {
		tokens = append(tokens, tok)
	}
	return tokens
}

func hasToken(tokens []chromalib.Token, tokenType chromalib.TokenType, value string) bool {
	for _, token := range tokens {
		if token.Type == tokenType && token.Value == value {
			return true
		}
	}
	return false
}

func hasTokenValue(tokens []chromalib.Token, value string) bool {
	for _, token := range tokens {
		if token.Value == value {
			return true
		}
	}
	return false
}
