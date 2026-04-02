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


package io.politepixels.piko.pk.css

import com.intellij.psi.TokenType
import com.intellij.psi.tree.IElementType
import io.politepixels.gen.pk.PKLexer
import io.politepixels.piko.pk.PKTokenTypes
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Tests that the PK lexer produces the correct token patterns for all
 * CSS class reference syntaxes used in templates.
 *
 * These tests verify the lexer-level foundation that [PKCssClassExtractor]
 * relies on to walk template body tokens and extract class names.
 */
class PKCssClassTokenPatternsTest {

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

    private fun tokenPairs(input: String): List<String> {
        return tokenize(input)
            .filter { it.first != TokenType.WHITE_SPACE }
            .map { "${it.first}('${it.second}')" }
    }

    private fun tokenTypes(input: String): List<String> {
        return tokenize(input)
            .filter { it.first != TokenType.WHITE_SPACE }
            .map { it.first.toString() }
    }

    @Test
    fun `static class attribute produces HTML_ATTR_NAME and HTML_ATTR_VALUE`() {
        val tokens = tokenPairs("<template><div class=\"card-img\"></div></template>")
        assertTrue(
            "Should contain HTML_ATTR_NAME('class')",
            tokens.contains("PK_HTML_ATTR_NAME('class')")
        )
        assertTrue(
            "Should contain HTML_ATTR_VALUE('card-img')",
            tokens.contains("PK_HTML_ATTR_VALUE('card-img')")
        )
    }

    @Test
    fun `static class with multiple classes produces single HTML_ATTR_VALUE`() {
        val tokens = tokenPairs("<template><div class=\"foo bar baz\"></div></template>")
        assertTrue(
            "Should contain the full value",
            tokens.contains("PK_HTML_ATTR_VALUE('foo bar baz')")
        )
    }

    @Test
    fun `p-class shorthand is tokenised as HTML_ATTR_NAME`() {
        val tokens = tokenPairs("<template><div p-class:active=\"isActive\"></div></template>")
        assertTrue(
            "p-class:active should be a single HTML_ATTR_NAME token",
            tokens.contains("PK_HTML_ATTR_NAME('p-class:active')")
        )
    }

    @Test
    fun `p-class shorthand with hyphenated name is single HTML_ATTR_NAME`() {
        val tokens = tokenPairs("<template><div p-class:is-visible=\"show\"></div></template>")
        assertTrue(
            "p-class:is-visible should be a single HTML_ATTR_NAME token",
            tokens.contains("PK_HTML_ATTR_NAME('p-class:is-visible')")
        )
    }

    @Test
    fun `p-class directive produces DIRECTIVE_NAME`() {
        val tokens = tokenPairs("<template><div p-class=\"{ 'active': true }\"></div></template>")
        assertTrue(
            "p-class should be DIRECTIVE_NAME",
            tokens.contains("PK_DIRECTIVE_NAME('p-class')")
        )
    }

    @Test
    fun `p-class object syntax contains EXPR_STRING for class names`() {
        val tokens = tokenPairs("<template><div p-class=\"{ 'active': true, 'hidden': false }\"></div></template>")
        assertTrue(
            "Should contain EXPR_STRING('active')",
            tokens.contains("PK_EXPR_STRING('active')")
        )
        assertTrue(
            "Should contain EXPR_STRING('hidden')",
            tokens.contains("PK_EXPR_STRING('hidden')")
        )
    }

    @Test
    fun `bind class directive produces DIRECTIVE_BIND`() {
        val tokens = tokenPairs("<template><div :class=\"'active'\"></div></template>")
        assertTrue(
            ":class should be DIRECTIVE_BIND",
            tokens.contains("PK_DIRECTIVE_BIND(':class')")
        )
    }

    @Test
    fun `bind class with string literal produces EXPR_STRING`() {
        val tokens = tokenPairs("<template><div :class=\"'card-img'\"></div></template>")
        assertTrue(
            "Should contain EXPR_STRING('card-img')",
            tokens.contains("PK_EXPR_STRING('card-img')")
        )
    }

    @Test
    fun `bind class with object syntax produces EXPR_STRING tokens`() {
        val tokens = tokenPairs("<template><div :class=\"{ 'highlight': isActive }\"></div></template>")
        assertTrue(
            "Should contain EXPR_STRING('highlight')",
            tokens.contains("PK_EXPR_STRING('highlight')")
        )
    }

    @Test
    fun `static class token sequence follows expected order`() {
        val tokens = tokenTypes("<template><div class=\"test\"></div></template>")
        val classIdx = tokens.indexOf("PK_HTML_ATTR_NAME")
        val eqIdx = tokens.indexOf("PK_HTML_ATTR_EQ")
        val quoteIdx = tokens.indexOf("PK_HTML_ATTR_QUOTE")
        val valueIdx = tokens.indexOf("PK_HTML_ATTR_VALUE")

        assertTrue("class attr name should come before eq", classIdx < eqIdx)
        assertTrue("eq should come before quote", eqIdx < quoteIdx)
        assertTrue("quote should come before value", quoteIdx < valueIdx)
    }

    @Test
    fun `p-class directive token sequence includes EXPR_BRACE tokens`() {
        val tokens = tokenTypes("<template><div p-class=\"{ 'x': true }\"></div></template>")
        assertTrue(
            "Should contain EXPR_BRACE_OPEN for object syntax",
            tokens.contains("PK_EXPR_BRACE_OPEN")
        )
        assertTrue(
            "Should contain EXPR_BRACE_CLOSE for object syntax",
            tokens.contains("PK_EXPR_BRACE_CLOSE")
        )
    }

    @Test
    fun `p-class with array syntax contains EXPR_STRING`() {
        val tokens = tokenPairs("<template><div p-class=\"['card', 'active']\"></div></template>")
        assertTrue(
            "Should contain EXPR_STRING('card')",
            tokens.contains("PK_EXPR_STRING('card')")
        )
        assertTrue(
            "Should contain EXPR_STRING('active')",
            tokens.contains("PK_EXPR_STRING('active')")
        )
    }

    @Test
    fun `p-class shorthand value is in plain attr context`() {
        val tokens = tokenTypes("<template><div p-class:hover=\"isHovered\"></div></template>")
        assertTrue(
            "Value should be a plain HTML attr value (JFlex longest-match catch-all)",
            tokens.contains("PK_HTML_ATTR_VALUE")
        )
    }

    @Test
    fun `multiple class patterns in same template produce correct tokens`() {
        val input = """<template><div class="static" p-class:dynamic="cond" :class="'bound'"></div></template>"""
        val tokens = tokenPairs(input)

        assertTrue(
            "Should have static class attr name",
            tokens.contains("PK_HTML_ATTR_NAME('class')")
        )
        assertTrue(
            "Should have static class value",
            tokens.contains("PK_HTML_ATTR_VALUE('static')")
        )
        assertTrue(
            "Should have p-class shorthand",
            tokens.contains("PK_HTML_ATTR_NAME('p-class:dynamic')")
        )
        assertTrue(
            "Should have :class bind",
            tokens.contains("PK_DIRECTIVE_BIND(':class')")
        )
        assertTrue(
            "Should have bound class string",
            tokens.contains("PK_EXPR_STRING('bound')")
        )
    }
}
