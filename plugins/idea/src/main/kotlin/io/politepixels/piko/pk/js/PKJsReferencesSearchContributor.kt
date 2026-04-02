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

import com.intellij.lang.injection.InjectedLanguageManager
import com.intellij.lang.javascript.psi.JSFunction
import com.intellij.lang.javascript.psi.JSVariable
import com.intellij.openapi.application.QueryExecutorBase
import com.intellij.openapi.util.TextRange
import com.intellij.psi.PsiElement
import com.intellij.psi.PsiFile
import com.intellij.psi.PsiRecursiveElementVisitor
import com.intellij.psi.PsiReference
import com.intellij.psi.PsiReferenceBase
import com.intellij.psi.search.searches.ReferencesSearch
import com.intellij.util.Processor
import io.politepixels.piko.pk.PKTokenTypes
import io.politepixels.piko.pk.psi.impl.PkJsScriptContentElementImpl

/**
 * Contributes Find Usages results for JavaScript functions defined in PK
 * script blocks.
 *
 * When the user invokes "Find Usages" on a JS function or variable inside a
 * PK file's `<script>` block, this contributor scans the same file's template
 * section for matching handler references in directive expressions:
 *
 * - **Function call references** - `p-on:click="openModal()"` where
 *   `openModal` is an `EXPR_FUNCTION_NAME` token.
 *
 * - **Bare identifier references** - `p-on:click="handler"` where `handler`
 *   is an `EXPR_IDENTIFIER` token.
 *
 * Results are returned as [PsiReference] objects that resolve back to the
 * original JS function or variable, enabling navigation from usage to
 * definition.
 */
class PKJsReferencesSearchContributor :
    QueryExecutorBase<PsiReference, ReferencesSearch.SearchParameters>(true) {

    /**
     * Processes a Find Usages query for a JS function inside a PK script block.
     *
     * @param parameters The search parameters containing the target element.
     * @param consumer   The processor that receives found references.
     */
    override fun processQuery(
        parameters: ReferencesSearch.SearchParameters,
        consumer: Processor<in PsiReference>
    ) {
        val target = parameters.elementToSearch
        val functionName = when (target) {
            is JSFunction -> target.name
            is JSVariable -> target.name
            else -> return
        } ?: return

        val project = target.project
        val host = InjectedLanguageManager.getInstance(project)
            .getInjectionHost(target) ?: return
        if (host !is PkJsScriptContentElementImpl) return

        val pkFile = host.containingFile ?: return
        scanForHandlerUsages(pkFile, functionName, target, consumer)
    }

    /**
     * Walks the PK file's PSI tree looking for template tokens that reference
     * the given function name.
     *
     * @param file     The PK file to scan.
     * @param functionName The JS function name to search for.
     * @param target   The JS function element that references resolve to.
     * @param consumer The processor that receives found references.
     */
    private fun scanForHandlerUsages(
        file: PsiFile,
        functionName: String,
        target: PsiElement,
        consumer: Processor<in PsiReference>
    ) {
        file.accept(object : PsiRecursiveElementVisitor() {
            override fun visitElement(element: PsiElement) {
                val elementType = element.node?.elementType

                if (elementType == PKTokenTypes.EXPR_FUNCTION_NAME ||
                    elementType == PKTokenTypes.EXPR_IDENTIFIER
                ) {
                    if (element.text == functionName) {
                        val range = TextRange(0, element.textLength)
                        val ref = PKJsFunctionReference(element, range, target)
                        if (!consumer.process(ref)) return
                    }
                }

                super.visitElement(element)
            }
        })
    }
}

/**
 * A lightweight PSI reference from a PK template expression token to a
 * JavaScript function or variable in the script block.
 *
 * @param element The PK template PSI element containing the handler reference.
 * @param range   The text range of the function name within the element.
 * @param target  The JS function or variable this reference resolves to.
 */
private class PKJsFunctionReference(
    element: PsiElement,
    range: TextRange,
    private val target: PsiElement
) : PsiReferenceBase<PsiElement>(element, range, true) {

    /**
     * Resolves this reference to the target JS function element.
     *
     * @return The JS function or variable this reference points to.
     */
    override fun resolve(): PsiElement = target
}
