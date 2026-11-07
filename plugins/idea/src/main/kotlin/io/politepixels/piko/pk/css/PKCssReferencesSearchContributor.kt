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


package io.politepixels.piko.pk.css

import com.intellij.lang.injection.InjectedLanguageManager
import com.intellij.openapi.application.QueryExecutorBase
import com.intellij.psi.PsiElement
import com.intellij.psi.PsiFile
import com.intellij.psi.PsiReference
import com.intellij.psi.PsiReferenceBase
import com.intellij.psi.TokenType
import com.intellij.psi.css.CssClass
import com.intellij.psi.search.searches.ReferencesSearch
import com.intellij.openapi.util.TextRange
import com.intellij.util.Processor
import io.politepixels.piko.pk.PKTemplateReferenceScanner
import io.politepixels.piko.pk.PKTokenTypes
import io.politepixels.piko.pk.PKTreeUtils
import io.politepixels.piko.pk.psi.impl.PkCssStyleContentElementImpl

/**
 * Contributes Find Usages results for CSS classes defined in PK style blocks.
 *
 * When the user invokes "Find Usages" on a CSS class selector (e.g. `.intro`)
 * inside a PK file's `<style>` block, this contributor scans the same file's
 * template section for matching class references in:
 *
 * - **Static class attributes** - `class="foo bar"` where each whitespace-
 *   delimited token is a class name.
 *
 * - **p-class shorthand directives** - `p-class:active="expr"` where the
 *   suffix after the colon is the class name.
 *
 * Results are returned as [PsiReference] objects that resolve back to the
 * original [CssClass] element, enabling navigation from usage to definition.
 */
class PKCssReferencesSearchContributor :
    QueryExecutorBase<PsiReference, ReferencesSearch.SearchParameters>(true) {

    /**
     * Processes a Find Usages query for a CSS class inside a PK style block.
     *
     * @param parameters The search parameters containing the target element.
     * @param consumer   The processor that receives found references.
     */
    override fun processQuery(
        parameters: ReferencesSearch.SearchParameters,
        consumer: Processor<in PsiReference>
    ) {
        val target = parameters.elementToSearch
        if (target !is CssClass) return

        val className = target.name ?: return

        val project = target.project
        val injectionManager = InjectedLanguageManager.getInstance(project)
        val host = injectionManager.getInjectionHost(target) ?: return
        if (host !is PkCssStyleContentElementImpl) return

        val pkFile = host.containingFile ?: return
        scanForClassUsages(pkFile, className, target, consumer)
    }

    /**
     * Scans a PK file for usages of a CSS class name.
     *
     * Stops as soon as the consumer returns false.
     *
     * @param file The PK file to scan.
     * @param className The CSS class name to search for.
     * @param target The CSS class element being searched.
     * @param consumer The processor to receive found references.
     */
    private fun scanForClassUsages(
        file: PsiFile,
        className: String,
        target: CssClass,
        consumer: Processor<in PsiReference>
    ) {
        PKTemplateReferenceScanner.scan(file, CANDIDATE_TYPES, consumer) { element ->
            when (element.node?.elementType) {
                PKTokenTypes.HTML_ATTR_VALUE -> attrValueReferences(element, className, target)
                PKTokenTypes.DIRECTIVE_BIND -> directiveBindReferences(element, className, target)
                else -> emptyList()
            }
        }
    }

    /**
     * Builds references for whole-word occurrences of a class name in a `class`
     * attribute value token.
     *
     * @param element The HTML_ATTR_VALUE element.
     * @param className The CSS class name to search for.
     * @param target The CSS class element being searched.
     * @return The references found in this token.
     */
    private fun attrValueReferences(
        element: PsiElement,
        className: String,
        target: CssClass
    ): List<PsiReference> {
        val attrName = PKTreeUtils.findOwningAttrName(element) ?: return emptyList()
        if (attrName.text != "class") return emptyList()

        val text = element.text
        val references = mutableListOf<PsiReference>()
        var offset = 0
        while (true) {
            val idx = text.indexOf(className, offset)
            if (idx == -1) break

            val endIdx = idx + className.length
            val isWordStart = idx == 0 || text[idx - 1].isWhitespace()
            val isWordEnd = endIdx == text.length || text[endIdx].isWhitespace()
            if (isWordStart && isWordEnd) {
                references.add(PKCssClassReference(element, TextRange(idx, endIdx), target))
            }
            offset = endIdx
        }
        return references
    }

    /**
     * Builds a reference for a `p-class:name` binding that names the class.
     *
     * @param element The DIRECTIVE_BIND element (e.g. `:active`).
     * @param className The CSS class name to search for.
     * @param target The CSS class element being searched.
     * @return The reference, or an empty list when the binding names another class.
     */
    private fun directiveBindReferences(
        element: PsiElement,
        className: String,
        target: CssClass
    ): List<PsiReference> {
        val text = element.text
        if (!text.startsWith(":")) return emptyList()
        if (text.substring(1) != className) return emptyList()

        val directiveName = findPrecedingDirectiveName(element) ?: return emptyList()
        if (directiveName.text != "p-class") return emptyList()

        return listOf(PKCssClassReference(element, TextRange(1, text.length), target))
    }

    /**
     * Walks backwards from a `DIRECTIVE_BIND` token to find the preceding
     * `DIRECTIVE_NAME`, skipping whitespace.
     *
     * @param element The starting PSI element.
     * @return The preceding directive name element, or null if not found.
     */
    private fun findPrecedingDirectiveName(element: PsiElement): PsiElement? {
        var prev = element.prevSibling
        while (prev != null) {
            when (prev.node?.elementType) {
                TokenType.WHITE_SPACE -> prev = prev.prevSibling
                PKTokenTypes.DIRECTIVE_NAME -> return prev
                else -> return null
            }
        }
        return null
    }

    companion object {
        /** Tokens that can contain a class name. */
        private val CANDIDATE_TYPES = setOf(PKTokenTypes.HTML_ATTR_VALUE, PKTokenTypes.DIRECTIVE_BIND)
    }
}

/**
 * A lightweight PSI reference from a PK template token to a CSS class
 * selector in the style block.
 *
 * @param element The PK template PSI element containing the class reference.
 * @param range   The text range of the class name within the element.
 * @param target  The CSS class selector this reference resolves to.
 */
private class PKCssClassReference(
    element: PsiElement,
    range: TextRange,
    private val target: CssClass
) : PsiReferenceBase<PsiElement>(element, range, true) {

    /**
     * Resolves this reference to the target CSS class element.
     *
     * @return The CSS class selector this reference points to.
     */
    override fun resolve(): PsiElement = target
}
