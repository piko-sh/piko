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
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class PKEditorFeaturesTest {

    @Test
    fun `PKContextAwareCommenter has no line comment prefix`() {
        val commenter = PKContextAwareCommenter()
        assertNull(commenter.lineCommentPrefix)
    }

    @Test
    fun `PKContextAwareCommenter uses HTML block comment prefix`() {
        val commenter = PKContextAwareCommenter()
        assertEquals("<!--", commenter.blockCommentPrefix)
    }

    @Test
    fun `PKContextAwareCommenter uses HTML block comment suffix`() {
        val commenter = PKContextAwareCommenter()
        assertEquals("-->", commenter.blockCommentSuffix)
    }

    @Test
    fun `PKContextAwareCommenter has no nested comment prefix`() {
        val commenter = PKContextAwareCommenter()
        assertNull(commenter.commentedBlockCommentPrefix)
    }

    @Test
    fun `PKContextAwareCommenter has no nested comment suffix`() {
        val commenter = PKContextAwareCommenter()
        assertNull(commenter.commentedBlockCommentSuffix)
    }

    @Test
    fun `PKBraceMatcher has brace pairs defined`() {
        val matcher = PKBraceMatcher()
        val pairs = matcher.pairs
        assertTrue("Should have multiple brace pairs", pairs.isNotEmpty())
    }

    @Test
    fun `PKBraceMatcher includes interpolation pair`() {
        val matcher = PKBraceMatcher()
        val pairs = matcher.pairs
        val hasInterpolationPair = pairs.any {
            it.leftBraceType == PKTokenTypes.INTERPOLATION_OPEN &&
                it.rightBraceType == PKTokenTypes.INTERPOLATION_CLOSE
        }
        assertTrue("Should have interpolation brace pair", hasInterpolationPair)
    }

    @Test
    fun `PKBraceMatcher includes template interpolation pair`() {
        val matcher = PKBraceMatcher()
        val pairs = matcher.pairs
        val hasTemplateInterpPair = pairs.any {
            it.leftBraceType == PKTokenTypes.TEMPLATE_INTERP_OPEN &&
                it.rightBraceType == PKTokenTypes.TEMPLATE_INTERP_CLOSE
        }
        assertTrue("Should have template interpolation brace pair", hasTemplateInterpPair)
    }

    @Test
    fun `PKBraceMatcher includes parenthesis pair`() {
        val matcher = PKBraceMatcher()
        val pairs = matcher.pairs
        val hasParenPair = pairs.any {
            it.leftBraceType == PKTokenTypes.EXPR_PAREN_OPEN &&
                it.rightBraceType == PKTokenTypes.EXPR_PAREN_CLOSE
        }
        assertTrue("Should have parenthesis brace pair", hasParenPair)
    }

    @Test
    fun `PKBraceMatcher includes bracket pair`() {
        val matcher = PKBraceMatcher()
        val pairs = matcher.pairs
        val hasBracketPair = pairs.any {
            it.leftBraceType == PKTokenTypes.EXPR_BRACKET_OPEN &&
                it.rightBraceType == PKTokenTypes.EXPR_BRACKET_CLOSE
        }
        assertTrue("Should have bracket brace pair", hasBracketPair)
    }

    @Test
    fun `PKBraceMatcher includes curly brace pair`() {
        val matcher = PKBraceMatcher()
        val pairs = matcher.pairs
        val hasBracePair = pairs.any {
            it.leftBraceType == PKTokenTypes.EXPR_BRACE_OPEN &&
                it.rightBraceType == PKTokenTypes.EXPR_BRACE_CLOSE
        }
        assertTrue("Should have curly brace pair", hasBracePair)
    }

    @Test
    fun `PKBraceMatcher has exactly five pairs`() {
        val matcher = PKBraceMatcher()
        assertEquals(5, matcher.pairs.size)
    }

    @Test
    fun `PKBraceMatcher allows paired braces before any type`() {
        val matcher = PKBraceMatcher()
        assertTrue(
            matcher.isPairedBracesAllowedBeforeType(
                PKTokenTypes.INTERPOLATION_OPEN,
                PKTokenTypes.TEXT_CONTENT
            )
        )
    }

    @Test
    fun `PKBraceMatcher allows paired braces before null type`() {
        val matcher = PKBraceMatcher()
        assertTrue(
            matcher.isPairedBracesAllowedBeforeType(
                PKTokenTypes.EXPR_PAREN_OPEN,
                null
            )
        )
    }

    @Test
    fun `PKBraceMatcher code construct start returns same offset`() {
        val matcher = PKBraceMatcher()
        assertEquals(42, matcher.getCodeConstructStart(null, 42))
    }

    @Test
    fun `PKQuoteHandler extends SimpleTokenSetQuoteHandler`() {
        val handler = PKQuoteHandler()
        assertTrue(
            "Should extend SimpleTokenSetQuoteHandler",
            com.intellij.codeInsight.editorActions.SimpleTokenSetQuoteHandler::class.java
                .isAssignableFrom(handler::class.java)
        )
    }

    @Test
    fun `PKQuoteHandler can be instantiated`() {
        val handler = PKQuoteHandler()
        assertNotNull(handler)
    }

    @Test
    fun `PKTypedHandler can be instantiated`() {
        val handler = PKTypedHandler()
        assertNotNull(handler)
    }

    @Test
    fun `PKTypedHandler extends TypedHandlerDelegate`() {
        val handler = PKTypedHandler()
        assertTrue(
            "Should extend TypedHandlerDelegate",
            com.intellij.codeInsight.editorActions.TypedHandlerDelegate::class.java
                .isAssignableFrom(handler::class.java)
        )
    }

    @Test
    fun `PKTemplateContextType can be instantiated`() {
        val contextType = PKTemplateContextType()
        assertNotNull(contextType)
    }

    @Test
    fun `PKTemplateContextType has correct presentation name`() {
        val contextType = PKTemplateContextType()
        assertEquals("Piko", contextType.presentableName)
    }
}
