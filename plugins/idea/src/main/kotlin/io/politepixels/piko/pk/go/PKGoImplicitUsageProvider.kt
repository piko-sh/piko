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


package io.politepixels.piko.pk.go

import com.intellij.codeInsight.daemon.ImplicitUsageProvider
import com.intellij.lang.injection.InjectedLanguageManager
import com.intellij.psi.PsiElement
import com.intellij.psi.PsiNamedElement
import com.intellij.psi.PsiRecursiveElementVisitor
import com.goide.psi.GoFunctionDeclaration
import com.goide.psi.GoMethodDeclaration
import io.politepixels.piko.pk.PKTokenTypes
import io.politepixels.piko.pk.psi.impl.PkGoScriptContentElementImpl

/**
 * Suppresses "unused function" warnings for Go functions inside PK script
 * blocks that are actually referenced from the template.
 *
 * IntelliJ's Go plugin marks functions as unused when it cannot find
 * references within the same file scope. Since PK template directives
 * (`p-on:click="openModal()"`, `{{ FormatPrice .Price }}`) reference these
 * functions via flat token elements that the Go plugin cannot resolve, this
 * provider scans the template for matching references and suppresses the
 * warning only when a reference exists.
 *
 * Functions that are genuinely unreferenced from the template will correctly
 * show "unused" warnings.
 *
 * The `Render` function is always considered used because the PK framework
 * calls it automatically as part of the component lifecycle.
 */
class PKGoImplicitUsageProvider : ImplicitUsageProvider {

    companion object {
        /**
         * PK framework lifecycle function names that are called automatically
         * and should never be marked as unused.
         */
        private val LIFECYCLE_FUNCTIONS = setOf(
            "Render",
        )
    }

    /**
     * Checks whether the given element should be considered implicitly used.
     *
     * Returns true for Go functions that live inside a PK file's script block
     * AND are either lifecycle functions or referenced from the template section.
     *
     * The Go plugin may pass the function declaration directly, or the name
     * identifier element. This method handles both cases by resolving the
     * declaration from either the element or its parent.
     *
     * @param element The PSI element to check.
     * @return True if the element is referenced from the PK template.
     */
    override fun isImplicitUsage(element: PsiElement): Boolean {
        val funcElement = resolveGoFunction(element) ?: return false
        val name = funcElement.name ?: return false

        val host = InjectedLanguageManager.getInstance(element.project)
            .getInjectionHost(funcElement) ?: return false
        if (host !is PkGoScriptContentElementImpl) return false

        if (name in LIFECYCLE_FUNCTIONS) return true

        val pkFile = host.containingFile ?: return false
        return isReferencedInTemplate(pkFile, name)
    }

    /**
     * Checks whether the given element should be considered implicitly read.
     *
     * @param element The PSI element to check.
     * @return Always false; read detection is not needed for Go symbols.
     */
    override fun isImplicitRead(element: PsiElement): Boolean = false

    /**
     * Checks whether the given element should be considered implicitly written.
     *
     * @param element The PSI element to check.
     * @return Always false; write detection is not needed for Go symbols.
     */
    override fun isImplicitWrite(element: PsiElement): Boolean = false

    /**
     * Resolves a Go function or method declaration from the given element.
     *
     * Handles two cases:
     * - The element is a [GoFunctionDeclaration] or [GoMethodDeclaration] directly.
     * - The element is the name identifier, whose parent is the declaration.
     *
     * @param element The PSI element to resolve.
     * @return The function/method declaration, or null if not applicable.
     */
    private fun resolveGoFunction(element: PsiElement): PsiNamedElement? {
        if (element is GoFunctionDeclaration) return element
        if (element is GoMethodDeclaration) return element

        val parent = element.parent
        if (parent is GoFunctionDeclaration) return parent
        if (parent is GoMethodDeclaration) return parent

        return null
    }

    /**
     * Scans the PK file's template section for expression tokens that match
     * the given function name.
     *
     * @param pkFile The PK file to scan.
     * @param name   The function name to search for.
     * @return True if a matching `EXPR_FUNCTION_NAME` or `EXPR_IDENTIFIER`
     *         token is found.
     */
    private fun isReferencedInTemplate(pkFile: PsiElement, name: String): Boolean {
        var found = false

        pkFile.accept(object : PsiRecursiveElementVisitor() {
            override fun visitElement(element: PsiElement) {
                if (found) return

                val elementType = element.node?.elementType
                if (elementType == PKTokenTypes.EXPR_FUNCTION_NAME ||
                    elementType == PKTokenTypes.EXPR_IDENTIFIER
                ) {
                    if (element.text == name) {
                        found = true
                        return
                    }
                }

                super.visitElement(element)
            }
        })

        return found
    }
}
