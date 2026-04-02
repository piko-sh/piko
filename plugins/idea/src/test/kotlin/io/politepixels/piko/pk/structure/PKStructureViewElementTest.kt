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


package io.politepixels.piko.pk.structure

import io.politepixels.piko.pk.PKTokenTypes
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class PKStructureViewElementTest {

    @Test
    fun `getNameForElementType returns template tag for TEMPLATE_BLOCK_ELEMENT`() {
        assertEquals(
            "<template>",
            PKStructureViewElement.getNameForElementType(PKTokenTypes.TEMPLATE_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `getNameForElementType returns script tag for SCRIPT_BLOCK_ELEMENT`() {
        assertEquals(
            "<script>",
            PKStructureViewElement.getNameForElementType(PKTokenTypes.SCRIPT_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `getNameForElementType returns style tag for STYLE_BLOCK_ELEMENT`() {
        assertEquals(
            "<style>",
            PKStructureViewElement.getNameForElementType(PKTokenTypes.STYLE_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `getNameForElementType returns i18n tag for I18N_BLOCK_ELEMENT`() {
        assertEquals(
            "<i18n>",
            PKStructureViewElement.getNameForElementType(PKTokenTypes.I18N_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `getNameForElementType returns content for TEMPLATE_BODY_ELEMENT`() {
        assertEquals(
            "content",
            PKStructureViewElement.getNameForElementType(PKTokenTypes.TEMPLATE_BODY_ELEMENT)
        )
    }

    @Test
    fun `getNameForElementType returns Go code for GO_SCRIPT_BODY_ELEMENT`() {
        assertEquals(
            "Go code",
            PKStructureViewElement.getNameForElementType(PKTokenTypes.GO_SCRIPT_BODY_ELEMENT)
        )
    }

    @Test
    fun `getNameForElementType returns JavaScript code for JS_SCRIPT_BODY_ELEMENT`() {
        assertEquals(
            "JavaScript code",
            PKStructureViewElement.getNameForElementType(PKTokenTypes.JS_SCRIPT_BODY_ELEMENT)
        )
    }

    @Test
    fun `getNameForElementType returns CSS rules for CSS_STYLE_BODY_ELEMENT`() {
        assertEquals(
            "CSS rules",
            PKStructureViewElement.getNameForElementType(PKTokenTypes.CSS_STYLE_BODY_ELEMENT)
        )
    }

    @Test
    fun `getNameForElementType returns translations for I18N_BODY_ELEMENT`() {
        assertEquals(
            "translations",
            PKStructureViewElement.getNameForElementType(PKTokenTypes.I18N_BODY_ELEMENT)
        )
    }

    @Test
    fun `getNameForElementType returns empty string for unknown type`() {
        assertEquals(
            "",
            PKStructureViewElement.getNameForElementType(PKTokenTypes.HTML_TAG_NAME)
        )
    }

    @Test
    fun `getLocationForElementType returns Go for GO_SCRIPT_BODY_ELEMENT`() {
        assertEquals(
            "Go",
            PKStructureViewElement.getLocationForElementType(PKTokenTypes.GO_SCRIPT_BODY_ELEMENT)
        )
    }

    @Test
    fun `getLocationForElementType returns JavaScript for JS_SCRIPT_BODY_ELEMENT`() {
        assertEquals(
            "JavaScript",
            PKStructureViewElement.getLocationForElementType(PKTokenTypes.JS_SCRIPT_BODY_ELEMENT)
        )
    }

    @Test
    fun `getLocationForElementType returns CSS for CSS_STYLE_BODY_ELEMENT`() {
        assertEquals(
            "CSS",
            PKStructureViewElement.getLocationForElementType(PKTokenTypes.CSS_STYLE_BODY_ELEMENT)
        )
    }

    @Test
    fun `getLocationForElementType returns JSON for I18N_BODY_ELEMENT`() {
        assertEquals(
            "JSON",
            PKStructureViewElement.getLocationForElementType(PKTokenTypes.I18N_BODY_ELEMENT)
        )
    }

    @Test
    fun `getLocationForElementType returns null for TEMPLATE_BLOCK_ELEMENT`() {
        assertNull(
            PKStructureViewElement.getLocationForElementType(PKTokenTypes.TEMPLATE_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `getLocationForElementType returns null for SCRIPT_BLOCK_ELEMENT`() {
        assertNull(
            PKStructureViewElement.getLocationForElementType(PKTokenTypes.SCRIPT_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `getLocationForElementType returns null for unknown type`() {
        assertNull(
            PKStructureViewElement.getLocationForElementType(PKTokenTypes.HTML_TAG_NAME)
        )
    }

    @Test
    fun `isBlockElementType returns true for TEMPLATE_BLOCK_ELEMENT`() {
        assertTrue(
            PKStructureViewElement.isBlockElementType(PKTokenTypes.TEMPLATE_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `isBlockElementType returns true for SCRIPT_BLOCK_ELEMENT`() {
        assertTrue(
            PKStructureViewElement.isBlockElementType(PKTokenTypes.SCRIPT_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `isBlockElementType returns true for STYLE_BLOCK_ELEMENT`() {
        assertTrue(
            PKStructureViewElement.isBlockElementType(PKTokenTypes.STYLE_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `isBlockElementType returns true for I18N_BLOCK_ELEMENT`() {
        assertTrue(
            PKStructureViewElement.isBlockElementType(PKTokenTypes.I18N_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `isBlockElementType returns false for body elements`() {
        assertFalse(
            PKStructureViewElement.isBlockElementType(PKTokenTypes.TEMPLATE_BODY_ELEMENT)
        )
        assertFalse(
            PKStructureViewElement.isBlockElementType(PKTokenTypes.GO_SCRIPT_BODY_ELEMENT)
        )
    }

    @Test
    fun `isBlockElementType returns false for token elements`() {
        assertFalse(
            PKStructureViewElement.isBlockElementType(PKTokenTypes.HTML_TAG_NAME)
        )
    }

    @Test
    fun `isBodyElementType returns true for TEMPLATE_BODY_ELEMENT`() {
        assertTrue(
            PKStructureViewElement.isBodyElementType(PKTokenTypes.TEMPLATE_BODY_ELEMENT)
        )
    }

    @Test
    fun `isBodyElementType returns true for GO_SCRIPT_BODY_ELEMENT`() {
        assertTrue(
            PKStructureViewElement.isBodyElementType(PKTokenTypes.GO_SCRIPT_BODY_ELEMENT)
        )
    }

    @Test
    fun `isBodyElementType returns true for JS_SCRIPT_BODY_ELEMENT`() {
        assertTrue(
            PKStructureViewElement.isBodyElementType(PKTokenTypes.JS_SCRIPT_BODY_ELEMENT)
        )
    }

    @Test
    fun `isBodyElementType returns true for CSS_STYLE_BODY_ELEMENT`() {
        assertTrue(
            PKStructureViewElement.isBodyElementType(PKTokenTypes.CSS_STYLE_BODY_ELEMENT)
        )
    }

    @Test
    fun `isBodyElementType returns true for I18N_BODY_ELEMENT`() {
        assertTrue(
            PKStructureViewElement.isBodyElementType(PKTokenTypes.I18N_BODY_ELEMENT)
        )
    }

    @Test
    fun `isBodyElementType returns false for block elements`() {
        assertFalse(
            PKStructureViewElement.isBodyElementType(PKTokenTypes.TEMPLATE_BLOCK_ELEMENT)
        )
        assertFalse(
            PKStructureViewElement.isBodyElementType(PKTokenTypes.SCRIPT_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `isBodyElementType returns false for token elements`() {
        assertFalse(
            PKStructureViewElement.isBodyElementType(PKTokenTypes.HTML_TAG_NAME)
        )
    }
}
