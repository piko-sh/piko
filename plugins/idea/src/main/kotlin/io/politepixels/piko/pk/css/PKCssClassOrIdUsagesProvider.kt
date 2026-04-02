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

import com.intellij.psi.PsiElement
import com.intellij.psi.TokenType
import com.intellij.psi.css.CssSelectorSuffix
import com.intellij.psi.css.CssSelectorSuffixType
import com.intellij.psi.css.usages.CssClassOrIdUsagesProvider
import io.politepixels.piko.pk.PKLanguage
import io.politepixels.piko.pk.PKTokenTypes

/**
 * Provides CSS class and ID usage detection for PK template files.
 *
 * When IntelliJ's CSS plugin searches for usages of a CSS selector (to
 * determine whether it is "unused"), it iterates over text matches and calls
 * [isUsage] for each candidate element. This provider confirms usages that
 * appear inside PK template attributes:
 *
 * - **Static class attributes** - `class="foo bar"` where class names are
 *   whitespace-delimited tokens inside an `HTML_ATTR_VALUE` preceded by an
 *   `HTML_ATTR_NAME` whose text is "class".
 *
 * - **p-class shorthand directives** - `p-class:active="expr"` where the
 *   class name is the suffix in a `DIRECTIVE_BIND` token preceded by a
 *   `DIRECTIVE_NAME` whose text is "p-class".
 *
 * Dynamic class bindings (`p-class="{ 'x': expr }"`, `:class="computed"`)
 * use runtime expressions that cannot be statically resolved, so selectors
 * referenced only via dynamic bindings will correctly appear as "unused."
 */
class PKCssClassOrIdUsagesProvider : CssClassOrIdUsagesProvider {

    /**
     * Determines whether the candidate PSI element at the given offset is a
     * usage of the specified CSS selector suffix.
     *
     * @param selectorSuffix The CSS `.class` or `#id` selector being searched.
     * @param candidate      The PSI element containing a text match.
     * @param offsetInCandidate Character offset of the match within the element.
     * @return True if the candidate is a confirmed usage of the selector.
     */
    override fun isUsage(
        selectorSuffix: CssSelectorSuffix,
        candidate: PsiElement,
        offsetInCandidate: Int
    ): Boolean {
        if (candidate.containingFile?.language != PKLanguage) return false
        if (selectorSuffix.type != CssSelectorSuffixType.CLASS) return false

        val elementType = candidate.node?.elementType ?: return false
        val className = selectorSuffix.name ?: return false

        if (elementType == PKTokenTypes.HTML_ATTR_VALUE) {
            return isStaticClassUsage(candidate, className, offsetInCandidate)
        }

        if (elementType == PKTokenTypes.DIRECTIVE_BIND) {
            return isPClassShorthandUsage(candidate, className)
        }

        return false
    }

    /**
     * Checks whether the candidate `HTML_ATTR_VALUE` token is inside a
     * `class="..."` attribute and the class name at the given offset is a
     * whole-word match.
     *
     * @param candidate The attribute value PSI element.
     * @param className The CSS class name to match.
     * @param offset    The character offset of the match within the element.
     * @return True if the class name appears as a whole word in a class attribute.
     */
    private fun isStaticClassUsage(
        candidate: PsiElement,
        className: String,
        offset: Int
    ): Boolean {
        val attrName = findPrecedingAttrName(candidate) ?: return false
        if (attrName.text != "class") return false

        val text = candidate.text
        val nameEnd = offset + className.length
        if (nameEnd > text.length) return false
        if (text.substring(offset, nameEnd) != className) return false

        if (offset > 0 && !text[offset - 1].isWhitespace()) return false
        if (nameEnd < text.length && !text[nameEnd].isWhitespace()) return false

        return true
    }

    /**
     * Checks whether the candidate `DIRECTIVE_BIND` token is a `p-class:name`
     * shorthand where the suffix matches the class name.
     *
     * @param candidate The directive bind PSI element.
     * @param className The CSS class name to match.
     * @return True if the bind suffix matches the class name on a p-class directive.
     */
    private fun isPClassShorthandUsage(
        candidate: PsiElement,
        className: String
    ): Boolean {
        val text = candidate.text
        if (!text.startsWith(":")) return false
        if (text.substring(1) != className) return false

        val directiveName = findPrecedingDirectiveName(candidate) ?: return false
        return directiveName.text == "p-class"
    }

    /**
     * Walks backwards through siblings from an `HTML_ATTR_VALUE` token to
     * find the preceding `HTML_ATTR_NAME`, skipping quotes, equals signs,
     * and whitespace.
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
     * Walks backwards through siblings from a `DIRECTIVE_BIND` token to
     * find the preceding `DIRECTIVE_NAME`, skipping whitespace.
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
