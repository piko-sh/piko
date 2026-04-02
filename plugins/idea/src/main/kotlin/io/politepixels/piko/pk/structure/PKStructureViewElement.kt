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


package io.politepixels.piko.pk.structure

import com.intellij.icons.AllIcons
import com.intellij.ide.projectView.PresentationData
import com.intellij.ide.structureView.StructureViewTreeElement
import com.intellij.ide.util.treeView.smartTree.SortableTreeElement
import com.intellij.ide.util.treeView.smartTree.TreeElement
import com.intellij.navigation.ItemPresentation
import com.intellij.psi.NavigatablePsiElement
import com.intellij.psi.PsiElement
import com.intellij.psi.PsiFile
import io.politepixels.piko.pk.PKConstants
import io.politepixels.piko.pk.PKIcons
import io.politepixels.piko.pk.PKTokenTypes
import io.politepixels.piko.pk.psi.impl.PkPsiElementImpl
import javax.swing.Icon

/**
 * Represents a single element in the Structure View tree.
 *
 * Each element displays with an icon and label based on its type.
 * Block elements show children, while body elements are leaves.
 */
class PKStructureViewElement(
    private val element: PsiElement
) : StructureViewTreeElement, SortableTreeElement {

    companion object {
        /** Maximum length for truncated element text display. */
        private const val MAX_TEXT_LENGTH = 30

        /**
         * Returns the display name for the given element type.
         *
         * @param elementType The element type.
         * @return The display name for the element type.
         */
        @JvmStatic
        fun getNameForElementType(
            elementType: com.intellij.psi.tree.IElementType
        ): String {
            return when (elementType) {
                PKTokenTypes.TEMPLATE_BLOCK_ELEMENT -> PKConstants.StructureDisplay.TEMPLATE_TAG
                PKTokenTypes.SCRIPT_BLOCK_ELEMENT -> PKConstants.StructureDisplay.SCRIPT_TAG
                PKTokenTypes.STYLE_BLOCK_ELEMENT -> PKConstants.StructureDisplay.STYLE_TAG
                PKTokenTypes.I18N_BLOCK_ELEMENT -> PKConstants.StructureDisplay.I18N_TAG
                PKTokenTypes.TEMPLATE_BODY_ELEMENT -> PKConstants.StructureDisplay.CONTENT
                PKTokenTypes.GO_SCRIPT_BODY_ELEMENT -> PKConstants.StructureDisplay.GO_CODE
                PKTokenTypes.JS_SCRIPT_BODY_ELEMENT -> PKConstants.StructureDisplay.JS_CODE
                PKTokenTypes.CSS_STYLE_BODY_ELEMENT -> PKConstants.StructureDisplay.CSS_RULES
                PKTokenTypes.I18N_BODY_ELEMENT -> PKConstants.StructureDisplay.TRANSLATIONS
                else -> ""
            }
        }

        /**
         * Returns the location string for the given element type.
         *
         * @param elementType The element type.
         * @return The location string, or null if not applicable.
         */
        @JvmStatic
        fun getLocationForElementType(
            elementType: com.intellij.psi.tree.IElementType
        ): String? {
            return when (elementType) {
                PKTokenTypes.GO_SCRIPT_BODY_ELEMENT -> PKConstants.LanguageDisplay.GO
                PKTokenTypes.JS_SCRIPT_BODY_ELEMENT -> PKConstants.LanguageDisplay.JAVASCRIPT
                PKTokenTypes.CSS_STYLE_BODY_ELEMENT -> PKConstants.LanguageDisplay.CSS
                PKTokenTypes.I18N_BODY_ELEMENT -> PKConstants.LanguageDisplay.JSON
                else -> null
            }
        }

        /**
         * Returns whether the element type represents a block element.
         *
         * @param elementType The element type.
         * @return true if the element type is a block element.
         */
        @JvmStatic
        fun isBlockElementType(
            elementType: com.intellij.psi.tree.IElementType
        ): Boolean {
            return elementType == PKTokenTypes.TEMPLATE_BLOCK_ELEMENT ||
                elementType == PKTokenTypes.SCRIPT_BLOCK_ELEMENT ||
                elementType == PKTokenTypes.STYLE_BLOCK_ELEMENT ||
                elementType == PKTokenTypes.I18N_BLOCK_ELEMENT
        }

        /**
         * Returns whether the element type represents a body element.
         *
         * @param elementType The element type.
         * @return true if the element type is a body element.
         */
        @JvmStatic
        fun isBodyElementType(
            elementType: com.intellij.psi.tree.IElementType
        ): Boolean {
            return elementType == PKTokenTypes.TEMPLATE_BODY_ELEMENT ||
                elementType == PKTokenTypes.GO_SCRIPT_BODY_ELEMENT ||
                elementType == PKTokenTypes.JS_SCRIPT_BODY_ELEMENT ||
                elementType == PKTokenTypes.CSS_STYLE_BODY_ELEMENT ||
                elementType == PKTokenTypes.I18N_BODY_ELEMENT
        }
    }

    /**
     * Returns the underlying PSI element.
     *
     * @return The PSI element this tree element wraps.
     */
    override fun getValue(): Any = element

    /**
     * Returns the sort key for alphabetical ordering.
     *
     * @return The element name used for sorting.
     */
    override fun getAlphaSortKey(): String = getElementName()

    /**
     * Returns the presentation data for display in the tree.
     *
     * @return The presentation containing name, location, and icon.
     */
    override fun getPresentation(): ItemPresentation {
        return PresentationData(
            getElementName(),
            getLocationString(),
            getIcon(),
            null
        )
    }

    /**
     * Returns the child elements to display in the tree.
     *
     * @return Array of child tree elements, or empty array if none.
     */
    override fun getChildren(): Array<TreeElement> {
        if (element is PsiFile) {
            return getFileChildren()
        }

        if (element !is PkPsiElementImpl) {
            return emptyArray()
        }

        return when (element.node.elementType) {
            PKTokenTypes.TEMPLATE_BLOCK_ELEMENT,
            PKTokenTypes.SCRIPT_BLOCK_ELEMENT,
            PKTokenTypes.STYLE_BLOCK_ELEMENT,
            PKTokenTypes.I18N_BLOCK_ELEMENT -> getBlockChildren()
            else -> emptyArray()
        }
    }

    /**
     * Navigates to this element in the editor.
     *
     * @param requestFocus Whether to request focus on the editor.
     */
    override fun navigate(requestFocus: Boolean) {
        if (element is NavigatablePsiElement) {
            element.navigate(requestFocus)
        }
    }

    /**
     * Returns whether this element can be navigated to.
     *
     * @return true if the element is navigatable.
     */
    override fun canNavigate(): Boolean {
        return element is NavigatablePsiElement && element.canNavigate()
    }

    /**
     * Returns whether this element can navigate to its source.
     *
     * @return true if the element can navigate to source.
     */
    override fun canNavigateToSource(): Boolean {
        return element is NavigatablePsiElement && element.canNavigateToSource()
    }

    /**
     * Returns the top-level block elements from the file.
     *
     * @return Array of tree elements for template, script, style, and i18n blocks.
     */
    private fun getFileChildren(): Array<TreeElement> {
        val children = mutableListOf<TreeElement>()
        var child = element.firstChild

        while (child != null) {
            if (child is PkPsiElementImpl) {
                val elementType = child.node.elementType
                if (isBlockElementType(elementType)) {
                    children.add(PKStructureViewElement(child))
                }
            }
            child = child.nextSibling
        }

        return children.toTypedArray()
    }

    /**
     * Returns the body elements within a block element.
     *
     * @return Array of tree elements for the block's content bodies.
     */
    private fun getBlockChildren(): Array<TreeElement> {
        val children = mutableListOf<TreeElement>()
        var child = element.firstChild

        while (child != null) {
            if (child is PkPsiElementImpl) {
                val elementType = child.node.elementType
                if (isBodyElementType(elementType)) {
                    children.add(PKStructureViewElement(child))
                }
            }
            child = child.nextSibling
        }

        return children.toTypedArray()
    }

    /**
     * Returns the display name for this element.
     *
     * @return The element name based on its type.
     */
    private fun getElementName(): String {
        if (element is PsiFile) {
            return element.name
        }

        if (element !is PkPsiElementImpl) {
            return element.text.take(MAX_TEXT_LENGTH)
        }

        return when (element.node.elementType) {
            PKTokenTypes.TEMPLATE_BLOCK_ELEMENT -> PKConstants.StructureDisplay.TEMPLATE_TAG
            PKTokenTypes.SCRIPT_BLOCK_ELEMENT -> getScriptBlockName()
            PKTokenTypes.STYLE_BLOCK_ELEMENT -> PKConstants.StructureDisplay.STYLE_TAG
            PKTokenTypes.I18N_BLOCK_ELEMENT -> PKConstants.StructureDisplay.I18N_TAG
            PKTokenTypes.TEMPLATE_BODY_ELEMENT -> PKConstants.StructureDisplay.CONTENT
            PKTokenTypes.GO_SCRIPT_BODY_ELEMENT -> PKConstants.StructureDisplay.GO_CODE
            PKTokenTypes.JS_SCRIPT_BODY_ELEMENT -> PKConstants.StructureDisplay.JS_CODE
            PKTokenTypes.CSS_STYLE_BODY_ELEMENT -> PKConstants.StructureDisplay.CSS_RULES
            PKTokenTypes.I18N_BODY_ELEMENT -> PKConstants.StructureDisplay.TRANSLATIONS
            else -> element.text.take(MAX_TEXT_LENGTH)
        }
    }

    /**
     * Returns the display name for a script block, including lang attribute.
     *
     * @return The script tag name with optional lang attribute.
     */
    private fun getScriptBlockName(): String {
        var child = element.firstChild
        while (child != null) {
            if (child.node?.elementType == PKTokenTypes.ATTR_VALUE) {
                val value = child.text.trim('"', '\'')
                if (PKConstants.ScriptLang.isJsLanguage(value)) {
                    return PKConstants.StructureDisplay.scriptTagWithLang(value)
                }
            }
            child = child.nextSibling
        }
        return PKConstants.StructureDisplay.SCRIPT_TAG
    }

    /**
     * Returns the location string for display in the structure view.
     *
     * @return The language name for body elements, or null.
     */
    private fun getLocationString(): String? {
        if (element !is PkPsiElementImpl) {
            return null
        }

        return when (element.node.elementType) {
            PKTokenTypes.GO_SCRIPT_BODY_ELEMENT -> PKConstants.LanguageDisplay.GO
            PKTokenTypes.JS_SCRIPT_BODY_ELEMENT -> PKConstants.LanguageDisplay.JAVASCRIPT
            PKTokenTypes.CSS_STYLE_BODY_ELEMENT -> PKConstants.LanguageDisplay.CSS
            PKTokenTypes.I18N_BODY_ELEMENT -> PKConstants.LanguageDisplay.JSON
            else -> null
        }
    }

    /**
     * Returns the icon for this element based on its type.
     *
     * @return The appropriate icon, or null for unknown types.
     */
    private fun getIcon(): Icon? {
        if (element is PsiFile) {
            return PKIcons.FILE
        }

        if (element !is PkPsiElementImpl) {
            return null
        }

        return when (element.node.elementType) {
            PKTokenTypes.TEMPLATE_BLOCK_ELEMENT -> AllIcons.FileTypes.Xml
            PKTokenTypes.SCRIPT_BLOCK_ELEMENT -> getScriptIcon()
            PKTokenTypes.STYLE_BLOCK_ELEMENT -> AllIcons.FileTypes.Css
            PKTokenTypes.I18N_BLOCK_ELEMENT -> AllIcons.FileTypes.Json
            PKTokenTypes.TEMPLATE_BODY_ELEMENT -> AllIcons.Nodes.Tag
            PKTokenTypes.GO_SCRIPT_BODY_ELEMENT -> AllIcons.Nodes.Function
            PKTokenTypes.JS_SCRIPT_BODY_ELEMENT -> AllIcons.Nodes.Function
            PKTokenTypes.CSS_STYLE_BODY_ELEMENT -> AllIcons.Nodes.Property
            PKTokenTypes.I18N_BODY_ELEMENT -> AllIcons.Nodes.Property
            else -> null
        }
    }

    /**
     * Returns the icon for a script block based on its language.
     *
     * @return JavaScript icon for JS/TS scripts, function icon for Go.
     */
    private fun getScriptIcon(): Icon {
        var child = element.firstChild
        while (child != null) {
            if (child.node?.elementType == PKTokenTypes.ATTR_VALUE) {
                val value = child.text.trim('"', '\'')
                if (PKConstants.ScriptLang.isJsLanguage(value)) {
                    return AllIcons.FileTypes.JavaScript
                }
            }
            child = child.nextSibling
        }
        return AllIcons.Nodes.Function
    }
}
