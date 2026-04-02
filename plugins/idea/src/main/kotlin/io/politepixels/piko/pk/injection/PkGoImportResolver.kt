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

import com.goide.psi.impl.GoPackage
import com.goide.psi.impl.GoPsiImplUtil
import com.goide.psi.impl.imports.GoImportResolver
import com.intellij.openapi.components.service
import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.module.Module
import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.VirtualFile
import com.intellij.psi.PsiManager
import com.intellij.psi.ResolveState
import com.intellij.lang.injection.InjectedLanguageManager
import io.politepixels.piko.pk.PKPsiFile
import io.politepixels.piko.pk.service.PikoModuleIndexService
import io.politepixels.piko.pk.service.PikoModuleInfo

/** Logger for the Go import resolver. */
private val LOG = Logger.getInstance(PkGoImportResolver::class.java)

/**
 * Resolves Go import paths for injected Go code in PK files.
 *
 * Handles import resolution for local modules and replacement modules defined
 * in go.mod. Defers standard library and external module imports to the
 * default Go import resolver.
 */
class PkGoImportResolver : GoImportResolver {

    companion object {
        /**
         * Checks if an import path matches a module path.
         *
         * @param importPath The import path to check.
         * @param modulePath The module path to match against.
         * @return True if the import path matches exactly or is a subpackage.
         */
        fun matchesModulePath(importPath: String, modulePath: String): Boolean {
            return importPath == modulePath || importPath.startsWith("$modulePath/")
        }
    }

    /**
     * Resolves a Go import path to its target package.
     *
     * First checks for module replacements, then checks if the import matches
     * the local module path. Returns null to defer resolution to other resolvers
     * if the import cannot be handled.
     *
     * @param importPath The Go import path to resolve.
     * @param project The current project.
     * @param module The current module, or null.
     * @param resolveState The resolve state containing context information.
     * @return A collection of matching packages, or null to defer.
     */
    override fun resolve(
        importPath: String,
        project: Project,
        module: Module?,
        resolveState: ResolveState?
    ): Collection<GoPackage>? {
        val hostVirtualFile = extractPkHostFile(project, resolveState) ?: return null

        LOG.debug("PkGoImportResolver: Handling import '$importPath' in PK context: ${hostVirtualFile.path}")

        if (importPath.endsWith(".pk")) {
            LOG.debug("  -> Deferring .pk import to PkGoImportPathReference.")
            return null
        }

        val moduleInfo = project.service<PikoModuleIndexService>().getModuleInfo(hostVirtualFile)
        if (moduleInfo == null) {
            LOG.debug("  -> Could not find module info for host file. Deferring.")
            return null
        }

        tryResolveReplacement(importPath, moduleInfo, project, module)?.let { return it }
        tryResolveLocalModule(importPath, moduleInfo, project, module)?.let { return it }

        LOG.debug("  -> FAILED to resolve '$importPath' as a local/replaced module path. Deferring.")
        return null
    }

    /**
     * Extracts the PK host file from the resolve context.
     *
     * Validates that the context element is within an injected Go block
     * inside a PK file with a valid virtual file.
     *
     * @param project The current project.
     * @param resolveState The resolve state containing context information.
     * @return The host virtual file, or null if not in a valid PK context.
     */
    private fun extractPkHostFile(project: Project, resolveState: ResolveState?): VirtualFile? {
        val contextElement = GoPsiImplUtil.getContextElement(resolveState)
        if (contextElement == null) {
            LOG.debug("PkGoImportResolver: Deferring. Context element is null.")
            return null
        }

        val containingFile = contextElement.containingFile
        if (containingFile == null || !containingFile.isValid) {
            LOG.debug("PkGoImportResolver: Deferring. Context element has a null or invalid containing file.")
            return null
        }

        val hostFile = InjectedLanguageManager.getInstance(project).getInjectionHost(contextElement)?.containingFile
        if (hostFile !is PKPsiFile) {
            LOG.debug("PkGoImportResolver: Deferring. Context is not PK.")
            return null
        }

        val hostVirtualFile = hostFile.virtualFile
        if (hostVirtualFile == null) {
            LOG.debug("PkGoImportResolver: Deferring. Host file has no virtual file (in-memory file).")
            return null
        }

        return hostVirtualFile
    }

    /**
     * Attempts to resolve an import via module replacements.
     *
     * Checks if the import path matches any replacement defined in go.mod.
     *
     * @param importPath The Go import path to resolve.
     * @param moduleInfo The parsed module information.
     * @param project The current project.
     * @param module The current module, or null.
     * @return The resolved packages, or null if no replacement matches.
     */
    private fun tryResolveReplacement(
        importPath: String,
        moduleInfo: PikoModuleInfo,
        project: Project,
        module: Module?
    ): Collection<GoPackage>? {
        for ((replacedPath, localReplacementDir) in moduleInfo.replacements) {
            if (!matchesModulePath(importPath, replacedPath)) continue

            val remainingPath = importPath.removePrefix(replacedPath).trimStart('/')
            val targetDir = localReplacementDir.findFileByRelativePath(remainingPath) ?: continue
            val packages = createPackageFromDir(targetDir, project, module) ?: continue

            LOG.debug("    -> SUCCESS (Replace): Resolved '$importPath' to dir '${targetDir.path}'")
            return packages
        }
        return null
    }

    /**
     * Attempts to resolve an import as a local module path.
     *
     * Checks if the import path starts with the module path from go.mod.
     *
     * @param importPath The Go import path to resolve.
     * @param moduleInfo The parsed module information.
     * @param project The current project.
     * @param module The current module, or null.
     * @return The resolved packages, or null if not a local module import.
     */
    private fun tryResolveLocalModule(
        importPath: String,
        moduleInfo: PikoModuleInfo,
        project: Project,
        module: Module?
    ): Collection<GoPackage>? {
        if (!importPath.startsWith(moduleInfo.modulePath + "/")) return null

        val relativePath = importPath.removePrefix(moduleInfo.modulePath).trimStart('/')
        val targetDir = moduleInfo.moduleRoot.findFileByRelativePath(relativePath) ?: return null
        val packages = createPackageFromDir(targetDir, project, module) ?: return null

        LOG.debug("    -> SUCCESS (Local): Resolved '$importPath' to dir '${targetDir.path}'")
        return packages
    }


    /**
     * Creates a Go package from a directory if valid.
     *
     * @param targetDir The target directory.
     * @param project The current project.
     * @param module The current module, or null.
     * @return The packages in the directory, or null if invalid.
     */
    private fun createPackageFromDir(
        targetDir: VirtualFile,
        project: Project,
        module: Module?
    ): Collection<GoPackage>? {
        if (!targetDir.isDirectory) return null
        val psiDir = PsiManager.getInstance(project).findDirectory(targetDir) ?: return null
        return GoPackage.`in`(psiDir, module)
    }
}
