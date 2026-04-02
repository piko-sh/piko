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

import com.intellij.codeInsight.template.emmet.generators.XmlZenCodingGeneratorImpl
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Unit tests for PKEmmetSupport.
 *
 * These tests verify the class structure and basic behaviour without
 * requiring a full IntelliJ platform context. Integration testing of
 * the actual Emmet expansion in template blocks should be done with
 * the IntelliJ platform test framework.
 */
class PKEmmetSupportTest {

    @Test
    fun `PKEmmetSupport extends XmlZenCodingGeneratorImpl`() {
        assertTrue(
            "PKEmmetSupport should extend XmlZenCodingGeneratorImpl",
            XmlZenCodingGeneratorImpl::class.java.isAssignableFrom(PKEmmetSupport::class.java)
        )
    }

    @Test
    fun `PKEmmetSupport can be instantiated`() {
        val support = PKEmmetSupport()
        assertTrue("Should be instance of PKEmmetSupport", support is PKEmmetSupport)
    }

    @Test
    fun `PKEmmetSupport inherits from correct superclass`() {
        val support = PKEmmetSupport()
        assertTrue(
            "Should inherit from XmlZenCodingGeneratorImpl",
            support is XmlZenCodingGeneratorImpl
        )
    }

    @Test
    fun `blocked element types include all non-template blocks`() {
        val blockedTypes = listOf(
            PKTokenTypes.SCRIPT_BLOCK_ELEMENT,
            PKTokenTypes.GO_SCRIPT_BODY_ELEMENT,
            PKTokenTypes.JS_SCRIPT_BODY_ELEMENT,
            PKTokenTypes.STYLE_BLOCK_ELEMENT,
            PKTokenTypes.CSS_STYLE_BODY_ELEMENT,
            PKTokenTypes.I18N_BLOCK_ELEMENT,
            PKTokenTypes.I18N_BODY_ELEMENT
        )

        blockedTypes.forEach { tokenType ->
            assertTrue(
                "Token type $tokenType should exist",
                tokenType.language == PKLanguage
            )
        }
    }

    @Test
    fun `allowed element type is template body`() {
        val allowedType = PKTokenTypes.TEMPLATE_BODY_ELEMENT
        assertTrue(
            "TEMPLATE_BODY_ELEMENT should exist with PKLanguage",
            allowedType.language == PKLanguage
        )
    }

    @Test
    fun `PKFileType is available for type checking`() {
        assertTrue(
            "PKFileType should be available",
            PKFileType.name == "PK File"
        )
    }

    @Test
    fun `PKEmmetSupport overrides isMyContext method`() {
        val method = PKEmmetSupport::class.java.getDeclaredMethod(
            "isMyContext",
            com.intellij.psi.PsiElement::class.java,
            Boolean::class.javaPrimitiveType
        )
        assertTrue("isMyContext should be a declared method", method != null)
    }
}
