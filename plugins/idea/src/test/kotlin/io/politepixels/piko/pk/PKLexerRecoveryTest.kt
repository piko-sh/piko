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

import com.intellij.psi.TokenType
import com.intellij.psi.tree.IElementType
import io.politepixels.gen.pk.PKLexer
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Covers nested templates, recovery from unterminated constructs, new
 * directives, interpolation inside attribute values and unknown script langs.
 */
class PKLexerRecoveryTest {

    private fun tokenise(input: String): List<Pair<IElementType, String>> {
        val lexer = PKLexer(null)
        lexer.reset(input, 0, input.length, PKLexer.YYINITIAL)

        val tokens = mutableListOf<Pair<IElementType, String>>()
        var token = lexer.advance()
        while (token != null) {
            tokens.add(token to input.substring(lexer.tokenStart, lexer.tokenEnd))
            token = lexer.advance()
        }
        return tokens
    }

    private fun nonWhitespace(input: String): List<Pair<IElementType, String>> =
        tokenise(input).filter { it.first != TokenType.WHITE_SPACE }

    private fun assertCoversInput(input: String) {
        assertEquals("Tokens should cover the whole input", input, tokenise(input).joinToString("") { it.second })
    }

    @Test
    fun `nested template end tag does not close the block`() {
        val input = "<template><div><template p-if=\"x\"><span></span></template></div></template>\n<script>package main</script>"
        val tokens = nonWhitespace(input)

        assertEquals(
            "Only the outer template should produce TEMPLATE_TAG_END",
            1,
            tokens.count { it.first == PKTokenTypes.TEMPLATE_TAG_END }
        )
        assertTrue(
            "The nested closing tag should be an HTML end tag",
            tokens.any { it.first == PKTokenTypes.HTML_TAG_NAME && it.second == "template" }
        )
        assertTrue(
            "The script block after the template should still be recognised",
            tokens.any { it.first == PKTokenTypes.GO_SCRIPT_CONTENT && it.second == "package main" }
        )
        assertCoversInput(input)
    }

    @Test
    fun `deeply nested templates are balanced`() {
        val input = "<template><template><template></template></template></template><style>a{}</style>"
        val tokens = nonWhitespace(input)
        assertEquals(1, tokens.count { it.first == PKTokenTypes.TEMPLATE_TAG_END })
        assertTrue(tokens.any { it.first == PKTokenTypes.CSS_STYLE_CONTENT })
    }

    @Test
    fun `self-closing nested template does not change depth`() {
        val input = "<template><template /></template><style>a{}</style>"
        val tokens = nonWhitespace(input)
        assertEquals(1, tokens.count { it.first == PKTokenTypes.TEMPLATE_TAG_END })
        assertTrue(tokens.any { it.first == PKTokenTypes.CSS_STYLE_CONTENT })
    }

    @Test
    fun `template prefix in another tag name is not nesting`() {
        val input = "<template><templates></templates></template><style>a{}</style>"
        val tokens = nonWhitespace(input)
        assertEquals(1, tokens.count { it.first == PKTokenTypes.TEMPLATE_TAG_END })
        assertTrue(tokens.any { it.first == PKTokenTypes.CSS_STYLE_CONTENT })
    }

    @Test
    fun `unterminated attribute value recovers at template end`() {
        val input = "<template><div class=\"broken></div></template><style>a{}</style>"
        val tokens = nonWhitespace(input)
        assertTrue(tokens.any { it.first == PKTokenTypes.TEMPLATE_TAG_END })
        assertTrue(tokens.any { it.first == PKTokenTypes.CSS_STYLE_CONTENT })
        assertCoversInput(input)
    }

    @Test
    fun `unterminated attribute value recovers at a new line starting a tag`() {
        val input = "<template><div class=\"broken>\n  <p>text</p>\n</template>"
        val tokens = nonWhitespace(input)
        assertTrue(
            "The <p> on the next line should be lexed as a tag",
            tokens.any { it.first == PKTokenTypes.HTML_TAG_NAME && it.second == "p" }
        )
        assertTrue(tokens.any { it.first == PKTokenTypes.TEMPLATE_TAG_END })
        assertCoversInput(input)
    }

    @Test
    fun `multi-line attribute values stay attribute values`() {
        val input = "<template><div class=\"a\n  b\n  c\"></div></template>"
        val tokens = nonWhitespace(input)
        val valueText = tokens.filter { it.first == PKTokenTypes.HTML_ATTR_VALUE }.joinToString("") { it.second }
        assertEquals("a\n  b\n  c", valueText)
        assertEquals(2, tokens.count { it.first == PKTokenTypes.HTML_ATTR_QUOTE })
    }

    @Test
    fun `unterminated single-quoted attribute value recovers at template end`() {
        val input = "<template><div title='broken></div></template><style>a{}</style>"
        val tokens = nonWhitespace(input)
        assertTrue(tokens.any { it.first == PKTokenTypes.CSS_STYLE_CONTENT })
    }

    @Test
    fun `unterminated interpolation recovers at template end`() {
        val input = "<template><p>{{ name </p></template><style>a{}</style>"
        val tokens = nonWhitespace(input)
        assertTrue(tokens.any { it.first == PKTokenTypes.TEMPLATE_TAG_END })
        assertTrue(tokens.any { it.first == PKTokenTypes.CSS_STYLE_CONTENT })
        assertCoversInput(input)
    }

    @Test
    fun `unterminated expression string recovers at template end`() {
        val input = "<template><p>{{ \"open </p></template><style>a{}</style>"
        val tokens = nonWhitespace(input)
        assertTrue(tokens.any { it.first == PKTokenTypes.TEMPLATE_TAG_END })
        assertTrue(tokens.any { it.first == PKTokenTypes.CSS_STYLE_CONTENT })
    }

    @Test
    fun `unterminated directive value recovers at template end`() {
        val input = "<template><p p-if=\"a && </p></template><style>a{}</style>"
        val tokens = nonWhitespace(input)
        assertTrue(tokens.any { it.first == PKTokenTypes.TEMPLATE_TAG_END })
        assertTrue(tokens.any { it.first == PKTokenTypes.CSS_STYLE_CONTENT })
    }

    @Test
    fun `unclosed tag recovers at template end`() {
        val input = "<template><div class=\"a\"</template><style>a{}</style>"
        val tokens = nonWhitespace(input)
        assertTrue(tokens.any { it.first == PKTokenTypes.TEMPLATE_TAG_END })
        assertTrue(tokens.any { it.first == PKTokenTypes.CSS_STYLE_CONTENT })
    }

    @Test
    fun `less-than inside a string stays in the string`() {
        val input = "<template>{{ \"a < b\" }}</template>"
        val tokens = nonWhitespace(input)
        val stringText = tokens.filter { it.first == PKTokenTypes.EXPR_STRING }.joinToString("") { it.second }
        assertEquals("a < b", stringText)
        assertTrue(tokens.any { it.first == PKTokenTypes.INTERPOLATION_CLOSE })
    }

    @Test
    fun `new directives are recognised`() {
        for (directive in listOf("p-memo", "p-timeline", "p-fragment", "p-param", "p-nil", "p-kf", "p-collection", "p-provider")) {
            val tokens = nonWhitespace("<template><div $directive=\"x\"></div></template>")
            assertTrue(
                "$directive should be a directive",
                tokens.any { it.first == PKTokenTypes.DIRECTIVE_NAME && it.second == directive }
            )
        }
    }

    @Test
    fun `longer attribute names sharing a directive prefix are plain attributes`() {
        val tokens = nonWhitespace("<template><div p-fragment-id=\"x\"></div></template>")
        assertTrue(tokens.any { it.first == PKTokenTypes.HTML_ATTR_NAME && it.second == "p-fragment-id" })
        assertFalse(tokens.any { it.first == PKTokenTypes.DIRECTIVE_NAME })
    }

    @Test
    fun `interpolation inside attribute value is highlighted`() {
        val input = "<template><a href=\"/posts/{{ post.ID }}/edit\"></a></template>"
        val tokens = nonWhitespace(input)
        assertTrue(tokens.any { it.first == PKTokenTypes.INTERPOLATION_OPEN })
        assertTrue(tokens.any { it.first == PKTokenTypes.EXPR_IDENTIFIER && it.second == "post" })
        assertTrue(tokens.any { it.first == PKTokenTypes.INTERPOLATION_CLOSE })
        assertTrue(tokens.any { it.first == PKTokenTypes.HTML_ATTR_VALUE && it.second == "/edit" })
        assertEquals(2, tokens.count { it.first == PKTokenTypes.HTML_ATTR_QUOTE })
        assertCoversInput(input)
    }

    @Test
    fun `interpolation inside single-quoted attribute value returns to the value`() {
        val tokens = nonWhitespace("<template><a title='{{ name }}!'></a></template>")
        assertTrue(tokens.any { it.first == PKTokenTypes.INTERPOLATION_CLOSE })
        assertTrue(tokens.any { it.first == PKTokenTypes.HTML_ATTR_VALUE && it.second == "!" })
    }

    @Test
    fun `backtick interpolation closes with TEMPLATE_INTERP_CLOSE`() {
        val tokens = nonWhitespace("<template>{{ `a \${b} c` }}</template>")
        assertTrue(tokens.any { it.first == PKTokenTypes.TEMPLATE_INTERP_OPEN })
        assertTrue(tokens.any { it.first == PKTokenTypes.TEMPLATE_INTERP_CLOSE })
        assertTrue(tokens.any { it.first == PKTokenTypes.INTERPOLATION_CLOSE })
    }

    @Test
    fun `unknown script lang is plain text`() {
        val tokens = nonWhitespace("<script lang=\"python\">print('x')</script>")
        assertTrue(tokens.any { it.first == PKTokenTypes.SCRIPT_TEXT_CONTENT })
        assertFalse(tokens.any { it.first == PKTokenTypes.GO_SCRIPT_CONTENT })
        assertFalse(tokens.any { it.first == PKTokenTypes.JS_SCRIPT_CONTENT })
    }

    @Test
    fun `go lang and default script stay Go`() {
        assertTrue(nonWhitespace("<script lang=\"go\">x</script>").any { it.first == PKTokenTypes.GO_SCRIPT_CONTENT })
        assertTrue(nonWhitespace("<script>x</script>").any { it.first == PKTokenTypes.GO_SCRIPT_CONTENT })
        assertTrue(
            nonWhitespace("<script type=\"application/x-go\">x</script>").any { it.first == PKTokenTypes.GO_SCRIPT_CONTENT }
        )
    }
}
