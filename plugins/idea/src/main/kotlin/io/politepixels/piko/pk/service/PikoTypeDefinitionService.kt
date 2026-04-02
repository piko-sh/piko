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

import com.intellij.openapi.Disposable
import com.intellij.openapi.components.Service
import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.LocalFileSystem
import com.intellij.openapi.vfs.VfsUtil
import com.intellij.openapi.vfs.VirtualFile
import com.intellij.openapi.vfs.VirtualFileManager
import com.intellij.openapi.vfs.newvfs.BulkFileListener
import com.intellij.openapi.vfs.newvfs.events.VFileContentChangeEvent
import com.intellij.openapi.vfs.newvfs.events.VFileCreateEvent
import com.intellij.openapi.vfs.newvfs.events.VFileEvent
import com.intellij.psi.PsiManager

/** Logger for the type definition service. */
private val LOG = Logger.getInstance(PikoTypeDefinitionService::class.java)

/**
 * Service that provides TypeScript type definitions for injection into
 * script blocks.
 *
 * Reads `.d.ts` files from the project's `dist/ts/` directory and caches
 * them for efficient injection into TypeScript/JavaScript script blocks.
 * Falls back to embedded resources bundled in the plugin JAR when the
 * on-disk files do not exist yet (i.e. before the first `piko run`).
 *
 * The type definitions provide IDE autocomplete and type checking for:
 * - The unified `piko` namespace (piko.nav, piko.form, etc.)
 * - Server-side action definitions
 *
 * This service watches for file changes in `dist/ts/` and automatically
 * refreshes the cache when type definitions are created or modified.
 */
@Service(Service.Level.PROJECT)
class PikoTypeDefinitionService(private val project: Project) : Disposable {

    /** Cached ambient type declarations for injection into script blocks. */
    private var cachedTypes: String = ""

    /** Timestamp of the last cache refresh, in epoch milliseconds. */
    private var lastRefreshTime: Long = 0

    companion object {
        /** Refresh interval in milliseconds (5 seconds). */
        private const val REFRESH_INTERVAL_MS = 5000L

        /** Relative path to type definitions from project root. */
        private const val DIST_TS_PATH = "dist/ts"

        /** Slim IDE-specific declaration file. */
        private const val IDE_TYPE_FILE = "piko-ide.d.ts"

        /** Action type definitions file. */
        private const val ACTIONS_TYPE_FILE = "piko-actions.d.ts"

        /** Resource path prefix for embedded fallback type definitions. */
        private const val RESOURCE_TYPES_PREFIX = "/types/"

        /** File names to watch for. */
        private val TYPE_FILE_NAMES = setOf(IDE_TYPE_FILE, ACTIONS_TYPE_FILE)

        /**
         * Gets the service instance for a project.
         *
         * @param project The project to get the service for.
         * @return The PikoTypeDefinitionService instance.
         */
        @JvmStatic
        fun getInstance(project: Project): PikoTypeDefinitionService {
            return project.getService(PikoTypeDefinitionService::class.java)
        }
    }

    init {
        setupFileWatcher()
    }

    /**
     * Gets the type definition prefix for injection into script blocks.
     *
     * Returns the cached type definitions wrapped in `declare global { }`
     * to make them truly global and prevent the TypeScript plugin from
     * trying to add import statements when selecting completions.
     *
     * @return Global type declarations, or empty string if no definitions
     *         are available.
     */
    fun getTypePrefix(): String {
        val currentTime = System.currentTimeMillis()
        if (currentTime - lastRefreshTime > REFRESH_INTERVAL_MS) {
            refresh()
            lastRefreshTime = currentTime
        }
        return cachedTypes
    }

    /**
     * Forces an immediate refresh of the cached type definitions.
     *
     * Loads `piko-ide.d.ts` for the piko namespace API and
     * `piko-actions.d.ts` for action type definitions.
     *
     * For each file, the on-disk version in `dist/ts/` takes priority.
     * If no on-disk version exists, falls back to the embedded resource
     * bundled in the plugin JAR.
     */
    fun refresh() {
        val distTsDir = findDistTsDirectory()

        val builder = StringBuilder()

        val ideContent = loadFromDiskOrResource(distTsDir, IDE_TYPE_FILE)
        if (ideContent.isNotEmpty()) {
            builder.append(ideContent)
            builder.append("\n")
        }

        val actionsContent = loadFromDiskOrResource(distTsDir, ACTIONS_TYPE_FILE)
        if (actionsContent.isNotEmpty()) {
            builder.append(actionsContent)
            builder.append("\n")
        }

        cachedTypes = builder.toString()
    }

    /**
     * Finds the `dist/ts` directory relative to the project root.
     *
     * @return The virtual file for the dist/ts directory, or null if
     *         not found.
     */
    private fun findDistTsDirectory(): VirtualFile? {
        val projectBasePath = project.basePath ?: return null
        val projectRoot = LocalFileSystem.getInstance().findFileByPath(projectBasePath)
            ?: return null

        return projectRoot.findFileByRelativePath(DIST_TS_PATH)
    }

    /**
     * Loads the text content of a virtual file.
     *
     * @param file The file to load.
     * @return The file content as a string, or empty string on error.
     */
    private fun loadFileContent(file: VirtualFile): String {
        return try {
            val content = VfsUtil.loadText(file)
            convertToAmbientDeclarations(content)
        } catch (e: Exception) {
            LOG.debug("Failed to load type definition file: ${file.path}", e)
            ""
        }
    }

    /**
     * Loads a type definition file from disk if available, falling back
     * to the embedded resource bundled in the plugin JAR.
     *
     * @param distTsDir The dist/ts directory, or null if not found.
     * @param fileName The file name to load.
     * @return The processed type definition content, or empty string.
     */
    private fun loadFromDiskOrResource(distTsDir: VirtualFile?, fileName: String): String {
        distTsDir?.findChild(fileName)?.let { file ->
            val content = loadFileContent(file)
            if (content.isNotEmpty()) {
                return content
            }
        }

        return loadEmbeddedResource(fileName)
    }

    /**
     * Loads a type definition from the plugin's embedded resources.
     *
     * @param fileName The resource file name (e.g. "piko-ide.d.ts").
     * @return The processed content, or empty string if not found.
     */
    private fun loadEmbeddedResource(fileName: String): String {
        return try {
            val resourcePath = "$RESOURCE_TYPES_PREFIX$fileName"
            val stream = javaClass.getResourceAsStream(resourcePath) ?: run {
                LOG.debug("Embedded type definition not found: $resourcePath")
                return ""
            }
            val content = stream.bufferedReader().use { it.readText() }
            convertToAmbientDeclarations(content)
        } catch (e: Exception) {
            LOG.debug("Failed to load embedded type definition: $fileName", e)
            ""
        }
    }

    /**
     * Converts ES module export declarations to ambient declarations.
     *
     * Files that already contain `declare global {` are returned unchanged,
     * as they have been processed by the build pipeline. For legacy files
     * that still use `export` keywords, strips the export prefixes:
     * `export declare` becomes `declare`, `export interface` becomes
     * `declare interface`, `export type` becomes `declare type`,
     * `export enum` becomes `declare enum`, and standalone
     * `export { ... }` statements are removed entirely.
     *
     * @param content The type definition content.
     * @return The content with exports converted to ambient declarations.
     */
    private fun convertToAmbientDeclarations(content: String): String {
        if (content.contains("declare global {")) {
            return content
        }

        return content
            .replace(Regex("^export declare ", RegexOption.MULTILINE), "declare ")
            .replace(Regex("^export interface ", RegexOption.MULTILINE), "declare interface ")
            .replace(Regex("^export type ", RegexOption.MULTILINE), "declare type ")
            .replace(Regex("^export enum ", RegexOption.MULTILINE), "declare enum ")
            .replace(Regex("^export \\{[^}]*\\};?\\s*$", setOf(RegexOption.MULTILINE)), "")
    }

    /**
     * Sets up a file watcher to detect changes to type definition files.
     *
     * Watches for creation and modification of `piko-ide.d.ts` and `piko-actions.d.ts`
     * in the project's `dist/ts/` directory. When changes are detected, refreshes
     * the cache and invalidates PSI to trigger re-injection.
     */
    private fun setupFileWatcher() {
        val connection = project.messageBus.connect(this)
        connection.subscribe(VirtualFileManager.VFS_CHANGES, object : BulkFileListener {
            override fun after(events: List<VFileEvent>) {
                if (project.isDisposed) return

                val hasRelevantChange = events.any { event ->
                    isRelevantTypeFile(event)
                }

                if (hasRelevantChange) {
                    LOG.info("Piko type definitions changed, refreshing cache")
                    onTypeFilesChanged()
                }
            }
        })
        LOG.info("Piko type definition file watcher registered")
    }

    /**
     * Checks if a file event is for a relevant type definition file.
     *
     * @param event The file event to check.
     * @return True if the event is for piko-ide.d.ts or piko-actions.d.ts in dist/ts/.
     */
    private fun isRelevantTypeFile(event: VFileEvent): Boolean {
        if (event !is VFileCreateEvent && event !is VFileContentChangeEvent) {
            return false
        }

        val fileName = event.file?.name ?: (event as? VFileCreateEvent)?.childName ?: return false
        if (fileName !in TYPE_FILE_NAMES) {
            return false
        }

        val filePath = event.path
        val projectBasePath = project.basePath ?: return false
        val expectedPath = "$projectBasePath/$DIST_TS_PATH/"

        return filePath.startsWith(expectedPath)
    }

    /**
     * Called when type definition files are created or modified.
     *
     * Refreshes the cache and invalidates PSI to trigger re-injection
     * of type definitions in all open .pk files.
     */
    private fun onTypeFilesChanged() {
        refresh()
        lastRefreshTime = System.currentTimeMillis()

        PsiManager.getInstance(project).dropPsiCaches()
    }

    /**
     * Disposes of resources when the service is shut down.
     */
    override fun dispose() {
    }
}
