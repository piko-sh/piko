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

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class GoLanguageInjectorLogicTest {

    @Test
    fun `package main detected as having package declaration`() {
        assertTrue(PKGoLanguageInjector.hasPackageDeclaration("package main"))
    }

    @Test
    fun `package foo detected as having package declaration`() {
        assertTrue(PKGoLanguageInjector.hasPackageDeclaration("package foo"))
    }

    @Test
    fun `leading whitespace before package still detected`() {
        assertTrue(PKGoLanguageInjector.hasPackageDeclaration("   package main"))
    }

    @Test
    fun `leading newlines before package still detected`() {
        assertTrue(PKGoLanguageInjector.hasPackageDeclaration("\n\npackage main"))
    }

    @Test
    fun `leading tabs before package still detected`() {
        assertTrue(PKGoLanguageInjector.hasPackageDeclaration("\t\tpackage main"))
    }

    @Test
    fun `func main without package is false`() {
        assertFalse(PKGoLanguageInjector.hasPackageDeclaration("func main() {}"))
    }

    @Test
    fun `empty string is false`() {
        assertFalse(PKGoLanguageInjector.hasPackageDeclaration(""))
    }

    @Test
    fun `whitespace only is false`() {
        assertFalse(PKGoLanguageInjector.hasPackageDeclaration("   \n\t   "))
    }

    @Test
    fun `comment before package is false`() {
        assertFalse(PKGoLanguageInjector.hasPackageDeclaration("// comment\npackage main"))
    }

    @Test
    fun `package in middle of content is false`() {
        assertFalse(PKGoLanguageInjector.hasPackageDeclaration("func foo() {}\npackage main"))
    }

    @Test
    fun `Package with capital P is false`() {
        assertFalse(PKGoLanguageInjector.hasPackageDeclaration("Package main"))
    }

    @Test
    fun `package without space after is false`() {
        assertFalse(PKGoLanguageInjector.hasPackageDeclaration("packagemain"))
    }

    @Test
    fun `package declaration with multiline content detected`() {
        val content = """
            package main

            import "fmt"

            func main() {
                fmt.Println("Hello")
            }
        """.trimIndent()
        assertTrue(PKGoLanguageInjector.hasPackageDeclaration(content))
    }

    @Test
    fun `computePrefix returns null when package present`() {
        assertNull(PKGoLanguageInjector.computePrefix("package main"))
    }

    @Test
    fun `computePrefix returns package main when no package`() {
        assertEquals(
            PKGoLanguageInjector.DEFAULT_PACKAGE_PREFIX,
            PKGoLanguageInjector.computePrefix("func main() {}")
        )
    }

    @Test
    fun `computePrefix with empty string returns package main`() {
        assertEquals(
            PKGoLanguageInjector.DEFAULT_PACKAGE_PREFIX,
            PKGoLanguageInjector.computePrefix("")
        )
    }

    @Test
    fun `computePrefix with func only returns package main`() {
        assertEquals(
            PKGoLanguageInjector.DEFAULT_PACKAGE_PREFIX,
            PKGoLanguageInjector.computePrefix("func hello() string { return \"hello\" }")
        )
    }

    @Test
    fun `prefix format is exactly package main with two newlines`() {
        assertEquals("package main\n\n", PKGoLanguageInjector.DEFAULT_PACKAGE_PREFIX)
    }

    @Test
    fun `computePrefix returns null for package with custom name`() {
        assertNull(PKGoLanguageInjector.computePrefix("package mypackage"))
    }

    @Test
    fun `computePrefix returns prefix for import statement only`() {
        assertEquals(
            PKGoLanguageInjector.DEFAULT_PACKAGE_PREFIX,
            PKGoLanguageInjector.computePrefix("import \"fmt\"")
        )
    }

    @Test
    fun `computePrefix returns prefix for variable declaration only`() {
        assertEquals(
            PKGoLanguageInjector.DEFAULT_PACKAGE_PREFIX,
            PKGoLanguageInjector.computePrefix("var x = 10")
        )
    }

    @Test
    fun `computePrefix returns prefix for type declaration only`() {
        assertEquals(
            PKGoLanguageInjector.DEFAULT_PACKAGE_PREFIX,
            PKGoLanguageInjector.computePrefix("type Props struct { Title string }")
        )
    }
}
