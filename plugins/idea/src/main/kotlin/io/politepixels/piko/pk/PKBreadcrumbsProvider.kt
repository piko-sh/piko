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


package io.politepixels.piko.pk

import com.intellij.lang.Language
import com.intellij.psi.PsiElement
import com.intellij.ui.breadcrumbs.BreadcrumbsProvider
import io.politepixels.piko.pk.psi.impl.PkPsiElementImpl

/**
 * Provides breadcrumb navigation for PK template files.
 *
 * Shows the structural path from the file root to the cursor position,
 * displaying block and body element names in the editor breadcrumb bar.
 */
class PKBreadcrumbsProvider : BreadcrumbsProvider {

    companion object {
        /**
         * Returns whether the given element type should appear in breadcrumbs.
         *
         * @param elementType The element type to check.
         * @return true if the element type should appear in breadcrumbs.
         */
        @JvmStatic
        fun isAcceptedElementType(elementType: com.intellij.psi.tree.IElementType): Boolean {
            return when (elementType) {
                PKTokenTypes.TEMPLATE_BLOCK_ELEMENT,
                PKTokenTypes.SCRIPT_BLOCK_ELEMENT,
                PKTokenTypes.STYLE_BLOCK_ELEMENT,
                PKTokenTypes.I18N_BLOCK_ELEMENT,
                PKTokenTypes.TEMPLATE_BODY_ELEMENT,
                PKTokenTypes.GO_SCRIPT_BODY_ELEMENT,
                PKTokenTypes.JS_SCRIPT_BODY_ELEMENT,
                PKTokenTypes.CSS_STYLE_BODY_ELEMENT,
                PKTokenTypes.I18N_BODY_ELEMENT -> true
                else -> false
            }
        }

        /**
         * Returns the display text for the given element type.
         *
         * @param elementType The element type to get display text for.
         * @return The display text for the element type.
         */
        @JvmStatic
        fun getElementInfoForType(elementType: com.intellij.psi.tree.IElementType): String {
            return when (elementType) {
                PKTokenTypes.TEMPLATE_BLOCK_ELEMENT -> PKConstants.BreadcrumbDisplay.TEMPLATE
                PKTokenTypes.STYLE_BLOCK_ELEMENT -> PKConstants.BreadcrumbDisplay.STYLE
                PKTokenTypes.I18N_BLOCK_ELEMENT -> PKConstants.BreadcrumbDisplay.I18N
                PKTokenTypes.TEMPLATE_BODY_ELEMENT -> PKConstants.BreadcrumbDisplay.CONTENT
                PKTokenTypes.GO_SCRIPT_BODY_ELEMENT -> PKConstants.LanguageDisplay.GO
                PKTokenTypes.JS_SCRIPT_BODY_ELEMENT -> PKConstants.LanguageDisplay.JAVASCRIPT
                PKTokenTypes.CSS_STYLE_BODY_ELEMENT -> PKConstants.LanguageDisplay.CSS
                PKTokenTypes.I18N_BODY_ELEMENT -> PKConstants.LanguageDisplay.JSON
                else -> ""
            }
        }

        /**
         * Returns the tooltip text for the given element type.
         *
         * @param elementType The element type to get tooltip text for.
         * @return The tooltip text, or null if no tooltip is defined.
         */
        @JvmStatic
        fun getTooltipForType(elementType: com.intellij.psi.tree.IElementType): String? {
            return when (elementType) {
                PKTokenTypes.TEMPLATE_BLOCK_ELEMENT -> "Template block - HTML content"
                PKTokenTypes.SCRIPT_BLOCK_ELEMENT -> "Script block - application logic"
                PKTokenTypes.STYLE_BLOCK_ELEMENT -> "Style block - CSS rules"
                PKTokenTypes.I18N_BLOCK_ELEMENT -> "Internationalisation block - translations"
                PKTokenTypes.TEMPLATE_BODY_ELEMENT -> "Template content"
                PKTokenTypes.GO_SCRIPT_BODY_ELEMENT -> "Go code"
                PKTokenTypes.JS_SCRIPT_BODY_ELEMENT -> "JavaScript code"
                PKTokenTypes.CSS_STYLE_BODY_ELEMENT -> "CSS content"
                PKTokenTypes.I18N_BODY_ELEMENT -> "JSON translations"
                else -> null
            }
        }
    }

    /**
     * Returns the languages this provider supports.
     *
     * @return Array containing only PKLanguage.
     */
    override fun getLanguages(): Array<Language> = arrayOf(PKLanguage)

    /**
     * Determines if an element should appear in the breadcrumb bar.
     *
     * @param element The PSI element to check.
     * @return true if the element is a block or body element.
     */
    override fun acceptElement(element: PsiElement): Boolean {
        val elementType = element.node?.elementType ?: return false
        return isAcceptedElementType(elementType)
    }

    /**
     * Returns the display text for a breadcrumb element.
     *
     * @param element The PSI element to display.
     * @return The breadcrumb label text.
     */
    override fun getElementInfo(element: PsiElement): String {
        val elementType = element.node?.elementType ?: return ""
        if (elementType == PKTokenTypes.SCRIPT_BLOCK_ELEMENT) {
            return getScriptInfo(element)
        }
        return getElementInfoForType(elementType)
    }

    /**
     * Returns the tooltip text for a breadcrumb element.
     *
     * @param element The PSI element to get tooltip for.
     * @return The tooltip text, or null if none.
     */
    override fun getElementTooltip(element: PsiElement): String? {
        val elementType = element.node?.elementType ?: return null
        return getTooltipForType(elementType)
    }

    /**
     * Returns the parent element for breadcrumb navigation.
     *
     * @param element The current PSI element.
     * @return The parent element, or null if at the root.
     */
    override fun getParent(element: PsiElement): PsiElement? {
        var parent = element.parent

        while (parent != null) {
            if (acceptElement(parent)) {
                return parent
            }
            if (parent is PKPsiFile) {
                return null
            }
            parent = parent.parent
        }

        return null
    }

    /**
     * Returns script-specific breadcrumb text indicating the language.
     *
     * @param element The script block element.
     * @return The script label with language suffix.
     */
    private fun getScriptInfo(element: PsiElement): String {
        if (element !is PkPsiElementImpl) {
            return PKConstants.BreadcrumbDisplay.SCRIPT
        }

        var child = element.firstChild
        while (child != null) {
            if (child.node?.elementType == PKTokenTypes.ATTR_VALUE) {
                val value = child.text.trim('"', '\'')
                if (value == PKConstants.ScriptLang.JS || value == PKConstants.ScriptLang.JAVASCRIPT) {
                    return PKConstants.BreadcrumbDisplay.SCRIPT_JS
                }
                if (value == PKConstants.ScriptLang.TYPESCRIPT) {
                    return PKConstants.BreadcrumbDisplay.SCRIPT_TS
                }
            }
            child = child.nextSibling
        }

        return PKConstants.BreadcrumbDisplay.SCRIPT_GO
    }
}
