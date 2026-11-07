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

package io.politepixels.piko.pk.lsp

import com.intellij.openapi.application.PathManager
import com.intellij.openapi.project.Project
import java.nio.file.Path

/**
 * Chooses where the Piko language server writes its debug log.
 */
object PikoLspLogFile {

    /** Directory name under the IDE log directory. */
    private const val LOG_DIRECTORY = "pikopls"

    /** Characters allowed in the per-project file name. */
    private val UNSAFE_FILE_NAME_CHARACTERS = Regex("[^A-Za-z0-9._-]")

    /**
     * Returns the log file path for a project.
     *
     * @param project The project the server runs for.
     * @return The absolute log file path; the file may not exist yet.
     */
    @JvmStatic
    fun forProject(project: Project): Path = forKey(project.locationHash)

    /**
     * Returns the log file path for a project key.
     *
     * @param projectKey A stable per-project identifier.
     * @return The absolute log file path; the file may not exist yet.
     */
    @JvmStatic
    fun forKey(projectKey: String): Path {
        val safeKey = projectKey.replace(UNSAFE_FILE_NAME_CHARACTERS, "_").ifEmpty { "default" }
        return Path.of(PathManager.getLogPath(), LOG_DIRECTORY, "pikopls-$safeKey.log")
    }
}
