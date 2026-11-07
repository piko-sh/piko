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

/**
 * Parses go.mod file content to extract module information.
 *
 * Extracts the module path and replace directives without requiring
 * IntelliJ platform dependencies, making it easily unit testable.
 */
object GoModParser {

    /** Pattern to extract the module path from a go.mod file. */
    private val modulePathRegex = Regex("""^\s*module\s+("[^"]+"|[^\s/]+(?:/[^\s/]+)*)""", RegexOption.MULTILINE)

    /** Matches the opening line of a `replace (` block. */
    private val replaceBlockStartRegex = Regex("""^replace\s*\($""")

    /** Matches a single-line `replace` directive and captures the replacement spec. */
    private val replaceLineRegex = Regex("""^replace\s+(.+)$""")

    /** Separator between tokens on a go.mod line. */
    private val WHITESPACE = Regex("\\s+")

    /** Matches a Windows absolute path such as `C:\\dir` or `C:/dir`. */
    private val windowsAbsolutePathRegex = Regex("""^[A-Za-z]:[\\/].*""")

    /**
     * Extracts the module path from go.mod content.
     *
     * @param content The go.mod file content.
     * @return The module path, or null if not found.
     */
    fun parseModulePath(content: String): String? =
        modulePathRegex.find(content)?.groupValues?.getOrNull(1)?.let(::unquote)

    /**
     * Extracts all replace directives from go.mod content.
     *
     * @param content The go.mod file content.
     * @return List of (oldPath, newPath) pairs. Versions on either side are dropped.
     */
    fun parseReplacements(content: String): List<Pair<String, String>> {
        val replacements = mutableListOf<Pair<String, String>>()
        var inReplaceBlock = false

        for (rawLine in content.lineSequence()) {
            val line = stripComment(rawLine).trim()
            if (line.isEmpty()) continue

            if (inReplaceBlock) {
                if (line == ")") {
                    inReplaceBlock = false
                } else {
                    parseReplaceSpec(line)?.let(replacements::add)
                }
                continue
            }

            if (replaceBlockStartRegex.matches(line)) {
                inReplaceBlock = true
                continue
            }

            replaceLineRegex.matchEntire(line)?.let { match ->
                parseReplaceSpec(match.groupValues[1])?.let(replacements::add)
            }
        }
        return replacements
    }

    /**
     * Reports whether a replacement target is a local file system path.
     *
     * Go treats a replacement as a directory when it starts with `./` or
     * `../`, or is an absolute path; anything else is a module path.
     *
     * @param path The replacement target.
     * @return True when the target is a local directory.
     */
    fun isLocalPath(path: String): Boolean =
        path == "." || path == ".." ||
            path.startsWith("./") || path.startsWith("../") ||
            path.startsWith(".\\") || path.startsWith("..\\") ||
            path.startsWith("/") || windowsAbsolutePathRegex.matches(path)

    /**
     * Parses `old [version] => new [version]`.
     *
     * @param spec The text after the `replace` keyword, or a line inside a block.
     * @return The (oldPath, newPath) pair, or null when malformed.
     */
    private fun parseReplaceSpec(spec: String): Pair<String, String>? {
        val arrow = spec.indexOf("=>")
        if (arrow < 0) return null
        val oldPath = spec.substring(0, arrow).trim().split(WHITESPACE).firstOrNull()?.let(::unquote)
        val newPath = spec.substring(arrow + 2).trim().split(WHITESPACE).firstOrNull()?.let(::unquote)
        if (oldPath.isNullOrEmpty() || newPath.isNullOrEmpty()) return null
        return oldPath to newPath
    }

    /**
     * Removes a trailing `//` comment that is not inside a quoted string.
     *
     * @param line The go.mod line.
     * @return The line without its comment.
     */
    private fun stripComment(line: String): String {
        var inQuotes = false
        var index = 0
        while (index < line.length) {
            val character = line[index]
            if (character == '"') {
                inQuotes = !inQuotes
            } else if (!inQuotes && line.startsWith("//", index)) {
                return line.substring(0, index)
            }
            index++
        }
        return line
    }

    /**
     * Removes surrounding double quotes or backticks from a go.mod token.
     *
     * @param token The token.
     * @return The unquoted token.
     */
    private fun unquote(token: String): String {
        if (token.length < 2) return token
        val quote = token.first()
        if ((quote == '"' || quote == '`') && token.last() == quote) {
            return token.substring(1, token.length - 1)
        }
        return token
    }
}
