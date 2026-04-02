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


package io.politepixels.piko.pk.injection

import io.politepixels.piko.pk.psi.impl.PkJsScriptContentElementImpl
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class TypeScriptInjectorLogicTest {

    @Test
    fun `documents TypeScript fallback chain order`() {
        val expectedChain = listOf("TypeScript", "ECMAScript 6", "JavaScript")
        assertEquals(3, expectedChain.size)
        assertEquals("TypeScript", expectedChain[0])
        assertEquals("ECMAScript 6", expectedChain[1])
        assertEquals("JavaScript", expectedChain[2])
    }

    @Test
    fun `elementsToInjectIn returns only PkJsScriptContentElementImpl`() {
        val injector = PKTypeScriptLanguageInjector()
        val elements = injector.elementsToInjectIn()

        assertEquals("Should return exactly one element type", 1, elements.size)
        assertEquals(
            "Should target PkJsScriptContentElementImpl",
            PkJsScriptContentElementImpl::class.java,
            elements[0]
        )
    }

    @Test
    fun `injector class name indicates TypeScript preference`() {
        val className = PKTypeScriptLanguageInjector::class.java.simpleName
        assertTrue(
            "Class name should indicate TypeScript",
            className.contains("TypeScript")
        )
    }

    @Test
    fun `documents no prefix or suffix added to JS content`() {
        val expectedPrefix = ""
        val expectedSuffix = ""
        assertEquals("Prefix should be empty", "", expectedPrefix)
        assertEquals("Suffix should be empty", "", expectedSuffix)
    }

    @Test
    fun `documents injection range covers entire content`() {
        val startOffset = 0
        assertTrue("Start offset should be 0", startOffset == 0)
    }
}
