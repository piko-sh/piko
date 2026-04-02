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

import com.intellij.openapi.fileTypes.SyntaxHighlighter
import com.intellij.openapi.fileTypes.SyntaxHighlighterFactory
import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.VirtualFile

/**
 * Factory for creating PK syntax highlighter instances.
 *
 * Registered via plugin.xml to provide syntax highlighting for PK files.
 */
class PKSyntaxHighlighterFactory : SyntaxHighlighterFactory() {

    /**
     * Creates a syntax highlighter for the given file.
     *
     * @param project The current project, or null if no project context.
     * @param virtualFile The file to highlight, or null for default behaviour.
     * @return A new PKSyntaxHighlighter instance.
     */
    override fun getSyntaxHighlighter(project: Project?, virtualFile: VirtualFile?): SyntaxHighlighter {
        return PKSyntaxHighlighter()
    }
}
