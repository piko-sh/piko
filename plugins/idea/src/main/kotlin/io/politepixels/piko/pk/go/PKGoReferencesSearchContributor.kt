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

import com.goide.psi.GoFile
import com.goide.psi.GoFunctionDeclaration
import com.goide.psi.GoFunctionOrMethodDeclaration
import com.goide.psi.GoMethodDeclaration
import com.intellij.lang.injection.InjectedLanguageManager
import com.intellij.openapi.application.QueryExecutorBase
import com.intellij.openapi.util.TextRange
import com.intellij.psi.PsiElement
import com.intellij.psi.PsiReference
import com.intellij.psi.PsiReferenceBase
import com.intellij.psi.search.searches.ReferencesSearch
import com.intellij.util.Processor
import io.politepixels.piko.pk.PKTemplateReferenceScanner
import io.politepixels.piko.pk.psi.impl.PkGoScriptContentElementImpl

/**
 * Finds template usages of Go functions and methods declared in a `.pk`
 * file's script block.
 *
 * Template expressions such as `{{ formatDate(post.Date) }}` and
 * `p-on:click="handleClick"` name script functions directly, but they live in
 * the PK template rather than Go code, so the Go plugin cannot find them.
 *
 * A bare identifier is only a candidate for a function and a member access
 * (`item.Method`) only for a method. Each candidate is then resolved against
 * the script block, and reported only when it resolves to the searched
 * declaration, so a struct field or loop variable with the same name is not
 * reported as a usage.
 */
class PKGoReferencesSearchContributor :
    QueryExecutorBase<PsiReference, ReferencesSearch.SearchParameters>(true) {

    /**
     * Processes a reference search for a Go function or method.
     *
     * @param parameters The search parameters containing the target element.
     * @param consumer The processor to receive found references.
     */
    override fun processQuery(
        parameters: ReferencesSearch.SearchParameters,
        consumer: Processor<in PsiReference>
    ) {
        val target = parameters.elementToSearch as? GoFunctionOrMethodDeclaration ?: return
        if (target !is GoFunctionDeclaration && target !is GoMethodDeclaration) return
        val name = target.name ?: return

        val host = InjectedLanguageManager.getInstance(target.project).getInjectionHost(target)
        if (host !is PkGoScriptContentElementImpl) return
        val goFile = target.containingFile as? GoFile ?: return
        val pkFile = host.containingFile ?: return
        val isMethod = target is GoMethodDeclaration
        val manager = target.manager

        val declaration = PKGoTemplateReference.resolveDeclaration(goFile, name, isMethod, target)
        if (!manager.areElementsEquivalent(declaration, target)) return

        PKTemplateReferenceScanner.scan(pkFile, PKTemplateReferenceScanner.IDENTIFIER_TYPES, consumer) { element ->
            if (element.text != name || PKTemplateReferenceScanner.isMemberAccess(element) != isMethod) {
                emptyList()
            } else {
                listOf(PKGoTemplateReference(element, goFile, isMethod, target))
            }
        }
    }
}

/**
 * Reference from a template identifier to a Go declaration in the script block.
 *
 * @param element The template identifier token.
 * @param goFile The injected Go file of the same `.pk` file.
 * @param isMethod Whether the identifier is a member access, resolved against methods.
 * @param preferred The searched declaration, chosen when several methods share the name.
 */
private class PKGoTemplateReference(
    element: PsiElement,
    private val goFile: GoFile,
    private val isMethod: Boolean,
    private val preferred: PsiElement
) : PsiReferenceBase<PsiElement>(element, TextRange(0, element.textLength), true) {

    /**
     * Resolves the identifier to a top-level function, or to a method of that name.
     *
     * Methods cannot be told apart without type information, so when several
     * receivers declare the name the searched declaration is preferred.
     *
     * @return The declaration, or null when the script declares none with this name.
     */
    override fun resolve(): PsiElement? = resolveDeclaration(goFile, element.text, isMethod, preferred)

    companion object {
        /**
         * Finds the script declaration a template identifier names.
         *
         * @param goFile The injected Go file of the `.pk` file.
         * @param name The identifier text.
         * @param isMethod Whether the identifier is a member access.
         * @param preferred The declaration chosen when several methods share the name.
         * @return The declaration, or null when the script declares none with this name.
         */
        fun resolveDeclaration(goFile: GoFile, name: String, isMethod: Boolean, preferred: PsiElement): PsiElement? {
            if (!isMethod) {
                return goFile.getFunctions(name).firstOrNull()
            }
            val methods = goFile.getMethods(name)
            return methods.firstOrNull { it.isEquivalentTo(preferred) } ?: methods.firstOrNull()
        }
    }
}
