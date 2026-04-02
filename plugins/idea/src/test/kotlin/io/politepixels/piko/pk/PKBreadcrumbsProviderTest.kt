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
import org.junit.Assert.assertNull
import org.junit.Test

class PKBreadcrumbsProviderTest {

    private val provider = PKBreadcrumbsProvider()

    @Test
    fun `getLanguages returns array containing PKLanguage`() {
        val languages = provider.languages
        assertEquals(1, languages.size)
        assertEquals(PKLanguage, languages[0])
    }

    @Test
    fun `getElementInfoForType returns template for TEMPLATE_BLOCK_ELEMENT`() {
        assertEquals(
            "template",
            PKBreadcrumbsProvider.getElementInfoForType(PKTokenTypes.TEMPLATE_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `getElementInfoForType returns style for STYLE_BLOCK_ELEMENT`() {
        assertEquals(
            "style",
            PKBreadcrumbsProvider.getElementInfoForType(PKTokenTypes.STYLE_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `getElementInfoForType returns i18n for I18N_BLOCK_ELEMENT`() {
        assertEquals(
            "i18n",
            PKBreadcrumbsProvider.getElementInfoForType(PKTokenTypes.I18N_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `getElementInfoForType returns content for TEMPLATE_BODY_ELEMENT`() {
        assertEquals(
            "content",
            PKBreadcrumbsProvider.getElementInfoForType(PKTokenTypes.TEMPLATE_BODY_ELEMENT)
        )
    }

    @Test
    fun `getElementInfoForType returns Go for GO_SCRIPT_BODY_ELEMENT`() {
        assertEquals(
            "Go",
            PKBreadcrumbsProvider.getElementInfoForType(PKTokenTypes.GO_SCRIPT_BODY_ELEMENT)
        )
    }

    @Test
    fun `getElementInfoForType returns JavaScript for JS_SCRIPT_BODY_ELEMENT`() {
        assertEquals(
            "JavaScript",
            PKBreadcrumbsProvider.getElementInfoForType(PKTokenTypes.JS_SCRIPT_BODY_ELEMENT)
        )
    }

    @Test
    fun `getElementInfoForType returns CSS for CSS_STYLE_BODY_ELEMENT`() {
        assertEquals(
            "CSS",
            PKBreadcrumbsProvider.getElementInfoForType(PKTokenTypes.CSS_STYLE_BODY_ELEMENT)
        )
    }

    @Test
    fun `getElementInfoForType returns JSON for I18N_BODY_ELEMENT`() {
        assertEquals(
            "JSON",
            PKBreadcrumbsProvider.getElementInfoForType(PKTokenTypes.I18N_BODY_ELEMENT)
        )
    }

    @Test
    fun `getElementInfoForType returns empty string for unknown type`() {
        assertEquals(
            "",
            PKBreadcrumbsProvider.getElementInfoForType(PKTokenTypes.HTML_TAG_NAME)
        )
    }

    @Test
    fun `getTooltipForType returns Template block for TEMPLATE_BLOCK_ELEMENT`() {
        assertEquals(
            "Template block - HTML content",
            PKBreadcrumbsProvider.getTooltipForType(PKTokenTypes.TEMPLATE_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `getTooltipForType returns Script block for SCRIPT_BLOCK_ELEMENT`() {
        assertEquals(
            "Script block - application logic",
            PKBreadcrumbsProvider.getTooltipForType(PKTokenTypes.SCRIPT_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `getTooltipForType returns null for unknown type`() {
        assertNull(PKBreadcrumbsProvider.getTooltipForType(PKTokenTypes.HTML_TAG_NAME))
    }

    @Test
    fun `isAcceptedElementType returns true for TEMPLATE_BLOCK_ELEMENT`() {
        assertEquals(
            true,
            PKBreadcrumbsProvider.isAcceptedElementType(PKTokenTypes.TEMPLATE_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `isAcceptedElementType returns true for SCRIPT_BLOCK_ELEMENT`() {
        assertEquals(
            true,
            PKBreadcrumbsProvider.isAcceptedElementType(PKTokenTypes.SCRIPT_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `isAcceptedElementType returns true for body elements`() {
        assertEquals(
            true,
            PKBreadcrumbsProvider.isAcceptedElementType(PKTokenTypes.TEMPLATE_BODY_ELEMENT)
        )
        assertEquals(
            true,
            PKBreadcrumbsProvider.isAcceptedElementType(PKTokenTypes.GO_SCRIPT_BODY_ELEMENT)
        )
    }

    @Test
    fun `isAcceptedElementType returns false for token elements`() {
        assertEquals(
            false,
            PKBreadcrumbsProvider.isAcceptedElementType(PKTokenTypes.HTML_TAG_NAME)
        )
        assertEquals(
            false,
            PKBreadcrumbsProvider.isAcceptedElementType(PKTokenTypes.TEXT_CONTENT)
        )
    }
}
