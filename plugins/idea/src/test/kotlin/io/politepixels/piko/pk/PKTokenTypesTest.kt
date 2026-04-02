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

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PKTokenTypesTest {

    @Test
    fun `COMMENTS contains HTML_COMMENT`() {
        assertTrue(
            "COMMENTS should contain HTML_COMMENT",
            PKTokenTypes.COMMENTS.contains(PKTokenTypes.HTML_COMMENT)
        )
    }

    @Test
    fun `COMMENTS does not contain non-comment tokens`() {
        assertFalse(
            "COMMENTS should not contain TEXT_CONTENT",
            PKTokenTypes.COMMENTS.contains(PKTokenTypes.TEXT_CONTENT)
        )
        assertFalse(
            "COMMENTS should not contain ATTR_VALUE",
            PKTokenTypes.COMMENTS.contains(PKTokenTypes.ATTR_VALUE)
        )
    }

    @Test
    fun `STRING_LITERALS contains ATTR_VALUE`() {
        assertTrue(
            "STRING_LITERALS should contain ATTR_VALUE",
            PKTokenTypes.STRING_LITERALS.contains(PKTokenTypes.ATTR_VALUE)
        )
    }

    @Test
    fun `STRING_LITERALS contains HTML_ATTR_VALUE`() {
        assertTrue(
            "STRING_LITERALS should contain HTML_ATTR_VALUE",
            PKTokenTypes.STRING_LITERALS.contains(PKTokenTypes.HTML_ATTR_VALUE)
        )
    }

    @Test
    fun `STRING_LITERALS contains EXPR_STRING`() {
        assertTrue(
            "STRING_LITERALS should contain EXPR_STRING",
            PKTokenTypes.STRING_LITERALS.contains(PKTokenTypes.EXPR_STRING)
        )
    }

    @Test
    fun `STRING_LITERALS does not contain quotes or escapes`() {
        assertFalse(
            "STRING_LITERALS should not contain EXPR_STRING_QUOTE",
            PKTokenTypes.STRING_LITERALS.contains(PKTokenTypes.EXPR_STRING_QUOTE)
        )
        assertFalse(
            "STRING_LITERALS should not contain EXPR_ESCAPE",
            PKTokenTypes.STRING_LITERALS.contains(PKTokenTypes.EXPR_ESCAPE)
        )
    }

    @Test
    fun `KEYWORDS contains all structural tag starts`() {
        assertTrue(
            "KEYWORDS should contain TEMPLATE_TAG_START",
            PKTokenTypes.KEYWORDS.contains(PKTokenTypes.TEMPLATE_TAG_START)
        )
        assertTrue(
            "KEYWORDS should contain SCRIPT_TAG_START",
            PKTokenTypes.KEYWORDS.contains(PKTokenTypes.SCRIPT_TAG_START)
        )
        assertTrue(
            "KEYWORDS should contain STYLE_TAG_START",
            PKTokenTypes.KEYWORDS.contains(PKTokenTypes.STYLE_TAG_START)
        )
        assertTrue(
            "KEYWORDS should contain I18N_TAG_START",
            PKTokenTypes.KEYWORDS.contains(PKTokenTypes.I18N_TAG_START)
        )
    }

    @Test
    fun `KEYWORDS contains all structural tag ends`() {
        assertTrue(
            "KEYWORDS should contain TEMPLATE_TAG_END",
            PKTokenTypes.KEYWORDS.contains(PKTokenTypes.TEMPLATE_TAG_END)
        )
        assertTrue(
            "KEYWORDS should contain SCRIPT_TAG_END",
            PKTokenTypes.KEYWORDS.contains(PKTokenTypes.SCRIPT_TAG_END)
        )
        assertTrue(
            "KEYWORDS should contain STYLE_TAG_END",
            PKTokenTypes.KEYWORDS.contains(PKTokenTypes.STYLE_TAG_END)
        )
        assertTrue(
            "KEYWORDS should contain I18N_TAG_END",
            PKTokenTypes.KEYWORDS.contains(PKTokenTypes.I18N_TAG_END)
        )
    }

    @Test
    fun `KEYWORDS contains directive tokens`() {
        assertTrue(
            "KEYWORDS should contain DIRECTIVE_NAME",
            PKTokenTypes.KEYWORDS.contains(PKTokenTypes.DIRECTIVE_NAME)
        )
        assertTrue(
            "KEYWORDS should contain DIRECTIVE_BIND",
            PKTokenTypes.KEYWORDS.contains(PKTokenTypes.DIRECTIVE_BIND)
        )
        assertTrue(
            "KEYWORDS should contain DIRECTIVE_EVENT",
            PKTokenTypes.KEYWORDS.contains(PKTokenTypes.DIRECTIVE_EVENT)
        )
    }

    @Test
    fun `KEYWORDS contains HTML_TAG_NAME`() {
        assertTrue(
            "KEYWORDS should contain HTML_TAG_NAME",
            PKTokenTypes.KEYWORDS.contains(PKTokenTypes.HTML_TAG_NAME)
        )
    }

    @Test
    fun `KEYWORDS contains EXPR_BOOLEAN`() {
        assertTrue(
            "KEYWORDS should contain EXPR_BOOLEAN",
            PKTokenTypes.KEYWORDS.contains(PKTokenTypes.EXPR_BOOLEAN)
        )
    }

    @Test
    fun `BRACKETS contains all bracket and paren tokens`() {
        assertTrue(
            "BRACKETS should contain EXPR_PAREN_OPEN",
            PKTokenTypes.BRACKETS.contains(PKTokenTypes.EXPR_PAREN_OPEN)
        )
        assertTrue(
            "BRACKETS should contain EXPR_PAREN_CLOSE",
            PKTokenTypes.BRACKETS.contains(PKTokenTypes.EXPR_PAREN_CLOSE)
        )
        assertTrue(
            "BRACKETS should contain EXPR_BRACKET_OPEN",
            PKTokenTypes.BRACKETS.contains(PKTokenTypes.EXPR_BRACKET_OPEN)
        )
        assertTrue(
            "BRACKETS should contain EXPR_BRACKET_CLOSE",
            PKTokenTypes.BRACKETS.contains(PKTokenTypes.EXPR_BRACKET_CLOSE)
        )
        assertTrue(
            "BRACKETS should contain EXPR_BRACE_OPEN",
            PKTokenTypes.BRACKETS.contains(PKTokenTypes.EXPR_BRACE_OPEN)
        )
        assertTrue(
            "BRACKETS should contain EXPR_BRACE_CLOSE",
            PKTokenTypes.BRACKETS.contains(PKTokenTypes.EXPR_BRACE_CLOSE)
        )
    }

    @Test
    fun `BRACKETS contains interpolation tokens`() {
        assertTrue(
            "BRACKETS should contain INTERPOLATION_OPEN",
            PKTokenTypes.BRACKETS.contains(PKTokenTypes.INTERPOLATION_OPEN)
        )
        assertTrue(
            "BRACKETS should contain INTERPOLATION_CLOSE",
            PKTokenTypes.BRACKETS.contains(PKTokenTypes.INTERPOLATION_CLOSE)
        )
    }

    @Test
    fun `BRACKETS contains HTML tag tokens`() {
        assertTrue(
            "BRACKETS should contain HTML_TAG_OPEN",
            PKTokenTypes.BRACKETS.contains(PKTokenTypes.HTML_TAG_OPEN)
        )
        assertTrue(
            "BRACKETS should contain HTML_TAG_CLOSE",
            PKTokenTypes.BRACKETS.contains(PKTokenTypes.HTML_TAG_CLOSE)
        )
        assertTrue(
            "BRACKETS should contain HTML_TAG_SELF_CLOSE",
            PKTokenTypes.BRACKETS.contains(PKTokenTypes.HTML_TAG_SELF_CLOSE)
        )
        assertTrue(
            "BRACKETS should contain HTML_END_TAG_OPEN",
            PKTokenTypes.BRACKETS.contains(PKTokenTypes.HTML_END_TAG_OPEN)
        )
        assertTrue(
            "BRACKETS should contain TAG_END_GT",
            PKTokenTypes.BRACKETS.contains(PKTokenTypes.TAG_END_GT)
        )
    }

    @Test
    fun `OPERATORS contains all operator tokens`() {
        assertTrue(
            "OPERATORS should contain EXPR_OP_COMPARISON",
            PKTokenTypes.OPERATORS.contains(PKTokenTypes.EXPR_OP_COMPARISON)
        )
        assertTrue(
            "OPERATORS should contain EXPR_OP_LOGICAL",
            PKTokenTypes.OPERATORS.contains(PKTokenTypes.EXPR_OP_LOGICAL)
        )
        assertTrue(
            "OPERATORS should contain EXPR_OP_ARITHMETIC",
            PKTokenTypes.OPERATORS.contains(PKTokenTypes.EXPR_OP_ARITHMETIC)
        )
        assertTrue(
            "OPERATORS should contain EXPR_OP_DOT",
            PKTokenTypes.OPERATORS.contains(PKTokenTypes.EXPR_OP_DOT)
        )
    }

    @Test
    fun `OPERATORS does not contain non-operator tokens`() {
        assertFalse(
            "OPERATORS should not contain EXPR_NUMBER",
            PKTokenTypes.OPERATORS.contains(PKTokenTypes.EXPR_NUMBER)
        )
        assertFalse(
            "OPERATORS should not contain EXPR_IDENTIFIER",
            PKTokenTypes.OPERATORS.contains(PKTokenTypes.EXPR_IDENTIFIER)
        )
    }

    @Test
    fun `PKElementType registers with PKLanguage`() {
        assertEquals(
            "Element type should be registered with PKLanguage",
            PKLanguage,
            PKTokenTypes.TEXT_CONTENT.language
        )
    }

    @Test
    fun `all element types have unique debug names`() {
        val debugNames = listOf(
            PKTokenTypes.TEMPLATE_TAG_START,
            PKTokenTypes.SCRIPT_TAG_START,
            PKTokenTypes.STYLE_TAG_START,
            PKTokenTypes.I18N_TAG_START,
            PKTokenTypes.TAG_END_GT,
            PKTokenTypes.TEXT_CONTENT,
            PKTokenTypes.HTML_COMMENT
        ).map { it.toString() }

        assertEquals(
            "All debug names should be unique",
            debugNames.size,
            debugNames.toSet().size
        )
    }

    @Test
    fun `body element types are distinct from block element types`() {
        val blockElements = setOf(
            PKTokenTypes.TEMPLATE_BLOCK_ELEMENT,
            PKTokenTypes.SCRIPT_BLOCK_ELEMENT,
            PKTokenTypes.STYLE_BLOCK_ELEMENT,
            PKTokenTypes.I18N_BLOCK_ELEMENT
        )
        val bodyElements = setOf(
            PKTokenTypes.TEMPLATE_BODY_ELEMENT,
            PKTokenTypes.GO_SCRIPT_BODY_ELEMENT,
            PKTokenTypes.JS_SCRIPT_BODY_ELEMENT,
            PKTokenTypes.CSS_STYLE_BODY_ELEMENT,
            PKTokenTypes.I18N_BODY_ELEMENT
        )

        assertTrue(
            "Block and body element sets should be disjoint",
            blockElements.intersect(bodyElements).isEmpty()
        )
    }

    @Test
    fun `script body elements are distinct for different languages`() {
        assertTrue(
            "GO_SCRIPT_BODY_ELEMENT should not equal JS_SCRIPT_BODY_ELEMENT",
            PKTokenTypes.GO_SCRIPT_BODY_ELEMENT != PKTokenTypes.JS_SCRIPT_BODY_ELEMENT
        )
    }
}
