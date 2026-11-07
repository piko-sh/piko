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
import com.intellij.psi.PsiNamedElement
import com.intellij.psi.PsiReference
import com.intellij.psi.PsiReferenceBase
import com.intellij.psi.search.searches.ReferencesSearch
import com.intellij.psi.util.PsiTreeUtil
import com.intellij.util.Processor
import io.politepixels.piko.pk.PKTemplateReferenceScanner
import io.politepixels.piko.pk.psi.impl.PkJsScriptContentElementImpl

/**
 * Finds template usages of JavaScript/TypeScript functions and variables
 * declared at the top level of a `.pk` or `.pkc` script block.
 *
 * Handlers such as `p-on:click="togglePanel"` name script declarations from
 * the PK template, which the JavaScript plugin cannot see. Only bare
 * identifiers are candidates, and each is resolved against the script's
 * top-level declarations before it is reported, so a nested local with the
 * same name is never matched.
 */
class PKJsReferencesSearchContributor :
    QueryExecutorBase<PsiReference, ReferencesSearch.SearchParameters>(true) {

    /**
     * Processes a reference search for a JS function or variable.
     *
     * @param parameters The search parameters containing the target element.
     * @param consumer The processor to receive found references.
     */
    override fun processQuery(
        parameters: ReferencesSearch.SearchParameters,
        consumer: Processor<in PsiReference>
    ) {
        val target = parameters.elementToSearch
        if (target !is JSFunction && target !is JSVariable) return
        val name = (target as? PsiNamedElement)?.name ?: return

        val project = target.project
        val host = InjectedLanguageManager.getInstance(project).getInjectionHost(target)
        if (host !is PkJsScriptContentElementImpl) return
        val scriptFile = target.containingFile ?: return
        val pkFile = host.containingFile ?: return
        val manager = target.manager

        val declaration = PKJsTemplateReference.findTopLevelDeclaration(scriptFile, name)
        if (!manager.areElementsEquivalent(declaration, target)) return

        PKTemplateReferenceScanner.scan(pkFile, PKTemplateReferenceScanner.IDENTIFIER_TYPES, consumer) { element ->
            if (element.text != name || PKTemplateReferenceScanner.isMemberAccess(element)) {
                emptyList()
            } else {
                listOf(PKJsTemplateReference(element, scriptFile))
            }
        }
    }
}

/**
 * Reference from a template identifier to a top-level script declaration.
 *
 * @param element The template identifier token.
 * @param scriptFile The injected JavaScript/TypeScript file of the same PK file.
 */
private class PKJsTemplateReference(
    element: PsiElement,
    private val scriptFile: PsiFile
) : PsiReferenceBase<PsiElement>(element, TextRange(0, element.textLength), true) {

    /**
     * Resolves the identifier to a top-level function or variable.
     *
     * @return The declaration, or null when the script declares none with this name.
     */
    override fun resolve(): PsiElement? = findTopLevelDeclaration(scriptFile, element.text)

    companion object {
        /**
         * Finds the top-level function or variable with the given name.
         *
         * @param scriptFile The injected JavaScript/TypeScript file.
         * @param name The identifier text.
         * @return The declaration, or null when the script declares none with this name.
         */
        fun findTopLevelDeclaration(scriptFile: PsiFile, name: String): PsiElement? =
            PsiTreeUtil.findChildrenOfAnyType(scriptFile, JSFunction::class.java, JSVariable::class.java)
                .asSequence()
                .filterIsInstance<PsiNamedElement>()
                .firstOrNull { declaration ->
                    declaration.name == name &&
                        PsiTreeUtil.getParentOfType(declaration, JSFunction::class.java, true) == null
                }
    }
}
