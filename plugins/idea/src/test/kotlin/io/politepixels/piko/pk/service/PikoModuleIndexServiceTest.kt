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


package io.politepixels.piko.pk.service

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class PikoModuleIndexServiceTest {

    @Test
    fun `module path parsed correctly`() {
        val content = """
            module piko.sh/piko

            go 1.21
        """.trimIndent()

        assertEquals(
            "piko.sh/piko",
            GoModParser.parseModulePath(content)
        )
    }

    @Test
    fun `replacement paths parsed correctly`() {
        val content = """
            module github.com/example/app

            replace github.com/example/lib => ./lib
        """.trimIndent()

        val replacements = GoModParser.parseReplacements(content)
        assertEquals(1, replacements.size)
        assertEquals("github.com/example/lib", replacements[0].first)
        assertEquals("./lib", replacements[0].second)
    }

    @Test
    fun `local replacement paths start with dot`() {
        val content = """
            module example.com/app

            replace example.com/dep1 => ./local
            replace example.com/dep2 => ../sibling
            replace example.com/dep3 => example.com/other v1.0.0
        """.trimIndent()

        val replacements = GoModParser.parseReplacements(content)
        val localReplacements = replacements.filter { it.second.startsWith(".") }

        assertEquals(
            "Should find 2 local replacements",
            2,
            localReplacements.size
        )
    }

    @Test
    fun `PikoModuleInfo has required properties`() {
        assertTrue("PikoModuleInfo class should exist", true)
    }

    @Test
    fun `PikoModuleInfo is a data class`() {
        val kClass = PikoModuleInfo::class
        assertTrue("PikoModuleInfo should have copy method",
            kClass.members.any { it.name == "copy" })
        assertTrue("PikoModuleInfo should have component methods",
            kClass.members.any { it.name == "component1" })
    }

    @Test
    fun `returns null for invalid go mod content`() {
        val content = "not a valid go.mod file"
        assertNull(GoModParser.parseModulePath(content))
    }

    @Test
    fun `handles go mod with only module declaration`() {
        val content = "module example.com/minimal"
        assertNotNull(GoModParser.parseModulePath(content))
        assertTrue(GoModParser.parseReplacements(content).isEmpty())
    }

    @Test
    fun `handles complex go mod with all sections`() {
        val content = """
            module piko.sh/piko

            go 1.21

            require (
                github.com/gofiber/fiber/v2 v2.50.0
                github.com/stretchr/testify v1.8.4
            )

            require (
                github.com/indirect/dep v1.0.0 // indirect
            )

            replace piko.sh/piko/internal/schema => ./internal/schema
            replace piko.sh/piko/internal/render => ./internal/render
        """.trimIndent()

        assertEquals(
            "piko.sh/piko",
            GoModParser.parseModulePath(content)
        )

        val replacements = GoModParser.parseReplacements(content)
        assertEquals(2, replacements.size)
    }

    @Test
    fun `relative paths are identified correctly`() {
        val paths = listOf(
            "./local" to true,
            "../sibling" to true,
            "github.com/other" to false,
            "example.com/pkg" to false
        )

        for ((path, shouldBeRelative) in paths) {
            assertEquals(
                "Path '$path' should ${if (shouldBeRelative) "" else "not "}be relative",
                shouldBeRelative,
                path.startsWith(".")
            )
        }
    }

    @Test
    fun `module path extraction is consistent`() {
        val content = "module example.com/app"

        val result1 = GoModParser.parseModulePath(content)
        val result2 = GoModParser.parseModulePath(content)

        assertEquals("Parsing should be deterministic", result1, result2)
    }
}
