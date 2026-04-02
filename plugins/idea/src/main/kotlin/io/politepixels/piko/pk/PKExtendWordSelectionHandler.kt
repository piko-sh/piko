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

import com.intellij.codeInsight.editorActions.ExtendWordSelectionHandlerBase
import com.intellij.openapi.editor.Editor
import com.intellij.openapi.util.TextRange
import com.intellij.psi.PsiElement
import com.intellij.psi.PsiFile
import io.politepixels.piko.pk.psi.impl.PkPsiElementImpl

/**
 * Extends word selection (Ctrl+W) for PK template files.
 *
 * Provides intelligent selection expansion through structural units:
 * interpolation expressions, attribute values, HTML tags, and blocks.
 */
class PKExtendWordSelectionHandler : ExtendWordSelectionHandlerBase() {

    companion object {
        /**
         * Returns whether the given element type represents interpolation content.
         *
         * @param elementType The element type to check.
         * @return true if the element type is interpolation content.
         */
        @JvmStatic
        fun isInterpolationContentType(
            elementType: com.intellij.psi.tree.IElementType
        ): Boolean {
            if (elementType == PKTokenTypes.INTERPOLATION_OPEN ||
                elementType == PKTokenTypes.INTERPOLATION_CLOSE
            ) {
                return true
            }

            return when (elementType) {
                PKTokenTypes.EXPR_BOOLEAN,
                PKTokenTypes.EXPR_NUMBER,
                PKTokenTypes.EXPR_STRING,
                PKTokenTypes.EXPR_STRING_QUOTE,
                PKTokenTypes.EXPR_IDENTIFIER,
                PKTokenTypes.EXPR_FUNCTION_NAME,
                PKTokenTypes.EXPR_CONTEXT_VAR,
                PKTokenTypes.EXPR_BUILTIN,
                PKTokenTypes.EXPR_OP_DOT,
                PKTokenTypes.EXPR_OP_COMPARISON,
                PKTokenTypes.EXPR_OP_LOGICAL,
                PKTokenTypes.EXPR_OP_ARITHMETIC,
                PKTokenTypes.EXPR_PAREN_OPEN,
                PKTokenTypes.EXPR_PAREN_CLOSE,
                PKTokenTypes.EXPR_BRACKET_OPEN,
                PKTokenTypes.EXPR_BRACKET_CLOSE,
                PKTokenTypes.EXPR_BRACE_OPEN,
                PKTokenTypes.EXPR_BRACE_CLOSE,
                PKTokenTypes.EXPR_COMMA,
                PKTokenTypes.EXPR_COLON -> true
                else -> false
            }
        }
    }

    /**
     * Determines if this handler can process the given element.
     *
     * @param e The PSI element at the cursor position.
     * @return true if the element is within a PK file.
     */
    override fun canSelect(e: PsiElement): Boolean {
        val file = e.containingFile ?: return false
        return file is PKPsiFile
    }

    /**
     * Returns extended selection ranges for the given element.
     *
     * @param e The PSI element at the cursor position.
     * @param editorText The full text content of the editor.
     * @param cursorOffset The current cursor position.
     * @param editor The active editor.
     * @return List of extended text ranges, or null if none apply.
     */
    override fun select(
        e: PsiElement,
        editorText: CharSequence,
        cursorOffset: Int,
        editor: Editor
    ): List<TextRange>? {
        val ranges = mutableListOf<TextRange>()

        addInterpolationRange(e, ranges)
        addAttributeRange(e, ranges)
        addTagRange(e, ranges)
        addBlockRange(e, ranges)

        return ranges.ifEmpty { null }
    }

    /**
     * Adds interpolation selection ranges for the given element.
     *
     * @param element The PSI element to check for interpolation context.
     * @param ranges The list to add discovered ranges to.
     */
    private fun addInterpolationRange(element: PsiElement, ranges: MutableList<TextRange>) {
        val interpolationElement = findInterpolationElement(element) ?: return
        val interpolationRange = findInterpolationBounds(interpolationElement) ?: return

        ranges.add(interpolationRange)
        addInnerInterpolationRange(interpolationRange, element, ranges)
    }

    /**
     * Finds the nearest ancestor element that is interpolation content.
     *
     * @param element The starting PSI element.
     * @return The interpolation content element, or null if not found.
     */
    private fun findInterpolationElement(element: PsiElement): PsiElement? {
        var current: PsiElement? = element
        while (current != null && current !is PsiFile) {
            if (isInterpolationContent(current)) {
                return current
            }
            current = current.parent
        }
        return null
    }

    /**
     * Adds the inner content range of an interpolation (excluding delimiters).
     *
     * @param interpolationRange The full interpolation range including delimiters.
     * @param element The original PSI element.
     * @param ranges The list to add the inner range to.
     */
    private fun addInnerInterpolationRange(
        interpolationRange: TextRange,
        element: PsiElement,
        ranges: MutableList<TextRange>
    ) {
        val innerRange = TextRange(
            interpolationRange.startOffset + 2,
            interpolationRange.endOffset - 2
        )
        if (innerRange.length > 0 && innerRange != element.textRange) {
            ranges.add(innerRange)
        }
    }

    /**
     * Adds attribute selection ranges for attribute value elements.
     *
     * @param element The PSI element to check for attribute context.
     * @param ranges The list to add discovered ranges to.
     */
    private fun addAttributeRange(element: PsiElement, ranges: MutableList<TextRange>) {
        val elementType = element.node?.elementType ?: return

        if (elementType == PKTokenTypes.HTML_ATTR_VALUE ||
            elementType == PKTokenTypes.ATTR_VALUE
        ) {
            val fullAttrRange = findFullAttributeRange(element)
            if (fullAttrRange != null) {
                ranges.add(fullAttrRange)
            }

            val valueWithoutQuotes = getValueWithoutQuotes(element)
            if (valueWithoutQuotes != null && valueWithoutQuotes != element.textRange) {
                ranges.add(valueWithoutQuotes)
            }
        }
    }

    /**
     * Adds HTML tag selection ranges by traversing ancestors.
     *
     * @param element The PSI element to check for tag context.
     * @param ranges The list to add discovered ranges to.
     */
    private fun addTagRange(element: PsiElement, ranges: MutableList<TextRange>) {
        var current: PsiElement? = element

        while (current != null && current !is PsiFile) {
            val elementType = current.node?.elementType

            if (elementType == PKTokenTypes.HTML_TAG_OPEN ||
                elementType == PKTokenTypes.HTML_TAG_NAME ||
                elementType == PKTokenTypes.PIKO_TAG_NAME
            ) {
                val tagRange = findTagBounds(current)
                if (tagRange != null) {
                    ranges.add(tagRange)
                }
            }

            current = current.parent
        }
    }

    /**
     * Adds block selection ranges by traversing ancestors.
     *
     * @param element The PSI element to check for block context.
     * @param ranges The list to add discovered ranges to.
     */
    private fun addBlockRange(element: PsiElement, ranges: MutableList<TextRange>) {
        var current: PsiElement? = element

        while (current != null && current !is PsiFile) {
            if (current is PkPsiElementImpl) {
                addBlockElementRanges(current, ranges)
            }
            current = current.parent
        }
    }

    /**
     * Adds ranges for block or body elements including parent blocks.
     *
     * @param element The PK PSI element to process.
     * @param ranges The list to add discovered ranges to.
     */
    private fun addBlockElementRanges(element: PkPsiElementImpl, ranges: MutableList<TextRange>) {
        val elementType = element.node.elementType

        if (isBodyElement(elementType)) {
            ranges.add(element.textRange)
            val parent = element.parent
            if (parent is PkPsiElementImpl) {
                ranges.add(parent.textRange)
            }
        } else if (isBlockElement(elementType)) {
            ranges.add(element.textRange)
        }
    }

    /**
     * Checks if the element type represents a body element.
     *
     * @param elementType The element type to check.
     * @return true if it is a body element type.
     */
    private fun isBodyElement(elementType: com.intellij.psi.tree.IElementType): Boolean {
        return elementType == PKTokenTypes.TEMPLATE_BODY_ELEMENT ||
            elementType == PKTokenTypes.GO_SCRIPT_BODY_ELEMENT ||
            elementType == PKTokenTypes.JS_SCRIPT_BODY_ELEMENT ||
            elementType == PKTokenTypes.CSS_STYLE_BODY_ELEMENT ||
            elementType == PKTokenTypes.I18N_BODY_ELEMENT
    }

    /**
     * Checks if the element type represents a block element.
     *
     * @param elementType The element type to check.
     * @return true if it is a block element type.
     */
    private fun isBlockElement(elementType: com.intellij.psi.tree.IElementType): Boolean {
        return elementType == PKTokenTypes.TEMPLATE_BLOCK_ELEMENT ||
            elementType == PKTokenTypes.SCRIPT_BLOCK_ELEMENT ||
            elementType == PKTokenTypes.STYLE_BLOCK_ELEMENT ||
            elementType == PKTokenTypes.I18N_BLOCK_ELEMENT
    }

    /**
     * Checks if an element is interpolation content.
     *
     * @param element The PSI element to check.
     * @return true if the element is interpolation content.
     */
    private fun isInterpolationContent(element: PsiElement): Boolean {
        val elementType = element.node?.elementType ?: return false
        return isInterpolationContentType(elementType)
    }

    /**
     * Finds the full text range of an interpolation expression.
     *
     * @param element The element within the interpolation.
     * @return The text range from open to close delimiter, or null.
     */
    private fun findInterpolationBounds(element: PsiElement): TextRange? {
        val startOffset = findInterpolationStart(element) ?: return null
        val endOffset = findInterpolationEnd(element) ?: return null
        return TextRange(startOffset, endOffset)
    }

    /**
     * Finds the start offset of the interpolation open delimiter.
     *
     * @param element The element within the interpolation.
     * @return The start offset, or null if not found.
     */
    private fun findInterpolationStart(element: PsiElement): Int? {
        val siblingStart = findInterpolationOpenInSiblings(element)
        if (siblingStart != null) return siblingStart
        return findInterpolationOpenInParents(element)
    }

    /**
     * Finds the end offset of the interpolation close delimiter.
     *
     * @param element The element within the interpolation.
     * @return The end offset, or null if not found.
     */
    private fun findInterpolationEnd(element: PsiElement): Int? {
        val siblingEnd = findInterpolationCloseInSiblings(element)
        if (siblingEnd != null) return siblingEnd
        return findInterpolationCloseInParents(element)
    }

    /**
     * Searches previous siblings for an interpolation open token.
     *
     * @param element The starting element.
     * @return The start offset of the open token, or null.
     */
    private fun findInterpolationOpenInSiblings(element: PsiElement): Int? {
        var sibling = element.prevSibling
        while (sibling != null) {
            if (sibling.node?.elementType == PKTokenTypes.INTERPOLATION_OPEN) {
                return sibling.textRange.startOffset
            }
            sibling = sibling.prevSibling
        }
        return null
    }

    /**
     * Searches parent siblings for an interpolation open token.
     *
     * @param element The starting element.
     * @return The start offset of the open token, or null.
     */
    private fun findInterpolationOpenInParents(element: PsiElement): Int? {
        var current = element
        while (current.parent != null && current.parent !is PsiFile) {
            val sibling = current.parent.prevSibling
            if (sibling?.node?.elementType == PKTokenTypes.INTERPOLATION_OPEN) {
                return sibling.textRange.startOffset
            }
            current = current.parent
        }
        return null
    }

    /**
     * Searches next siblings for an interpolation close token.
     *
     * @param element The starting element.
     * @return The end offset of the close token, or null.
     */
    private fun findInterpolationCloseInSiblings(element: PsiElement): Int? {
        var sibling = element.nextSibling
        while (sibling != null) {
            if (sibling.node?.elementType == PKTokenTypes.INTERPOLATION_CLOSE) {
                return sibling.textRange.endOffset
            }
            sibling = sibling.nextSibling
        }
        return null
    }

    /**
     * Searches parent siblings for an interpolation close token.
     *
     * @param element The starting element.
     * @return The end offset of the close token, or null.
     */
    private fun findInterpolationCloseInParents(element: PsiElement): Int? {
        var current = element
        while (current.parent != null && current.parent !is PsiFile) {
            val sibling = current.parent.nextSibling
            if (sibling?.node?.elementType == PKTokenTypes.INTERPOLATION_CLOSE) {
                return sibling.textRange.endOffset
            }
            current = current.parent
        }
        return null
    }

    /**
     * Finds the full range of an attribute from name to value.
     *
     * @param valueElement The attribute value element.
     * @return The full attribute range, or null if name not found.
     */
    private fun findFullAttributeRange(valueElement: PsiElement): TextRange? {
        var sibling = valueElement.prevSibling
        var attrNameStart: Int? = null

        while (sibling != null) {
            val siblingType = sibling.node?.elementType
            when (siblingType) {
                PKTokenTypes.HTML_ATTR_NAME,
                PKTokenTypes.ATTR_NAME,
                PKTokenTypes.DIRECTIVE_NAME,
                PKTokenTypes.DIRECTIVE_BIND,
                PKTokenTypes.DIRECTIVE_EVENT -> {
                    attrNameStart = sibling.textRange.startOffset
                    break
                }
            }
            sibling = sibling.prevSibling
        }

        return if (attrNameStart != null) {
            TextRange(attrNameStart, valueElement.textRange.endOffset)
        } else {
            null
        }
    }

    /**
     * Returns the text range excluding surrounding quotes.
     *
     * @param element The element containing a potentially quoted value.
     * @return The inner range without quotes, or null if not quoted.
     */
    private fun getValueWithoutQuotes(element: PsiElement): TextRange? {
        val text = element.text
        val range = element.textRange

        if (!isQuotedString(text)) {
            return null
        }

        return TextRange(range.startOffset + 1, range.endOffset - 1)
    }

    /**
     * Checks if a string is surrounded by matching quotes.
     *
     * @param text The text to check.
     * @return true if surrounded by single or double quotes.
     */
    private fun isQuotedString(text: String): Boolean {
        if (text.length < 2) return false
        val isDoubleQuoted = text.startsWith("\"") && text.endsWith("\"")
        val isSingleQuoted = text.startsWith("'") && text.endsWith("'")
        return isDoubleQuoted || isSingleQuoted
    }

    /**
     * Finds the bounds of an HTML tag from open to close.
     *
     * @param element An element within the tag.
     * @return The full tag range, or null if bounds not found.
     */
    private fun findTagBounds(element: PsiElement): TextRange? {
        val tagOpenStart = findTagOpenStart(element) ?: return null
        val tagInfo = scanForTagInfo(element)

        if (tagInfo.selfCloseEnd != null) {
            return TextRange(tagOpenStart, tagInfo.selfCloseEnd)
        }

        if (tagInfo.tagName != null) {
            val closeTagEnd = findMatchingCloseTag(element, tagInfo.tagName)
            if (closeTagEnd != null) {
                return TextRange(tagOpenStart, closeTagEnd)
            }
        }

        return null
    }

    /**
     * Finds the start offset of the tag open token.
     *
     * @param element An element within the tag.
     * @return The start offset, or null if not found.
     */
    private fun findTagOpenStart(element: PsiElement): Int? {
        var current = element
        while (current.prevSibling != null) {
            current = current.prevSibling
            if (current.node?.elementType == PKTokenTypes.HTML_TAG_OPEN) {
                return current.textRange.startOffset
            }
        }

        if (element.node?.elementType == PKTokenTypes.HTML_TAG_OPEN) {
            return element.textRange.startOffset
        }

        return null
    }

    /**
     * Holds information about a tag including name and self-close position.
     *
     * @property tagName The name of the tag, or null if not found.
     * @property selfCloseEnd The end offset if self-closing, or null.
     */
    private data class TagInfo(val tagName: String?, val selfCloseEnd: Int?)

    /**
     * Scans forward from an element to gather tag information.
     *
     * @param element The starting element within the tag.
     * @return Tag information including name and self-close status.
     */
    private fun scanForTagInfo(element: PsiElement): TagInfo {
        var tagName: String? = null
        var selfCloseEnd: Int? = null

        var current = element
        while (current.nextSibling != null) {
            current = current.nextSibling
            val elementType = current.node?.elementType

            if (isTagNameElement(elementType)) {
                tagName = current.text
            }

            if (elementType == PKTokenTypes.HTML_TAG_SELF_CLOSE) {
                selfCloseEnd = current.textRange.endOffset
                break
            }

            if (elementType == PKTokenTypes.HTML_TAG_CLOSE) {
                break
            }
        }

        return TagInfo(tagName, selfCloseEnd)
    }

    /**
     * Checks if an element type represents a tag name.
     *
     * @param elementType The element type to check.
     * @return true if it is an HTML or Piko tag name type.
     */
    private fun isTagNameElement(elementType: com.intellij.psi.tree.IElementType?): Boolean {
        return elementType == PKTokenTypes.HTML_TAG_NAME ||
            elementType == PKTokenTypes.PIKO_TAG_NAME
    }

    /**
     * Finds the matching close tag for a given tag name.
     *
     * @param startElement The element to start searching from.
     * @param tagName The tag name to match.
     * @return The end offset of the close tag, or null if not found.
     */
    private fun findMatchingCloseTag(startElement: PsiElement, tagName: String): Int? {
        var depth = 1
        var current: PsiElement? = startElement.nextSibling

        while (current != null) {
            depth = updateTagDepth(current, tagName, depth)
            if (depth == 0) {
                return findCloseTagEnd(current)
            }
            current = current.nextSibling
        }

        return null
    }

    /**
     * Updates the nesting depth based on tag open/close elements.
     *
     * @param element The current element being examined.
     * @param tagName The tag name being matched.
     * @param currentDepth The current nesting depth.
     * @return The updated depth after processing this element.
     */
    private fun updateTagDepth(element: PsiElement, tagName: String, currentDepth: Int): Int {
        val elementType = element.node?.elementType ?: return currentDepth

        return when (elementType) {
            PKTokenTypes.HTML_TAG_OPEN -> {
                val name = findTagNameAfter(element)
                if (name == tagName && !isSelfClosing(element)) currentDepth + 1 else currentDepth
            }
            PKTokenTypes.HTML_END_TAG_OPEN -> {
                val name = findTagNameAfter(element)
                if (name == tagName) currentDepth - 1 else currentDepth
            }
            else -> currentDepth
        }
    }

    /**
     * Finds the tag name following a tag open element.
     *
     * @param element The tag open element.
     * @return The tag name text, or null if not found.
     */
    private fun findTagNameAfter(element: PsiElement): String? = PKTreeUtils.findTagNameFromElement(element)

    /**
     * Checks if a tag is self-closing.
     *
     * @param tagOpenElement The tag open element to check.
     * @return true if the tag ends with a self-close token.
     */
    private fun isSelfClosing(tagOpenElement: PsiElement): Boolean = PKTreeUtils.isSelfClosingElement(tagOpenElement)

    /**
     * Finds the end offset of a close tag.
     *
     * @param endTagOpenElement The end tag open element.
     * @return The end offset of the close token.
     */
    private fun findCloseTagEnd(endTagOpenElement: PsiElement): Int? {
        val closeTag = PKTreeUtils.findNextSiblingOfType(endTagOpenElement, PKTokenTypes.HTML_TAG_CLOSE)
        return closeTag?.textRange?.endOffset ?: endTagOpenElement.textRange.endOffset
    }
}
