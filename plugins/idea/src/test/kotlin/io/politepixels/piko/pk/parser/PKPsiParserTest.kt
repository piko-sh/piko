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


package io.politepixels.piko.pk.parser

import com.intellij.psi.TokenType
import com.intellij.psi.tree.IElementType
import io.politepixels.gen.pk.PKLexer
import io.politepixels.piko.pk.PKTokenTypes
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

class PKPsiParserTest {

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
    fun `template block starts with TEMPLATE_TAG_START`() {
        val tokens = tokenTypes("<template></template>")
        assertEquals("PK_TEMPLATE_TAG_START", tokens[0])
    }

    @Test
    fun `template block ends with TEMPLATE_TAG_END`() {
        val tokens = tokenTypes("<template></template>")
        assertTrue(tokens.contains("PK_TEMPLATE_TAG_END"))
    }

    @Test
    fun `template with content produces TAG_END_GT between start and content`() {
        val tokens = tokenTypes("<template><div></div></template>")
        assertEquals("PK_TEMPLATE_TAG_START", tokens[0])
        assertEquals("PK_TAG_END_GT", tokens[1])
        assertEquals("PK_HTML_TAG_OPEN", tokens[2])
    }

    @Test
    fun `go script block has GO_SCRIPT_CONTENT token`() {
        val tokens = tokenTypes("<script>func main() {}</script>")
        assertTrue("Should contain GO_SCRIPT_CONTENT", tokens.contains("PK_GO_SCRIPT_CONTENT"))
    }

    @Test
    fun `js script block has JS_SCRIPT_CONTENT token`() {
        val tokens = tokenTypes("<script lang=\"js\">const x = 1;</script>")
        assertTrue("Should contain JS_SCRIPT_CONTENT", tokens.contains("PK_JS_SCRIPT_CONTENT"))
    }

    @Test
    fun `script language detection uses lang attribute`() {
        val goTokens = tokenTypes("<script>code</script>")
        val jsTokens = tokenTypes("<script lang=\"js\">code</script>")
        val tsTokens = tokenTypes("<script lang=\"ts\">code</script>")

        assertTrue("Default should be Go", goTokens.contains("PK_GO_SCRIPT_CONTENT"))
        assertTrue("lang=js should be JS", jsTokens.contains("PK_JS_SCRIPT_CONTENT"))
        assertTrue("lang=ts should be JS", tsTokens.contains("PK_JS_SCRIPT_CONTENT"))
    }

    @Test
    fun `style block has CSS_STYLE_CONTENT token`() {
        val tokens = tokenTypes("<style>.test { color: red; }</style>")
        assertTrue("Should contain CSS_STYLE_CONTENT", tokens.contains("PK_CSS_STYLE_CONTENT"))
    }

    @Test
    fun `style block starts with STYLE_TAG_START`() {
        val tokens = tokenTypes("<style></style>")
        assertEquals("PK_STYLE_TAG_START", tokens[0])
    }

    @Test
    fun `i18n block has I18N_CONTENT token`() {
        val tokens = tokenTypes("<i18n>{\"en\": {}}</i18n>")
        assertTrue("Should contain I18N_CONTENT", tokens.contains("PK_I18N_CONTENT"))
    }

    @Test
    fun `i18n block starts with I18N_TAG_START`() {
        val tokens = tokenTypes("<i18n></i18n>")
        assertEquals("PK_I18N_TAG_START", tokens[0])
    }

    @Test
    fun `file with all four block types tokenises correctly`() {
        val input = """
            <template><div></div></template>
            <script>func main() {}</script>
            <style>.test {}</style>
            <i18n>{}</i18n>
        """.trimIndent()

        val tokens = tokenTypes(input)

        assertTrue("Should have template block", tokens.contains("PK_TEMPLATE_TAG_START"))
        assertTrue("Should have script block", tokens.contains("PK_SCRIPT_TAG_START"))
        assertTrue("Should have style block", tokens.contains("PK_STYLE_TAG_START"))
        assertTrue("Should have i18n block", tokens.contains("PK_I18N_TAG_START"))
    }

    @Test
    fun `multiple script blocks are independent`() {
        val input = """
            <script>func go() {}</script>
            <script lang="js">const js = 1;</script>
        """.trimIndent()

        val tokens = tokenTypes(input)
        val scriptStarts = tokens.count { it == "PK_SCRIPT_TAG_START" }
        val scriptEnds = tokens.count { it == "PK_SCRIPT_TAG_END" }

        assertEquals("Should have 2 script starts", 2, scriptStarts)
        assertEquals("Should have 2 script ends", 2, scriptEnds)
    }

    @Test
    fun `body element types are defined`() {
        assertNotNull("TEMPLATE_BODY_ELEMENT should exist", PKTokenTypes.TEMPLATE_BODY_ELEMENT)
        assertNotNull("GO_SCRIPT_BODY_ELEMENT should exist", PKTokenTypes.GO_SCRIPT_BODY_ELEMENT)
        assertNotNull("JS_SCRIPT_BODY_ELEMENT should exist", PKTokenTypes.JS_SCRIPT_BODY_ELEMENT)
        assertNotNull("CSS_STYLE_BODY_ELEMENT should exist", PKTokenTypes.CSS_STYLE_BODY_ELEMENT)
        assertNotNull("I18N_BODY_ELEMENT should exist", PKTokenTypes.I18N_BODY_ELEMENT)
    }

    @Test
    fun `block element types are defined`() {
        assertNotNull("TEMPLATE_BLOCK_ELEMENT should exist", PKTokenTypes.TEMPLATE_BLOCK_ELEMENT)
        assertNotNull("SCRIPT_BLOCK_ELEMENT should exist", PKTokenTypes.SCRIPT_BLOCK_ELEMENT)
        assertNotNull("STYLE_BLOCK_ELEMENT should exist", PKTokenTypes.STYLE_BLOCK_ELEMENT)
        assertNotNull("I18N_BLOCK_ELEMENT should exist", PKTokenTypes.I18N_BLOCK_ELEMENT)
    }

    @Test
    fun `empty file produces no tokens`() {
        val tokens = tokenTypes("")
        assertTrue("Empty file should produce no tokens", tokens.isEmpty())
    }

    @Test
    fun `whitespace between blocks is handled`() {
        val input = """
            <template></template>

            <script></script>
        """.trimIndent()

        val tokens = tokenTypes(input)
        assertTrue("Should have template start", tokens.contains("PK_TEMPLATE_TAG_START"))
        assertTrue("Should have script start", tokens.contains("PK_SCRIPT_TAG_START"))
    }
}
