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

import com.intellij.openapi.editor.Document
import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.VirtualFile
import com.intellij.psi.PsiElement
import com.intellij.psi.PsiManager
import com.intellij.psi.tree.IElementType
import com.intellij.psi.util.CachedValueProvider
import com.intellij.psi.util.CachedValuesManager
import com.intellij.xdebugger.breakpoints.XLineBreakpointType
import io.politepixels.piko.pk.PKFileType
import io.politepixels.piko.pk.PKPsiFile
import io.politepixels.piko.pk.PKTokenTypes

/**
 * Gutter shim that lets users toggle Delve breakpoints on `.pk` lines.
 *
 * GoLand's own Delve line breakpoint type only offers itself in Go files,
 * so this type claims the eligible `.pk` lines instead. A breakpoint of
 * this type never lives for long: [PKBreakpointBridge] replaces it with a
 * real Delve breakpoint on the same file and line, which Delve resolves
 * through the `//line` directives the Piko generator writes into
 * `generated.go`. If the line already holds a Delve breakpoint, the bridge
 * removes both, which gives the gutter its usual toggle behaviour.
 *
 * Registered in `piko-go.xml`, so it only exists when the Go plugin does.
 */
class PKLineBreakpointType : XLineBreakpointType<PKBreakpointProperties>(
    ID,
    "Piko Line Breakpoint"
) {

    /**
     * Determines whether a breakpoint can be placed at a given line.
     *
     * Accepts mappable Go script lines and template lines in `.pk` files,
     * and any line that already holds a Delve breakpoint so it can be
     * toggled off. Refuses `.pkc` files, JS script lines, style and i18n
     * blocks.
     *
     * @param file The virtual file being edited.
     * @param line The zero-based line number.
     * @param project The current project.
     * @return True if a breakpoint is allowed at this location.
     */
    override fun canPutAt(file: VirtualFile, line: Int, project: Project): Boolean {
        if (!isPkPage(file) || line < 0) return false
        if (PKDelveBreakpoints.findAtLine(project, file, line).isNotEmpty()) return true

        val psiFile = PsiManager.getInstance(project).findFile(file) as? PKPsiFile ?: return false
        val document = psiFile.viewProvider.document ?: return false
        return isBreakableLine(psiFile, document, line)
    }

    /**
     * Creates breakpoint properties for a new shim breakpoint.
     *
     * @param file The file containing the breakpoint.
     * @param line The line number (zero-based).
     * @return A new properties instance.
     */
    override fun createBreakpointProperties(file: VirtualFile, line: Int): PKBreakpointProperties {
        return PKBreakpointProperties()
    }

    companion object {
        /** Persisted identifier of the shim type, kept for migrating old breakpoints. */
        const val ID: String = "piko-line"

        /**
         * Template tokens that start a mapped template construct: an element,
         * a directive, or an interpolation.
         */
        private val MAPPED_TEMPLATE_TOKENS: Set<IElementType> = setOf(
            PKTokenTypes.HTML_TAG_OPEN,
            PKTokenTypes.DIRECTIVE_NAME,
            PKTokenTypes.DIRECTIVE_BIND,
            PKTokenTypes.DIRECTIVE_EVENT,
            PKTokenTypes.INTERPOLATION_OPEN,
        )

        /**
         * Reports whether a file is a `.pk` page, partial, email or PDF
         * template rather than a client-side `.pkc` component.
         *
         * @param file The file to check.
         * @return True for `.pk` files.
         */
        fun isPkPage(file: VirtualFile): Boolean =
            file.fileType == PKFileType && file.extension.equals("pk", ignoreCase = true)

        /**
         * Reports whether a document line holds code that the generator maps.
         *
         * @param psiFile The parsed `.pk` file.
         * @param document The document backing [psiFile].
         * @param line The zero-based line number.
         * @return True for mappable Go script lines and template lines.
         */
        fun isBreakableLine(psiFile: PKPsiFile, document: Document, line: Int): Boolean {
            if (line >= document.lineCount) return false

            val lineStartOffset = document.getLineStartOffset(line)
            val lineEndOffset = document.getLineEndOffset(line)

            var offset = lineStartOffset
            while (offset < lineEndOffset) {
                val element = psiFile.findElementAt(offset) ?: return false
                val elementType = element.node.elementType

                when {
                    elementType == PKTokenTypes.GO_SCRIPT_CONTENT ->
                        return isBreakableGoLine(element, document, line)
                    elementType == PKTokenTypes.JS_SCRIPT_CONTENT -> return false
                    elementType in MAPPED_TEMPLATE_TOKENS -> return true
                }

                val nextOffset = element.textRange.endOffset
                offset = if (nextOffset > offset) nextOffset else offset + 1
            }

            return false
        }

        /**
         * Checks a Go script line against the whole script body, since the
         * lexer splits Go content into several leaves at each `<`.
         */
        private fun isBreakableGoLine(leaf: PsiElement, document: Document, line: Int): Boolean {
            val script = leaf.parent?.takeIf { it.node.elementType == PKTokenTypes.GO_SCRIPT_BODY_ELEMENT } ?: leaf
            val relativeLine = line - document.getLineNumber(script.textRange.startOffset)
            val breakable = CachedValuesManager.getCachedValue(script) {
                CachedValueProvider.Result.create(PKGoScriptLines.breakableLines(script.text), script)
            }
            return relativeLine in breakable.indices && breakable[relativeLine]
        }
    }
}
