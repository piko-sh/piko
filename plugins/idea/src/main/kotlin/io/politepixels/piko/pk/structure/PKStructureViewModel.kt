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

import com.intellij.ide.structureView.StructureViewModel
import com.intellij.ide.structureView.StructureViewModelBase
import com.intellij.ide.structureView.StructureViewTreeElement
import com.intellij.ide.util.treeView.smartTree.Sorter
import com.intellij.openapi.editor.Editor
import com.intellij.psi.PsiFile
import io.politepixels.piko.pk.PKTokenTypes
import io.politepixels.piko.pk.psi.impl.PkPsiElementImpl

/**
 * View model for the Structure View of PK files.
 *
 * Defines the root element and navigation behaviour for the structure tree,
 * showing template, script, style, and i18n blocks as top-level elements.
 */
class PKStructureViewModel(
    psiFile: PsiFile,
    editor: Editor?
) : StructureViewModelBase(psiFile, editor, PKStructureViewElement(psiFile)),
    StructureViewModel.ElementInfoProvider {

    /**
     * Returns the sorters available for the structure view.
     *
     * @return Array containing the alphabetical sorter.
     */
    override fun getSorters(): Array<Sorter> = arrayOf(Sorter.ALPHA_SORTER)

    /**
     * Determines if the given element should always be shown as a leaf.
     *
     * @param element The element to check.
     * @return true if the element has no meaningful children to display.
     */
    override fun isAlwaysLeaf(element: StructureViewTreeElement): Boolean {
        val value = element.value
        if (value !is PkPsiElementImpl) {
            return false
        }

        return when (value.node.elementType) {
            PKTokenTypes.GO_SCRIPT_BODY_ELEMENT,
            PKTokenTypes.JS_SCRIPT_BODY_ELEMENT,
            PKTokenTypes.CSS_STYLE_BODY_ELEMENT,
            PKTokenTypes.I18N_BODY_ELEMENT -> true
            else -> false
        }
    }

    /**
     * Determines if the given element should always be shown in the structure.
     *
     * @param element The element to check.
     * @return true if the element is a significant structural component.
     */
    override fun isAlwaysShowsPlus(element: StructureViewTreeElement): Boolean {
        val value = element.value
        if (value !is PkPsiElementImpl) {
            return value is PsiFile
        }

        return when (value.node.elementType) {
            PKTokenTypes.TEMPLATE_BLOCK_ELEMENT,
            PKTokenTypes.SCRIPT_BLOCK_ELEMENT,
            PKTokenTypes.STYLE_BLOCK_ELEMENT,
            PKTokenTypes.I18N_BLOCK_ELEMENT -> true
            else -> false
        }
    }
}
