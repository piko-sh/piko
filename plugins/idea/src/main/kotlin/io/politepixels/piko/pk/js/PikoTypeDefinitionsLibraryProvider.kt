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

package io.politepixels.piko.pk.js

import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.project.Project
import com.intellij.openapi.project.guessProjectDir
import com.intellij.openapi.roots.AdditionalLibraryRootsProvider
import com.intellij.openapi.roots.SyntheticLibrary
import com.intellij.openapi.vfs.VfsUtil
import com.intellij.openapi.vfs.VirtualFile

/**
 * Exposes the Piko TypeScript declarations as a library.
 */
class PikoTypeDefinitionsLibraryProvider : AdditionalLibraryRootsProvider() {

    /** Logger for this provider. */
    private val log = Logger.getInstance(PikoTypeDefinitionsLibraryProvider::class.java)

    /**
     * Returns the Piko declaration library for a project.
     *
     * @param project The current project.
     * @return A single synthetic library, or empty when no declarations are available.
     */
    override fun getAdditionalProjectLibraries(project: Project): Collection<SyntheticLibrary> {
        val roots = declarationRoots(project)
        if (roots.isEmpty()) return emptyList()
        return listOf(SyntheticLibrary.newImmutableLibrary(LIBRARY_ID, roots, emptyList(), emptySet(), null))
    }

    /**
     * Watches the project's `dist/ts` directory so regenerated declarations are re-indexed.
     *
     * @param project The current project.
     * @return The directory to watch, or empty when it does not exist.
     */
    override fun getRootsToWatch(project: Project): Collection<VirtualFile> =
        listOfNotNull(findDistTsDirectory(project))

    /**
     * Chooses the directories that hold the declarations.
     *
     * @param project The current project.
     * @return The project's `dist/ts` directory when it has the IDE declarations,
     *         otherwise the bundled declarations directory.
     */
    private fun declarationRoots(project: Project): List<VirtualFile> {
        val distTs = findDistTsDirectory(project)
        if (distTs != null && distTs.findChild(IDE_TYPE_FILE) != null) {
            return listOf(distTs)
        }
        return listOfNotNull(bundledDeclarationsDirectory())
    }

    /**
     * Finds the project's `dist/ts` directory without refreshing the VFS.
     *
     * @param project The current project.
     * @return The directory, or null when it does not exist.
     */
    private fun findDistTsDirectory(project: Project): VirtualFile? {
        val projectDir = project.guessProjectDir() ?: return null
        return projectDir.findFileByRelativePath(DIST_TS_PATH)?.takeIf { it.isDirectory }
    }

    /**
     * Locates the declarations bundled in the plugin JAR.
     *
     * @return The bundled `types` directory, or null when it is missing.
     */
    private fun bundledDeclarationsDirectory(): VirtualFile? {
        val resource = PikoTypeDefinitionsLibraryProvider::class.java.getResource("/types/$IDE_TYPE_FILE")
        if (resource == null) {
            log.debug("Bundled Piko type definitions not found")
            return null
        }
        return VfsUtil.findFileByURL(resource)?.parent
    }

    companion object {
        /** Identifier of the synthetic library. */
        private const val LIBRARY_ID = "Piko type definitions"

        /** Relative path to generated type definitions from the project root. */
        private const val DIST_TS_PATH = "dist/ts"

        /** IDE declaration file for the `piko` namespace. */
        private const val IDE_TYPE_FILE = "piko-ide.d.ts"
    }
}
