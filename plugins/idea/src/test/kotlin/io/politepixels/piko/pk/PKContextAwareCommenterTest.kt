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

class PKContextAwareCommenterTest {

    private val commenter = PKContextAwareCommenter()

    @Test
    fun `HTML_COMMENT_PREFIX is correct`() {
        assertEquals("<!--", PKContextAwareCommenter.HTML_COMMENT_PREFIX)
    }

    @Test
    fun `HTML_COMMENT_SUFFIX is correct`() {
        assertEquals("-->", PKContextAwareCommenter.HTML_COMMENT_SUFFIX)
    }

    @Test
    fun `EXPR_COMMENT_PREFIX is correct`() {
        assertEquals("/*", PKContextAwareCommenter.EXPR_COMMENT_PREFIX)
    }

    @Test
    fun `EXPR_COMMENT_SUFFIX is correct`() {
        assertEquals("*/", PKContextAwareCommenter.EXPR_COMMENT_SUFFIX)
    }

    @Test
    fun `EXPRESSION_DIRECTIVES contains p-if`() {
        assertTrue(PKContextAwareCommenter.EXPRESSION_DIRECTIVES.contains("p-if"))
    }

    @Test
    fun `EXPRESSION_DIRECTIVES contains p-for`() {
        assertTrue(PKContextAwareCommenter.EXPRESSION_DIRECTIVES.contains("p-for"))
    }

    @Test
    fun `EXPRESSION_DIRECTIVES contains p-show`() {
        assertTrue(PKContextAwareCommenter.EXPRESSION_DIRECTIVES.contains("p-show"))
    }

    @Test
    fun `EXPRESSION_DIRECTIVES contains p-model`() {
        assertTrue(PKContextAwareCommenter.EXPRESSION_DIRECTIVES.contains("p-model"))
    }

    @Test
    fun `EXPRESSION_DIRECTIVES does not contain p-key`() {
        assertFalse(PKContextAwareCommenter.EXPRESSION_DIRECTIVES.contains("p-key"))
    }

    @Test
    fun `NON_EXPRESSION_DIRECTIVES contains p-key`() {
        assertTrue(PKContextAwareCommenter.NON_EXPRESSION_DIRECTIVES.contains("p-key"))
    }

    @Test
    fun `NON_EXPRESSION_DIRECTIVES contains p-slot`() {
        assertTrue(PKContextAwareCommenter.NON_EXPRESSION_DIRECTIVES.contains("p-slot"))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_BOOLEAN`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_BOOLEAN))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_NUMBER`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_NUMBER))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_STRING`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_STRING))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_IDENTIFIER`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_IDENTIFIER))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_FUNCTION_NAME`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_FUNCTION_NAME))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_CONTEXT_VAR`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_CONTEXT_VAR))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_BUILTIN`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_BUILTIN))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_OP_DOT`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_OP_DOT))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_OP_COMPARISON`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_OP_COMPARISON))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_OP_LOGICAL`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_OP_LOGICAL))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_OP_ARITHMETIC`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_OP_ARITHMETIC))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_PAREN_OPEN`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_PAREN_OPEN))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_PAREN_CLOSE`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_PAREN_CLOSE))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_BRACKET_OPEN`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_BRACKET_OPEN))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_BRACKET_CLOSE`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_BRACKET_CLOSE))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_BRACE_OPEN`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_BRACE_OPEN))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_BRACE_CLOSE`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_BRACE_CLOSE))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_COMMA`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_COMMA))
    }

    @Test
    fun `isExpressionToken returns true for EXPR_COLON`() {
        assertTrue(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.EXPR_COLON))
    }

    @Test
    fun `isExpressionToken returns false for HTML_TAG_NAME`() {
        assertFalse(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.HTML_TAG_NAME))
    }

    @Test
    fun `isExpressionToken returns false for TEXT_CONTENT`() {
        assertFalse(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.TEXT_CONTENT))
    }

    @Test
    fun `isExpressionToken returns false for TEMPLATE_BLOCK_ELEMENT`() {
        assertFalse(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.TEMPLATE_BLOCK_ELEMENT))
    }

    @Test
    fun `isExpressionToken returns false for INTERPOLATION_OPEN`() {
        assertFalse(PKContextAwareCommenter.isExpressionToken(PKTokenTypes.INTERPOLATION_OPEN))
    }

    @Test
    fun `getLineCommentPrefix returns null`() {
        assertEquals(null, commenter.lineCommentPrefix)
    }

    @Test
    fun `getBlockCommentPrefix returns HTML comment prefix`() {
        assertEquals("<!--", commenter.blockCommentPrefix)
    }

    @Test
    fun `getBlockCommentSuffix returns HTML comment suffix`() {
        assertEquals("-->", commenter.blockCommentSuffix)
    }

    @Test
    fun `getCommentedBlockCommentPrefix returns null`() {
        assertEquals(null, commenter.commentedBlockCommentPrefix)
    }

    @Test
    fun `getCommentedBlockCommentSuffix returns null`() {
        assertEquals(null, commenter.commentedBlockCommentSuffix)
    }
}
