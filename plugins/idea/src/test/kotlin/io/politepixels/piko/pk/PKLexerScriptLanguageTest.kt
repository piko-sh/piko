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
import org.junit.Assert.assertTrue
import org.junit.Test

class PKLexerScriptLanguageTest {

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

    private fun hasJsContent(tokens: List<String>): Boolean = tokens.contains("PK_JS_SCRIPT_CONTENT")
    private fun hasGoContent(tokens: List<String>): Boolean = tokens.contains("PK_GO_SCRIPT_CONTENT")

    @Test
    fun `test script lang js produces JS content`() {
        val tokens = tokenTypes("<script lang=\"js\">const x = 1;</script>")
        assertTrue("Should produce JS content", hasJsContent(tokens))
    }

    @Test
    fun `test script lang javascript produces JS content`() {
        val tokens = tokenTypes("<script lang=\"javascript\">const x = 1;</script>")
        assertTrue("Should produce JS content", hasJsContent(tokens))
    }

    @Test
    fun `test script lang ts produces JS content`() {
        val tokens = tokenTypes("<script lang=\"ts\">const x: number = 1;</script>")
        assertTrue("Should produce JS content for TypeScript", hasJsContent(tokens))
    }

    @Test
    fun `test script lang typescript produces JS content`() {
        val tokens = tokenTypes("<script lang=\"typescript\">const x: number = 1;</script>")
        assertTrue("Should produce JS content for TypeScript", hasJsContent(tokens))
    }

    @Test
    fun `test script lang js with single quotes`() {
        val tokens = tokenTypes("<script lang='js'>const x = 1;</script>")
        assertTrue("Should produce JS content with single quotes", hasJsContent(tokens))
    }

    @Test
    fun `test script lang ts with single quotes`() {
        val tokens = tokenTypes("<script lang='ts'>const x: number = 1;</script>")
        assertTrue("Should produce JS content with single quotes", hasJsContent(tokens))
    }

    @Test
    fun `test script lang typescript with single quotes`() {
        val tokens = tokenTypes("<script lang='typescript'>const x: number = 1;</script>")
        assertTrue("Should produce JS content with single quotes", hasJsContent(tokens))
    }

    @Test
    fun `test script type application javascript`() {
        val tokens = tokenTypes("<script type=\"application/javascript\">const x = 1;</script>")
        assertTrue("Should produce JS content for application/javascript", hasJsContent(tokens))
    }

    @Test
    fun `test script type text javascript`() {
        val tokens = tokenTypes("<script type=\"text/javascript\">const x = 1;</script>")
        assertTrue("Should produce JS content for text/javascript", hasJsContent(tokens))
    }

    @Test
    fun `test script type application x-go produces Go content`() {
        val tokens = tokenTypes("<script type=\"application/x-go\">func main() {}</script>")
        assertTrue("Should produce Go content for application/x-go", hasGoContent(tokens))
    }

    @Test
    fun `test script type text x-go produces Go content`() {
        val tokens = tokenTypes("<script type=\"text/x-go\">func main() {}</script>")
        assertTrue("Should produce Go content for text/x-go", hasGoContent(tokens))
    }

    @Test
    fun `test script lang JS uppercase produces JS content`() {
        val tokens = tokenTypes("<script lang=\"JS\">const x = 1;</script>")
        assertTrue("Should handle uppercase JS", hasJsContent(tokens))
    }

    @Test
    fun `test script lang TypeScript mixed case produces JS content`() {
        val tokens = tokenTypes("<script lang=\"TypeScript\">const x: number = 1;</script>")
        assertTrue("Should handle mixed case TypeScript", hasJsContent(tokens))
    }

    @Test
    fun `test script lang JAVASCRIPT uppercase produces JS content`() {
        val tokens = tokenTypes("<script lang=\"JAVASCRIPT\">const x = 1;</script>")
        assertTrue("Should handle uppercase JAVASCRIPT", hasJsContent(tokens))
    }

    @Test
    fun `test script lang go lowercase produces Go content`() {
        val tokens = tokenTypes("<script lang=\"go\">func main() {}</script>")
        assertTrue("Should handle lowercase go", hasGoContent(tokens))
    }

    @Test
    fun `test script lang GO uppercase produces Go content`() {
        val tokens = tokenTypes("<script lang=\"GO\">func main() {}</script>")
        assertTrue("Should handle uppercase GO", hasGoContent(tokens))
    }

    @Test
    fun `test script without lang defaults to Go content`() {
        val tokens = tokenTypes("<script>func main() {}</script>")
        assertTrue("Should default to Go content", hasGoContent(tokens))
    }

    @Test
    fun `test script with unknown lang defaults to Go content`() {
        val tokens = tokenTypes("<script lang=\"python\">def main(): pass</script>")
        assertTrue("Should default to Go for unknown lang", hasGoContent(tokens))
    }

    @Test
    fun `test script with empty lang defaults to Go content`() {
        val tokens = tokenTypes("<script lang=\"\">func main() {}</script>")
        assertTrue("Should default to Go for empty lang", hasGoContent(tokens))
    }

    @Test
    fun `test script lang with space before equals`() {
        val tokens = tokenTypes("<script lang =\"js\">const x = 1;</script>")
        assertTrue("Should handle space before equals", hasJsContent(tokens))
    }

    @Test
    fun `test script lang with space after equals`() {
        val tokens = tokenTypes("<script lang= \"js\">const x = 1;</script>")
        assertTrue("Should handle space after equals", hasJsContent(tokens))
    }

    @Test
    fun `test script lang with spaces around equals`() {
        val tokens = tokenTypes("<script lang = \"js\">const x = 1;</script>")
        assertTrue("Should handle spaces around equals", hasJsContent(tokens))
    }

    @Test
    fun `test script with lang and other attributes`() {
        val tokens = tokenTypes("<script lang=\"js\" id=\"main\">const x = 1;</script>")
        assertTrue("Should produce JS content with multiple attributes", hasJsContent(tokens))
    }

    @Test
    fun `test script with type attribute before lang`() {
        val tokens = tokenTypes("<script type=\"module\" lang=\"ts\">const x: number = 1;</script>")
        assertTrue("Should produce JS content when type appears before lang", hasJsContent(tokens))
    }

    @Test
    fun `test script tag structure includes ATTR_NAME for lang`() {
        val tokens = tokenTypes("<script lang=\"js\">const x = 1;</script>")
        assertTrue("Should include ATTR_NAME token", tokens.contains("PK_ATTR_NAME"))
    }

    @Test
    fun `test script tag closing structure`() {
        val tokens = tokenTypes("<script lang=\"js\">const x = 1;</script>")
        assertEquals(
            listOf(
                "PK_SCRIPT_TAG_START",
                "PK_ATTR_NAME",
                "PK_TAG_END_GT",
                "PK_JS_SCRIPT_CONTENT",
                "PK_SCRIPT_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }
}
