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

import com.intellij.ide.structureView.StructureViewBuilder
import com.intellij.ide.structureView.StructureViewModel
import com.intellij.ide.structureView.TreeBasedStructureViewBuilder
import com.intellij.lang.PsiStructureViewFactory
import com.intellij.openapi.editor.Editor
import com.intellij.psi.PsiFile
import io.politepixels.piko.pk.PKPsiFile

/**
 * Factory for creating Structure View instances for PK files.
 *
 * The Structure View provides a hierarchical outline of the file's content,
 * showing top-level blocks (template, script, style, i18n) and their structure.
 */
class PKStructureViewFactory : PsiStructureViewFactory {

    /**
     * Creates a structure view builder for the given PK file.
     *
     * @param psiFile The PK file to create a structure view for.
     * @return A structure view builder, or null if the file is not a PK file.
     */
    override fun getStructureViewBuilder(psiFile: PsiFile): StructureViewBuilder? {
        if (psiFile !is PKPsiFile) {
            return null
        }

        return object : TreeBasedStructureViewBuilder() {
            override fun createStructureViewModel(editor: Editor?): StructureViewModel {
                return PKStructureViewModel(psiFile, editor)
            }
        }
    }
}
