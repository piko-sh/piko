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

import com.goide.psi.GoFile
import com.goide.psi.impl.GoPackage
import com.intellij.lang.injection.InjectedLanguageManager
import com.intellij.openapi.module.Module
import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.VirtualFile
import com.intellij.psi.PsiElement
import com.intellij.psi.PsiFile
import com.intellij.psi.PsiManager
import com.intellij.psi.search.GlobalSearchScope
import com.intellij.psi.util.PsiTreeUtil
import com.intellij.util.Processor
import io.politepixels.piko.pk.PKPsiFile
import io.politepixels.piko.pk.psi.impl.PkGoScriptContentElementImpl
import io.politepixels.piko.pk.service.PikoModuleInfo
import java.util.function.Predicate

/**
 * Where a `.pk` component import points, relative to a module root.
 *
 * @property replacedModulePath The replaced module path from go.mod, or null for the current module.
 * @property relativePath The `.pk` file path relative to that module's root.
 */
data class PkImportTarget(val replacedModulePath: String?, val relativePath: String)

/**
 * Maps `.pk` component import paths to files, mirroring the Piko compiler.
 */
object PkComponentImports {

    /** Prefix for imports relative to the current module root. */
    private const val MODULE_ALIAS_PREFIX = "@/"

    /** File extension of Piko components. */
    private const val PK_EXTENSION = ".pk"

    /**
     * Reports whether an import path refers to a `.pk` component.
     *
     * @param importPath The Go import path.
     * @return True for `.pk` component imports.
     */
    @JvmStatic
    fun isComponentImport(importPath: String): Boolean = importPath.endsWith(PK_EXTENSION)

    /**
     * Splits a component import into its module and relative path.
     *
     * @param importPath The `.pk` import path.
     * @param modulePath The current module path from go.mod.
     * @param replacedModulePaths Module paths that go.mod replaces with local directories.
     * @return The target, or null when the import is outside every known module.
     */
    @JvmStatic
    fun target(importPath: String, modulePath: String, replacedModulePaths: Collection<String>): PkImportTarget? {
        if (!isComponentImport(importPath)) return null

        val target = when {
            importPath.startsWith(MODULE_ALIAS_PREFIX) ->
                PkImportTarget(null, importPath.removePrefix(MODULE_ALIAS_PREFIX))
            importPath.startsWith("$modulePath/") ->
                PkImportTarget(null, importPath.removePrefix("$modulePath/"))
            else -> {
                val replaced = replacedModulePaths
                    .filter { importPath.startsWith("$it/") }
                    .maxByOrNull { it.length }
                    ?: return null
                PkImportTarget(replaced, importPath.removePrefix("$replaced/"))
            }
        }
        return target.takeIf { isContainedRelativePath(it.relativePath) }
    }

    /**
     * Reports whether a relative path stays inside its module root.
     *
     * @param relativePath The path relative to a module root.
     * @return True when every path element is a plain name.
     */
    private fun isContainedRelativePath(relativePath: String): Boolean =
        relativePath.split('/').none { it.isEmpty() || it == "." || it == ".." || it.contains('\\') }

    /**
     * Resolves a component import to its `.pk` file.
     *
     * @param importPath The `.pk` import path.
     * @param moduleInfo The module containing the importing file.
     * @return The component file, or null when it does not exist.
     */
    @JvmStatic
    fun resolveFile(importPath: String, moduleInfo: PikoModuleInfo): VirtualFile? {
        val target = target(importPath, moduleInfo.modulePath, moduleInfo.replacements.keys) ?: return null
        val root = if (target.replacedModulePath == null) {
            moduleInfo.moduleRoot
        } else {
            moduleInfo.replacements[target.replacedModulePath]
        } ?: return null
        return root.findFileByRelativePath(target.relativePath)?.takeIf { !it.isDirectory }
    }

    /**
     * Finds the injected Go file for a component's script block.
     *
     * @param project The current project.
     * @param componentFile The `.pk` component file.
     * @return The injected Go file, or null when the component has no Go script.
     */
    @JvmStatic
    fun findInjectedGoFile(project: Project, componentFile: VirtualFile): GoFile? {
        val pkFile = PsiManager.getInstance(project).findFile(componentFile) as? PKPsiFile ?: return null
        val host = PsiTreeUtil.findChildOfType(pkFile, PkGoScriptContentElementImpl::class.java) ?: return null
        val injected = InjectedLanguageManager.getInstance(project).getInjectedPsiFiles(host) ?: return null
        return injected.firstNotNullOfOrNull { it.first.containingFile as? GoFile }
    }
}

/**
 * A Go package backed by the injected script block of one `.pk` component.
 *
 * @param project The current project.
 * @param goFile The injected Go file of the component's script block.
 * @param componentFile The `.pk` component file.
 * @param componentDirectory The directory containing the component, used as the package directory.
 * @param importPath The import path as written in the importing file.
 */
class PkComponentGoPackage(
    project: Project,
    private val goFile: GoFile,
    private val componentFile: VirtualFile,
    componentDirectory: VirtualFile,
    private val importPath: String
) : GoPackage(project, goFile.packageName ?: DEFAULT_PACKAGE_NAME, componentDirectory) {

    /**
     * Returns the injected Go file as the package's only file.
     *
     * @return A single-element collection.
     */
    override fun files(): Collection<PsiFile> = listOf(goFile)

    /**
     * Processes the package's only file.
     *
     * @param processor The file processor.
     * @return False when the processor stopped early.
     */
    override fun processFiles(processor: Processor<in PsiFile>): Boolean = processor.process(goFile)

    /**
     * Processes the package's only file when the filter accepts it.
     *
     * @param processor The file processor.
     * @param filter The virtual file filter.
     * @return False when the processor stopped early.
     */
    override fun processFiles(processor: Processor<in PsiFile>, filter: Predicate<VirtualFile>): Boolean {
        val virtualFile = goFile.virtualFile
        if (virtualFile != null && !filter.test(virtualFile)) return true
        return processor.process(goFile)
    }

    /**
     * Processes the package's only file; script blocks have no build constraints.
     *
     * @param contextFile The file the request comes from, unused.
     * @param module The module, unused.
     * @param processor The file processor.
     * @return False when the processor stopped early.
     */
    override fun processBuildableFiles(contextFile: PsiFile?, module: Module?, processor: Processor<in PsiFile>): Boolean =
        processor.process(goFile)

    /**
     * Processes the package's only file when the filter accepts it.
     *
     * @param contextFile The file the request comes from, unused.
     * @param module The module, unused.
     * @param processor The file processor.
     * @param filter The virtual file filter.
     * @return False when the processor stopped early.
     */
    override fun processBuildableFiles(
        contextFile: PsiFile?,
        module: Module?,
        processor: Processor<in PsiFile>,
        filter: Predicate<VirtualFile>
    ): Boolean = processFiles(processor, filter)

    /**
     * Returns the injected file when it matches.
     *
     * @param predicate The file predicate.
     * @return The injected file, or null when it does not match.
     */
    override fun find(predicate: Predicate<PsiFile>): PsiFile? = goFile.takeIf { predicate.test(it) }

    /**
     * Restricts the package scope to the component file.
     *
     * @return A scope containing the component file.
     */
    override fun getScope(): GlobalSearchScope = GlobalSearchScope.fileScope(project, componentFile)

    /**
     * Restricts the package scope to the component file.
     *
     * @param contextFile The file the request comes from, unused.
     * @return A scope containing the component file.
     */
    override fun getScope(contextFile: PsiFile?): GlobalSearchScope = scope

    /**
     * Returns the import path as written.
     *
     * @param withVendoring Unused; components are never vendored.
     * @return The `.pk` import path.
     */
    override fun getImportPath(withVendoring: Boolean): String = importPath

    /**
     * Returns the import path as written.
     *
     * @param withVendoring Unused; components are never vendored.
     * @return A single-element collection with the `.pk` import path.
     */
    override fun getAllImportPaths(withVendoring: Boolean): Collection<String> = listOf(importPath)

    /**
     * Navigates to the component file.
     *
     * @return The component PSI file, or the injected file when the component cannot be found.
     */
    override fun getNavigableElement(): PsiElement? =
        PsiManager.getInstance(project).findFile(componentFile) ?: goFile

    /**
     * Reports whether both the component and its injected file are still valid.
     *
     * @return True when valid.
     */
    override fun isValid(): Boolean = componentFile.isValid && goFile.isValid

    /**
     * Compares by component file and import path.
     *
     * @param other The object to compare.
     * @return True for the same component imported by the same path.
     */
    override fun equals(other: Any?): Boolean =
        other is PkComponentGoPackage && other.componentFile == componentFile && other.importPath == importPath

    /**
     * Hashes by component file and import path.
     *
     * @return The hash code.
     */
    override fun hashCode(): Int = 31 * componentFile.hashCode() + importPath.hashCode()

    companion object {
        /** Package name used when the script block has no package clause. */
        private const val DEFAULT_PACKAGE_NAME = "main"
    }
}
