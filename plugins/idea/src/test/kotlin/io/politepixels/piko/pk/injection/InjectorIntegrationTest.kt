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

import io.politepixels.piko.pk.psi.impl.PkCssStyleContentElementImpl
import io.politepixels.piko.pk.psi.impl.PkGoScriptContentElementImpl
import io.politepixels.piko.pk.psi.impl.PkI18nContentElementImpl
import io.politepixels.piko.pk.psi.impl.PkJsScriptContentElementImpl
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

class InjectorIntegrationTest {

    @Test
    fun `go injector targets PkGoScriptContentElementImpl`() {
        val injector = PKGoLanguageInjector()
        val targetElements = injector.elementsToInjectIn()

        assertEquals("Should target exactly one element type", 1, targetElements.size)
        assertEquals(
            "Should target PkGoScriptContentElementImpl",
            PkGoScriptContentElementImpl::class.java,
            targetElements[0]
        )
    }

    @Test
    fun `go injector package prefix is correctly formatted`() {
        assertEquals(
            "Prefix should be 'package main' with two newlines",
            "package main\n\n",
            PKGoLanguageInjector.DEFAULT_PACKAGE_PREFIX
        )
    }

    @Test
    fun `go injector companion object functions are accessible`() {
        assertNotNull(PKGoLanguageInjector.Companion)
        assertTrue("hasPackageDeclaration should work", PKGoLanguageInjector.hasPackageDeclaration("package main"))
    }

    @Test
    fun `typescript injector targets PkJsScriptContentElementImpl`() {
        val injector = PKTypeScriptLanguageInjector()
        val targetElements = injector.elementsToInjectIn()

        assertEquals("Should target exactly one element type", 1, targetElements.size)
        assertEquals(
            "Should target PkJsScriptContentElementImpl",
            PkJsScriptContentElementImpl::class.java,
            targetElements[0]
        )
    }

    @Test
    fun `css injector targets PkCssStyleContentElementImpl`() {
        val injector = PKCssLanguageInjector()
        val targetElements = injector.elementsToInjectIn()

        assertEquals("Should target exactly one element type", 1, targetElements.size)
        assertEquals(
            "Should target PkCssStyleContentElementImpl",
            PkCssStyleContentElementImpl::class.java,
            targetElements[0]
        )
    }

    @Test
    fun `json injector targets PkI18nContentElementImpl`() {
        val injector = PKJsonLanguageInjector()
        val targetElements = injector.elementsToInjectIn()

        assertEquals("Should target exactly one element type", 1, targetElements.size)
        assertEquals(
            "Should target PkI18nContentElementImpl",
            PkI18nContentElementImpl::class.java,
            targetElements[0]
        )
    }

    @Test
    fun `all injectors target different element types`() {
        val goTarget = PKGoLanguageInjector().elementsToInjectIn()[0]
        val jsTarget = PKTypeScriptLanguageInjector().elementsToInjectIn()[0]
        val cssTarget = PKCssLanguageInjector().elementsToInjectIn()[0]
        val jsonTarget = PKJsonLanguageInjector().elementsToInjectIn()[0]

        val targets = setOf(goTarget, jsTarget, cssTarget, jsonTarget)
        assertEquals("All injectors should have distinct targets", 4, targets.size)
    }

    @Test
    fun `injector classes are properly named for their languages`() {
        assertTrue(
            "Go injector class name should contain 'Go'",
            PKGoLanguageInjector::class.java.simpleName.contains("Go")
        )
        assertTrue(
            "TypeScript injector class name should contain 'TypeScript'",
            PKTypeScriptLanguageInjector::class.java.simpleName.contains("TypeScript")
        )
        assertTrue(
            "CSS injector class name should contain 'Css'",
            PKCssLanguageInjector::class.java.simpleName.contains("Css")
        )
        assertTrue(
            "JSON injector class name should contain 'Json'",
            PKJsonLanguageInjector::class.java.simpleName.contains("Json")
        )
    }

    @Test
    fun `all injectors implement MultiHostInjector`() {
        val goInjector = PKGoLanguageInjector()
        val jsInjector = PKTypeScriptLanguageInjector()
        val cssInjector = PKCssLanguageInjector()
        val jsonInjector = PKJsonLanguageInjector()

        assertTrue("Go injector should be MultiHostInjector",
            goInjector is com.intellij.lang.injection.MultiHostInjector)
        assertTrue("JS injector should be MultiHostInjector",
            jsInjector is com.intellij.lang.injection.MultiHostInjector)
        assertTrue("CSS injector should be MultiHostInjector",
            cssInjector is com.intellij.lang.injection.MultiHostInjector)
        assertTrue("JSON injector should be MultiHostInjector",
            jsonInjector is com.intellij.lang.injection.MultiHostInjector)
    }

    @Test
    fun `matchesModulePath returns true for exact match`() {
        assertTrue(PkGoImportResolver.matchesModulePath(
            "github.com/example/app",
            "github.com/example/app"
        ))
    }

    @Test
    fun `matchesModulePath returns true for subpackage`() {
        assertTrue(PkGoImportResolver.matchesModulePath(
            "github.com/example/app/internal/service",
            "github.com/example/app"
        ))
    }

    @Test
    fun `matchesModulePath returns false for different module`() {
        assertFalse(PkGoImportResolver.matchesModulePath(
            "github.com/other/app",
            "github.com/example/app"
        ))
    }

    @Test
    fun `matchesModulePath returns false for partial prefix`() {
        assertFalse(PkGoImportResolver.matchesModulePath(
            "github.com/example/application",
            "github.com/example/app"
        ))
    }

    @Test
    fun `matchesModulePath handles single segment modules`() {
        assertTrue(PkGoImportResolver.matchesModulePath("mymodule/pkg", "mymodule"))
        assertFalse(PkGoImportResolver.matchesModulePath("mymodule2/pkg", "mymodule"))
    }
}
