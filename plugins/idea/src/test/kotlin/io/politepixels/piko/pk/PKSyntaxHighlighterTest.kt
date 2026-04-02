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

import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class PKSyntaxHighlighterTest {

    private val highlighter = PKSyntaxHighlighter()

    @Test
    fun `test TEMPLATE_TAG_START maps to PK_STRUCTURAL_TAG`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.TEMPLATE_TAG_START)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_STRUCTURAL_TAG), highlights)
    }

    @Test
    fun `test TEMPLATE_TAG_END maps to PK_STRUCTURAL_TAG`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.TEMPLATE_TAG_END)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_STRUCTURAL_TAG), highlights)
    }

    @Test
    fun `test SCRIPT_TAG_START maps to PK_STRUCTURAL_TAG`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.SCRIPT_TAG_START)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_STRUCTURAL_TAG), highlights)
    }

    @Test
    fun `test SCRIPT_TAG_END maps to PK_STRUCTURAL_TAG`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.SCRIPT_TAG_END)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_STRUCTURAL_TAG), highlights)
    }

    @Test
    fun `test STYLE_TAG_START maps to PK_STRUCTURAL_TAG`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.STYLE_TAG_START)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_STRUCTURAL_TAG), highlights)
    }

    @Test
    fun `test STYLE_TAG_END maps to PK_STRUCTURAL_TAG`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.STYLE_TAG_END)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_STRUCTURAL_TAG), highlights)
    }

    @Test
    fun `test I18N_TAG_START maps to PK_STRUCTURAL_TAG`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.I18N_TAG_START)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_STRUCTURAL_TAG), highlights)
    }

    @Test
    fun `test I18N_TAG_END maps to PK_STRUCTURAL_TAG`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.I18N_TAG_END)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_STRUCTURAL_TAG), highlights)
    }

    @Test
    fun `test TAG_END_GT maps to PK_TAG_BRACKET`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.TAG_END_GT)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_TAG_BRACKET), highlights)
    }

    @Test
    fun `test HTML_TAG_NAME maps to PK_HTML_TAG_NAME`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.HTML_TAG_NAME)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_HTML_TAG_NAME), highlights)
    }

    @Test
    fun `test HTML_TAG_OPEN maps to PK_HTML_TAG_BRACKET`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.HTML_TAG_OPEN)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_HTML_TAG_BRACKET), highlights)
    }

    @Test
    fun `test HTML_TAG_CLOSE maps to PK_HTML_TAG_BRACKET`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.HTML_TAG_CLOSE)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_HTML_TAG_BRACKET), highlights)
    }

    @Test
    fun `test HTML_TAG_SELF_CLOSE maps to PK_HTML_TAG_BRACKET`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.HTML_TAG_SELF_CLOSE)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_HTML_TAG_BRACKET), highlights)
    }

    @Test
    fun `test HTML_END_TAG_OPEN maps to PK_HTML_TAG_BRACKET`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.HTML_END_TAG_OPEN)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_HTML_TAG_BRACKET), highlights)
    }

    @Test
    fun `test PIKO_TAG_NAME maps to PK_PIKO_COMPONENT`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.PIKO_TAG_NAME)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_PIKO_COMPONENT), highlights)
    }

    @Test
    fun `test ATTR_NAME maps to PK_ATTRIBUTE`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.ATTR_NAME)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_ATTRIBUTE), highlights)
    }

    @Test
    fun `test ATTR_VALUE maps to PK_ATTRIBUTE_VALUE`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.ATTR_VALUE)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_ATTRIBUTE_VALUE), highlights)
    }

    @Test
    fun `test HTML_ATTR_NAME maps to PK_ATTRIBUTE`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.HTML_ATTR_NAME)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_ATTRIBUTE), highlights)
    }

    @Test
    fun `test HTML_ATTR_VALUE maps to PK_ATTRIBUTE_VALUE`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.HTML_ATTR_VALUE)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_ATTRIBUTE_VALUE), highlights)
    }

    @Test
    fun `test HTML_ATTR_QUOTE maps to PK_ATTRIBUTE_QUOTE`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.HTML_ATTR_QUOTE)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_ATTRIBUTE_QUOTE), highlights)
    }

    @Test
    fun `test DIRECTIVE_NAME maps to PK_DIRECTIVE`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.DIRECTIVE_NAME)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_DIRECTIVE), highlights)
    }

    @Test
    fun `test DIRECTIVE_BIND maps to PK_DIRECTIVE_BIND`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.DIRECTIVE_BIND)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_DIRECTIVE_BIND), highlights)
    }

    @Test
    fun `test DIRECTIVE_EVENT maps to PK_DIRECTIVE_EVENT`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.DIRECTIVE_EVENT)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_DIRECTIVE_EVENT), highlights)
    }

    @Test
    fun `test INTERPOLATION_OPEN maps to PK_INTERPOLATION_BRACKET`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.INTERPOLATION_OPEN)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_INTERPOLATION_BRACKET), highlights)
    }

    @Test
    fun `test INTERPOLATION_CLOSE maps to PK_INTERPOLATION_BRACKET`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.INTERPOLATION_CLOSE)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_INTERPOLATION_BRACKET), highlights)
    }

    @Test
    fun `test EXPR_BOOLEAN maps to PK_EXPR_BOOLEAN`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_BOOLEAN)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_BOOLEAN), highlights)
    }

    @Test
    fun `test EXPR_NUMBER maps to PK_EXPR_NUMBER`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_NUMBER)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_NUMBER), highlights)
    }

    @Test
    fun `test EXPR_STRING maps to PK_EXPR_STRING`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_STRING)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_STRING), highlights)
    }

    @Test
    fun `test EXPR_STRING_QUOTE maps to PK_EXPR_STRING_QUOTE`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_STRING_QUOTE)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_STRING_QUOTE), highlights)
    }

    @Test
    fun `test EXPR_ESCAPE maps to PK_EXPR_ESCAPE`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_ESCAPE)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_ESCAPE), highlights)
    }

    @Test
    fun `test EXPR_OP_COMPARISON maps to PK_EXPR_OPERATOR`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_OP_COMPARISON)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_OPERATOR), highlights)
    }

    @Test
    fun `test EXPR_OP_LOGICAL maps to PK_EXPR_OPERATOR`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_OP_LOGICAL)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_OPERATOR), highlights)
    }

    @Test
    fun `test EXPR_OP_ARITHMETIC maps to PK_EXPR_OPERATOR`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_OP_ARITHMETIC)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_OPERATOR), highlights)
    }

    @Test
    fun `test EXPR_OP_DOT maps to PK_EXPR_DOT`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_OP_DOT)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_DOT), highlights)
    }

    @Test
    fun `test EXPR_PAREN_OPEN maps to PK_EXPR_PAREN`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_PAREN_OPEN)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_PAREN), highlights)
    }

    @Test
    fun `test EXPR_PAREN_CLOSE maps to PK_EXPR_PAREN`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_PAREN_CLOSE)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_PAREN), highlights)
    }

    @Test
    fun `test EXPR_BRACKET_OPEN maps to PK_EXPR_BRACKET`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_BRACKET_OPEN)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_BRACKET), highlights)
    }

    @Test
    fun `test EXPR_BRACKET_CLOSE maps to PK_EXPR_BRACKET`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_BRACKET_CLOSE)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_BRACKET), highlights)
    }

    @Test
    fun `test EXPR_BRACE_OPEN maps to PK_EXPR_BRACE`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_BRACE_OPEN)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_BRACE), highlights)
    }

    @Test
    fun `test EXPR_BRACE_CLOSE maps to PK_EXPR_BRACE`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_BRACE_CLOSE)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_BRACE), highlights)
    }

    @Test
    fun `test EXPR_COMMA maps to PK_EXPR_COMMA`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_COMMA)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_COMMA), highlights)
    }

    @Test
    fun `test EXPR_COLON maps to PK_EXPR_COLON`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_COLON)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_COLON), highlights)
    }

    @Test
    fun `test EXPR_BUILTIN maps to PK_EXPR_BUILTIN`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_BUILTIN)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_BUILTIN), highlights)
    }

    @Test
    fun `test EXPR_CONTEXT_VAR maps to PK_EXPR_CONTEXT_VAR`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_CONTEXT_VAR)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_CONTEXT_VAR), highlights)
    }

    @Test
    fun `test EXPR_FUNCTION_NAME maps to PK_EXPR_FUNCTION`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_FUNCTION_NAME)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_FUNCTION), highlights)
    }

    @Test
    fun `test EXPR_IDENTIFIER maps to PK_EXPR_IDENTIFIER`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.EXPR_IDENTIFIER)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_EXPR_IDENTIFIER), highlights)
    }

    @Test
    fun `test TEMPLATE_INTERP_OPEN maps to PK_TEMPLATE_INTERP`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.TEMPLATE_INTERP_OPEN)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_TEMPLATE_INTERP), highlights)
    }

    @Test
    fun `test TEMPLATE_INTERP_CLOSE maps to PK_TEMPLATE_INTERP`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.TEMPLATE_INTERP_CLOSE)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_TEMPLATE_INTERP), highlights)
    }

    @Test
    fun `test TEXT_CONTENT maps to PK_TEXT`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.TEXT_CONTENT)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_TEXT), highlights)
    }

    @Test
    fun `test HTML_COMMENT maps to PK_COMMENT`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.HTML_COMMENT)
        assertArrayEquals(arrayOf(PKSyntaxHighlighter.PK_COMMENT), highlights)
    }

    @Test
    fun `test GO_SCRIPT_CONTENT returns empty keys`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.GO_SCRIPT_CONTENT)
        assertTrue("Embedded Go content should return empty keys", highlights.isEmpty())
    }

    @Test
    fun `test JS_SCRIPT_CONTENT returns empty keys`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.JS_SCRIPT_CONTENT)
        assertTrue("Embedded JS content should return empty keys", highlights.isEmpty())
    }

    @Test
    fun `test CSS_STYLE_CONTENT returns empty keys`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.CSS_STYLE_CONTENT)
        assertTrue("Embedded CSS content should return empty keys", highlights.isEmpty())
    }

    @Test
    fun `test I18N_CONTENT returns empty keys`() {
        val highlights = highlighter.getTokenHighlights(PKTokenTypes.I18N_CONTENT)
        assertTrue("Embedded i18n content should return empty keys", highlights.isEmpty())
    }

    @Test
    fun `test null token returns empty keys`() {
        val highlights = highlighter.getTokenHighlights(null)
        assertTrue("Null token should return empty keys", highlights.isEmpty())
    }

    @Test
    fun `test unknown token returns empty keys`() {
        val unknownToken = PKElementType("PK_UNKNOWN_TEST_TOKEN")
        val highlights = highlighter.getTokenHighlights(unknownToken)
        assertTrue("Unknown token should return empty keys", highlights.isEmpty())
    }

    @Test
    fun `test getHighlightingLexer returns PKLexerAdapter`() {
        val lexer = highlighter.highlightingLexer
        assertEquals("PKLexerAdapter", lexer.javaClass.simpleName)
    }
}
