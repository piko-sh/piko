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

package io.politepixels.piko.pk.debug

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PKGoScriptLinesTest {

    private fun breakable(vararg lines: String): List<Boolean> =
        PKGoScriptLines.breakableLines(lines.joinToString("\n")).toList()

    @Test
    fun `statements in a function are breakable`() {
        assertEquals(
            listOf(true, true, true),
            breakable("func Render() Response {", "    x := 1", "    return Response{X: x}"),
        )
    }

    @Test
    fun `blank and whitespace lines are refused`() {
        assertEquals(listOf(false, false, true), breakable("", "   \t", "x := 1"))
    }

    @Test
    fun `line comments are refused but trailing comments keep the line`() {
        assertEquals(listOf(false, true), breakable("    // a comment", "    x := 1 // trailing"))
    }

    @Test
    fun `block comments are refused across lines`() {
        assertEquals(
            listOf(false, false, false, true, true),
            breakable("/* start", "middle", "end */", "/* inline */ x := 1", "y := 2 /* open"),
        )
    }

    @Test
    fun `code after a block comment closes is breakable`() {
        assertEquals(listOf(false, true), breakable("/* one", "two */ x := 1"))
    }

    @Test
    fun `package clause is refused`() {
        assertEquals(listOf(false, true), breakable("package main", "var x = 1"))
    }

    @Test
    fun `single and grouped imports are refused`() {
        assertEquals(
            listOf(false, false, false, false, false, false, true),
            breakable(
                "import \"fmt\"",
                "import alias \"strings\"",
                "import (",
                "    \"piko.sh/piko\"",
                "    card \"example.com/partials/card.pk\"",
                ")",
                "var answer = 42",
            ),
        )
    }

    @Test
    fun `method signatures are refused but their bodies are breakable`() {
        assertEquals(
            listOf(false, true, true),
            breakable("func (p Props) Title() string {", "    t := p.Name", "    return t"),
        )
    }

    @Test
    fun `function literals at statement start are breakable`() {
        assertEquals(listOf(true, true), breakable("func() {", "}()"))
    }

    @Test
    fun `top level declarations are breakable`() {
        assertEquals(
            listOf(true, true, true),
            breakable("type Props struct {", "var answer = 42", "func Render() {"),
        )
    }

    @Test
    fun `continuation lines of a raw string are refused`() {
        assertEquals(
            listOf(true, false, false, true),
            breakable("q := `SELECT *", "FROM items", "WHERE x`", "y := 2"),
        )
    }

    @Test
    fun `comment markers inside strings are code`() {
        assertEquals(listOf(true, true), breakable("u := \"http://example.com\"", "r := '/'"))
    }

    @Test
    fun `escaped quotes do not end a string`() {
        assertTrue(breakable("s := \"a \\\" // b\"")[0])
    }

    @Test
    fun `carriage returns are ignored`() {
        assertEquals(listOf(false, true), breakable("package main\r", "x := 1\r"))
    }

    @Test
    fun `empty script has one refused line`() {
        val lines = PKGoScriptLines.breakableLines("")
        assertEquals(1, lines.size)
        assertFalse(lines[0])
    }
}
