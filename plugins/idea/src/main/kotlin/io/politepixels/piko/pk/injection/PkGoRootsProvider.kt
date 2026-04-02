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


package io.politepixels.piko.pk.injection

import com.goide.project.GoRootsProvider
import com.intellij.openapi.components.service
import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.module.Module
import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.VirtualFile
import com.intellij.psi.PsiManager
import com.intellij.lang.injection.InjectedLanguageManager
import com.intellij.util.ThreeState
import io.politepixels.piko.pk.PKPsiFile
import io.politepixels.piko.pk.service.PikoModuleIndexService

/** Logger for the Go roots provider. */
private val LOG = Logger.getInstance(PkGoRootsProvider::class.java)

/**
 * Provides Go source roots for injected Go code in PK files.
 *
 * Tells the Go plugin where to find Go source files when resolving imports
 * from within PK script blocks. Uses the PikoModuleIndexService to find
 * the go.mod file and any module replacements.
 */
class PkGoRootsProvider : GoRootsProvider {

    /**
     * Returns vendor directories to include in import resolution scope.
     *
     * For PK files, returns the module root and any replacement directories
     * defined in go.mod. This enables import resolution for local modules.
     *
     * @param project The current project.
     * @param module The current module, or null.
     * @param file The file being processed, or null.
     * @return A collection of directories to include, or null to defer.
     */
    override fun getVendorDirectoriesInResolveScope(
        project: Project,
        module: Module?,
        file: VirtualFile?
    ): Collection<VirtualFile>? {
        if (file == null) return null

        if (file.extension != "pk" && file.fileType.name != "go") {
            return emptyList()
        }

        val psiFile = PsiManager.getInstance(project).findFile(file) ?: return null

        val hostFile = InjectedLanguageManager.getInstance(project).getInjectionHost(psiFile)?.containingFile
        if (hostFile !is PKPsiFile && psiFile !is PKPsiFile) {
            return emptyList()
        }

        LOG.debug("PkGoRootsProvider: Providing source roots for injected file '${file.name}'.")
        val service = project.service<PikoModuleIndexService>()

        val contextFileForModuleLookup = hostFile?.virtualFile ?: file
        val moduleInfo = service.getModuleInfo(contextFileForModuleLookup)

        if (moduleInfo == null) {
            LOG.debug(" -> No module info found for '${contextFileForModuleLookup.path}'. Returning empty source roots.")
            return emptyList()
        }

        val allRoots = mutableSetOf<VirtualFile>()
        allRoots.add(moduleInfo.moduleRoot)
        allRoots.addAll(moduleInfo.replacements.values)

        LOG.debug(" -> Providing ${allRoots.size} source roots: ${allRoots.joinToString { it.name }}")
        return allRoots
    }

    /**
     * Returns GOPATH root directories.
     *
     * @param project The current project.
     * @param module The current module, or null.
     * @return An empty collection as PK does not use GOPATH.
     */
    override fun getGoPathRoots(project: Project?, module: Module?): Collection<VirtualFile> = emptyList()

    /**
     * Returns GOPATH source directories.
     *
     * @param project The current project.
     * @param module The current module, or null.
     * @return An empty collection as PK does not use GOPATH.
     */
    override fun getGoPathSourcesRoots(project: Project?, module: Module?): Collection<VirtualFile> = emptyList()

    /**
     * Returns GOPATH bin directories.
     *
     * @param project The current project.
     * @param module The current module, or null.
     * @return An empty collection as PK does not use GOPATH.
     */
    override fun getGoPathBinRoots(project: Project?, module: Module?): Collection<VirtualFile> = emptyList()

    /**
     * Indicates whether this provider handles external dependencies.
     *
     * @return False as this provider handles local module sources only.
     */
    override fun isExternal(): Boolean = false

    /**
     * Indicates whether GOPATH sources should be indexed.
     *
     * @param project The current project.
     * @return YES to enable indexing.
     */
    override fun indexGoPathSources(project: Project): ThreeState = ThreeState.YES
}
