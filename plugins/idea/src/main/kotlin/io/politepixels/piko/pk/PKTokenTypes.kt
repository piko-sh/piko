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


package io.politepixels.piko.pk

import com.intellij.psi.tree.IElementType
import com.intellij.psi.tree.TokenSet

/**
 * Token and element types for PK files.
 *
 * Defines all lexer tokens and PSI element types used when parsing Piko templates.
 * These types enable syntax highlighting for template content, directives,
 * expressions, and embedded language blocks.
 */
object PKTokenTypes {

    /** Opening tag for template blocks (`<template`). */
    @JvmField val TEMPLATE_TAG_START = PKElementType("PK_TEMPLATE_TAG_START")

    /** Opening tag for script blocks (`<script`). */
    @JvmField val SCRIPT_TAG_START = PKElementType("PK_SCRIPT_TAG_START")

    /** Opening tag for style blocks (`<style`). */
    @JvmField val STYLE_TAG_START = PKElementType("PK_STYLE_TAG_START")

    /** Opening tag for i18n blocks (`<i18n`). */
    @JvmField val I18N_TAG_START = PKElementType("PK_I18N_TAG_START")

    /** Closing angle bracket for structural tags (`>`). */
    @JvmField val TAG_END_GT = PKElementType("PK_TAG_END_GT")

    /** Closing tag for template blocks (`</template>`). */
    @JvmField val TEMPLATE_TAG_END = PKElementType("PK_TEMPLATE_TAG_END")

    /** Closing tag for script blocks (`</script>`). */
    @JvmField val SCRIPT_TAG_END = PKElementType("PK_SCRIPT_TAG_END")

    /** Closing tag for style blocks (`</style>`). */
    @JvmField val STYLE_TAG_END = PKElementType("PK_STYLE_TAG_END")

    /** Closing tag for i18n blocks (`</i18n>`). */
    @JvmField val I18N_TAG_END = PKElementType("PK_I18N_TAG_END")

    /** Attribute name on structural tags (e.g. `lang` in `<script lang="ts">`). */
    @JvmField val ATTR_NAME = PKElementType("PK_ATTR_NAME")

    /** Equals sign between attribute name and value on structural tags. */
    @JvmField val ATTR_EQ = PKElementType("PK_ATTR_EQ")

    /** Attribute value on structural tags (e.g. `"ts"` in `lang="ts"`). */
    @JvmField val ATTR_VALUE = PKElementType("PK_ATTR_VALUE")

    /** Raw Go source content inside a script block. */
    @JvmField val GO_SCRIPT_CONTENT = PKElementType("PK_GO_SCRIPT_CONTENT")

    /** Raw JavaScript/TypeScript source content inside a script block. */
    @JvmField val JS_SCRIPT_CONTENT = PKElementType("PK_JS_SCRIPT_CONTENT")

    /** Raw CSS source content inside a style block. */
    @JvmField val CSS_STYLE_CONTENT = PKElementType("PK_CSS_STYLE_CONTENT")

    /** Raw JSON content inside an i18n block. */
    @JvmField val I18N_CONTENT = PKElementType("PK_I18N_CONTENT")

    /** Opening angle bracket for HTML tags (`<`). */
    @JvmField val HTML_TAG_OPEN = PKElementType("PK_HTML_TAG_OPEN")

    /** Closing angle bracket for HTML tags (`>`). */
    @JvmField val HTML_TAG_CLOSE = PKElementType("PK_HTML_TAG_CLOSE")

    /** Self-closing slash for HTML tags (`/>`). */
    @JvmField val HTML_TAG_SELF_CLOSE = PKElementType("PK_HTML_TAG_SELF_CLOSE")

    /** Opening bracket for HTML end tags (`</`). */
    @JvmField val HTML_END_TAG_OPEN = PKElementType("PK_HTML_END_TAG_OPEN")

    /** HTML tag name (e.g. `div`, `span`, `button`). */
    @JvmField val HTML_TAG_NAME = PKElementType("PK_HTML_TAG_NAME")

    /** Piko component tag name (e.g. `piko:card`). */
    @JvmField val PIKO_TAG_NAME = PKElementType("PK_PIKO_TAG_NAME")

    /** Attribute name on HTML elements (e.g. `class`, `id`). */
    @JvmField val HTML_ATTR_NAME = PKElementType("PK_HTML_ATTR_NAME")

    /** Equals sign between HTML attribute name and value. */
    @JvmField val HTML_ATTR_EQ = PKElementType("PK_HTML_ATTR_EQ")

    /** Attribute value on HTML elements (e.g. `"container"`). */
    @JvmField val HTML_ATTR_VALUE = PKElementType("PK_HTML_ATTR_VALUE")

    /** Quote character surrounding HTML attribute values. */
    @JvmField val HTML_ATTR_QUOTE = PKElementType("PK_HTML_ATTR_QUOTE")

    /** Directive name (e.g. `p-if`, `p-for`, `p-on`). */
    @JvmField val DIRECTIVE_NAME = PKElementType("PK_DIRECTIVE_NAME")

    /** Bind shorthand suffix (e.g. `:title` in `p-bind:title`). */
    @JvmField val DIRECTIVE_BIND = PKElementType("PK_DIRECTIVE_BIND")

    /** Event shorthand (e.g. `@click`). */
    @JvmField val DIRECTIVE_EVENT = PKElementType("PK_DIRECTIVE_EVENT")

    /** Opening interpolation delimiter (`{{`). */
    @JvmField val INTERPOLATION_OPEN = PKElementType("PK_INTERPOLATION_OPEN")

    /** Closing interpolation delimiter (`}}`). */
    @JvmField val INTERPOLATION_CLOSE = PKElementType("PK_INTERPOLATION_CLOSE")

    /** Boolean literal (`true`, `false`, `nil`). */
    @JvmField val EXPR_BOOLEAN = PKElementType("PK_EXPR_BOOLEAN")

    /** Numeric literal. */
    @JvmField val EXPR_NUMBER = PKElementType("PK_EXPR_NUMBER")

    /** String literal content (excluding quotes). */
    @JvmField val EXPR_STRING = PKElementType("PK_EXPR_STRING")

    /** String quote character (single or double). */
    @JvmField val EXPR_STRING_QUOTE = PKElementType("PK_EXPR_STRING_QUOTE")

    /** Escape sequence inside a string literal. */
    @JvmField val EXPR_ESCAPE = PKElementType("PK_EXPR_ESCAPE")

    /** Comparison operator (`==`, `!=`, `<`, `>`, `<=`, `>=`). */
    @JvmField val EXPR_OP_COMPARISON = PKElementType("PK_EXPR_OP_COMPARISON")

    /** Logical operator (`&&`, `||`, `!`). */
    @JvmField val EXPR_OP_LOGICAL = PKElementType("PK_EXPR_OP_LOGICAL")

    /** Arithmetic operator (`+`, `-`, `*`, `/`, `%`). */
    @JvmField val EXPR_OP_ARITHMETIC = PKElementType("PK_EXPR_OP_ARITHMETIC")

    /** Dot accessor operator (`.`). */
    @JvmField val EXPR_OP_DOT = PKElementType("PK_EXPR_OP_DOT")

    /** Opening parenthesis in expressions (`(`). */
    @JvmField val EXPR_PAREN_OPEN = PKElementType("PK_EXPR_PAREN_OPEN")

    /** Closing parenthesis in expressions (`)`). */
    @JvmField val EXPR_PAREN_CLOSE = PKElementType("PK_EXPR_PAREN_CLOSE")

    /** Opening square bracket in expressions (`[`). */
    @JvmField val EXPR_BRACKET_OPEN = PKElementType("PK_EXPR_BRACKET_OPEN")

    /** Closing square bracket in expressions (`]`). */
    @JvmField val EXPR_BRACKET_CLOSE = PKElementType("PK_EXPR_BRACKET_CLOSE")

    /** Opening curly brace in expressions (`{`). */
    @JvmField val EXPR_BRACE_OPEN = PKElementType("PK_EXPR_BRACE_OPEN")

    /** Closing curly brace in expressions (`}`). */
    @JvmField val EXPR_BRACE_CLOSE = PKElementType("PK_EXPR_BRACE_CLOSE")

    /** Comma separator in expressions. */
    @JvmField val EXPR_COMMA = PKElementType("PK_EXPR_COMMA")

    /** Colon separator in expressions. */
    @JvmField val EXPR_COLON = PKElementType("PK_EXPR_COLON")

    /** Built-in function name (e.g. `len`, `cap`, `make`). */
    @JvmField val EXPR_BUILTIN = PKElementType("PK_EXPR_BUILTIN")

    /** Context variable (e.g. `props`, `state`, `partial`). */
    @JvmField val EXPR_CONTEXT_VAR = PKElementType("PK_EXPR_CONTEXT_VAR")

    /** Function name in a call expression. */
    @JvmField val EXPR_FUNCTION_NAME = PKElementType("PK_EXPR_FUNCTION_NAME")

    /** General identifier in expressions. */
    @JvmField val EXPR_IDENTIFIER = PKElementType("PK_EXPR_IDENTIFIER")

    /** Opening template literal interpolation marker (`${`). */
    @JvmField val TEMPLATE_INTERP_OPEN = PKElementType("PK_TEMPLATE_INTERP_OPEN")

    /** Closing template literal interpolation marker (`}`). */
    @JvmField val TEMPLATE_INTERP_CLOSE = PKElementType("PK_TEMPLATE_INTERP_CLOSE")

    /** Plain text content between HTML tags. */
    @JvmField val TEXT_CONTENT = PKElementType("PK_TEXT_CONTENT")

    /** HTML comment (`<!-- ... -->`). */
    @JvmField val HTML_COMMENT = PKElementType("PK_HTML_COMMENT")

    /** PSI element wrapping an entire template block. */
    @JvmField val TEMPLATE_BLOCK_ELEMENT = PKElementType("PK_TEMPLATE_BLOCK")

    /** PSI element wrapping an entire script block. */
    @JvmField val SCRIPT_BLOCK_ELEMENT = PKElementType("PK_SCRIPT_BLOCK")

    /** PSI element wrapping an entire style block. */
    @JvmField val STYLE_BLOCK_ELEMENT = PKElementType("PK_STYLE_BLOCK")

    /** PSI element wrapping an entire i18n block. */
    @JvmField val I18N_BLOCK_ELEMENT = PKElementType("PK_I18N_BLOCK")

    /** PSI element for template body content (injection host). */
    @JvmField val TEMPLATE_BODY_ELEMENT = PKElementType("PK_TEMPLATE_BODY")

    /** PSI element for Go script body content (injection host). */
    @JvmField val GO_SCRIPT_BODY_ELEMENT = PKElementType("PK_GO_SCRIPT_BODY")

    /** PSI element for JavaScript script body content (injection host). */
    @JvmField val JS_SCRIPT_BODY_ELEMENT = PKElementType("PK_JS_SCRIPT_BODY")

    /** PSI element for CSS style body content (injection host). */
    @JvmField val CSS_STYLE_BODY_ELEMENT = PKElementType("PK_CSS_STYLE_BODY")

    /** PSI element for i18n body content (injection host). */
    @JvmField val I18N_BODY_ELEMENT = PKElementType("PK_I18N_BODY")

    /** Token set containing all comment types for brace matching and folding. */
    val COMMENTS: TokenSet = TokenSet.create(HTML_COMMENT)

    /** Token set containing all string literal types for syntax highlighting. */
    val STRING_LITERALS: TokenSet = TokenSet.create(
        ATTR_VALUE,
        HTML_ATTR_VALUE,
        EXPR_STRING
    )

    /** Token set containing all keyword types for syntax highlighting. */
    val KEYWORDS: TokenSet = TokenSet.create(
        TEMPLATE_TAG_START,
        TEMPLATE_TAG_END,
        SCRIPT_TAG_START,
        SCRIPT_TAG_END,
        STYLE_TAG_START,
        STYLE_TAG_END,
        I18N_TAG_START,
        I18N_TAG_END,
        HTML_TAG_NAME,
        DIRECTIVE_NAME,
        DIRECTIVE_BIND,
        DIRECTIVE_EVENT,
        EXPR_BOOLEAN
    )

    /** Token set containing all bracket types for brace matching. */
    val BRACKETS: TokenSet = TokenSet.create(
        TAG_END_GT,
        HTML_TAG_OPEN,
        HTML_TAG_CLOSE,
        HTML_TAG_SELF_CLOSE,
        HTML_END_TAG_OPEN,
        INTERPOLATION_OPEN,
        INTERPOLATION_CLOSE,
        EXPR_PAREN_OPEN,
        EXPR_PAREN_CLOSE,
        EXPR_BRACKET_OPEN,
        EXPR_BRACKET_CLOSE,
        EXPR_BRACE_OPEN,
        EXPR_BRACE_CLOSE
    )

    /** Token set containing all operator types for syntax highlighting. */
    val OPERATORS: TokenSet = TokenSet.create(
        EXPR_OP_COMPARISON,
        EXPR_OP_LOGICAL,
        EXPR_OP_ARITHMETIC,
        EXPR_OP_DOT
    )
}

/**
 * Custom element type for PK language tokens.
 *
 * Each token type is registered with the PK language to ensure proper
 * association during lexing and parsing.
 *
 * @param debugName The identifier shown in PSI tree viewers and debug output.
 */
class PKElementType(debugName: String) : IElementType(debugName, PKLanguage)
