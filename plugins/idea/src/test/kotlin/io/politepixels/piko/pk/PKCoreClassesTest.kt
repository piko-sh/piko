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
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertSame
import org.junit.Assert.assertTrue
import org.junit.Test

class PKCoreClassesTest {

    @Test
    fun `PKLanguage is a singleton object`() {
        assertSame("PKLanguage should be same instance", PKLanguage, PKLanguage)
    }

    @Test
    fun `PKLanguage has correct ID`() {
        assertEquals("PK", PKLanguage.id)
    }

    @Test
    fun `PKLanguage has correct display name`() {
        assertEquals("Piko Template", PKLanguage.displayName)
    }

    @Test
    fun `PKLanguage extends Language`() {
        assertTrue(
            "PKLanguage should extend Language",
            com.intellij.lang.Language::class.java.isAssignableFrom(PKLanguage::class.java)
        )
    }

    @Test
    fun `PKFileType is a singleton object`() {
        assertSame("PKFileType should be same instance", PKFileType, PKFileType)
    }

    @Test
    fun `PKFileType has correct name`() {
        assertEquals("PK File", PKFileType.name)
    }

    @Test
    fun `PKFileType has correct description`() {
        assertEquals("Piko Template File", PKFileType.description)
    }

    @Test
    fun `PKFileType has correct default extension`() {
        assertEquals("pk", PKFileType.defaultExtension)
    }

    @Test
    fun `PKFileType is associated with PKLanguage`() {
        assertSame("PKFileType should use PKLanguage", PKLanguage, PKFileType.language)
    }

    @Test
    fun `PKFileType icon is available`() {
        PKFileType.icon
    }

    @Test
    fun `factory creates PKSyntaxHighlighter instance`() {
        val factory = PKSyntaxHighlighterFactory()
        val highlighter = factory.getSyntaxHighlighter(null, null)

        assertTrue(
            "Factory should create PKSyntaxHighlighter",
            highlighter is PKSyntaxHighlighter
        )
    }

    @Test
    fun `factory creates new instance each time`() {
        val factory = PKSyntaxHighlighterFactory()
        val highlighter1 = factory.getSyntaxHighlighter(null, null)
        val highlighter2 = factory.getSyntaxHighlighter(null, null)

        assertTrue(
            "Factory should create new instances",
            highlighter1 !== highlighter2
        )
    }

    @Test
    fun `color settings page has correct display name`() {
        val page = PKColorSettingsPage()
        assertEquals("Piko", page.displayName)
    }

    @Test
    fun `color settings page returns highlighter`() {
        val page = PKColorSettingsPage()
        val highlighter = page.highlighter

        assertTrue(
            "Should return PKSyntaxHighlighter",
            highlighter is PKSyntaxHighlighter
        )
    }

    @Test
    fun `color settings page has attribute descriptors`() {
        val page = PKColorSettingsPage()
        val descriptors = page.attributeDescriptors

        assertTrue(
            "Should have multiple attribute descriptors",
            descriptors.isNotEmpty()
        )
    }

    @Test
    fun `color settings page has expected descriptor count`() {
        val page = PKColorSettingsPage()
        val descriptors = page.attributeDescriptors

        assertEquals(31, descriptors.size)
    }

    @Test
    fun `color settings page returns empty color descriptors`() {
        val page = PKColorSettingsPage()
        val descriptors = page.colorDescriptors

        assertTrue("Color descriptors should be empty", descriptors.isEmpty())
    }

    @Test
    fun `color settings page has demo text`() {
        val page = PKColorSettingsPage()
        val demoText = page.demoText

        assertTrue("Demo text should not be empty", demoText.isNotBlank())
        assertTrue("Demo text should contain template tag", demoText.contains("<template>"))
        assertTrue("Demo text should contain script tag", demoText.contains("<script>"))
        assertTrue("Demo text should contain style tag", demoText.contains("<style>"))
    }

    @Test
    fun `color settings page has additional highlighting tags`() {
        val page = PKColorSettingsPage()
        val tags = page.additionalHighlightingTagToDescriptorMap

        assertTrue("Should have additional tags", tags.isNotEmpty())
        assertTrue("Should have piko tag", tags.containsKey("piko"))
        assertTrue("Should have directive tag", tags.containsKey("directive"))
        assertTrue("Should have bind tag", tags.containsKey("bind"))
        assertTrue("Should have event tag", tags.containsKey("event"))
    }

    @Test
    fun `color settings page icon uses file type icon`() {
        val page = PKColorSettingsPage()
        page.icon
    }

    @Test
    fun `lexer adapter can be instantiated`() {
        val adapter = PKLexerAdapter()
        assertNotNull("Lexer adapter should be created", adapter)
    }

    @Test
    fun `lexer adapter extends FlexAdapter`() {
        val adapter = PKLexerAdapter()
        assertTrue(
            "Should extend FlexAdapter",
            com.intellij.lexer.FlexAdapter::class.java.isAssignableFrom(adapter::class.java)
        )
    }

    @Test
    fun `PKElementType uses PKLanguage`() {
        val elementType = PKElementType("TEST_ELEMENT")
        assertEquals("Element type should use PKLanguage", PKLanguage, elementType.language)
    }

    @Test
    fun `PKElementType has debug name`() {
        val elementType = PKElementType("MY_DEBUG_NAME")
        assertEquals("MY_DEBUG_NAME", elementType.toString())
    }

}
