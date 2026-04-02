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
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class GoModParserTest {

    @Test
    fun `parses simple module path`() {
        val content = "module example.com/mymodule"
        assertEquals("example.com/mymodule", GoModParser.parseModulePath(content))
    }

    @Test
    fun `parses module path with nested packages`() {
        val content = "module piko.sh/piko/internal/render"
        assertEquals("piko.sh/piko/internal/render", GoModParser.parseModulePath(content))
    }

    @Test
    fun `handles leading whitespace before module`() {
        val content = "  module example.com/mymodule"
        assertEquals("example.com/mymodule", GoModParser.parseModulePath(content))
    }

    @Test
    fun `handles leading tabs before module`() {
        val content = "\tmodule example.com/mymodule"
        assertEquals("example.com/mymodule", GoModParser.parseModulePath(content))
    }

    @Test
    fun `returns null for empty content`() {
        assertNull(GoModParser.parseModulePath(""))
    }

    @Test
    fun `returns null when no module directive`() {
        val content = """
            go 1.21

            require (
                github.com/foo/bar v1.0.0
            )
        """.trimIndent()
        assertNull(GoModParser.parseModulePath(content))
    }

    @Test
    fun `ignores module in comment`() {
        val content = """
            // module commented.example.com/wrong
            module example.com/correct
        """.trimIndent()
        assertEquals("example.com/correct", GoModParser.parseModulePath(content))
    }

    @Test
    fun `handles module followed by go directive`() {
        val content = """
            module example.com/mymodule

            go 1.21
        """.trimIndent()
        assertEquals("example.com/mymodule", GoModParser.parseModulePath(content))
    }

    @Test
    fun `handles module with version suffix in path`() {
        val content = "module example.com/mymodule/v2"
        assertEquals("example.com/mymodule/v2", GoModParser.parseModulePath(content))
    }

    @Test
    fun `parses single replace directive`() {
        val content = """
            module example.com/mymodule

            replace example.com/old => ./local
        """.trimIndent()
        val replacements = GoModParser.parseReplacements(content)
        assertEquals(1, replacements.size)
        assertEquals("example.com/old" to "./local", replacements[0])
    }

    @Test
    fun `parses multiple replace directives`() {
        val content = """
            module example.com/mymodule

            replace example.com/old1 => ./local1
            replace example.com/old2 => ./local2
        """.trimIndent()
        val replacements = GoModParser.parseReplacements(content)
        assertEquals(2, replacements.size)
        assertEquals("example.com/old1" to "./local1", replacements[0])
        assertEquals("example.com/old2" to "./local2", replacements[1])
    }

    @Test
    fun `parses replace with local path`() {
        val content = "replace example.com/dep => ../sibling"
        val replacements = GoModParser.parseReplacements(content)
        assertEquals(1, replacements.size)
        assertEquals("example.com/dep" to "../sibling", replacements[0])
    }

    @Test
    fun `parses replace with remote path`() {
        val content = "replace example.com/old => example.com/new v1.0.0"
        val replacements = GoModParser.parseReplacements(content)
        assertEquals(1, replacements.size)
        assertEquals("example.com/old" to "example.com/new", replacements[0])
    }

    @Test
    fun `handles whitespace around arrow`() {
        val content = "replace example.com/old   =>   ./local"
        val replacements = GoModParser.parseReplacements(content)
        assertEquals(1, replacements.size)
        assertEquals("example.com/old" to "./local", replacements[0])
    }

    @Test
    fun `handles tabs in replace directive`() {
        val content = "\treplace\texample.com/old\t=>\t./local"
        val replacements = GoModParser.parseReplacements(content)
        assertEquals(1, replacements.size)
        assertEquals("example.com/old" to "./local", replacements[0])
    }

    @Test
    fun `returns empty list when no replacements`() {
        val content = """
            module example.com/mymodule

            go 1.21
        """.trimIndent()
        val replacements = GoModParser.parseReplacements(content)
        assertTrue("Should return empty list", replacements.isEmpty())
    }

    @Test
    fun `ignores replace in comment`() {
        val content = """
            module example.com/mymodule

            // replace example.com/commented => ./wrong
            replace example.com/actual => ./correct
        """.trimIndent()
        val replacements = GoModParser.parseReplacements(content)
        assertEquals(1, replacements.size)
        assertEquals("example.com/actual" to "./correct", replacements[0])
    }

    @Test
    fun `parses real piko gomod format`() {
        val content = """
            module piko.sh/piko

            go 1.21

            require (
                github.com/gofiber/fiber/v2 v2.50.0
                github.com/google/flatbuffers v23.5.26+incompatible
            )

            replace piko.sh/piko/internal/schema => ./internal/schema
        """.trimIndent()

        assertEquals("piko.sh/piko", GoModParser.parseModulePath(content))

        val replacements = GoModParser.parseReplacements(content)
        assertEquals(1, replacements.size)
        assertEquals(
            "piko.sh/piko/internal/schema" to "./internal/schema",
            replacements[0]
        )
    }

    @Test
    fun `handles require block without affecting module`() {
        val content = """
            module example.com/app

            require (
                example.com/dep1 v1.0.0
                example.com/dep2 v2.0.0
            )
        """.trimIndent()

        assertEquals("example.com/app", GoModParser.parseModulePath(content))
        assertTrue("Should have no replacements", GoModParser.parseReplacements(content).isEmpty())
    }

    @Test
    fun `handles indirect comments in require`() {
        val content = """
            module example.com/app

            require (
                example.com/dep1 v1.0.0 // indirect
                example.com/dep2 v2.0.0
            )
        """.trimIndent()

        assertEquals("example.com/app", GoModParser.parseModulePath(content))
    }

    @Test
    fun `handles single-line replace directives only`() {
        val content = """
            module example.com/app

            replace example.com/old1 => ./local1
            replace example.com/old2 => ./local2
        """.trimIndent()

        val replacements = GoModParser.parseReplacements(content)
        assertEquals(2, replacements.size)
    }

    @Test
    fun `handles module path with hyphen`() {
        val content = "module example.com/my-module"
        assertEquals("example.com/my-module", GoModParser.parseModulePath(content))
    }

    @Test
    fun `handles module path with underscore`() {
        val content = "module example.com/my_module"
        assertEquals("example.com/my_module", GoModParser.parseModulePath(content))
    }
}
