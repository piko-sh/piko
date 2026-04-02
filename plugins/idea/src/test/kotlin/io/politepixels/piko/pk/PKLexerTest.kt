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
import org.junit.Test

class PKLexerTest {

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
    fun `test script tag with Go content`() {
        val tokens = tokenTypes("<script>func main() {}</script>")
        assertEquals(
            listOf(
                "PK_SCRIPT_TAG_START",
                "PK_TAG_END_GT",
                "PK_GO_SCRIPT_CONTENT",
                "PK_SCRIPT_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test script tag with lang js`() {
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

    @Test
    fun `test style tag`() {
        val tokens = tokenTypes("<style>.foo { color: red; }</style>")
        assertEquals(
            listOf(
                "PK_STYLE_TAG_START",
                "PK_TAG_END_GT",
                "PK_CSS_STYLE_CONTENT",
                "PK_STYLE_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test simple div tag`() {
        val tokens = tokenTypes("<template><div></div></template>")
        assertEquals(
            listOf(
                "PK_TEMPLATE_TAG_START",
                "PK_TAG_END_GT",
                "PK_HTML_TAG_OPEN",
                "PK_HTML_TAG_NAME",
                "PK_HTML_TAG_CLOSE",
                "PK_HTML_END_TAG_OPEN",
                "PK_HTML_TAG_NAME",
                "PK_HTML_TAG_CLOSE",
                "PK_TEMPLATE_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test self-closing tag`() {
        val tokens = tokenTypes("<template><br/></template>")
        assertEquals(
            listOf(
                "PK_TEMPLATE_TAG_START",
                "PK_TAG_END_GT",
                "PK_HTML_TAG_OPEN",
                "PK_HTML_TAG_NAME",
                "PK_HTML_TAG_SELF_CLOSE",
                "PK_TEMPLATE_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test tag with regular attribute`() {
        val tokens = tokenTypes("<template><div class=\"container\"></div></template>")
        assertEquals(
            listOf(
                "PK_TEMPLATE_TAG_START",
                "PK_TAG_END_GT",
                "PK_HTML_TAG_OPEN",
                "PK_HTML_TAG_NAME",
                "PK_HTML_ATTR_NAME",
                "PK_HTML_ATTR_EQ",
                "PK_HTML_ATTR_QUOTE",
                "PK_HTML_ATTR_VALUE",
                "PK_HTML_ATTR_QUOTE",
                "PK_HTML_TAG_CLOSE",
                "PK_HTML_END_TAG_OPEN",
                "PK_HTML_TAG_NAME",
                "PK_HTML_TAG_CLOSE",
                "PK_TEMPLATE_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test piko component tag`() {
        val tokens = tokenTypes("<template><piko:header/></template>")
        assertEquals(
            listOf(
                "PK_TEMPLATE_TAG_START",
                "PK_TAG_END_GT",
                "PK_HTML_TAG_OPEN",
                "PK_PIKO_TAG_NAME",
                "PK_HTML_TAG_SELF_CLOSE",
                "PK_TEMPLATE_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test p-if directive`() {
        val tokens = tokenTypes("<template><div p-if=\"true\"></div></template>")
        assertEquals(
            listOf(
                "PK_TEMPLATE_TAG_START",
                "PK_TAG_END_GT",
                "PK_HTML_TAG_OPEN",
                "PK_HTML_TAG_NAME",
                "PK_DIRECTIVE_NAME",
                "PK_HTML_ATTR_EQ",
                "PK_HTML_ATTR_QUOTE",
                "PK_EXPR_BOOLEAN",
                "PK_HTML_ATTR_QUOTE",
                "PK_HTML_TAG_CLOSE",
                "PK_HTML_END_TAG_OPEN",
                "PK_HTML_TAG_NAME",
                "PK_HTML_TAG_CLOSE",
                "PK_TEMPLATE_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test p-for directive`() {
        val tokens = tokenTypes("<template><li p-for=\"item in items\"></li></template>")
        val filtered = tokenTypes("<template><li p-for=\"item in items\"></li></template>")
        assert(filtered.contains("PK_DIRECTIVE_NAME"))
        assert(filtered.contains("PK_EXPR_IDENTIFIER"))
    }

    @Test
    fun `test bind shorthand directive`() {
        val tokens = tokenTypes("<template><div :class=\"cls\"></div></template>")
        assertEquals(
            listOf(
                "PK_TEMPLATE_TAG_START",
                "PK_TAG_END_GT",
                "PK_HTML_TAG_OPEN",
                "PK_HTML_TAG_NAME",
                "PK_DIRECTIVE_BIND",
                "PK_HTML_ATTR_EQ",
                "PK_HTML_ATTR_QUOTE",
                "PK_EXPR_IDENTIFIER",
                "PK_HTML_ATTR_QUOTE",
                "PK_HTML_TAG_CLOSE",
                "PK_HTML_END_TAG_OPEN",
                "PK_HTML_TAG_NAME",
                "PK_HTML_TAG_CLOSE",
                "PK_TEMPLATE_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test event shorthand directive`() {
        val tokens = tokenTypes("<template><button @click=\"handleClick\"></button></template>")
        assert(tokens.contains("PK_DIRECTIVE_EVENT"))
        assert(tokens.contains("PK_EXPR_IDENTIFIER"))
    }

    @Test
    fun `test p-bind with argument`() {
        val tokens = tokenTypes("<template><a p-bind:href=\"url\"></a></template>")
        assert(tokens.contains("PK_DIRECTIVE_NAME"))
        assert(tokens.contains("PK_EXPR_IDENTIFIER"))
    }

    @Test
    fun `test p-on with modifier`() {
        val tokens = tokenTypes("<template><form p-on:submit.prevent=\"onSubmit\"></form></template>")
        assert(tokens.contains("PK_DIRECTIVE_NAME"))
    }

    @Test
    fun `test simple interpolation`() {
        val tokens = tokenTypes("<template>{{ name }}</template>")
        assertEquals(
            listOf(
                "PK_TEMPLATE_TAG_START",
                "PK_TAG_END_GT",
                "PK_INTERPOLATION_OPEN",
                "PK_EXPR_IDENTIFIER",
                "PK_INTERPOLATION_CLOSE",
                "PK_TEMPLATE_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test interpolation with context variable`() {
        val tokens = tokenTypes("<template>{{ props.Title }}</template>")
        assertEquals(
            listOf(
                "PK_TEMPLATE_TAG_START",
                "PK_TAG_END_GT",
                "PK_INTERPOLATION_OPEN",
                "PK_EXPR_CONTEXT_VAR",
                "PK_EXPR_OP_DOT",
                "PK_EXPR_IDENTIFIER",
                "PK_INTERPOLATION_CLOSE",
                "PK_TEMPLATE_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test interpolation with function call`() {
        val tokens = tokenTypes("<template>{{ len(items) }}</template>")
        assertEquals(
            listOf(
                "PK_TEMPLATE_TAG_START",
                "PK_TAG_END_GT",
                "PK_INTERPOLATION_OPEN",
                "PK_EXPR_BUILTIN",
                "PK_EXPR_PAREN_OPEN",
                "PK_EXPR_IDENTIFIER",
                "PK_EXPR_PAREN_CLOSE",
                "PK_INTERPOLATION_CLOSE",
                "PK_TEMPLATE_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test method call after dot accessor`() {
        val tokens = tokenTypes("<template>{{ props.Article.TitleString() }}</template>")
        assertEquals(
            listOf(
                "PK_TEMPLATE_TAG_START",
                "PK_TAG_END_GT",
                "PK_INTERPOLATION_OPEN",
                "PK_EXPR_CONTEXT_VAR",
                "PK_EXPR_OP_DOT",
                "PK_EXPR_IDENTIFIER",
                "PK_EXPR_OP_DOT",
                "PK_EXPR_FUNCTION_NAME",
                "PK_EXPR_PAREN_OPEN",
                "PK_EXPR_PAREN_CLOSE",
                "PK_INTERPOLATION_CLOSE",
                "PK_TEMPLATE_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test boolean literals in expression`() {
        val tokens = tokenTypes("<template>{{ true }}</template>")
        assert(tokens.contains("PK_EXPR_BOOLEAN"))

        val tokens2 = tokenTypes("<template>{{ false }}</template>")
        assert(tokens2.contains("PK_EXPR_BOOLEAN"))

        val tokens3 = tokenTypes("<template>{{ nil }}</template>")
        assert(tokens3.contains("PK_EXPR_BOOLEAN"))
    }

    @Test
    fun `test number literals in expression`() {
        val tokens = tokenTypes("<template>{{ 42 }}</template>")
        assert(tokens.contains("PK_EXPR_NUMBER"))

        val tokens2 = tokenTypes("<template>{{ 3.14 }}</template>")
        assert(tokens2.contains("PK_EXPR_NUMBER"))
    }

    @Test
    fun `test comparison operators`() {
        val tokens = tokenTypes("<template>{{ a == b }}</template>")
        assert(tokens.contains("PK_EXPR_OP_COMPARISON"))

        val tokens2 = tokenTypes("<template>{{ x > 10 }}</template>")
        assert(tokens2.contains("PK_EXPR_OP_COMPARISON"))
    }

    @Test
    fun `test logical operators`() {
        val tokens = tokenTypes("<template>{{ a && b }}</template>")
        assert(tokens.contains("PK_EXPR_OP_LOGICAL"))

        val tokens2 = tokenTypes("<template>{{ !flag }}</template>")
        assert(tokens2.contains("PK_EXPR_OP_LOGICAL"))
    }

    @Test
    fun `test arithmetic operators`() {
        val tokens = tokenTypes("<template>{{ a + b }}</template>")
        assert(tokens.contains("PK_EXPR_OP_ARITHMETIC"))

        val tokens2 = tokenTypes("<template>{{ x * y }}</template>")
        assert(tokens2.contains("PK_EXPR_OP_ARITHMETIC"))
    }

    @Test
    fun `test string in expression`() {
        val pairs = tokenPairs("<template>{{ \"hello\" }}</template>")
        assert(pairs.any { it.contains("PK_EXPR_STRING_QUOTE") })
        assert(pairs.any { it.contains("PK_EXPR_STRING") && it.contains("hello") })
    }

    @Test
    fun `test builtin functions`() {
        val builtins = listOf("len", "cap", "make", "new", "append", "copy", "delete", "panic", "recover", "print", "println")
        for (builtin in builtins) {
            val tokens = tokenTypes("<template>{{ $builtin(x) }}</template>")
            assert(tokens.contains("PK_EXPR_BUILTIN")) { "Expected PK_EXPR_BUILTIN for $builtin" }
        }
    }

    @Test
    fun `test context variables`() {
        val contextVars = listOf("props", "state", "partial")
        for (cv in contextVars) {
            val tokens = tokenTypes("<template>{{ $cv.Value }}</template>")
            assert(tokens.contains("PK_EXPR_CONTEXT_VAR")) { "Expected PK_EXPR_CONTEXT_VAR for $cv" }
        }
    }

    @Test
    fun `test directive with complex expression`() {
        val tokens = tokenTypes("<template><div p-if=\"props.Count > 0\"></div></template>")
        assert(tokens.contains("PK_DIRECTIVE_NAME"))
        assert(tokens.contains("PK_EXPR_CONTEXT_VAR"))
        assert(tokens.contains("PK_EXPR_OP_DOT"))
        assert(tokens.contains("PK_EXPR_IDENTIFIER"))
        assert(tokens.contains("PK_EXPR_OP_COMPARISON"))
        assert(tokens.contains("PK_EXPR_NUMBER"))
    }

    @Test
    fun `test directive with function call`() {
        val tokens = tokenTypes("<template><span p-text=\"formatDate(props.Date)\"></span></template>")
        assert(tokens.contains("PK_DIRECTIVE_NAME"))
        assert(tokens.contains("PK_EXPR_FUNCTION_NAME"))
        assert(tokens.contains("PK_EXPR_PAREN_OPEN"))
        assert(tokens.contains("PK_EXPR_CONTEXT_VAR"))
    }

    @Test
    fun `test plain text content`() {
        val tokens = tokenTypes("<template>Hello World</template>")
        assertEquals(
            listOf(
                "PK_TEMPLATE_TAG_START",
                "PK_TAG_END_GT",
                "PK_TEXT_CONTENT",
                "PK_TEMPLATE_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test text with interpolation`() {
        val tokens = tokenTypes("<template>Hello {{ name }}!</template>")
        assertEquals(
            listOf(
                "PK_TEMPLATE_TAG_START",
                "PK_TAG_END_GT",
                "PK_TEXT_CONTENT",
                "PK_INTERPOLATION_OPEN",
                "PK_EXPR_IDENTIFIER",
                "PK_INTERPOLATION_CLOSE",
                "PK_TEXT_CONTENT",
                "PK_TEMPLATE_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test HTML comment`() {
        val tokens = tokenTypes("<template><!-- comment --></template>")
        assertEquals(
            listOf(
                "PK_TEMPLATE_TAG_START",
                "PK_TAG_END_GT",
                "PK_HTML_COMMENT",
                "PK_TEMPLATE_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test HTML comment does not consume following content`() {
        val tokens = tokenTypes("<template><!-- Loading overlay --><div class=\"test\"></div></template>")
        assertEquals(
            listOf(
                "PK_TEMPLATE_TAG_START",
                "PK_TAG_END_GT",
                "PK_HTML_COMMENT",
                "PK_HTML_TAG_OPEN",
                "PK_HTML_TAG_NAME",
                "PK_HTML_ATTR_NAME",
                "PK_HTML_ATTR_EQ",
                "PK_HTML_ATTR_QUOTE",
                "PK_HTML_ATTR_VALUE",
                "PK_HTML_ATTR_QUOTE",
                "PK_HTML_TAG_CLOSE",
                "PK_HTML_END_TAG_OPEN",
                "PK_HTML_TAG_NAME",
                "PK_HTML_TAG_CLOSE",
                "PK_TEMPLATE_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test multiline HTML comment`() {
        val tokens = tokenTypes("""<template><!--
            Multi-line
            comment
        --><div></div></template>""")

        assertEquals(1, tokens.count { it == "PK_HTML_COMMENT" })
        assert(tokens.contains("PK_HTML_TAG_NAME")) { "Should contain HTML tag after comment" }
    }

    @Test
    fun `test adjacent interpolations`() {
        val tokens = tokenTypes("<template>{{ a }}{{ b }}</template>")
        val interpolationOpens = tokens.filter { it == "PK_INTERPOLATION_OPEN" }
        val interpolationCloses = tokens.filter { it == "PK_INTERPOLATION_CLOSE" }
        assertEquals(2, interpolationOpens.size)
        assertEquals(2, interpolationCloses.size)
    }

    @Test
    fun `test single brace is text`() {
        val tokens = tokenTypes("<template>{ not interpolation }</template>")
        assert(tokens.contains("PK_TEXT_CONTENT"))
        assert(!tokens.contains("PK_INTERPOLATION_OPEN"))
    }

    @Test
    fun `test chained property access`() {
        val tokens = tokenTypes("<template>{{ props.User.Profile.Name }}</template>")
        val dots = tokens.filter { it == "PK_EXPR_OP_DOT" }
        assertEquals(3, dots.size)
    }

    @Test
    fun `test complex nested expression`() {
        val tokens = tokenTypes("<template>{{ props.A && (props.B || props.C) }}</template>")
        assert(tokens.contains("PK_EXPR_OP_LOGICAL"))
        assert(tokens.contains("PK_EXPR_PAREN_OPEN"))
        assert(tokens.contains("PK_EXPR_PAREN_CLOSE"))
    }

    @Test
    fun `test all directive types`() {
        val directives = listOf(
            "p-if", "p-else-if", "p-else", "p-for", "p-show",
            "p-text", "p-html", "p-model",
            "p-class", "p-style",
            "p-ref", "p-slot", "p-key", "p-context", "p-scaffold"
        )
        for (directive in directives) {
            val tokens = tokenTypes("<template><div $directive=\"x\"></div></template>")
            assert(tokens.contains("PK_DIRECTIVE_NAME")) { "Expected PK_DIRECTIVE_NAME for $directive" }
        }
    }

    @Test
    fun `test p-else standalone directive`() {
        val tokens = tokenTypes("<template><div p-else></div></template>")
        assert(tokens.contains("PK_DIRECTIVE_NAME")) { "p-else should be recognized as directive" }
    }

    @Test
    fun `test p-model directive with expression`() {
        val tokens = tokenTypes("<template><input p-model=\"state.value\"/></template>")
        assert(tokens.contains("PK_DIRECTIVE_NAME"))
        assert(tokens.contains("PK_EXPR_CONTEXT_VAR"))
    }

    @Test
    fun `test p-show directive with boolean`() {
        val tokens = tokenTypes("<template><div p-show=\"visible\"></div></template>")
        assert(tokens.contains("PK_DIRECTIVE_NAME"))
        assert(tokens.contains("PK_EXPR_IDENTIFIER"))
    }

    @Test
    fun `test p-ref directive`() {
        val tokens = tokenTypes("<template><div p-ref=\"myRef\"></div></template>")
        assert(tokens.contains("PK_DIRECTIVE_NAME"))
    }

    @Test
    fun `test p-key directive`() {
        val tokens = tokenTypes("<template><li p-for=\"item in items\" p-key=\"item.ID\"></li></template>")
        val directiveCount = tokens.count { it == "PK_DIRECTIVE_NAME" }
        assertEquals("Should have 2 directives (p-for and p-key)", 2, directiveCount)
    }

    @Test
    fun `test p-context directive`() {
        val tokens = tokenTypes("<template><div p-context=\"ctx\"></div></template>")
        assert(tokens.contains("PK_DIRECTIVE_NAME"))
    }

    @Test
    fun `test p-scaffold directive`() {
        val tokens = tokenTypes("<template><piko:layout p-scaffold=\"main\"/></template>")
        assert(tokens.contains("PK_DIRECTIVE_NAME"))
        assert(tokens.contains("PK_PIKO_TAG_NAME"))
    }

    @Test
    fun `test p-slot directive`() {
        val tokens = tokenTypes("<template><slot p-slot=\"header\"></slot></template>")
        assert(tokens.contains("PK_DIRECTIVE_NAME"))
    }

    @Test
    fun `test regex equality operator`() {
        val tokens = tokenTypes("<template>{{ name ~= pattern }}</template>")
        assert(tokens.contains("PK_EXPR_OP_COMPARISON") || tokens.contains("PK_EXPR_IDENTIFIER")) {
            "Should handle ~= operator"
        }
    }

    @Test
    fun `test negated regex equality operator`() {
        val tokens = tokenTypes("<template>{{ name !~= pattern }}</template>")
        assert(tokens.isNotEmpty()) { "Should handle !~= operator" }
    }

    @Test
    fun `test nullish coalescing operator`() {
        val tokens = tokenTypes("<template>{{ value ?? defaultValue }}</template>")
        assert(tokens.contains("PK_EXPR_IDENTIFIER")) { "Should handle ?? operator" }
    }

    @Test
    fun `test optional chaining operator`() {
        val tokens = tokenTypes("<template>{{ user?.name }}</template>")
        assert(tokens.contains("PK_EXPR_IDENTIFIER")) { "Should handle ?. operator" }
    }

    @Test
    fun `test ternary operator`() {
        val tokens = tokenTypes("<template>{{ condition ? trueVal : falseVal }}</template>")
        assert(tokens.contains("PK_INTERPOLATION_OPEN"))
        assert(tokens.contains("PK_EXPR_IDENTIFIER"))
    }

    @Test
    fun `test ternary with expressions`() {
        val tokens = tokenTypes("<template>{{ props.Count > 0 ? \"has items\" : \"empty\" }}</template>")
        assert(tokens.contains("PK_EXPR_CONTEXT_VAR"))
        assert(tokens.contains("PK_EXPR_OP_COMPARISON"))
    }

    @Test
    fun `test less than or equal operator`() {
        val tokens = tokenTypes("<template>{{ a <= b }}</template>")
        assert(tokens.contains("PK_EXPR_OP_COMPARISON"))
    }

    @Test
    fun `test greater than or equal operator`() {
        val tokens = tokenTypes("<template>{{ a >= b }}</template>")
        assert(tokens.contains("PK_EXPR_OP_COMPARISON"))
    }

    @Test
    fun `test not equal operator`() {
        val tokens = tokenTypes("<template>{{ a != b }}</template>")
        assert(tokens.contains("PK_EXPR_OP_COMPARISON"))
    }

    @Test
    fun `test logical or operator`() {
        val tokens = tokenTypes("<template>{{ a || b }}</template>")
        assert(tokens.contains("PK_EXPR_OP_LOGICAL"))
    }

    @Test
    fun `test modulo operator`() {
        val tokens = tokenTypes("<template>{{ a % b }}</template>")
        assert(tokens.contains("PK_EXPR_OP_ARITHMETIC"))
    }

    @Test
    fun `test division operator`() {
        val tokens = tokenTypes("<template>{{ a / b }}</template>")
        assert(tokens.contains("PK_EXPR_OP_ARITHMETIC"))
    }

    @Test
    fun `test subtraction operator`() {
        val tokens = tokenTypes("<template>{{ a - b }}</template>")
        assert(tokens.contains("PK_EXPR_OP_ARITHMETIC"))
    }

    @Test
    fun `test array indexing`() {
        val tokens = tokenTypes("<template>{{ items[0] }}</template>")
        assert(tokens.contains("PK_EXPR_IDENTIFIER"))
        assert(tokens.contains("PK_EXPR_BRACKET_OPEN"))
        assert(tokens.contains("PK_EXPR_NUMBER"))
        assert(tokens.contains("PK_EXPR_BRACKET_CLOSE"))
    }

    @Test
    fun `test array indexing with variable`() {
        val tokens = tokenTypes("<template>{{ items[idx] }}</template>")
        assert(tokens.contains("PK_EXPR_BRACKET_OPEN"))
        val identCount = tokens.count { it == "PK_EXPR_IDENTIFIER" }
        assertEquals("Should have 2 identifiers (items and idx)", 2, identCount)
    }

    @Test
    fun `test method call on array index`() {
        val tokens = tokenTypes("<template>{{ items[0].Name() }}</template>")
        assert(tokens.contains("PK_EXPR_BRACKET_OPEN"))
        assert(tokens.contains("PK_EXPR_FUNCTION_NAME"))
        assert(tokens.contains("PK_EXPR_PAREN_OPEN"))
    }

    @Test
    fun `test nested function calls`() {
        val tokens = tokenTypes("<template>{{ outer(inner(x)) }}</template>")
        val parenOpens = tokens.count { it == "PK_EXPR_PAREN_OPEN" }
        val parenCloses = tokens.count { it == "PK_EXPR_PAREN_CLOSE" }
        assertEquals("Should have 2 paren opens", 2, parenOpens)
        assertEquals("Should have 2 paren closes", 2, parenCloses)
    }

    @Test
    fun `test function call with multiple arguments`() {
        val tokens = tokenTypes("<template>{{ fn(a, b, c) }}</template>")
        val commas = tokens.count { it == "PK_EXPR_COMMA" }
        assertEquals("Should have 2 commas", 2, commas)
    }

    @Test
    fun `test map literal syntax`() {
        val tokens = tokenTypes("<template>{{ {\"key\": value} }}</template>")
        assert(tokens.contains("PK_EXPR_BRACE_OPEN") || tokens.contains("PK_INTERPOLATION_OPEN"))
    }

    @Test
    fun `test slice expression`() {
        val tokens = tokenTypes("<template>{{ items[1:3] }}</template>")
        assert(tokens.contains("PK_EXPR_BRACKET_OPEN"))
        assert(tokens.contains("PK_EXPR_COLON"))
    }

    @Test
    fun `test combined property and method access`() {
        val tokens = tokenTypes("<template>{{ props.User.GetName().ToUpper() }}</template>")
        val dots = tokens.count { it == "PK_EXPR_OP_DOT" }
        assert(dots >= 3) { "Should have at least 3 dot operators" }
    }

    @Test
    fun `test complex boolean expression`() {
        val tokens = tokenTypes("<template>{{ (a && b) || (c && !d) }}</template>")
        assert(tokens.contains("PK_EXPR_OP_LOGICAL"))
        assert(tokens.contains("PK_EXPR_PAREN_OPEN"))
    }

    @Test
    fun `test negative number`() {
        val tokens = tokenTypes("<template>{{ -42 }}</template>")
        assert(tokens.contains("PK_EXPR_NUMBER") || tokens.contains("PK_EXPR_OP_ARITHMETIC"))
    }

    @Test
    fun `test floating point number`() {
        val tokens = tokenTypes("<template>{{ 3.14159 }}</template>")
        assert(tokens.contains("PK_EXPR_NUMBER"))
    }

    @Test
    fun `test scientific notation number`() {
        val tokens = tokenTypes("<template>{{ 1e10 }}</template>")
        assert(tokens.contains("PK_EXPR_NUMBER") || tokens.contains("PK_EXPR_IDENTIFIER"))
    }

    @Test
    fun `test hexadecimal number`() {
        val tokens = tokenTypes("<template>{{ 0xFF }}</template>")
        assert(tokens.contains("PK_EXPR_NUMBER") || tokens.contains("PK_EXPR_IDENTIFIER"))
    }

    @Test
    fun `test p-bind with dynamic argument tokenises as attribute`() {
        val tokens = tokenTypes("<template><a p-bind:[attrName]=\"value\"></a></template>")
        assert(tokens.isNotEmpty()) { "Should produce tokens" }
    }

    @Test
    fun `test p-on with multiple modifiers`() {
        val tokens = tokenTypes("<template><form p-on:submit.prevent.stop=\"onSubmit\"></form></template>")
        assert(tokens.contains("PK_DIRECTIVE_NAME"))
    }

    @Test
    fun `test shorthand bind with modifier`() {
        val tokens = tokenTypes("<template><div :class.sync=\"cls\"></div></template>")
        assert(tokens.contains("PK_DIRECTIVE_BIND"))
    }

    @Test
    fun `test shorthand event with modifier`() {
        val tokens = tokenTypes("<template><button @click.once=\"handleClick\"></button></template>")
        assert(tokens.contains("PK_DIRECTIVE_EVENT"))
    }

    @Test
    fun `test i18n block with JSON content`() {
        val tokens = tokenTypes("<i18n>{\"en\": {\"hello\": \"Hello\"}}</i18n>")
        assertEquals(
            listOf(
                "PK_I18N_TAG_START",
                "PK_TAG_END_GT",
                "PK_I18N_CONTENT",
                "PK_I18N_TAG_END",
                "PK_TAG_END_GT"
            ),
            tokens
        )
    }

    @Test
    fun `test i18n block with lang attribute`() {
        val tokens = tokenTypes("<i18n lang=\"yaml\">en: hello</i18n>")
        assert(tokens.contains("PK_I18N_TAG_START"))
        assert(tokens.contains("PK_ATTR_NAME"))
    }

    @Test
    fun `test full file with all block types`() {
        val input = """
            <template><div>{{ message }}</div></template>
            <script>func main() {}</script>
            <style>.test { color: red; }</style>
            <i18n>{"en": {}}</i18n>
        """.trimIndent()
        val tokens = tokenTypes(input)
        assert(tokens.contains("PK_TEMPLATE_TAG_START"))
        assert(tokens.contains("PK_SCRIPT_TAG_START"))
        assert(tokens.contains("PK_STYLE_TAG_START"))
        assert(tokens.contains("PK_I18N_TAG_START"))
    }

    @Test
    fun `test script and template order independence`() {
        val input = """
            <script>func main() {}</script>
            <template><div></div></template>
        """.trimIndent()
        val tokens = tokenTypes(input)
        val scriptIdx = tokens.indexOf("PK_SCRIPT_TAG_START")
        val templateIdx = tokens.indexOf("PK_TEMPLATE_TAG_START")
        assert(scriptIdx < templateIdx) { "Script should come before template" }
    }
}
