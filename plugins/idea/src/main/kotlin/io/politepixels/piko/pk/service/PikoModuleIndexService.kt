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

import com.intellij.openapi.components.Service
import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.VfsUtil
import com.intellij.openapi.vfs.VirtualFile
import com.intellij.psi.util.CachedValue
import com.intellij.psi.util.CachedValueProvider
import com.intellij.psi.util.CachedValuesManager
import java.nio.file.Paths
import java.util.concurrent.ConcurrentHashMap

/**
 * Provides cached access to Go module information for PK files.
 *
 * Parses go.mod files to extract the module path and any local replace
 * directives. Results are cached and invalidated when the go.mod file changes.
 *
 * @param project The project this service belongs to.
 */
@Service(Service.Level.PROJECT)
class PikoModuleIndexService(private val project: Project) {

    /** Logger for this service. */
    private val log = Logger.getInstance(PikoModuleIndexService::class.java)

    /** Cache of parsed module info keyed by go.mod file. Invalidates when the file changes. */
    private val cache = ConcurrentHashMap<VirtualFile, CachedValue<PikoModuleInfo?>>()

    /**
     * Returns module information for the go.mod containing the given file.
     *
     * Searches upward from the context file to find the nearest go.mod file,
     * parses it, and caches the result. Returns null if no go.mod is found.
     *
     * @param contextFile The file to find module information for.
     * @return The parsed module info, or null if not found.
     */
    fun getModuleInfo(contextFile: VirtualFile): PikoModuleInfo? {
        val goModVFile = findGoMod(contextFile) ?: return null

        val cachedValue = cache.computeIfAbsent(goModVFile) {
            CachedValuesManager.getManager(project).createCachedValue {
                log.debug("Computing PikoModuleInfo for ${goModVFile.path}")
                val moduleInfo = computeModuleInfo(goModVFile)
                CachedValueProvider.Result.create(moduleInfo, goModVFile)
            }
        }
        return cachedValue.value
    }

    /**
     * Parses a go.mod file to extract module information.
     *
     * @param goModVFile The go.mod file to parse.
     * @return The parsed module info, or null if parsing fails.
     */
    private fun computeModuleInfo(goModVFile: VirtualFile): PikoModuleInfo? {
        val moduleRoot = goModVFile.parent ?: return null
        val goModContent = VfsUtil.loadText(goModVFile)

        val modulePath = GoModParser.parseModulePath(goModContent)
        if (modulePath == null) {
            log.warn("Could not parse module path from ${goModVFile.path}")
            return null
        }

        val replacements = mutableMapOf<String, VirtualFile>()
        GoModParser.parseReplacements(goModContent).forEach { (oldPath, newPath) ->
            if (newPath.startsWith(".")) {
                val normalizedPath = Paths.get(moduleRoot.path, newPath).normalize().toString()
                val replacedDir = VfsUtil.findFileByIoFile(Paths.get(normalizedPath).toFile(), true)
                if (replacedDir != null && replacedDir.isDirectory) {
                    replacements[oldPath] = replacedDir
                }
            }
        }

        return PikoModuleInfo(modulePath, moduleRoot, replacements)
    }

    /**
     * Searches upward from a file to find the nearest go.mod file.
     *
     * @param startFile The file to start searching from.
     * @return The go.mod file, or null if not found.
     */
    private fun findGoMod(startFile: VirtualFile): VirtualFile? {
        var current: VirtualFile? = if (startFile.isDirectory) startFile else startFile.parent
        while (current != null) {
            val goMod = current.findChild("go.mod")
            if (goMod != null && !goMod.isDirectory) {
                return goMod
            }
            if (current == project.basePath?.let { VfsUtil.findFileByIoFile(java.io.File(it), false) }) break
            current = current.parent
        }
        return project.basePath?.let { VfsUtil.findFileByIoFile(java.io.File(it), false)?.findChild("go.mod") }
    }
}
