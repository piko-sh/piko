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
    private val modulePathRegex = Regex("""^\s*module\s+([^\s]+)""", RegexOption.MULTILINE)

    /** Pattern to extract replace directives from a go.mod file. */
    private val replaceRegex = Regex("""^\s*replace\s+([^\s]+)\s*=>\s*([^\s]+)""", RegexOption.MULTILINE)

    /**
     * Extracts the module path from go.mod content.
     *
     * @param content The go.mod file content.
     * @return The module path, or null if not found.
     */
    fun parseModulePath(content: String): String? =
        modulePathRegex.find(content)?.groupValues?.getOrNull(1)

    /**
     * Extracts all replace directives from go.mod content.
     *
     * @param content The go.mod file content.
     * @return List of (oldPath, newPath) pairs.
     */
    fun parseReplacements(content: String): List<Pair<String, String>> =
        replaceRegex.findAll(content).map { it.groupValues[1] to it.groupValues[2] }.toList()
}
