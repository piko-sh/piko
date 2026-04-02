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

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PKExtendWordSelectionHandlerTest {

    @Test
    fun `isInterpolationContentType returns true for INTERPOLATION_OPEN`() {
        assertTrue(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.INTERPOLATION_OPEN)
        )
    }

    @Test
    fun `isInterpolationContentType returns true for INTERPOLATION_CLOSE`() {
        assertTrue(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.INTERPOLATION_CLOSE)
        )
    }

    @Test
    fun `isInterpolationContentType returns true for EXPR_BOOLEAN`() {
        assertTrue(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.EXPR_BOOLEAN)
        )
    }

    @Test
    fun `isInterpolationContentType returns true for EXPR_NUMBER`() {
        assertTrue(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.EXPR_NUMBER)
        )
    }

    @Test
    fun `isInterpolationContentType returns true for EXPR_STRING`() {
        assertTrue(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.EXPR_STRING)
        )
    }

    @Test
    fun `isInterpolationContentType returns true for EXPR_IDENTIFIER`() {
        assertTrue(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.EXPR_IDENTIFIER)
        )
    }

    @Test
    fun `isInterpolationContentType returns true for EXPR_FUNCTION_NAME`() {
        assertTrue(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.EXPR_FUNCTION_NAME)
        )
    }

    @Test
    fun `isInterpolationContentType returns true for EXPR_CONTEXT_VAR`() {
        assertTrue(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.EXPR_CONTEXT_VAR)
        )
    }

    @Test
    fun `isInterpolationContentType returns true for EXPR_BUILTIN`() {
        assertTrue(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.EXPR_BUILTIN)
        )
    }

    @Test
    fun `isInterpolationContentType returns true for EXPR_OP_DOT`() {
        assertTrue(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.EXPR_OP_DOT)
        )
    }

    @Test
    fun `isInterpolationContentType returns true for EXPR_OP_COMPARISON`() {
        assertTrue(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.EXPR_OP_COMPARISON)
        )
    }

    @Test
    fun `isInterpolationContentType returns true for EXPR_OP_LOGICAL`() {
        assertTrue(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.EXPR_OP_LOGICAL)
        )
    }

    @Test
    fun `isInterpolationContentType returns true for EXPR_OP_ARITHMETIC`() {
        assertTrue(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.EXPR_OP_ARITHMETIC)
        )
    }

    @Test
    fun `isInterpolationContentType returns true for EXPR_PAREN_OPEN`() {
        assertTrue(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.EXPR_PAREN_OPEN)
        )
    }

    @Test
    fun `isInterpolationContentType returns true for EXPR_PAREN_CLOSE`() {
        assertTrue(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.EXPR_PAREN_CLOSE)
        )
    }

    @Test
    fun `isInterpolationContentType returns true for EXPR_COMMA`() {
        assertTrue(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.EXPR_COMMA)
        )
    }

    @Test
    fun `isInterpolationContentType returns true for EXPR_COLON`() {
        assertTrue(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.EXPR_COLON)
        )
    }

    @Test
    fun `isInterpolationContentType returns false for HTML_TAG_NAME`() {
        assertFalse(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.HTML_TAG_NAME)
        )
    }

    @Test
    fun `isInterpolationContentType returns false for TEXT_CONTENT`() {
        assertFalse(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.TEXT_CONTENT)
        )
    }

    @Test
    fun `isInterpolationContentType returns false for TEMPLATE_BLOCK_ELEMENT`() {
        assertFalse(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.TEMPLATE_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `isInterpolationContentType returns false for HTML_ATTR_NAME`() {
        assertFalse(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.HTML_ATTR_NAME)
        )
    }

    @Test
    fun `isInterpolationContentType returns false for DIRECTIVE_NAME`() {
        assertFalse(
            PKExtendWordSelectionHandler.isInterpolationContentType(PKTokenTypes.DIRECTIVE_NAME)
        )
    }
}
