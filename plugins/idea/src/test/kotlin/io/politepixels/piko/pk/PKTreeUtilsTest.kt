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

class PKTreeUtilsTest {

    @Test
    fun `TAG_NAME_TYPES contains expected element types`() {
        assertTrue(
            "Should contain HTML_TAG_NAME",
            PKTreeUtils.TAG_NAME_TYPES.contains(PKTokenTypes.HTML_TAG_NAME)
        )
        assertTrue(
            "Should contain PIKO_TAG_NAME",
            PKTreeUtils.TAG_NAME_TYPES.contains(PKTokenTypes.PIKO_TAG_NAME)
        )
        assertEquals("Should have exactly 2 types", 2, PKTreeUtils.TAG_NAME_TYPES.size)
    }

    @Test
    fun `TAG_CLOSE_TYPES contains expected element types`() {
        assertTrue(
            "Should contain HTML_TAG_CLOSE",
            PKTreeUtils.TAG_CLOSE_TYPES.contains(PKTokenTypes.HTML_TAG_CLOSE)
        )
        assertTrue(
            "Should contain HTML_TAG_SELF_CLOSE",
            PKTreeUtils.TAG_CLOSE_TYPES.contains(PKTokenTypes.HTML_TAG_SELF_CLOSE)
        )
        assertEquals("Should have exactly 2 types", 2, PKTreeUtils.TAG_CLOSE_TYPES.size)
    }

    @Test
    fun `ATTR_NAME_TYPES contains expected element types`() {
        assertTrue(
            "Should contain HTML_ATTR_NAME",
            PKTreeUtils.ATTR_NAME_TYPES.contains(PKTokenTypes.HTML_ATTR_NAME)
        )
        assertTrue(
            "Should contain ATTR_NAME",
            PKTreeUtils.ATTR_NAME_TYPES.contains(PKTokenTypes.ATTR_NAME)
        )
        assertTrue(
            "Should contain DIRECTIVE_NAME",
            PKTreeUtils.ATTR_NAME_TYPES.contains(PKTokenTypes.DIRECTIVE_NAME)
        )
        assertTrue(
            "Should contain DIRECTIVE_BIND",
            PKTreeUtils.ATTR_NAME_TYPES.contains(PKTokenTypes.DIRECTIVE_BIND)
        )
        assertTrue(
            "Should contain DIRECTIVE_EVENT",
            PKTreeUtils.ATTR_NAME_TYPES.contains(PKTokenTypes.DIRECTIVE_EVENT)
        )
        assertEquals("Should have exactly 5 types", 5, PKTreeUtils.ATTR_NAME_TYPES.size)
    }

    @Test
    fun `TAG_NAME_TYPES is immutable set`() {
        val originalSize = PKTreeUtils.TAG_NAME_TYPES.size
        try {
            @Suppress("UNCHECKED_CAST")
            (PKTreeUtils.TAG_NAME_TYPES as MutableSet<Any>).add(PKTokenTypes.TEXT_CONTENT)
        } catch (_: UnsupportedOperationException) {}
        assertEquals("Set should remain unchanged", originalSize, PKTreeUtils.TAG_NAME_TYPES.size)
    }

    @Test
    fun `TAG_CLOSE_TYPES is immutable set`() {
        val originalSize = PKTreeUtils.TAG_CLOSE_TYPES.size
        try {
            @Suppress("UNCHECKED_CAST")
            (PKTreeUtils.TAG_CLOSE_TYPES as MutableSet<Any>).add(PKTokenTypes.TEXT_CONTENT)
        } catch (_: UnsupportedOperationException) {}
        assertEquals("Set should remain unchanged", originalSize, PKTreeUtils.TAG_CLOSE_TYPES.size)
    }

    @Test
    fun `TAG_NAME_TYPES does not contain unrelated types`() {
        assertFalse(
            "Should not contain TEXT_CONTENT",
            PKTreeUtils.TAG_NAME_TYPES.contains(PKTokenTypes.TEXT_CONTENT)
        )
        assertFalse(
            "Should not contain INTERPOLATION_OPEN",
            PKTreeUtils.TAG_NAME_TYPES.contains(PKTokenTypes.INTERPOLATION_OPEN)
        )
    }
}
