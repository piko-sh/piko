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
import com.intellij.psi.PsiRecursiveElementVisitor
import com.intellij.psi.PsiReference
import com.intellij.psi.PsiReferenceBase
import com.intellij.psi.TokenType
import com.intellij.psi.css.CssClass
import com.intellij.psi.search.searches.ReferencesSearch
import com.intellij.openapi.util.TextRange
import com.intellij.util.Processor
import io.politepixels.piko.pk.PKTokenTypes
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
     * Walks the PK file's PSI tree looking for template tokens that reference
     * the given CSS class name.
     *
     * @param file      The PK file to scan.
     * @param className The CSS class name to search for.
     * @param target    The CSS class element that references resolve to.
     * @param consumer  The processor that receives found references.
     */
    private fun scanForClassUsages(
        file: PsiFile,
        className: String,
        target: CssClass,
        consumer: Processor<in PsiReference>
    ) {
        file.accept(object : PsiRecursiveElementVisitor() {
            override fun visitElement(element: PsiElement) {
                val elementType = element.node?.elementType

                if (elementType == PKTokenTypes.HTML_ATTR_VALUE) {
                    if (!processAttrValue(element, className, target, consumer)) return
                }

                if (elementType == PKTokenTypes.DIRECTIVE_BIND) {
                    if (!processDirectiveBind(element, className, target, consumer)) return
                }

                super.visitElement(element)
            }
        })
    }

    /**
     * Checks an `HTML_ATTR_VALUE` token for class name references.
     *
     * Verifies the preceding attribute name is "class", then finds all
     * whole-word occurrences of the class name within the value.
     *
     * @return False if the consumer signalled to stop processing.
     */
    private fun processAttrValue(
        element: PsiElement,
        className: String,
        target: CssClass,
        consumer: Processor<in PsiReference>
    ): Boolean {
        val attrName = findPrecedingAttrName(element) ?: return true
        if (attrName.text != "class") return true

        val text = element.text
        var offset = 0

        while (true) {
            val idx = text.indexOf(className, offset)
            if (idx == -1) break

            val endIdx = idx + className.length
            val isWordStart = idx == 0 || text[idx - 1].isWhitespace()
            val isWordEnd = endIdx == text.length || text[endIdx].isWhitespace()

            if (isWordStart && isWordEnd) {
                val ref = PKCssClassReference(element, TextRange(idx, endIdx), target)
                if (!consumer.process(ref)) return false
            }

            offset = endIdx
        }

        return true
    }

    /**
     * Checks a `DIRECTIVE_BIND` token for a `p-class:name` shorthand usage.
     *
     * @return False if the consumer signalled to stop processing.
     */
    private fun processDirectiveBind(
        element: PsiElement,
        className: String,
        target: CssClass,
        consumer: Processor<in PsiReference>
    ): Boolean {
        val text = element.text
        if (!text.startsWith(":")) return true
        if (text.substring(1) != className) return true

        val directiveName = findPrecedingDirectiveName(element) ?: return true
        if (directiveName.text != "p-class") return true

        val ref = PKCssClassReference(element, TextRange(1, text.length), target)
        return consumer.process(ref)
    }

    /**
     * Walks backwards from an `HTML_ATTR_VALUE` token to find the preceding
     * `HTML_ATTR_NAME`, skipping quotes, equals signs, and whitespace.
     *
     * @param element The starting PSI element.
     * @return The preceding attribute name element, or null if not found.
     */
    private fun findPrecedingAttrName(element: PsiElement): PsiElement? {
        var prev = element.prevSibling
        while (prev != null) {
            when (prev.node?.elementType) {
                PKTokenTypes.HTML_ATTR_QUOTE,
                PKTokenTypes.HTML_ATTR_EQ,
                TokenType.WHITE_SPACE -> prev = prev.prevSibling
                PKTokenTypes.HTML_ATTR_NAME -> return prev
                else -> return null
            }
        }
        return null
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
