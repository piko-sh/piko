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

import com.intellij.codeInsight.editorActions.TypedHandlerDelegate
import com.intellij.openapi.editor.Editor
import com.intellij.openapi.project.Project
import com.intellij.psi.PsiFile

/**
 * Handles auto-completion of paired constructs in PK files.
 */
class PKTypedHandler : TypedHandlerDelegate() {

    /**
     * Auto-closes interpolation braces when typing `{{`.
     *
     * @param c The character that was typed.
     * @param project The current project.
     * @param editor The active editor.
     * @param file The file being edited.
     * @return STOP if auto-completion was performed, CONTINUE otherwise.
     */
    override fun charTyped(c: Char, project: Project, editor: Editor, file: PsiFile): Result {
        if (file.language != PKLanguage && file.fileType != PKFileType) {
            return Result.CONTINUE
        }

        val document = editor.document
        val offset = editor.caretModel.offset

        when (c) {
            '{' -> {
                if (offset >= 2) {
                    val prevChar = document.charsSequence.getOrNull(offset - 2)
                    if (prevChar == '{') {
                        document.insertString(offset, "}}")
                        return Result.STOP
                    }
                }
            }
        }

        return Result.CONTINUE
    }
}
