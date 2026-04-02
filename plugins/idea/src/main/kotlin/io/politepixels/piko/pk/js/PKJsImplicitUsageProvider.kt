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

import com.intellij.codeInsight.daemon.ImplicitUsageProvider
import com.intellij.lang.injection.InjectedLanguageManager
import com.intellij.lang.javascript.psi.JSFunction
import com.intellij.lang.javascript.psi.JSVariable
import com.intellij.psi.PsiElement
import com.intellij.psi.PsiRecursiveElementVisitor
import io.politepixels.piko.pk.PKTokenTypes
import io.politepixels.piko.pk.psi.impl.PkJsScriptContentElementImpl

/**
 * Suppresses "unused function" and "unused variable" warnings for JavaScript
 * and TypeScript symbols inside PK script blocks that are actually referenced
 * from the template.
 *
 * IntelliJ's JavaScript plugin marks functions and variables as unused when
 * it cannot find references within the same file scope. Since PK template
 * directives (`p-on:click="openModal()"`) reference these functions via
 * flat token elements that the JS plugin cannot resolve, this provider
 * scans the template for matching handler references and suppresses the
 * warning only when a reference exists.
 *
 * Functions and variables that are genuinely unreferenced from the template
 * will correctly show "unused" warnings.
 *
 * Lifecycle functions (`onConnected`, `onDisconnected`, etc.) are always
 * considered used because the PK framework calls them automatically.
 */
class PKJsImplicitUsageProvider : ImplicitUsageProvider {

    companion object {
        /**
         * PK framework lifecycle function names that are called automatically
         * and should never be marked as unused.
         */
        private val LIFECYCLE_FUNCTIONS = setOf(
            "onConnected",
            "onDisconnected",
            "onBeforeRender",
            "onAfterRender",
            "onUpdated",
            "onCleanup",
        )
    }

    /**
     * Checks whether the given element should be considered implicitly used.
     *
     * Returns true for JS functions and variables that live inside a PK
     * file's script block AND are referenced from the template section.
     *
     * @param element The PSI element to check.
     * @return True if the element is referenced from the PK template.
     */
    override fun isImplicitUsage(element: PsiElement): Boolean {
        val name = when (element) {
            is JSFunction -> element.name
            is JSVariable -> element.name
            else -> return false
        } ?: return false

        val host = InjectedLanguageManager.getInstance(element.project)
            .getInjectionHost(element) ?: return false
        if (host !is PkJsScriptContentElementImpl) return false

        if (name in LIFECYCLE_FUNCTIONS) return true

        val pkFile = host.containingFile ?: return false
        return isReferencedInTemplate(pkFile, name)
    }

    /**
     * Checks whether the given element should be considered implicitly read.
     *
     * @param element The PSI element to check.
     * @return Always false; read detection is not needed for JS symbols.
     */
    override fun isImplicitRead(element: PsiElement): Boolean = false

    /**
     * Checks whether the given element should be considered implicitly written.
     *
     * @param element The PSI element to check.
     * @return Always false; write detection is not needed for JS symbols.
     */
    override fun isImplicitWrite(element: PsiElement): Boolean = false

    /**
     * Scans the PK file's template section for expression tokens that match
     * the given function or variable name.
     *
     * @param pkFile The PK file to scan.
     * @param name   The function or variable name to search for.
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
