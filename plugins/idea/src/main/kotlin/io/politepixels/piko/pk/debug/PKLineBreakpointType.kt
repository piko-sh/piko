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


package io.politepixels.piko.pk.debug

import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.VirtualFile
import com.intellij.psi.PsiManager
import com.intellij.xdebugger.breakpoints.XLineBreakpointType
import io.politepixels.piko.pk.PKFileType
import io.politepixels.piko.pk.PKPsiFile
import io.politepixels.piko.pk.PKTokenTypes

/**
 * Line breakpoint type for Piko template files.
 *
 * Allows users to set breakpoints in the gutter of `.pk` files.
 * Breakpoints are only permitted within `<template>` and `<script>` blocks,
 * since those are the blocks that produce debuggable Go code via `//line`
 * directives in the generated output.
 *
 * When a Go debug session is active, Delve resolves these breakpoints
 * through DWARF line tables produced by the `//line` directives that the
 * Piko code generator emits.
 */
class PKLineBreakpointType : XLineBreakpointType<PKBreakpointProperties>(
    "piko-line",
    "Piko Line Breakpoint"
) {

    /**
     * Determines whether a breakpoint can be placed at a given line.
     *
     * Returns true only for `.pk` files (not `.pkc`) when the line falls
     * within a `<template>` or `<script>` block. Lines in `<style>` and
     * `<i18n>` blocks do not produce debuggable code.
     *
     * @param file The virtual file being edited.
     * @param line The zero-based line number.
     * @param project The current project.
     * @return True if a breakpoint is allowed at this location.
     */
    override fun canPutAt(file: VirtualFile, line: Int, project: Project): Boolean {
        if (file.fileType != PKFileType) return false
        if (file.extension == "pkc") return false

        val psiFile = PsiManager.getInstance(project).findFile(file) as? PKPsiFile ?: return false
        val document = psiFile.viewProvider.document ?: return false

        if (line < 0 || line >= document.lineCount) return false

        val lineStartOffset = document.getLineStartOffset(line)
        val lineEndOffset = document.getLineEndOffset(line)

        var offset = lineStartOffset
        while (offset <= lineEndOffset) {
            val element = psiFile.findElementAt(offset) ?: break
            val elementType = element.node.elementType

            if (elementType == PKTokenTypes.GO_SCRIPT_CONTENT ||
                elementType == PKTokenTypes.JS_SCRIPT_CONTENT ||
                elementType in DEBUGGABLE_TEMPLATE_TOKENS
            ) {
                return true
            }

            val nextOffset = element.textRange.endOffset
            offset = if (nextOffset > offset) nextOffset else offset + 1
        }

        return false
    }

    /**
     * Creates breakpoint properties for a new PK breakpoint.
     *
     * @param file The file containing the breakpoint.
     * @param line The line number (zero-based).
     * @return A new properties instance.
     */
    override fun createBreakpointProperties(file: VirtualFile, line: Int): PKBreakpointProperties {
        return PKBreakpointProperties()
    }

    companion object {
        /**
         * PSI element types that represent debuggable template content.
         * Breakpoints are allowed on lines containing any of these tokens.
         */
        private val DEBUGGABLE_TEMPLATE_TOKENS: Set<com.intellij.psi.tree.IElementType> = setOf(
            PKTokenTypes.HTML_TAG_OPEN,
            PKTokenTypes.HTML_TAG_CLOSE,
            PKTokenTypes.HTML_TAG_SELF_CLOSE,
            PKTokenTypes.HTML_END_TAG_OPEN,
            PKTokenTypes.HTML_TAG_NAME,
            PKTokenTypes.PIKO_TAG_NAME,
            PKTokenTypes.HTML_ATTR_NAME,
            PKTokenTypes.HTML_ATTR_VALUE,
            PKTokenTypes.DIRECTIVE_NAME,
            PKTokenTypes.DIRECTIVE_BIND,
            PKTokenTypes.DIRECTIVE_EVENT,
            PKTokenTypes.INTERPOLATION_OPEN,
            PKTokenTypes.INTERPOLATION_CLOSE,
            PKTokenTypes.TEXT_CONTENT,
            PKTokenTypes.EXPR_CONTEXT_VAR,
            PKTokenTypes.EXPR_IDENTIFIER,
            PKTokenTypes.EXPR_FUNCTION_NAME,
            PKTokenTypes.EXPR_BUILTIN,
            PKTokenTypes.EXPR_BOOLEAN,
            PKTokenTypes.EXPR_NUMBER,
            PKTokenTypes.EXPR_STRING,
            PKTokenTypes.EXPR_OP_DOT,
            PKTokenTypes.EXPR_OP_COMPARISON,
            PKTokenTypes.EXPR_OP_LOGICAL,
            PKTokenTypes.EXPR_OP_ARITHMETIC,
        )
    }
}
