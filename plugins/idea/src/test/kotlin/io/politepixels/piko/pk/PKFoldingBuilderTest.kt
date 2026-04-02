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
import org.junit.Test

class PKFoldingBuilderTest {

    @Test
    fun `getPlaceholderForElementType returns template placeholder for TEMPLATE_BLOCK_ELEMENT`() {
        assertEquals(
            "<template>...",
            PKFoldingBuilder.getPlaceholderForElementType(PKTokenTypes.TEMPLATE_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `getPlaceholderForElementType returns script placeholder for SCRIPT_BLOCK_ELEMENT`() {
        assertEquals(
            "<script>...",
            PKFoldingBuilder.getPlaceholderForElementType(PKTokenTypes.SCRIPT_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `getPlaceholderForElementType returns style placeholder for STYLE_BLOCK_ELEMENT`() {
        assertEquals(
            "<style>...",
            PKFoldingBuilder.getPlaceholderForElementType(PKTokenTypes.STYLE_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `getPlaceholderForElementType returns i18n placeholder for I18N_BLOCK_ELEMENT`() {
        assertEquals(
            "<i18n>...",
            PKFoldingBuilder.getPlaceholderForElementType(PKTokenTypes.I18N_BLOCK_ELEMENT)
        )
    }

    @Test
    fun `getPlaceholderForElementType returns comment placeholder for HTML_COMMENT`() {
        assertEquals(
            "<!--...-->",
            PKFoldingBuilder.getPlaceholderForElementType(PKTokenTypes.HTML_COMMENT)
        )
    }

    @Test
    fun `getPlaceholderForElementType returns interpolation placeholder for INTERPOLATION_OPEN`() {
        assertEquals(
            "{{...}}",
            PKFoldingBuilder.getPlaceholderForElementType(PKTokenTypes.INTERPOLATION_OPEN)
        )
    }

    @Test
    fun `getPlaceholderForElementType returns tag placeholder for HTML_TAG_OPEN`() {
        assertEquals(
            "<...>",
            PKFoldingBuilder.getPlaceholderForElementType(PKTokenTypes.HTML_TAG_OPEN)
        )
    }

    @Test
    fun `getPlaceholderForElementType returns ellipsis for unknown type`() {
        assertEquals(
            "...",
            PKFoldingBuilder.getPlaceholderForElementType(PKTokenTypes.TEXT_CONTENT)
        )
    }

    @Test
    fun `getPlaceholderForElementType returns ellipsis for HTML_TAG_NAME`() {
        assertEquals(
            "...",
            PKFoldingBuilder.getPlaceholderForElementType(PKTokenTypes.HTML_TAG_NAME)
        )
    }
}
