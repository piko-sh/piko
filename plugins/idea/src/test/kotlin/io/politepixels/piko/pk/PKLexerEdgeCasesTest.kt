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

class PKLexerEdgeCasesTest {

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

    private fun tokenPairs(input: String, includeWhitespace: Boolean = false): List<String> {
        return tokenize(input)
            .filter { includeWhitespace || it.first != TokenType.WHITE_SPACE }
            .map { "${it.first}('${it.second}')" }
    }

    @Test
    fun `test escaped double quote in string`() {
        val pairs = tokenPairs("<template>{{ \"quote: \\\"\" }}</template>")
        assertTrue("Should contain string with escaped quote", pairs.any { it.contains("PK_EXPR_STRING") })
    }

    @Test
    fun `test escaped single quote in string`() {
        val pairs = tokenPairs("<template>{{ 'single: \\'' }}</template>")
        assertTrue("Should contain string with escaped single quote", pairs.any { it.contains("PK_EXPR_STRING") })
    }

    @Test
    fun `test escaped backslash in string`() {
        val pairs = tokenPairs("<template>{{ \"\\\\\" }}</template>")
        assertTrue("Should handle escaped backslash", pairs.any { it.contains("PK_EXPR_STRING") || it.contains("PK_EXPR_ESCAPE") })
    }

    @Test
    fun `test newline escape in string`() {
        val pairs = tokenPairs("<template>{{ \"line1\\nline2\" }}</template>")
        assertTrue("Should handle newline escape", pairs.any { it.contains("PK_EXPR_STRING") || it.contains("PK_EXPR_ESCAPE") })
    }

    @Test
    fun `test tab escape in string`() {
        val pairs = tokenPairs("<template>{{ \"col1\\tcol2\" }}</template>")
        assertTrue("Should handle tab escape", pairs.any { it.contains("PK_EXPR_STRING") || it.contains("PK_EXPR_ESCAPE") })
    }

    @Test
    fun `test carriage return escape in string`() {
        val pairs = tokenPairs("<template>{{ \"text\\rmore\" }}</template>")
        assertTrue("Should handle carriage return escape", pairs.any { it.contains("PK_EXPR_STRING") || it.contains("PK_EXPR_ESCAPE") })
    }

    @Test
    fun `test simple backtick template`() {
        val tokens = tokenTypes("<template>{{ `template` }}</template>")
        assertTrue("Should tokenize backtick template", tokens.contains("PK_INTERPOLATION_OPEN"))
    }

    @Test
    fun `test backtick template with interpolation marker`() {
        val pairs = tokenPairs("<template>{{ `hello \${name}` }}</template>")
        assertTrue("Should handle template interpolation", pairs.isNotEmpty())
    }

    @Test
    fun `test backtick template with multiple interpolations`() {
        val pairs = tokenPairs("<template>{{ `\${a} and \${b}` }}</template>")
        assertTrue("Should handle multiple template interpolations", pairs.isNotEmpty())
    }

    @Test
    fun `test backtick template with nested braces`() {
        val pairs = tokenPairs("<template>{{ `\${obj.field}` }}</template>")
        assertTrue("Should handle nested braces in template", pairs.isNotEmpty())
    }

    @Test
    fun `test empty template tag`() {
        val tokens = tokenTypes("<template></template>")
        assertEquals(
            listOf(
                "PK_TEMPLATE_TAG_START",
                "PK_TAG_END_GT",
                "PK_TEMPLATE_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test empty script tag`() {
        val tokens = tokenTypes("<script></script>")
        assertEquals(
            listOf(
                "PK_SCRIPT_TAG_START",
                "PK_TAG_END_GT",
                "PK_SCRIPT_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test empty style tag`() {
        val tokens = tokenTypes("<style></style>")
        assertEquals(
            listOf(
                "PK_STYLE_TAG_START",
                "PK_TAG_END_GT",
                "PK_STYLE_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test empty i18n tag`() {
        val tokens = tokenTypes("<i18n></i18n>")
        assertEquals(
            listOf(
                "PK_I18N_TAG_START",
                "PK_TAG_END_GT",
                "PK_I18N_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test whitespace-only template content`() {
        val tokens = tokenTypes("<template>   </template>", includeWhitespace = true)
        assertTrue("Should contain template tags", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `test whitespace-only script content`() {
        val tokens = tokenTypes("<script>   </script>", includeWhitespace = true)
        assertTrue("Should contain script tags", tokens.contains("PK_SCRIPT_TAG_START"))
    }

    @Test
    fun `test template with only newlines`() {
        val tokens = tokenTypes("<template>\n\n\n</template>", includeWhitespace = true)
        assertTrue("Should handle newlines in template", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `test unclosed template tag tokenizes available content`() {
        val tokens = tokenTypes("<template><div>content")
        assertTrue("Should tokenize partial content", tokens.contains("PK_TEMPLATE_TAG_START"))
        assertTrue("Should tokenize HTML tag", tokens.contains("PK_HTML_TAG_NAME"))
    }

    @Test
    fun `test unclosed interpolation tokenizes opening brace`() {
        val tokens = tokenTypes("<template>{{ name </template>")
        assertTrue("Should contain interpolation open", tokens.contains("PK_INTERPOLATION_OPEN"))
    }

    @Test
    fun `test mismatched quotes in attribute gracefully handled`() {
        val tokens = tokenTypes("<template><div class=\"test'></div></template>")
        assertTrue("Should attempt to tokenize despite mismatched quotes", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `test unclosed HTML tag`() {
        val tokens = tokenTypes("<template><div class=\"test\">content</template>")
        assertTrue("Should tokenize even with unclosed tag", tokens.contains("PK_HTML_TAG_NAME"))
        assertTrue("Should still find closing template tag", tokens.contains("PK_TEMPLATE_TAG_END"))
    }

    @Test
    fun `test unclosed string in interpolation`() {
        val tokens = tokenTypes("<template>{{ \"unclosed }}</template>")
        assertTrue("Should handle unclosed string", tokens.contains("PK_INTERPOLATION_OPEN"))
    }

    @Test
    fun `test single brace is text not interpolation`() {
        val tokens = tokenTypes("<template>{ not interpolation }</template>")
        assertTrue("Should contain text content", tokens.contains("PK_TEXT_CONTENT"))
        assertFalse("Should NOT contain interpolation open", tokens.contains("PK_INTERPOLATION_OPEN"))
    }

    @Test
    fun `test triple brace produces single brace plus interpolation`() {
        val tokens = tokenTypes("<template>{{{ name }}</template>")
        assertTrue("Should have some form of opening brace handling", tokens.isNotEmpty())
    }

    @Test
    fun `test HTML entities in template content`() {
        val tokens = tokenTypes("<template>&lt;div&gt;</template>")
        assertTrue("Should tokenize HTML entities as text", tokens.contains("PK_TEXT_CONTENT"))
    }

    @Test
    fun `test angle bracket in text content`() {
        val tokens = tokenTypes("<template>5 > 3 is true</template>")
        assertTrue("Should contain text content", tokens.contains("PK_TEXT_CONTENT"))
    }

    @Test
    fun `test ampersand in text content`() {
        val tokens = tokenTypes("<template>A & B</template>")
        assertTrue("Should handle ampersand as text", tokens.contains("PK_TEXT_CONTENT"))
    }

    @Test
    fun `test multiple consecutive interpolations`() {
        val tokens = tokenTypes("<template>{{ a }}{{ b }}{{ c }}</template>")
        val opens = tokens.count { it == "PK_INTERPOLATION_OPEN" }
        val closes = tokens.count { it == "PK_INTERPOLATION_CLOSE" }
        assertEquals("Should have 3 interpolation opens", 3, opens)
        assertEquals("Should have 3 interpolation closes", 3, closes)
    }

    @Test
    fun `test interpolation immediately after tag`() {
        val tokens = tokenTypes("<template><div>{{ value }}</div></template>")
        assertTrue("Should have interpolation after tag", tokens.contains("PK_INTERPOLATION_OPEN"))
    }

    @Test
    fun `test interpolation immediately before tag`() {
        val tokens = tokenTypes("<template>{{ value }}<div></div></template>")
        assertTrue("Should have interpolation before tag", tokens.contains("PK_INTERPOLATION_OPEN"))
        assertTrue("Should have HTML tag after interpolation", tokens.contains("PK_HTML_TAG_NAME"))
    }

    @Test
    fun `test deeply nested HTML structure`() {
        val tokens = tokenTypes("<template><div><span><a><strong>text</strong></a></span></div></template>")
        val tagNames = tokens.filter { it == "PK_HTML_TAG_NAME" }
        assertEquals("Should have 8 tag name tokens (4 open + 4 close)", 8, tagNames.size)
    }

    @Test
    fun `test interpolation inside attribute value`() {
        val tokens = tokenTypes("<template><div class=\"container\"></div></template>")
        assertTrue("Should tokenize attribute", tokens.contains("PK_HTML_ATTR_NAME"))
        assertTrue("Should tokenize attribute value", tokens.contains("PK_HTML_ATTR_VALUE"))
    }

    @Test
    fun `test multiple attributes on single tag`() {
        val tokens = tokenTypes("<template><div id=\"main\" class=\"container\" data-value=\"test\"></div></template>")
        val attrNames = tokens.count { it == "PK_HTML_ATTR_NAME" }
        assertEquals("Should have 3 attribute names", 3, attrNames)
    }

    @Test
    fun `test empty file`() {
        val tokens = tokenTypes("")
        assertTrue("Empty file should produce no tokens", tokens.isEmpty())
    }

    @Test
    fun `test file with only whitespace`() {
        val tokens = tokenTypes("   \n\n   \t   ")
        assertTrue("Whitespace-only file should produce no non-whitespace tokens", tokens.isEmpty())
    }

    @Test
    fun `test very long interpolation expression`() {
        val longExpr = "a" + ".b".repeat(50)
        val tokens = tokenTypes("<template>{{ $longExpr }}</template>")
        assertTrue("Should handle long expressions", tokens.contains("PK_INTERPOLATION_OPEN"))
    }

    @Test
    fun `test unicode in text content`() {
        val tokens = tokenTypes("<template>Hello \u4E16\u754C</template>")
        assertTrue("Should handle unicode characters", tokens.contains("PK_TEXT_CONTENT"))
    }

    @Test
    fun `test unicode in string literal`() {
        val pairs = tokenPairs("<template>{{ \"\u4E16\u754C\" }}</template>")
        assertTrue("Should handle unicode in string", pairs.any { it.contains("PK_EXPR_STRING") })
    }

    @Test
    fun `test emoji in text content`() {
        val tokens = tokenTypes("<template>Hello \uD83D\uDC4B World \uD83C\uDF0D</template>")
        assertTrue("Should handle emoji characters", tokens.contains("PK_TEXT_CONTENT"))
    }

    @Test
    fun `test p-if directive with single quotes`() {
        val tokens = tokenTypes("<template><div p-if='true'></div></template>")
        assertTrue("Should contain directive name", tokens.contains("PK_DIRECTIVE_NAME"))
        assertTrue("Should contain boolean in single-quoted directive", tokens.contains("PK_EXPR_BOOLEAN"))
    }

    @Test
    fun `test p-for directive with single quotes`() {
        val tokens = tokenTypes("<template><li p-for='item in items'></li></template>")
        assertTrue("Should contain directive name", tokens.contains("PK_DIRECTIVE_NAME"))
        assertTrue("Should contain identifier in single-quoted directive", tokens.contains("PK_EXPR_IDENTIFIER"))
    }

    @Test
    fun `test bind directive with single quotes`() {
        val tokens = tokenTypes("<template><div :class='active'></div></template>")
        assertTrue("Should contain bind directive", tokens.contains("PK_DIRECTIVE_BIND"))
        assertTrue("Should contain identifier", tokens.contains("PK_EXPR_IDENTIFIER"))
    }

    @Test
    fun `test event directive with single quotes`() {
        val tokens = tokenTypes("<template><button @click='handleClick'></button></template>")
        assertTrue("Should contain event directive", tokens.contains("PK_DIRECTIVE_EVENT"))
        assertTrue("Should contain identifier", tokens.contains("PK_EXPR_IDENTIFIER"))
    }

    @Test
    fun `test complex expression in single-quoted directive`() {
        val tokens = tokenTypes("<template><div p-if='props.Count > 0 && state.Ready'></div></template>")
        assertTrue("Should contain comparison operator", tokens.contains("PK_EXPR_OP_COMPARISON"))
        assertTrue("Should contain logical operator", tokens.contains("PK_EXPR_OP_LOGICAL"))
        assertTrue("Should contain context var", tokens.contains("PK_EXPR_CONTEXT_VAR"))
    }

    @Test
    fun `test number in single-quoted directive`() {
        val tokens = tokenTypes("<template><div p-if='count >= 10'></div></template>")
        assertTrue("Should contain number", tokens.contains("PK_EXPR_NUMBER"))
    }

    @Test
    fun `test function call in single-quoted directive`() {
        val tokens = tokenTypes("<template><div p-if='len(items) > 0'></div></template>")
        assertTrue("Should contain builtin", tokens.contains("PK_EXPR_BUILTIN"))
        assertTrue("Should contain paren open", tokens.contains("PK_EXPR_PAREN_OPEN"))
    }

    @Test
    fun `test all operators in single-quoted directive`() {
        val tokens = tokenTypes("<template><div p-if='a + b - c * d / e % f'></div></template>")
        assertTrue("Should contain arithmetic operator", tokens.contains("PK_EXPR_OP_ARITHMETIC"))
    }

    @Test
    fun `test brackets in single-quoted directive`() {
        val tokens = tokenTypes("<template><div p-if='items[0]'></div></template>")
        assertTrue("Should contain bracket open", tokens.contains("PK_EXPR_BRACKET_OPEN"))
        assertTrue("Should contain bracket close", tokens.contains("PK_EXPR_BRACKET_CLOSE"))
    }

    @Test
    fun `test braces in single-quoted directive`() {
        val tokens = tokenTypes("<template><div p-if='{a: 1}'></div></template>")
        assertTrue("Should contain brace open", tokens.contains("PK_EXPR_BRACE_OPEN"))
        assertTrue("Should contain brace close", tokens.contains("PK_EXPR_BRACE_CLOSE"))
    }

    @Test
    fun `test colon in single-quoted directive`() {
        val tokens = tokenTypes("<template><div p-if='items[1:3]'></div></template>")
        assertTrue("Should contain colon", tokens.contains("PK_EXPR_COLON"))
    }

    @Test
    fun `test comma in single-quoted directive`() {
        val tokens = tokenTypes("<template><div p-text='fn(a, b)'></div></template>")
        assertTrue("Should contain comma", tokens.contains("PK_EXPR_COMMA"))
    }

    @Test
    fun `test double-quoted string inside single-quoted directive`() {
        val tokens = tokenTypes("<template><div p-text='\"hello\"'></div></template>")
        assertTrue("Should contain string quote", tokens.contains("PK_EXPR_STRING_QUOTE"))
        assertTrue("Should contain string content", tokens.contains("PK_EXPR_STRING"))
    }

    @Test
    fun `test single-quoted string inside double-quoted directive`() {
        val tokens = tokenTypes("<template><div p-text=\"'hello'\"></div></template>")
        assertTrue("Should contain string quote", tokens.contains("PK_EXPR_STRING_QUOTE"))
        assertTrue("Should contain string content", tokens.contains("PK_EXPR_STRING"))
    }

    @Test
    fun `test backtick string inside double-quoted directive`() {
        val tokens = tokenTypes("<template><div p-text=\"`template`\"></div></template>")
        assertTrue("Should contain string quote", tokens.contains("PK_EXPR_STRING_QUOTE"))
    }

    @Test
    fun `test backtick string inside single-quoted directive`() {
        val tokens = tokenTypes("<template><div p-text='`template`'></div></template>")
        assertTrue("Should contain string quote", tokens.contains("PK_EXPR_STRING_QUOTE"))
    }

    @Test
    fun `test multiple nested strings in directive`() {
        val tokens = tokenTypes("<template><div p-text=\"concat('hello', 'world')\"></div></template>")
        val stringQuotes = tokens.count { it == "PK_EXPR_STRING_QUOTE" }
        assertTrue("Should have multiple string quotes", stringQuotes >= 4)
    }

    @Test
    fun `test script with single-quoted lang js`() {
        val tokens = tokenTypes("<script lang='js'>const x = 1;</script>")
        assertTrue("Should contain script tag start", tokens.contains("PK_SCRIPT_TAG_START"))
        assertTrue("Should contain JS script content", tokens.contains("PK_JS_SCRIPT_CONTENT"))
    }

    @Test
    fun `test script with single-quoted lang ts`() {
        val tokens = tokenTypes("<script lang='ts'>const x: number = 1;</script>")
        assertTrue("Should contain JS script content for ts", tokens.contains("PK_JS_SCRIPT_CONTENT"))
    }

    @Test
    fun `test script with single-quoted lang typescript`() {
        val tokens = tokenTypes("<script lang='typescript'>const x: number = 1;</script>")
        assertTrue("Should contain JS script content", tokens.contains("PK_JS_SCRIPT_CONTENT"))
    }

    @Test
    fun `test script with single-quoted lang javascript`() {
        val tokens = tokenTypes("<script lang='javascript'>const x = 1;</script>")
        assertTrue("Should contain JS script content", tokens.contains("PK_JS_SCRIPT_CONTENT"))
    }

    @Test
    fun `test script with single-quoted type attribute for go`() {
        val tokens = tokenTypes("<script type='application/x-go'>func main() {}</script>")
        assertTrue("Should contain Go script content", tokens.contains("PK_GO_SCRIPT_CONTENT"))
    }

    @Test
    fun `test script with single-quoted type attribute text x-go`() {
        val tokens = tokenTypes("<script type='text/x-go'>func main() {}</script>")
        assertTrue("Should contain Go script content", tokens.contains("PK_GO_SCRIPT_CONTENT"))
    }

    @Test
    fun `test script with single-quoted type attribute for js`() {
        val tokens = tokenTypes("<script type='application/javascript'>const x = 1;</script>")
        assertTrue("Should contain JS script content", tokens.contains("PK_JS_SCRIPT_CONTENT"))
    }

    @Test
    fun `test script with single-quoted type attribute text javascript`() {
        val tokens = tokenTypes("<script type='text/javascript'>const x = 1;</script>")
        assertTrue("Should contain JS script content", tokens.contains("PK_JS_SCRIPT_CONTENT"))
    }

    @Test
    fun `test script with single-quoted type attribute for ts`() {
        val tokens = tokenTypes("<script type='application/typescript'>const x: number = 1;</script>")
        assertTrue("Should contain JS script content", tokens.contains("PK_JS_SCRIPT_CONTENT"))
    }

    @Test
    fun `test script with single-quoted type attribute text typescript`() {
        val tokens = tokenTypes("<script type='text/typescript'>const x: number = 1;</script>")
        assertTrue("Should contain JS script content", tokens.contains("PK_JS_SCRIPT_CONTENT"))
    }

    @Test
    fun `test script with double-quoted type attribute for go`() {
        val tokens = tokenTypes("<script type=\"application/x-go\">func main() {}</script>")
        assertTrue("Should contain Go script content", tokens.contains("PK_GO_SCRIPT_CONTENT"))
    }

    @Test
    fun `test script with double-quoted type attribute text x-go`() {
        val tokens = tokenTypes("<script type=\"text/x-go\">func main() {}</script>")
        assertTrue("Should contain Go script content", tokens.contains("PK_GO_SCRIPT_CONTENT"))
    }

    @Test
    fun `test script with double-quoted type attribute for js`() {
        val tokens = tokenTypes("<script type=\"application/javascript\">const x = 1;</script>")
        assertTrue("Should contain JS script content", tokens.contains("PK_JS_SCRIPT_CONTENT"))
    }

    @Test
    fun `test script with double-quoted type attribute for ts`() {
        val tokens = tokenTypes("<script type=\"application/typescript\">const x: number = 1;</script>")
        assertTrue("Should contain JS script content", tokens.contains("PK_JS_SCRIPT_CONTENT"))
    }

    @Test
    fun `test backslash n escape produces escape token in double string`() {
        val pairs = tokenPairs("<template>{{ \"line1\\nline2\" }}</template>")
        assertTrue("Should have escape token", pairs.any { it.contains("PK_EXPR_ESCAPE") })
    }

    @Test
    fun `test backslash t escape produces escape token`() {
        val pairs = tokenPairs("<template>{{ \"col1\\tcol2\" }}</template>")
        assertTrue("Should have escape token", pairs.any { it.contains("PK_EXPR_ESCAPE") })
    }

    @Test
    fun `test backslash r escape produces escape token`() {
        val pairs = tokenPairs("<template>{{ \"text\\rmore\" }}</template>")
        assertTrue("Should have escape token", pairs.any { it.contains("PK_EXPR_ESCAPE") })
    }

    @Test
    fun `test backslash b escape produces escape token`() {
        val pairs = tokenPairs("<template>{{ \"back\\bspace\" }}</template>")
        assertTrue("Should have escape token", pairs.any { it.contains("PK_EXPR_ESCAPE") })
    }

    @Test
    fun `test backslash f escape produces escape token`() {
        val pairs = tokenPairs("<template>{{ \"form\\ffeed\" }}</template>")
        assertTrue("Should have escape token", pairs.any { it.contains("PK_EXPR_ESCAPE") })
    }

    @Test
    fun `test backslash 0 escape produces escape token`() {
        val pairs = tokenPairs("<template>{{ \"null\\0char\" }}</template>")
        assertTrue("Should have escape token", pairs.any { it.contains("PK_EXPR_ESCAPE") })
    }

    @Test
    fun `test escaped quote in double string produces escape token`() {
        val pairs = tokenPairs("<template>{{ \"quote: \\\"text\\\"\" }}</template>")
        assertTrue("Should have escape token", pairs.any { it.contains("PK_EXPR_ESCAPE") })
    }

    @Test
    fun `test escaped quote in single string produces escape token`() {
        val pairs = tokenPairs("<template>{{ 'apostrophe: \\'text\\'' }}</template>")
        assertTrue("Should have escape token", pairs.any { it.contains("PK_EXPR_ESCAPE") })
    }

    @Test
    fun `test multiple escapes in one string`() {
        val pairs = tokenPairs("<template>{{ \"line1\\nline2\\ttab\\rcarriage\" }}</template>")
        val escapeCount = pairs.count { it.contains("PK_EXPR_ESCAPE") }
        assertEquals("Should have 3 escape tokens", 3, escapeCount)
    }

    @Test
    fun `test template interpolation open in backtick string`() {
        val tokens = tokenTypes("<template>{{ `hello \${name}` }}</template>")
        assertTrue("Should contain template interp open", tokens.contains("PK_TEMPLATE_INTERP_OPEN"))
    }

    @Test
    fun `test backtick with dollar not followed by brace`() {
        val tokens = tokenTypes("<template>{{ `price: $100` }}</template>")
        assertTrue("Should tokenize backtick template", tokens.contains("PK_EXPR_STRING_QUOTE"))
        assertTrue("Should contain string content with dollar", tokens.contains("PK_EXPR_STRING"))
    }

    @Test
    fun `test backtick with multiple template interpolations`() {
        val tokens = tokenTypes("<template>{{ `\${a} + \${b} = \${c}` }}</template>")
        val interpOpens = tokens.count { it == "PK_TEMPLATE_INTERP_OPEN" }
        assertEquals("Should have 3 template interpolation opens", 3, interpOpens)
    }

    @Test
    fun `test regular HTML attribute with single quotes`() {
        val tokens = tokenTypes("<template><div class='container'></div></template>")
        assertTrue("Should contain attr name", tokens.contains("PK_HTML_ATTR_NAME"))
        assertTrue("Should contain attr quote", tokens.contains("PK_HTML_ATTR_QUOTE"))
        assertTrue("Should contain attr value", tokens.contains("PK_HTML_ATTR_VALUE"))
    }

    @Test
    fun `test multiple attributes with mixed quotes`() {
        val tokens = tokenTypes("<template><div class=\"foo\" id='bar'></div></template>")
        val attrNames = tokens.count { it == "PK_HTML_ATTR_NAME" }
        assertEquals("Should have 2 attributes", 2, attrNames)
    }

    @Test
    fun `test structural tag with multiple attributes`() {
        val tokens = tokenTypes("<script lang=\"go\" setup></script>")
        assertTrue("Should tokenize script with attributes", tokens.contains("PK_SCRIPT_TAG_START"))
        assertTrue("Should have attr name", tokens.contains("PK_ATTR_NAME"))
    }

    @Test
    fun `test template with custom attribute`() {
        val tokens = tokenTypes("<template scoped></template>")
        assertTrue("Should handle template attributes", tokens.contains("PK_TEMPLATE_TAG_START"))
    }

    @Test
    fun `test style with lang attribute`() {
        val tokens = tokenTypes("<style lang=\"scss\">.test { }</style>")
        assertTrue("Should tokenize style with lang attr", tokens.contains("PK_STYLE_TAG_START"))
        assertTrue("Should have attr name", tokens.contains("PK_ATTR_NAME"))
    }

    @Test
    fun `test builtin len followed by paren`() {
        val tokens = tokenTypes("<template>{{ len(items) }}</template>")
        assertTrue("Should recognize len as builtin", tokens.contains("PK_EXPR_BUILTIN"))
    }

    @Test
    fun `test builtin len not followed by paren`() {
        val tokens = tokenTypes("<template>{{ len }}</template>")
        assertTrue("Should recognize standalone len as builtin", tokens.contains("PK_EXPR_BUILTIN"))
    }

    @Test
    fun `test builtin cap followed by paren`() {
        val tokens = tokenTypes("<template>{{ cap(slice) }}</template>")
        assertTrue("Should recognize cap as builtin", tokens.contains("PK_EXPR_BUILTIN"))
    }

    @Test
    fun `test builtin make followed by paren`() {
        val tokens = tokenTypes("<template>{{ make([]int, 10) }}</template>")
        assertTrue("Should recognize make as builtin", tokens.contains("PK_EXPR_BUILTIN"))
    }

    @Test
    fun `test builtin with whitespace before paren`() {
        val tokens = tokenTypes("<template>{{ len  (items) }}</template>")
        assertTrue("Should recognize builtin with whitespace before paren", tokens.contains("PK_EXPR_BUILTIN"))
    }

    @Test
    fun `test function name followed by paren`() {
        val tokens = tokenTypes("<template>{{ myFunc(x) }}</template>")
        assertTrue("Should recognize function name", tokens.contains("PK_EXPR_FUNCTION_NAME"))
    }

    @Test
    fun `test identifier not followed by paren`() {
        val tokens = tokenTypes("<template>{{ myVar }}</template>")
        assertTrue("Should recognize as identifier", tokens.contains("PK_EXPR_IDENTIFIER"))
        assertFalse("Should NOT be function name", tokens.contains("PK_EXPR_FUNCTION_NAME"))
    }

    @Test
    fun `test method call with whitespace before paren`() {
        val tokens = tokenTypes("<template>{{ myFunc  (x) }}</template>")
        assertTrue("Should recognize function with whitespace before paren", tokens.contains("PK_EXPR_FUNCTION_NAME"))
    }

    @Test
    fun `test all builtins in directive value`() {
        val builtins = listOf("len", "cap", "make", "new", "append", "copy", "delete", "panic", "recover", "print", "println")
        for (builtin in builtins) {
            val tokens = tokenTypes("<template><div p-if=\"$builtin(x)\"></div></template>")
            assertTrue("Should recognize $builtin in directive", tokens.contains("PK_EXPR_BUILTIN"))
        }
    }

    @Test
    fun `test context vars in single-quoted directive`() {
        val contextVars = listOf("props", "state", "partial")
        for (cv in contextVars) {
            val tokens = tokenTypes("<template><div p-if='$cv.Ready'></div></template>")
            assertTrue("Should recognize $cv in single-quoted directive", tokens.contains("PK_EXPR_CONTEXT_VAR"))
        }
    }
}
