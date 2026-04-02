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
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

class PKLexerNegativePathTest {

    private fun tokenize(input: String): List<Pair<IElementType, String>> {
        val lexer = PKLexer(null)
        lexer.reset(input, 0, input.length, PKLexer.YYINITIAL)

        val tokens = mutableListOf<Pair<IElementType, String>>()
        var token = lexer.advance()
        while (token != null) {
            val text = input.substring(lexer.tokenStart, lexer.tokenEnd)
            tokens.add(token to text)
            token = lexer.advance()
        }
        return tokens
    }

    private fun tokenTypes(input: String, includeWhitespace: Boolean = false): List<String> {
        return tokenize(input)
            .filter { includeWhitespace || it.first != TokenType.WHITE_SPACE }
            .map { it.first.toString() }
    }

    @Test
    fun `lexer handles null-like characters gracefully`() {
        val input = "<template>\u0000</template>"
        val tokens = tokenize(input)
        assertNotNull("Should produce tokens", tokens)
        assertTrue("Should have some tokens", tokens.isNotEmpty())
    }

    @Test
    fun `lexer handles extremely long attribute value`() {
        val longValue = "a".repeat(10000)
        val input = "<template><div class=\"$longValue\"></div></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should contain template tag", tokens.contains("PK_TEMPLATE_TAG_START"))
        assertTrue("Should contain attribute value", tokens.contains("PK_HTML_ATTR_VALUE"))
    }

    @Test
    fun `lexer handles extremely long interpolation`() {
        val longExpr = "a" + ".prop".repeat(1000)
        val input = "<template>{{ $longExpr }}</template>"
        val tokens = tokenTypes(input)
        assertTrue("Should contain interpolation", tokens.contains("PK_INTERPOLATION_OPEN"))
    }

    @Test
    fun `lexer handles deeply nested parentheses`() {
        val nested = "((".repeat(50) + "x" + "))".repeat(50)
        val input = "<template>{{ $nested }}</template>"
        val tokens = tokenTypes(input)
        assertTrue("Should contain parentheses", tokens.contains("PK_EXPR_PAREN_OPEN"))
    }

    @Test
    fun `lexer handles unclosed block tags at EOF`() {
        val input = "<template><div>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize partial content", tokens.contains("PK_TEMPLATE_TAG_START"))
        assertTrue("Should tokenize HTML tag", tokens.contains("PK_HTML_TAG_NAME"))
    }

    @Test
    fun `lexer handles unclosed script at EOF`() {
        val input = "<script>func main() {"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize script start", tokens.contains("PK_SCRIPT_TAG_START"))
    }

    @Test
    fun `lexer handles unclosed style at EOF`() {
        val input = "<style>.class {"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize style start", tokens.contains("PK_STYLE_TAG_START"))
    }

    @Test
    fun `lexer handles script tag without content`() {
        val input = "<script></script>"
        val tokens = tokenTypes(input)
        assertEquals(
            listOf("PK_SCRIPT_TAG_START", "PK_TAG_END_GT", "PK_SCRIPT_TAG_END", "PK_TAG_END_GT"),
            tokens
        )
    }

    @Test
    fun `lexer handles malformed HTML entities`() {
        val inputs = listOf(
            "<template>&</template>",
            "<template>&&</template>",
            "<template>&amp</template>",
            "<template>&invalid;</template>",
            "<template>&123;</template>"
        )
        for (input in inputs) {
            val tokens = tokenTypes(input)
            assertTrue("Should tokenize malformed entity as text: $input", tokens.contains("PK_TEXT_CONTENT"))
        }
    }

    @Test
    fun `lexer handles invalid directive names`() {
        val inputs = listOf(
            "<template><div p-></div></template>",
            "<template><div p-123></div></template>",
            "<template><div p-!invalid></div></template>"
        )
        for (input in inputs) {
            val tokens = tokenTypes(input)
            assertTrue("Should still tokenize template: $input", tokens.contains("PK_TEMPLATE_TAG_START"))
        }
    }

    @Test
    fun `lexer handles consecutive angle brackets`() {
        val input = "<template><<<>>></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize template", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer handles consecutive braces`() {
        val input = "<template>{{{{{}}}}}</template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize template", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer handles mixed quote types in single attribute`() {
        val input = "<template><div class=\"test'value\"></div></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize despite mismatched quotes", tokens.contains("PK_HTML_ATTR_NAME"))
    }

    @Test
    fun `lexer handles attribute without value`() {
        val input = "<template><div disabled></div></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize boolean attribute", tokens.contains("PK_HTML_ATTR_NAME"))
    }

    @Test
    fun `lexer handles attribute with equals but no value`() {
        val input = "<template><div class=></div></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize partial attribute", tokens.contains("PK_HTML_ATTR_NAME"))
    }

    @Test
    fun `lexer handles self-closing tag without space`() {
        val input = "<template><br/></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize self-closing tag", tokens.contains("PK_HTML_TAG_SELF_CLOSE"))
    }

    @Test
    fun `lexer handles CDATA-like content`() {
        val input = "<template><![CDATA[some data]]></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize template", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer handles processing instructions`() {
        val input = "<?xml version=\"1.0\"?><template></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should still find template", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer handles DOCTYPE`() {
        val input = "<!DOCTYPE html><template></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should still find template", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer handles expression with unclosed bracket`() {
        val input = "<template>{{ items[0 }}</template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize interpolation", tokens.contains("PK_INTERPOLATION_OPEN"))
        assertTrue("Should tokenize bracket", tokens.contains("PK_EXPR_BRACKET_OPEN"))
    }

    @Test
    fun `lexer handles expression with unclosed brace`() {
        val input = "<template>{{ {key: val }}</template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize interpolation", tokens.contains("PK_INTERPOLATION_OPEN"))
        assertTrue("Should tokenize brace", tokens.contains("PK_EXPR_BRACE_OPEN"))
    }

    @Test
    fun `lexer handles expression with unclosed parenthesis`() {
        val input = "<template>{{ fn(x }}</template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize interpolation", tokens.contains("PK_INTERPOLATION_OPEN"))
        assertTrue("Should tokenize paren", tokens.contains("PK_EXPR_PAREN_OPEN"))
    }

    @Test
    fun `lexer handles script with invalid lang attribute`() {
        val input = "<script lang=\"invalid\">content</script>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize script", tokens.contains("PK_SCRIPT_TAG_START"))
    }

    @Test
    fun `lexer handles newlines in attribute value`() {
        val input = "<template><div class=\"line1\nline2\"></div></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize attribute with newlines", tokens.contains("PK_HTML_ATTR_VALUE"))
    }

    @Test
    fun `lexer handles tabs in attribute value`() {
        val input = "<template><div class=\"col1\tcol2\"></div></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize attribute with tabs", tokens.contains("PK_HTML_ATTR_VALUE"))
    }

    @Test
    fun `lexer handles carriage returns`() {
        val input = "<template>\r\n<div></div>\r\n</template>"
        val tokens = tokenTypes(input)
        assertTrue("Should handle CRLF line endings", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer handles form feed character`() {
        val input = "<template>\u000C</template>"
        val tokens = tokenTypes(input)
        assertTrue("Should handle form feed", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer handles vertical tab`() {
        val input = "<template>\u000B</template>"
        val tokens = tokenTypes(input)
        assertTrue("Should handle vertical tab", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer handles BOM at start of file`() {
        val input = "\uFEFF<template></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should handle BOM and find template", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer handles zero-width characters`() {
        val input = "<template>\u200B\u200C\u200D</template>"
        val tokens = tokenTypes(input)
        assertTrue("Should handle zero-width chars", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer handles RTL characters`() {
        val input = "<template>\u200F\u202B</template>"
        val tokens = tokenTypes(input)
        assertTrue("Should handle RTL chars", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer handles surrogate pairs correctly`() {
        val input = "<template>\uD83D\uDE00</template>"
        val tokens = tokenTypes(input)
        assertTrue("Should handle emoji surrogate pair", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer does not hang on pathological input`() {
        val pathological = "{{" + "(".repeat(100) + "x" + ")".repeat(100) + "}}"
        val input = "<template>$pathological</template>"
        val startTime = System.currentTimeMillis()
        val tokens = tokenTypes(input)
        val elapsed = System.currentTimeMillis() - startTime
        assertTrue("Should complete in reasonable time", elapsed < 5000)
        assertTrue("Should produce tokens", tokens.isNotEmpty())
    }

    @Test
    fun `lexer handles comment with double dashes inside`() {
        val input = "<template><!-- -- comment --></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize comment", tokens.contains("PK_HTML_COMMENT"))
    }

    @Test
    fun `lexer handles unclosed comment`() {
        val input = "<template><!-- unclosed comment"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize template start", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer handles nested comments attempt`() {
        val input = "<template><!-- outer <!-- inner --> outer --></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize template", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer handles empty attribute name`() {
        val input = "<template><div =\"value\"></div></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize template", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer handles duplicate attributes`() {
        val input = "<template><div class=\"a\" class=\"b\"></div></template>"
        val tokens = tokenTypes(input)
        val attrCount = tokens.count { it == "PK_HTML_ATTR_NAME" }
        assertEquals("Should tokenize both attributes", 2, attrCount)
    }

    @Test
    fun `lexer handles numeric tag names`() {
        val input = "<template><123></123></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize template", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer handles tag name starting with hyphen`() {
        val input = "<template><-custom></-custom></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize template", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer handles expression with only operators`() {
        val input = "<template>{{ + - * / }}</template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize interpolation", tokens.contains("PK_INTERPOLATION_OPEN"))
    }

    @Test
    fun `lexer handles expression with consecutive operators`() {
        val input = "<template>{{ a ++ b }}</template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize interpolation", tokens.contains("PK_INTERPOLATION_OPEN"))
    }

    @Test
    fun `lexer handles very long tag name`() {
        val longName = "a".repeat(1000)
        val input = "<template><$longName></$longName></template>"
        val tokens = tokenTypes(input)
        assertTrue("Should tokenize long tag name", tokens.contains("PK_HTML_TAG_NAME"))
    }

    @Test
    fun `lexer handles all whitespace input`() {
        val tokens = tokenTypes("    \n\t\r\n    ", includeWhitespace = false)
        assertTrue("Should produce no non-whitespace tokens", tokens.isEmpty())
    }

    @Test
    fun `lexer handles only interpolation without template`() {
        val tokens = tokenTypes("{{ value }}")
        assertFalse("Should not find template tag", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `lexer handles only script without wrapper`() {
        val tokens = tokenTypes("func main() {}")
        assertFalse("Should not find script tag", tokens.contains("PK_SCRIPT_TAG_START"))
    }
}
