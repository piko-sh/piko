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

import com.intellij.extapi.psi.PsiFileBase
import com.intellij.openapi.fileTypes.FileType
import com.intellij.psi.FileViewProvider

/**
 * PSI file representation for Piko template files.
 *
 * Serves as the root of the PSI tree for `.pk` files, containing template,
 * script, style, and i18n blocks as child elements.
 *
 * @param viewProvider The file view provider supplying the file content.
 */
class PKPsiFile(viewProvider: FileViewProvider) : PsiFileBase(viewProvider, PKLanguage) {

    /**
     * Returns the file type associated with this PSI file.
     *
     * @return The PKFileType singleton.
     */
    override fun getFileType(): FileType = PKFileType

    /**
     * Returns a string representation for debugging purposes.
     *
     * @return The string "PK File".
     */
    override fun toString(): String = "PK File"
}
