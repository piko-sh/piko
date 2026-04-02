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

import com.intellij.codeInsight.generation.CommenterDataHolder
import com.intellij.codeInsight.generation.SelfManagingCommenter
import com.intellij.codeInsight.generation.SelfManagingCommenterUtil
import com.intellij.lang.Commenter
import com.intellij.openapi.editor.Document
import com.intellij.openapi.util.TextRange
import com.intellij.psi.PsiFile
import com.intellij.util.text.CharArrayUtil

/**
 * Context-aware commenter for PK template files.
 *
 * Intelligently chooses the appropriate comment style based on cursor position:
 * expression comments for interpolations and expression directives,
 * HTML comments for regular template content.
 */
class PKContextAwareCommenter : SelfManagingCommenter<PKCommentContext>, Commenter {

    companion object {
        /** HTML comment opening delimiter. */
        internal const val HTML_COMMENT_PREFIX = "<!--"

        /** HTML comment closing delimiter. */
        internal const val HTML_COMMENT_SUFFIX = "-->"

        /** Expression comment opening delimiter (C-style). */
        internal const val EXPR_COMMENT_PREFIX = "/*"

        /** Expression comment closing delimiter (C-style). */
        internal const val EXPR_COMMENT_SUFFIX = "*/"

        /** Directive names that expect expression values in their attributes. */
        internal val EXPRESSION_DIRECTIVES = setOf(
            "p-if", "p-else-if", "p-else", "p-for", "p-show",
            "p-text", "p-html", "p-model", "p-class", "p-style",
            "p-bind", "p-on", "p-event"
        )

        /** Directive names that do not expect expression values. */
        @Suppress("unused")
        internal val NON_EXPRESSION_DIRECTIVES = setOf(
            "p-key", "p-slot", "p-ref", "p-context", "p-scaffold"
        )

        /**
         * Determines whether the given element type represents an expression token.
         *
         * @param elementType The element type to check.
         * @return true if the element type is an expression token.
         */
        @JvmStatic
        fun isExpressionToken(elementType: com.intellij.psi.tree.IElementType): Boolean {
            return when (elementType) {
                PKTokenTypes.EXPR_BOOLEAN,
                PKTokenTypes.EXPR_NUMBER,
                PKTokenTypes.EXPR_STRING,
                PKTokenTypes.EXPR_STRING_QUOTE,
                PKTokenTypes.EXPR_ESCAPE,
                PKTokenTypes.EXPR_OP_COMPARISON,
                PKTokenTypes.EXPR_OP_LOGICAL,
                PKTokenTypes.EXPR_OP_ARITHMETIC,
                PKTokenTypes.EXPR_OP_DOT,
                PKTokenTypes.EXPR_PAREN_OPEN,
                PKTokenTypes.EXPR_PAREN_CLOSE,
                PKTokenTypes.EXPR_BRACKET_OPEN,
                PKTokenTypes.EXPR_BRACKET_CLOSE,
                PKTokenTypes.EXPR_BRACE_OPEN,
                PKTokenTypes.EXPR_BRACE_CLOSE,
                PKTokenTypes.EXPR_COMMA,
                PKTokenTypes.EXPR_COLON,
                PKTokenTypes.EXPR_BUILTIN,
                PKTokenTypes.EXPR_CONTEXT_VAR,
                PKTokenTypes.EXPR_FUNCTION_NAME,
                PKTokenTypes.EXPR_IDENTIFIER -> true
                else -> false
            }
        }
    }

    /**
     * Returns the line comment prefix.
     *
     * @return null as PK files do not support line comments.
     */
    override fun getLineCommentPrefix(): String? = null

    /**
     * Returns the default block comment prefix.
     *
     * @return The HTML comment prefix.
     */
    override fun getBlockCommentPrefix(): String = HTML_COMMENT_PREFIX

    /**
     * Returns the default block comment suffix.
     *
     * @return The HTML comment suffix.
     */
    override fun getBlockCommentSuffix(): String = HTML_COMMENT_SUFFIX

    /**
     * Returns the prefix for nested block comments.
     *
     * @return null as nested comments are not supported.
     */
    override fun getCommentedBlockCommentPrefix(): String? = null

    /**
     * Returns the suffix for nested block comments.
     *
     * @return null as nested comments are not supported.
     */
    override fun getCommentedBlockCommentSuffix(): String? = null

    /**
     * Creates state for line commenting operations.
     *
     * Determines the comment context by checking the first non-whitespace character
     * on the start line to decide whether to use HTML or expression comments.
     *
     * @param startLine The starting line number.
     * @param endLine The ending line number.
     * @param document The document being edited.
     * @param file The PSI file.
     * @return The comment context for the lines.
     */
    override fun createLineCommentingState(
        startLine: Int,
        endLine: Int,
        document: Document,
        file: PsiFile
    ): PKCommentContext {
        val lineStartOffset = document.getLineStartOffset(startLine)
        val lineEndOffset = document.getLineEndOffset(startLine)
        val lineText = document.charsSequence.subSequence(lineStartOffset, lineEndOffset)

        val firstNonWhitespace = CharArrayUtil.shiftForward(lineText, 0, " \t")
        val checkOffset = if (firstNonWhitespace < lineText.length) {
            lineStartOffset + firstNonWhitespace
        } else {
            lineStartOffset
        }

        val isExpression = isExpressionContext(file, checkOffset, checkOffset)
        return PKCommentContext(isExpression)
    }

    /**
     * Creates state for block commenting operations.
     *
     * @param selectionStart The start offset of the selection.
     * @param selectionEnd The end offset of the selection.
     * @param document The document being edited.
     * @param file The PSI file.
     * @return The comment context indicating whether expression comments should be used.
     */
    override fun createBlockCommentingState(
        selectionStart: Int,
        selectionEnd: Int,
        document: Document,
        file: PsiFile
    ): PKCommentContext {
        val isExpression = isExpressionContext(file, selectionStart, selectionEnd)
        return PKCommentContext(isExpression)
    }

    /**
     * Returns the block comment prefix for the given context.
     *
     * @param selectionStart The start offset of the selection.
     * @param document The document being edited.
     * @param data The comment context.
     * @return Expression comment prefix if in expression context, HTML comment prefix otherwise.
     */
    override fun getBlockCommentPrefix(
        selectionStart: Int,
        document: Document,
        data: PKCommentContext
    ): String = if (data.isExpressionContext) EXPR_COMMENT_PREFIX else HTML_COMMENT_PREFIX

    /**
     * Returns the block comment suffix for the given context.
     *
     * @param selectionEnd The end offset of the selection.
     * @param document The document being edited.
     * @param data The comment context.
     * @return Expression comment suffix if in expression context, HTML comment suffix otherwise.
     */
    override fun getBlockCommentSuffix(
        selectionEnd: Int,
        document: Document,
        data: PKCommentContext
    ): String = if (data.isExpressionContext) EXPR_COMMENT_SUFFIX else HTML_COMMENT_SUFFIX

    /**
     * Inserts a block comment around the specified range.
     *
     * @param startOffset The start offset of the range to comment.
     * @param endOffset The end offset of the range to comment.
     * @param document The document being edited.
     * @param data The comment context.
     * @return The text range of the inserted comment.
     */
    override fun insertBlockComment(
        startOffset: Int,
        endOffset: Int,
        document: Document,
        data: PKCommentContext?
    ): TextRange {
        val prefix = if (data?.isExpressionContext == true) EXPR_COMMENT_PREFIX else HTML_COMMENT_PREFIX
        val suffix = if (data?.isExpressionContext == true) EXPR_COMMENT_SUFFIX else HTML_COMMENT_SUFFIX

        return SelfManagingCommenterUtil.insertBlockComment(
            startOffset,
            endOffset,
            document,
            prefix,
            suffix
        )
    }

    /**
     * Removes a block comment from the specified range.
     *
     * @param startOffset The start offset of the commented range.
     * @param endOffset The end offset of the commented range.
     * @param document The document being edited.
     * @param data The comment context.
     */
    override fun uncommentBlockComment(
        startOffset: Int,
        endOffset: Int,
        document: Document,
        data: PKCommentContext?
    ) {
        val text = document.charsSequence

        if (CharArrayUtil.regionMatches(text, startOffset, EXPR_COMMENT_PREFIX) &&
            CharArrayUtil.regionMatches(text, endOffset - EXPR_COMMENT_SUFFIX.length, EXPR_COMMENT_SUFFIX)
        ) {
            SelfManagingCommenterUtil.uncommentBlockComment(
                startOffset,
                endOffset,
                document,
                EXPR_COMMENT_PREFIX,
                EXPR_COMMENT_SUFFIX
            )
        } else {
            SelfManagingCommenterUtil.uncommentBlockComment(
                startOffset,
                endOffset,
                document,
                HTML_COMMENT_PREFIX,
                HTML_COMMENT_SUFFIX
            )
        }
    }

    /**
     * Finds the range of a block comment containing the selection.
     *
     * @param selectionStart The start offset of the selection.
     * @param selectionEnd The end offset of the selection.
     * @param document The document being edited.
     * @param data The comment context.
     * @return The text range of the block comment, or null if not found.
     */
    override fun getBlockCommentRange(
        selectionStart: Int,
        selectionEnd: Int,
        document: Document,
        data: PKCommentContext
    ): TextRange? {
        var commentStart = SelfManagingCommenterUtil.getBlockCommentRange(
            selectionStart,
            selectionEnd,
            document,
            EXPR_COMMENT_PREFIX,
            EXPR_COMMENT_SUFFIX
        )

        if (commentStart != null) {
            return commentStart
        }

        commentStart = SelfManagingCommenterUtil.getBlockCommentRange(
            selectionStart,
            selectionEnd,
            document,
            HTML_COMMENT_PREFIX,
            HTML_COMMENT_SUFFIX
        )

        return commentStart
    }

    /**
     * Returns the comment prefix for a specific line.
     *
     * Returns the appropriate block-style prefix based on context:
     * expression context uses C-style, HTML context uses HTML comments.
     *
     * @param line The line number.
     * @param document The document being edited.
     * @param data The comment context.
     * @return The comment prefix to use.
     */
    override fun getCommentPrefix(line: Int, document: Document, data: PKCommentContext): String {
        return if (data.isExpressionContext) EXPR_COMMENT_PREFIX else HTML_COMMENT_PREFIX
    }

    /**
     * Checks if a line is commented using block-style line comments.
     *
     * Detects both HTML comments (`<!-- ... -->`) and expression comments (`/* ... */`).
     *
     * @param line The line number.
     * @param offset The offset within the line.
     * @param document The document being edited.
     * @param data The comment context.
     * @return true if the line is wrapped in a block comment.
     */
    override fun isLineCommented(
        line: Int,
        offset: Int,
        document: Document,
        data: PKCommentContext
    ): Boolean {
        val lineStartOffset = document.getLineStartOffset(line)
        val lineEndOffset = document.getLineEndOffset(line)
        val lineText = document.charsSequence.subSequence(lineStartOffset, lineEndOffset)

        val trimmedStart = CharArrayUtil.shiftForward(lineText, 0, " \t")
        val trimmedEnd = CharArrayUtil.shiftBackward(lineText, lineText.length - 1, " \t") + 1

        if (trimmedStart >= trimmedEnd) return false

        val trimmedText = lineText.subSequence(trimmedStart, trimmedEnd)

        return isCommentedWith(trimmedText, HTML_COMMENT_PREFIX, HTML_COMMENT_SUFFIX) ||
            isCommentedWith(trimmedText, EXPR_COMMENT_PREFIX, EXPR_COMMENT_SUFFIX)
    }

    /**
     * Checks if the text is wrapped with the given comment delimiters.
     *
     * @param text The text to check.
     * @param prefix The comment prefix.
     * @param suffix The comment suffix.
     * @return true if the text starts with prefix and ends with suffix.
     */
    private fun isCommentedWith(text: CharSequence, prefix: String, suffix: String): Boolean {
        return text.length >= prefix.length + suffix.length &&
            CharArrayUtil.regionMatches(text, 0, prefix) &&
            CharArrayUtil.regionMatches(text, text.length - suffix.length, suffix)
    }

    /**
     * Comments a single line using block-style line comments.
     *
     * Wraps the line content with the appropriate comment delimiters based on context.
     * HTML context uses `<!-- content -->`, expression context uses `/* content */`.
     *
     * @param line The line number.
     * @param offset The offset within the line.
     * @param document The document being edited.
     * @param data The comment context.
     */
    override fun commentLine(line: Int, offset: Int, document: Document, data: PKCommentContext) {
        val lineStartOffset = document.getLineStartOffset(line)
        val lineEndOffset = document.getLineEndOffset(line)
        val lineText = document.charsSequence.subSequence(lineStartOffset, lineEndOffset)

        val firstNonWhitespace = CharArrayUtil.shiftForward(lineText, 0, " \t")
        val lastNonWhitespace = CharArrayUtil.shiftBackward(lineText, lineText.length - 1, " \t") + 1

        val prefix = if (data.isExpressionContext) EXPR_COMMENT_PREFIX else HTML_COMMENT_PREFIX
        val suffix = if (data.isExpressionContext) EXPR_COMMENT_SUFFIX else HTML_COMMENT_SUFFIX

        if (firstNonWhitespace >= lastNonWhitespace) {
            document.insertString(lineEndOffset, "$prefix $suffix")
            return
        }

        val contentStart = lineStartOffset + firstNonWhitespace
        val contentEnd = lineStartOffset + lastNonWhitespace

        document.insertString(contentEnd, " $suffix")
        document.insertString(contentStart, "$prefix ")
    }

    /**
     * Uncomments a single line by removing block-style line comment delimiters.
     *
     * Removes both HTML comment delimiters (`<!-- -->`) and expression comment
     * delimiters (`/* */`) depending on what is present.
     *
     * @param line The line number.
     * @param offset The offset within the line.
     * @param document The document being edited.
     * @param data The comment context.
     */
    override fun uncommentLine(line: Int, offset: Int, document: Document, data: PKCommentContext) {
        val lineStartOffset = document.getLineStartOffset(line)
        val lineEndOffset = document.getLineEndOffset(line)
        val lineText = document.charsSequence.subSequence(lineStartOffset, lineEndOffset)

        val trimmedStart = CharArrayUtil.shiftForward(lineText, 0, " \t")
        val trimmedEnd = CharArrayUtil.shiftBackward(lineText, lineText.length - 1, " \t") + 1

        if (trimmedStart >= trimmedEnd) return

        val trimmedText = lineText.subSequence(trimmedStart, trimmedEnd)

        val (prefix, suffix) = when {
            isCommentedWith(trimmedText, HTML_COMMENT_PREFIX, HTML_COMMENT_SUFFIX) ->
                HTML_COMMENT_PREFIX to HTML_COMMENT_SUFFIX
            isCommentedWith(trimmedText, EXPR_COMMENT_PREFIX, EXPR_COMMENT_SUFFIX) ->
                EXPR_COMMENT_PREFIX to EXPR_COMMENT_SUFFIX
            else -> return
        }

        val prefixStart = lineStartOffset + trimmedStart
        val suffixStart = lineStartOffset + trimmedEnd - suffix.length

        var prefixEnd = prefixStart + prefix.length
        if (prefixEnd < lineEndOffset && document.charsSequence[prefixEnd] == ' ') {
            prefixEnd++
        }

        var adjustedSuffixStart = suffixStart
        if (adjustedSuffixStart > prefixEnd && document.charsSequence[adjustedSuffixStart - 1] == ' ') {
            adjustedSuffixStart--
        }

        document.deleteString(adjustedSuffixStart, suffixStart + suffix.length)
        document.deleteString(prefixStart, prefixEnd)
    }

    /**
     * Determines if the selection is within an expression context.
     *
     * @param file The PSI file.
     * @param selectionStart The start offset of the selection.
     * @param selectionEnd The end offset of the selection.
     * @return true if the selection is within an expression context.
     */
    private fun isExpressionContext(
        file: PsiFile,
        selectionStart: Int,
        selectionEnd: Int
    ): Boolean {
        val elementAtStart = file.findElementAt(selectionStart) ?: return false
        val elementAtEnd = if (selectionEnd > selectionStart) {
            file.findElementAt(selectionEnd - 1) ?: elementAtStart
        } else {
            elementAtStart
        }

        return isInExpressionContext(elementAtStart) || isInExpressionContext(elementAtEnd)
    }

    /**
     * Checks if an element is within an expression context.
     *
     * @param element The PSI element to check.
     * @return true if the element is within an expression context.
     */
    private fun isInExpressionContext(element: com.intellij.psi.PsiElement): Boolean {
        var current: com.intellij.psi.PsiElement? = element

        while (current != null && current !is PsiFile) {
            val curr = current
            val node = curr.node
            if (node == null) {
                current = curr.parent
                continue
            }

            val elementType = node.elementType

            if (isExpressionTokenType(elementType)) {
                return true
            }

            if (isInsideInterpolation(curr)) {
                return true
            }

            if (isInsideExpressionDirective(curr)) {
                return true
            }

            current = curr.parent
        }

        return false
    }

    /**
     * Delegates to the companion object method for expression token checking.
     *
     * @param elementType The element type to check.
     * @return true if the element type is an expression token.
     */
    private fun isExpressionTokenType(elementType: com.intellij.psi.tree.IElementType): Boolean {
        return isExpressionToken(elementType)
    }

    /**
     * Checks if an element is inside an interpolation expression.
     *
     * @param element The PSI element to check.
     * @return true if the element is inside an interpolation.
     */
    private fun isInsideInterpolation(element: com.intellij.psi.PsiElement): Boolean {
        var sibling = element.prevSibling
        var depth = 0

        while (sibling != null) {
            val elementType = sibling.node?.elementType
            when (elementType) {
                PKTokenTypes.INTERPOLATION_CLOSE -> depth++
                PKTokenTypes.INTERPOLATION_OPEN -> {
                    if (depth == 0) return true
                    depth--
                }
            }
            sibling = sibling.prevSibling
        }

        var parent = element.parent
        while (parent != null && parent !is PsiFile) {
            val parentSibling = parent.prevSibling
            if (parentSibling?.node?.elementType == PKTokenTypes.INTERPOLATION_OPEN) {
                return true
            }
            parent = parent.parent
        }

        return false
    }

    /**
     * Checks if an element is inside an expression directive.
     *
     * @param element The PSI element to check.
     * @return true if the element is inside an expression directive.
     */
    private fun isInsideExpressionDirective(element: com.intellij.psi.PsiElement): Boolean {
        var current: com.intellij.psi.PsiElement? = element

        while (current != null && current !is PsiFile) {
            val sibling = current.prevSibling
            if (sibling != null && checkSiblingForExpressionDirective(sibling)) {
                return true
            }
            current = current.parent
        }

        return false
    }

    /**
     * Checks if a sibling element indicates an expression directive context.
     *
     * @param sibling The sibling PSI element to check.
     * @return true if the sibling indicates an expression directive.
     */
    private fun checkSiblingForExpressionDirective(sibling: com.intellij.psi.PsiElement): Boolean {
        val siblingType = sibling.node?.elementType ?: return false

        return when (siblingType) {
            PKTokenTypes.DIRECTIVE_BIND,
            PKTokenTypes.DIRECTIVE_EVENT -> true
            PKTokenTypes.DIRECTIVE_NAME -> isExpressionDirectiveName(sibling.text)
            PKTokenTypes.HTML_ATTR_EQ -> checkAttrEqForExpressionDirective(sibling)
            else -> false
        }
    }

    /**
     * Checks if a directive name is an expression directive.
     *
     * @param directiveName The directive name to check.
     * @return true if the directive name matches an expression directive.
     */
    private fun isExpressionDirectiveName(directiveName: String): Boolean {
        return EXPRESSION_DIRECTIVES.any { directiveName.startsWith(it) }
    }

    /**
     * Checks the attribute before an equals sign for expression directive context.
     *
     * @param attrEqSibling The HTML_ATTR_EQ element.
     * @return true if the attribute indicates an expression directive.
     */
    private fun checkAttrEqForExpressionDirective(attrEqSibling: com.intellij.psi.PsiElement): Boolean {
        val attrNameSibling = attrEqSibling.prevSibling ?: return false
        val attrType = attrNameSibling.node?.elementType ?: return false

        return when (attrType) {
            PKTokenTypes.DIRECTIVE_NAME -> isExpressionDirectiveName(attrNameSibling.text)
            PKTokenTypes.DIRECTIVE_BIND,
            PKTokenTypes.DIRECTIVE_EVENT -> true
            else -> false
        }
    }
}

/**
 * Holds context data for the commenter during a comment operation.
 *
 * @property isExpressionContext True if the cursor is inside an expression
 *           context (interpolation or expression directive), false for HTML context.
 */
class PKCommentContext(
    val isExpressionContext: Boolean
) : CommenterDataHolder()
