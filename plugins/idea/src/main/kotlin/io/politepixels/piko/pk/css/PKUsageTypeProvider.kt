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
import com.intellij.psi.PsiElement
import com.intellij.psi.TokenType
import com.intellij.usages.impl.rules.UsageType
import com.intellij.usages.impl.rules.UsageTypeProvider
import io.politepixels.piko.pk.PKLanguage
import io.politepixels.piko.pk.PKTokenTypes
import io.politepixels.piko.pk.psi.impl.PkCssStyleContentElementImpl
import io.politepixels.piko.pk.psi.impl.PkGoScriptContentElementImpl
import io.politepixels.piko.pk.psi.impl.PkJsScriptContentElementImpl

/**
 * Classifies usages found in PK files so that the Find Usages panel groups
 * them under meaningful labels instead of "Unclassified".
 *
 * Handles five cases:
 * - Template class references (`class="foo"`, `p-class:foo`) in the PK
 *   template section.
 * - CSS class definitions (`.foo { }`) in PK-injected style blocks.
 * - Event handler references (`p-on:click="openModal()"`) in the PK
 *   template section.
 * - JS function definitions in PK-injected script blocks.
 * - Go function definitions in PK-injected script blocks.
 */
class PKUsageTypeProvider : UsageTypeProvider {

    /**
     * Returns the usage type for a PSI element found during a Find Usages search.
     *
     * Delegates to [classifyTemplateElement] for elements in PK files, or
     * [classifyInjectedElement] for elements in injected language files.
     *
     * @param element The PSI element to classify.
     * @return The usage type for grouping in the Find Usages panel, or null.
     */
    override fun getUsageType(element: PsiElement): UsageType? {
        val language = element.containingFile?.language

        if (language == PKLanguage) {
            return classifyTemplateElement(element)
        }

        return classifyInjectedElement(element)
    }

    /**
     * Classifies PK template tokens that reference CSS classes or JS handlers.
     *
     * @param element The PK template PSI element to classify.
     * @return The usage type, or null if the element is not a recognised reference.
     */
    private fun classifyTemplateElement(element: PsiElement): UsageType? {
        when (element.node?.elementType) {
            PKTokenTypes.HTML_ATTR_VALUE -> {
                if (isPrecededByClassAttr(element)) return CSS_CLASS_IN_TEMPLATE
            }
            PKTokenTypes.DIRECTIVE_BIND -> {
                if (isPrecededByPClassDirective(element)) return CSS_CLASS_IN_TEMPLATE
            }
            PKTokenTypes.EXPR_FUNCTION_NAME,
            PKTokenTypes.EXPR_IDENTIFIER -> return JS_HANDLER_IN_TEMPLATE
        }

        return null
    }

    /**
     * Classifies elements that live inside a PK-injected style or script block.
     *
     * @param element The injected PSI element to classify.
     * @return The usage type, or null if the host is not a recognised PK block.
     */
    private fun classifyInjectedElement(element: PsiElement): UsageType? {
        val host = InjectedLanguageManager.getInstance(element.project)
            .getInjectionHost(element)

        if (host is PkCssStyleContentElementImpl) {
            return CSS_CLASS_IN_STYLE
        }

        if (host is PkJsScriptContentElementImpl) {
            return JS_FUNCTION_IN_SCRIPT
        }

        if (host is PkGoScriptContentElementImpl) {
            return GO_FUNCTION_IN_SCRIPT
        }

        return null
    }

    /**
     * Checks whether the element is preceded by a `class` attribute name.
     *
     * Walks backwards through siblings, skipping quotes, equals signs,
     * and whitespace to find the attribute name.
     *
     * @param element The PSI element to check.
     * @return True if the preceding attribute name is "class".
     */
    private fun isPrecededByClassAttr(element: PsiElement): Boolean {
        var prev = element.prevSibling
        while (prev != null) {
            when (prev.node?.elementType) {
                PKTokenTypes.HTML_ATTR_QUOTE,
                PKTokenTypes.HTML_ATTR_EQ,
                TokenType.WHITE_SPACE -> prev = prev.prevSibling
                PKTokenTypes.HTML_ATTR_NAME -> return prev.text == "class"
                else -> return false
            }
        }
        return false
    }

    /**
     * Checks whether the element is preceded by a `p-class` directive name.
     *
     * Walks backwards through siblings, skipping whitespace, to find the
     * directive name.
     *
     * @param element The PSI element to check.
     * @return True if the preceding directive name is "p-class".
     */
    private fun isPrecededByPClassDirective(element: PsiElement): Boolean {
        var prev = element.prevSibling
        while (prev != null) {
            when (prev.node?.elementType) {
                TokenType.WHITE_SPACE -> prev = prev.prevSibling
                PKTokenTypes.DIRECTIVE_NAME -> return prev.text == "p-class"
                else -> return false
            }
        }
        return false
    }

    companion object {
        /** Usage type for CSS class references in template attributes. */
        private val CSS_CLASS_IN_TEMPLATE = UsageType { "CSS class reference in template" }

        /** Usage type for CSS class definitions in style blocks. */
        private val CSS_CLASS_IN_STYLE = UsageType { "CSS class definition in style" }

        /** Usage type for event handler references in template directives. */
        private val JS_HANDLER_IN_TEMPLATE = UsageType { "Event handler reference in template" }

        /** Usage type for function definitions in JavaScript script blocks. */
        private val JS_FUNCTION_IN_SCRIPT = UsageType { "Function definition in script" }

        /** Usage type for function definitions in Go script blocks. */
        private val GO_FUNCTION_IN_SCRIPT = UsageType { "Go function definition in script" }
    }
}
