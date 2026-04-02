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

import com.intellij.lexer.Lexer
import com.intellij.openapi.editor.DefaultLanguageHighlighterColors
import com.intellij.openapi.editor.HighlighterColors
import com.intellij.openapi.editor.colors.TextAttributesKey
import com.intellij.openapi.editor.colors.TextAttributesKey.createTextAttributesKey
import com.intellij.openapi.fileTypes.SyntaxHighlighterBase
import com.intellij.psi.tree.IElementType

/**
 * Provides syntax highlighting for PK template files.
 *
 * Maps token types from the lexer to text attribute keys that define colours
 * and styles. Covers structural tags, HTML elements, directives, interpolation
 * expressions, and embedded content blocks.
 */
class PKSyntaxHighlighter : SyntaxHighlighterBase() {

    companion object {
        /** Text attributes for template, script, style, and i18n tags. */
        val PK_STRUCTURAL_TAG = createTextAttributesKey("PK_STRUCTURAL_TAG", DefaultLanguageHighlighterColors.KEYWORD)

        /** Text attributes for angle brackets in structural tags. */
        val PK_TAG_BRACKET = createTextAttributesKey("PK_TAG_BRACKET", DefaultLanguageHighlighterColors.BRACKETS)

        /** Text attributes for HTML tag names like div, span, etc. */
        val PK_HTML_TAG_NAME = createTextAttributesKey("PK_HTML_TAG_NAME", DefaultLanguageHighlighterColors.KEYWORD)

        /** Text attributes for angle brackets in HTML tags. */
        val PK_HTML_TAG_BRACKET = createTextAttributesKey("PK_HTML_TAG_BRACKET", DefaultLanguageHighlighterColors.BRACKETS)

        /** Text attributes for Piko component tags like piko:card. */
        val PK_PIKO_COMPONENT = createTextAttributesKey("PK_PIKO_COMPONENT", DefaultLanguageHighlighterColors.STATIC_METHOD)

        /** Text attributes for attribute names. */
        val PK_ATTRIBUTE = createTextAttributesKey("PK_ATTRIBUTE", DefaultLanguageHighlighterColors.METADATA)

        /** Text attributes for attribute values. */
        val PK_ATTRIBUTE_VALUE = createTextAttributesKey("PK_ATTRIBUTE_VALUE", DefaultLanguageHighlighterColors.STRING)

        /** Text attributes for attribute quote characters. */
        val PK_ATTRIBUTE_QUOTE = createTextAttributesKey("PK_ATTRIBUTE_QUOTE", DefaultLanguageHighlighterColors.STRING)

        /** Text attributes for directive names like p-if, p-for. */
        val PK_DIRECTIVE = createTextAttributesKey("PK_DIRECTIVE", DefaultLanguageHighlighterColors.METADATA)

        /** Text attributes for bind shorthand :attr syntax. */
        val PK_DIRECTIVE_BIND = createTextAttributesKey("PK_DIRECTIVE_BIND", DefaultLanguageHighlighterColors.METADATA)

        /** Text attributes for event shorthand @event syntax. */
        val PK_DIRECTIVE_EVENT = createTextAttributesKey("PK_DIRECTIVE_EVENT", DefaultLanguageHighlighterColors.METADATA)

        /** Text attributes for interpolation brackets {{ and }}. */
        val PK_INTERPOLATION_BRACKET = createTextAttributesKey("PK_INTERPOLATION_BRACKET", DefaultLanguageHighlighterColors.TEMPLATE_LANGUAGE_COLOR)

        /** Text attributes for boolean literals true, false, nil. */
        val PK_EXPR_BOOLEAN = createTextAttributesKey("PK_EXPR_BOOLEAN", DefaultLanguageHighlighterColors.KEYWORD)

        /** Text attributes for number literals. */
        val PK_EXPR_NUMBER = createTextAttributesKey("PK_EXPR_NUMBER", DefaultLanguageHighlighterColors.NUMBER)

        /** Text attributes for string literal content. */
        val PK_EXPR_STRING = createTextAttributesKey("PK_EXPR_STRING", DefaultLanguageHighlighterColors.STRING)

        /** Text attributes for string quote characters. */
        val PK_EXPR_STRING_QUOTE = createTextAttributesKey("PK_EXPR_STRING_QUOTE", DefaultLanguageHighlighterColors.STRING)

        /** Text attributes for escape sequences in strings. */
        val PK_EXPR_ESCAPE = createTextAttributesKey("PK_EXPR_ESCAPE", DefaultLanguageHighlighterColors.VALID_STRING_ESCAPE)

        /** Text attributes for comparison, logical, and arithmetic operators. */
        val PK_EXPR_OPERATOR = createTextAttributesKey("PK_EXPR_OPERATOR", DefaultLanguageHighlighterColors.KEYWORD)

        /** Text attributes for dot accessor. */
        val PK_EXPR_DOT = createTextAttributesKey("PK_EXPR_DOT", DefaultLanguageHighlighterColors.DOT)

        /** Text attributes for parentheses in expressions. */
        val PK_EXPR_PAREN = createTextAttributesKey("PK_EXPR_PAREN", DefaultLanguageHighlighterColors.PARENTHESES)

        /** Text attributes for square brackets in expressions. */
        val PK_EXPR_BRACKET = createTextAttributesKey("PK_EXPR_BRACKET", DefaultLanguageHighlighterColors.BRACKETS)

        /** Text attributes for curly braces in expressions. */
        val PK_EXPR_BRACE = createTextAttributesKey("PK_EXPR_BRACE", DefaultLanguageHighlighterColors.BRACES)

        /** Text attributes for comma separator. */
        val PK_EXPR_COMMA = createTextAttributesKey("PK_EXPR_COMMA", DefaultLanguageHighlighterColors.COMMA)

        /** Text attributes for colon separator. */
        val PK_EXPR_COLON = createTextAttributesKey("PK_EXPR_COLON", DefaultLanguageHighlighterColors.OPERATION_SIGN)

        /** Text attributes for built-in functions like len, cap, make. */
        val PK_EXPR_BUILTIN = createTextAttributesKey("PK_EXPR_BUILTIN", DefaultLanguageHighlighterColors.PREDEFINED_SYMBOL)

        /** Text attributes for context variables like props, state, partial. */
        val PK_EXPR_CONTEXT_VAR = createTextAttributesKey("PK_EXPR_CONTEXT_VAR", DefaultLanguageHighlighterColors.INSTANCE_FIELD)

        /** Text attributes for function names. */
        val PK_EXPR_FUNCTION = createTextAttributesKey("PK_EXPR_FUNCTION", DefaultLanguageHighlighterColors.FUNCTION_DECLARATION)

        /** Text attributes for general identifiers. */
        val PK_EXPR_IDENTIFIER = createTextAttributesKey("PK_EXPR_IDENTIFIER", DefaultLanguageHighlighterColors.IDENTIFIER)

        /** Text attributes for template literal interpolation markers. */
        val PK_TEMPLATE_INTERP = createTextAttributesKey("PK_TEMPLATE_INTERP", DefaultLanguageHighlighterColors.BRACES)

        /** Text attributes for plain text content. */
        val PK_TEXT = createTextAttributesKey("PK_TEXT", HighlighterColors.TEXT)

        /** Text attributes for HTML comments. */
        val PK_COMMENT = createTextAttributesKey("PK_COMMENT", DefaultLanguageHighlighterColors.BLOCK_COMMENT)

        /** Dispatch key for structural tag highlighting. */
        private val STRUCTURAL_TAG_KEYS = arrayOf(PK_STRUCTURAL_TAG)

        /** Dispatch key for structural tag bracket highlighting. */
        private val TAG_BRACKET_KEYS = arrayOf(PK_TAG_BRACKET)

        /** Dispatch key for HTML tag name highlighting. */
        private val HTML_TAG_NAME_KEYS = arrayOf(PK_HTML_TAG_NAME)

        /** Dispatch key for HTML tag bracket highlighting. */
        private val HTML_TAG_BRACKET_KEYS = arrayOf(PK_HTML_TAG_BRACKET)

        /** Dispatch key for Piko component tag highlighting. */
        private val PIKO_COMPONENT_KEYS = arrayOf(PK_PIKO_COMPONENT)

        /** Dispatch key for attribute name highlighting. */
        private val ATTRIBUTE_KEYS = arrayOf(PK_ATTRIBUTE)

        /** Dispatch key for attribute value highlighting. */
        private val ATTRIBUTE_VALUE_KEYS = arrayOf(PK_ATTRIBUTE_VALUE)

        /** Dispatch key for attribute quote highlighting. */
        private val ATTRIBUTE_QUOTE_KEYS = arrayOf(PK_ATTRIBUTE_QUOTE)

        /** Dispatch key for directive name highlighting. */
        private val DIRECTIVE_KEYS = arrayOf(PK_DIRECTIVE)

        /** Dispatch key for bind shorthand highlighting. */
        private val DIRECTIVE_BIND_KEYS = arrayOf(PK_DIRECTIVE_BIND)

        /** Dispatch key for event shorthand highlighting. */
        private val DIRECTIVE_EVENT_KEYS = arrayOf(PK_DIRECTIVE_EVENT)

        /** Dispatch key for interpolation bracket highlighting. */
        private val INTERPOLATION_BRACKET_KEYS = arrayOf(PK_INTERPOLATION_BRACKET)

        /** Dispatch key for boolean literal highlighting. */
        private val EXPR_BOOLEAN_KEYS = arrayOf(PK_EXPR_BOOLEAN)

        /** Dispatch key for number literal highlighting. */
        private val EXPR_NUMBER_KEYS = arrayOf(PK_EXPR_NUMBER)

        /** Dispatch key for string literal highlighting. */
        private val EXPR_STRING_KEYS = arrayOf(PK_EXPR_STRING)

        /** Dispatch key for string quote highlighting. */
        private val EXPR_STRING_QUOTE_KEYS = arrayOf(PK_EXPR_STRING_QUOTE)

        /** Dispatch key for escape sequence highlighting. */
        private val EXPR_ESCAPE_KEYS = arrayOf(PK_EXPR_ESCAPE)

        /** Dispatch key for operator highlighting. */
        private val EXPR_OPERATOR_KEYS = arrayOf(PK_EXPR_OPERATOR)

        /** Dispatch key for dot accessor highlighting. */
        private val EXPR_DOT_KEYS = arrayOf(PK_EXPR_DOT)

        /** Dispatch key for parenthesis highlighting. */
        private val EXPR_PAREN_KEYS = arrayOf(PK_EXPR_PAREN)

        /** Dispatch key for square bracket highlighting. */
        private val EXPR_BRACKET_KEYS = arrayOf(PK_EXPR_BRACKET)

        /** Dispatch key for curly brace highlighting. */
        private val EXPR_BRACE_KEYS = arrayOf(PK_EXPR_BRACE)

        /** Dispatch key for comma separator highlighting. */
        private val EXPR_COMMA_KEYS = arrayOf(PK_EXPR_COMMA)

        /** Dispatch key for colon separator highlighting. */
        private val EXPR_COLON_KEYS = arrayOf(PK_EXPR_COLON)

        /** Dispatch key for built-in function highlighting. */
        private val EXPR_BUILTIN_KEYS = arrayOf(PK_EXPR_BUILTIN)

        /** Dispatch key for context variable highlighting. */
        private val EXPR_CONTEXT_VAR_KEYS = arrayOf(PK_EXPR_CONTEXT_VAR)

        /** Dispatch key for function name highlighting. */
        private val EXPR_FUNCTION_KEYS = arrayOf(PK_EXPR_FUNCTION)

        /** Dispatch key for general identifier highlighting. */
        private val EXPR_IDENTIFIER_KEYS = arrayOf(PK_EXPR_IDENTIFIER)

        /** Dispatch key for template interpolation marker highlighting. */
        private val TEMPLATE_INTERP_KEYS = arrayOf(PK_TEMPLATE_INTERP)

        /** Dispatch key for plain text highlighting. */
        private val TEXT_KEYS = arrayOf(PK_TEXT)

        /** Dispatch key for comment highlighting. */
        private val COMMENT_KEYS = arrayOf(PK_COMMENT)

        /** Empty dispatch key for tokens with no highlighting. */
        private val EMPTY_KEYS = emptyArray<TextAttributesKey>()
    }

    /**
     * Returns the lexer used for tokenising file content during highlighting.
     *
     * @return A new PKLexerAdapter instance.
     */
    override fun getHighlightingLexer(): Lexer = PKLexerAdapter()

    /**
     * Maps a token type to its corresponding text attribute keys.
     *
     * @param tokenType The token type to look up.
     * @return An array of text attribute keys for highlighting, or empty if none.
     */
    override fun getTokenHighlights(tokenType: IElementType?): Array<TextAttributesKey> {
        return when (tokenType) {
            PKTokenTypes.TEMPLATE_TAG_START,
            PKTokenTypes.TEMPLATE_TAG_END,
            PKTokenTypes.SCRIPT_TAG_START,
            PKTokenTypes.SCRIPT_TAG_END,
            PKTokenTypes.STYLE_TAG_START,
            PKTokenTypes.STYLE_TAG_END,
            PKTokenTypes.I18N_TAG_START,
            PKTokenTypes.I18N_TAG_END -> STRUCTURAL_TAG_KEYS

            PKTokenTypes.TAG_END_GT -> TAG_BRACKET_KEYS

            PKTokenTypes.ATTR_NAME -> ATTRIBUTE_KEYS
            PKTokenTypes.ATTR_VALUE -> ATTRIBUTE_VALUE_KEYS

            PKTokenTypes.HTML_TAG_NAME -> HTML_TAG_NAME_KEYS

            PKTokenTypes.HTML_TAG_OPEN,
            PKTokenTypes.HTML_TAG_CLOSE,
            PKTokenTypes.HTML_TAG_SELF_CLOSE,
            PKTokenTypes.HTML_END_TAG_OPEN -> HTML_TAG_BRACKET_KEYS

            PKTokenTypes.PIKO_TAG_NAME -> PIKO_COMPONENT_KEYS

            PKTokenTypes.HTML_ATTR_NAME -> ATTRIBUTE_KEYS
            PKTokenTypes.HTML_ATTR_VALUE -> ATTRIBUTE_VALUE_KEYS
            PKTokenTypes.HTML_ATTR_QUOTE -> ATTRIBUTE_QUOTE_KEYS

            PKTokenTypes.DIRECTIVE_NAME -> DIRECTIVE_KEYS
            PKTokenTypes.DIRECTIVE_BIND -> DIRECTIVE_BIND_KEYS
            PKTokenTypes.DIRECTIVE_EVENT -> DIRECTIVE_EVENT_KEYS

            PKTokenTypes.INTERPOLATION_OPEN,
            PKTokenTypes.INTERPOLATION_CLOSE -> INTERPOLATION_BRACKET_KEYS

            PKTokenTypes.EXPR_BOOLEAN -> EXPR_BOOLEAN_KEYS
            PKTokenTypes.EXPR_NUMBER -> EXPR_NUMBER_KEYS
            PKTokenTypes.EXPR_STRING -> EXPR_STRING_KEYS
            PKTokenTypes.EXPR_STRING_QUOTE -> EXPR_STRING_QUOTE_KEYS
            PKTokenTypes.EXPR_ESCAPE -> EXPR_ESCAPE_KEYS

            PKTokenTypes.EXPR_OP_COMPARISON,
            PKTokenTypes.EXPR_OP_LOGICAL,
            PKTokenTypes.EXPR_OP_ARITHMETIC -> EXPR_OPERATOR_KEYS

            PKTokenTypes.EXPR_OP_DOT -> EXPR_DOT_KEYS

            PKTokenTypes.EXPR_PAREN_OPEN,
            PKTokenTypes.EXPR_PAREN_CLOSE -> EXPR_PAREN_KEYS

            PKTokenTypes.EXPR_BRACKET_OPEN,
            PKTokenTypes.EXPR_BRACKET_CLOSE -> EXPR_BRACKET_KEYS

            PKTokenTypes.EXPR_BRACE_OPEN,
            PKTokenTypes.EXPR_BRACE_CLOSE -> EXPR_BRACE_KEYS

            PKTokenTypes.EXPR_COMMA -> EXPR_COMMA_KEYS
            PKTokenTypes.EXPR_COLON -> EXPR_COLON_KEYS

            PKTokenTypes.EXPR_BUILTIN -> EXPR_BUILTIN_KEYS
            PKTokenTypes.EXPR_CONTEXT_VAR -> EXPR_CONTEXT_VAR_KEYS
            PKTokenTypes.EXPR_FUNCTION_NAME -> EXPR_FUNCTION_KEYS
            PKTokenTypes.EXPR_IDENTIFIER -> EXPR_IDENTIFIER_KEYS

            PKTokenTypes.TEMPLATE_INTERP_OPEN,
            PKTokenTypes.TEMPLATE_INTERP_CLOSE -> TEMPLATE_INTERP_KEYS

            PKTokenTypes.TEXT_CONTENT -> TEXT_KEYS

            PKTokenTypes.HTML_COMMENT -> COMMENT_KEYS

            PKTokenTypes.GO_SCRIPT_CONTENT,
            PKTokenTypes.JS_SCRIPT_CONTENT,
            PKTokenTypes.CSS_STYLE_CONTENT,
            PKTokenTypes.I18N_CONTENT -> EMPTY_KEYS

            else -> EMPTY_KEYS
        }
    }
}
