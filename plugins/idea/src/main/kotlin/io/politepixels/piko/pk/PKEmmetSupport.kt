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

import com.intellij.codeInsight.template.emmet.generators.XmlZenCodingGeneratorImpl
import com.intellij.psi.PsiElement
import com.intellij.psi.tree.IElementType

/**
 * Enables Emmet/Zen Coding support for Piko template blocks.
 *
 * Emmet abbreviations expand only when the caret is inside
 * a template block's body content, not in script, style, or i18n blocks.
 * This ensures that Emmet doesn't interfere with Go code, CSS, or JSON editing.
 */
class PKEmmetSupport : XmlZenCodingGeneratorImpl() {

    companion object {
        /**
         * Element types where Emmet should NOT be enabled.
         * These are non-template blocks where code editing should not trigger Emmet expansion.
         */
        private val BLOCKED_ELEMENT_TYPES: Set<IElementType> = setOf(
            PKTokenTypes.SCRIPT_BLOCK_ELEMENT,
            PKTokenTypes.GO_SCRIPT_BODY_ELEMENT,
            PKTokenTypes.JS_SCRIPT_BODY_ELEMENT,
            PKTokenTypes.STYLE_BLOCK_ELEMENT,
            PKTokenTypes.CSS_STYLE_BODY_ELEMENT,
            PKTokenTypes.I18N_BLOCK_ELEMENT,
            PKTokenTypes.I18N_BODY_ELEMENT
        )
    }

    /**
     * Determines whether Emmet abbreviation expansion should be enabled at the given context.
     *
     * @param context The PSI element at the current caret position.
     * @param wrapping Whether this is for wrapping existing content.
     * @return true if Emmet should be active, false otherwise.
     */
    override fun isMyContext(context: PsiElement, wrapping: Boolean): Boolean {
        if (context.containingFile?.fileType !is PKFileType) {
            return false
        }

        var current: PsiElement? = context
        while (current != null) {
            val elementType = current.node?.elementType

            if (elementType == PKTokenTypes.TEMPLATE_BODY_ELEMENT) {
                return true
            }

            if (elementType in BLOCKED_ELEMENT_TYPES) {
                return false
            }

            current = current.parent
        }

        return false
    }
}
