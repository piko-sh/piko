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

class PKConstantsTest {

    @Test
    fun `ScriptLang JS_LANGUAGES contains expected values`() {
        assertTrue("Should contain js", PKConstants.ScriptLang.JS in PKConstants.ScriptLang.JS_LANGUAGES)
        assertTrue("Should contain javascript", PKConstants.ScriptLang.JAVASCRIPT in PKConstants.ScriptLang.JS_LANGUAGES)
        assertTrue("Should contain typescript", PKConstants.ScriptLang.TYPESCRIPT in PKConstants.ScriptLang.JS_LANGUAGES)
        assertEquals("Should have exactly 3 languages", 3, PKConstants.ScriptLang.JS_LANGUAGES.size)
    }

    @Test
    fun `ScriptLang isJsLanguage returns true for valid languages`() {
        assertTrue("js should be JS language", PKConstants.ScriptLang.isJsLanguage("js"))
        assertTrue("javascript should be JS language", PKConstants.ScriptLang.isJsLanguage("javascript"))
        assertTrue("typescript should be JS language", PKConstants.ScriptLang.isJsLanguage("typescript"))
    }

    @Test
    fun `ScriptLang isJsLanguage returns false for invalid languages`() {
        assertFalse("go should not be JS language", PKConstants.ScriptLang.isJsLanguage("go"))
        assertFalse("python should not be JS language", PKConstants.ScriptLang.isJsLanguage("python"))
        assertFalse("empty should not be JS language", PKConstants.ScriptLang.isJsLanguage(""))
    }

    @Test
    fun `FoldingPlaceholder values are correct`() {
        assertEquals("<template>...", PKConstants.FoldingPlaceholder.TEMPLATE)
        assertEquals("<script>...", PKConstants.FoldingPlaceholder.SCRIPT)
        assertEquals("<style>...", PKConstants.FoldingPlaceholder.STYLE)
        assertEquals("<i18n>...", PKConstants.FoldingPlaceholder.I18N)
        assertEquals("<!--...-->", PKConstants.FoldingPlaceholder.COMMENT)
        assertEquals("{{...}}", PKConstants.FoldingPlaceholder.INTERPOLATION)
        assertEquals("<...>", PKConstants.FoldingPlaceholder.TAG)
        assertEquals("...", PKConstants.FoldingPlaceholder.DEFAULT)
    }

    @Test
    fun `FoldingPlaceholder scriptWithLang formats correctly`() {
        assertEquals("<script lang=\"js\">...", PKConstants.FoldingPlaceholder.scriptWithLang("js"))
        assertEquals("<script lang=\"typescript\">...", PKConstants.FoldingPlaceholder.scriptWithLang("typescript"))
    }

    @Test
    fun `FoldingPlaceholder tagWithName formats correctly`() {
        assertEquals("<div>...", PKConstants.FoldingPlaceholder.tagWithName("div"))
        assertEquals("<span>...", PKConstants.FoldingPlaceholder.tagWithName("span"))
        assertEquals("<custom-element>...", PKConstants.FoldingPlaceholder.tagWithName("custom-element"))
    }

    @Test
    fun `LanguageDisplay values are correct`() {
        assertEquals("Go", PKConstants.LanguageDisplay.GO)
        assertEquals("JavaScript", PKConstants.LanguageDisplay.JAVASCRIPT)
        assertEquals("TypeScript", PKConstants.LanguageDisplay.TYPESCRIPT)
        assertEquals("CSS", PKConstants.LanguageDisplay.CSS)
        assertEquals("JSON", PKConstants.LanguageDisplay.JSON)
    }

    @Test
    fun `StructureDisplay values are correct`() {
        assertEquals("<template>", PKConstants.StructureDisplay.TEMPLATE_TAG)
        assertEquals("<script>", PKConstants.StructureDisplay.SCRIPT_TAG)
        assertEquals("<style>", PKConstants.StructureDisplay.STYLE_TAG)
        assertEquals("<i18n>", PKConstants.StructureDisplay.I18N_TAG)
        assertEquals("content", PKConstants.StructureDisplay.CONTENT)
        assertEquals("Go code", PKConstants.StructureDisplay.GO_CODE)
        assertEquals("JavaScript code", PKConstants.StructureDisplay.JS_CODE)
        assertEquals("CSS rules", PKConstants.StructureDisplay.CSS_RULES)
        assertEquals("translations", PKConstants.StructureDisplay.TRANSLATIONS)
    }

    @Test
    fun `StructureDisplay scriptTagWithLang formats correctly`() {
        assertEquals("<script lang=\"js\">", PKConstants.StructureDisplay.scriptTagWithLang("js"))
        assertEquals("<script lang=\"typescript\">", PKConstants.StructureDisplay.scriptTagWithLang("typescript"))
    }

    @Test
    fun `BreadcrumbDisplay values are correct`() {
        assertEquals("template", PKConstants.BreadcrumbDisplay.TEMPLATE)
        assertEquals("script", PKConstants.BreadcrumbDisplay.SCRIPT)
        assertEquals("style", PKConstants.BreadcrumbDisplay.STYLE)
        assertEquals("i18n", PKConstants.BreadcrumbDisplay.I18N)
        assertEquals("content", PKConstants.BreadcrumbDisplay.CONTENT)
        assertEquals("script (Go)", PKConstants.BreadcrumbDisplay.SCRIPT_GO)
        assertEquals("script (JS)", PKConstants.BreadcrumbDisplay.SCRIPT_JS)
        assertEquals("script (TS)", PKConstants.BreadcrumbDisplay.SCRIPT_TS)
    }

    @Test
    fun `BlockTags values are correct`() {
        assertEquals("template", PKConstants.BlockTags.TEMPLATE)
        assertEquals("script", PKConstants.BlockTags.SCRIPT)
        assertEquals("style", PKConstants.BlockTags.STYLE)
        assertEquals("i18n", PKConstants.BlockTags.I18N)
    }

    @Test
    fun `Comments values are correct`() {
        assertEquals("<!--", PKConstants.Comments.HTML_START)
        assertEquals("-->", PKConstants.Comments.HTML_END)
        assertEquals("/*", PKConstants.Comments.EXPR_START)
        assertEquals("*/", PKConstants.Comments.EXPR_END)
    }

    @Test
    fun `ScriptLang constants are lowercase`() {
        assertEquals("js", PKConstants.ScriptLang.JS)
        assertEquals("javascript", PKConstants.ScriptLang.JAVASCRIPT)
        assertEquals("typescript", PKConstants.ScriptLang.TYPESCRIPT)
    }
}
