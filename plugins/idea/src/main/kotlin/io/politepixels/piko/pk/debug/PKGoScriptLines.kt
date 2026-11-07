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

/**
 * Decides which lines of a Go script block can hold a breakpoint.
 */
object PKGoScriptLines {

    private enum class State { CODE, BLOCK_COMMENT, RAW_STRING }

    /**
     * Computes the breakable lines of a Go script block.
     *
     * @param scriptText The raw text of the script block content.
     * @return One flag per line of [scriptText], true where a breakpoint is allowed.
     */
    fun breakableLines(scriptText: String): BooleanArray {
        val lines = scriptText.split('\n')
        val result = BooleanArray(lines.size)
        var state = State.CODE
        var inImportBlock = false

        for ((index, rawLine) in lines.withIndex()) {
            val startedInRawString = state == State.RAW_STRING
            val scan = scanLine(rawLine.removeSuffix("\r"), state)
            state = scan.endState

            val code = scan.code.trim()
            if (code.isEmpty() || startedInRawString) {
                continue
            }

            if (inImportBlock) {
                if (code.startsWith(")")) inImportBlock = false
                continue
            }

            if (isImportBlockOpener(code)) {
                inImportBlock = true
                continue
            }

            result[index] = !isUnmappableDeclaration(code)
        }

        return result
    }

    private fun isImportBlockOpener(code: String): Boolean =
        IMPORT_BLOCK_PATTERN.matches(code)

    private fun isUnmappableDeclaration(code: String): Boolean =
        PACKAGE_PATTERN.containsMatchIn(code) ||
            IMPORT_PATTERN.containsMatchIn(code) ||
            METHOD_SIGNATURE_PATTERN.containsMatchIn(code)

    private class LineScan(val code: String, val endState: State)

    /**
     * Strips comments and string contents from one line, carrying multi-line
     * block comment and raw string state across lines.
     */
    private fun scanLine(line: String, initialState: State): LineScan {
        val cursor = LineCursor(line, initialState)
        while (!cursor.finished) {
            when (cursor.state) {
                State.BLOCK_COMMENT -> cursor.stepInBlockComment()
                State.RAW_STRING -> cursor.stepInRawString()
                State.CODE -> cursor.stepInCode()
            }
        }
        return LineScan(cursor.code.toString(), cursor.state)
    }

    /**
     * Walks one line of Go source, collecting the code outside comments.
     *
     * @property line The line being scanned.
     * @property state The lexical state at the current position.
     */
    private class LineCursor(private val line: String, var state: State) {

        /** The code collected so far, with strings replaced by placeholders. */
        val code = StringBuilder()

        /** The offset of the next character to read. */
        private var position = 0

        /** Whether a line comment cut the rest of the line. */
        private var stopped = false

        /** Whether the whole line has been read. */
        val finished: Boolean get() = stopped || position >= line.length

        private val current: Char get() = line[position]

        private val next: Char? get() = line.getOrNull(position + 1)

        fun stepInBlockComment() {
            if (current == '*' && next == '/') {
                state = State.CODE
                position++
            }
            position++
        }

        fun stepInRawString() {
            if (current == '`') {
                state = State.CODE
                code.append(STRING_PLACEHOLDER)
            }
            position++
        }

        fun stepInCode() {
            val character = current
            when {
                character == '/' && next == '/' -> stopped = true
                character == '/' && next == '*' -> {
                    state = State.BLOCK_COMMENT
                    position++
                }
                character == '`' -> state = State.RAW_STRING
                character == '"' || character == '\'' -> {
                    position = skipQuoted(line, position, character)
                    code.append(STRING_PLACEHOLDER)
                }
                else -> code.append(character)
            }
            position++
        }
    }

    private fun skipQuoted(line: String, start: Int, quote: Char): Int {
        var position = start + 1
        while (position < line.length) {
            when (line[position]) {
                '\\' -> position++
                quote -> return position
            }
            position++
        }
        return line.length
    }

    private const val STRING_PLACEHOLDER = "\"\""

    private val PACKAGE_PATTERN = Regex("""^package\s""")

    private val IMPORT_PATTERN = Regex("""^import(\s|\(|"|$)""")

    private val IMPORT_BLOCK_PATTERN = Regex("""^import\s*\($""")

    private val METHOD_SIGNATURE_PATTERN = Regex("""^func\s+\(""")
}
