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

import com.intellij.lang.ASTNode
import com.intellij.lang.folding.FoldingBuilderEx
import com.intellij.lang.folding.FoldingDescriptor
import com.intellij.openapi.editor.Document
import com.intellij.openapi.project.DumbAware
import com.intellij.openapi.util.TextRange
import com.intellij.psi.PsiElement
import com.intellij.psi.util.PsiTreeUtil
import java.util.regex.Pattern

/**
 * Provides code folding support for PK template files.
 *
 * Creates foldable regions for top-level blocks, multi-line HTML tags,
 * interpolation expressions, and HTML comments.
 */
class PKFoldingBuilder : FoldingBuilderEx(), DumbAware {

    /**
     * Builds fold regions for the given PSI element.
     *
     * Traverses the PSI tree to identify foldable elements and creates
     * descriptors for each region that can be collapsed.
     *
     * @param root The root PSI element of the file.
     * @param document The document containing the file content.
     * @param quick If true, only fast folding regions should be returned.
     * @return Array of folding descriptors for all foldable regions.
     */
    override fun buildFoldRegions(
        root: PsiElement,
        document: Document,
        quick: Boolean
    ): Array<FoldingDescriptor> {
        val descriptors = mutableListOf<FoldingDescriptor>()

        PsiTreeUtil.processElements(root) { element ->
            val node = element.node ?: return@processElements true

            when (node.elementType) {
                PKTokenTypes.TEMPLATE_BLOCK_ELEMENT,
                PKTokenTypes.SCRIPT_BLOCK_ELEMENT,
                PKTokenTypes.STYLE_BLOCK_ELEMENT,
                PKTokenTypes.I18N_BLOCK_ELEMENT -> {
                    addBlockFoldRegion(node, document, descriptors)
                }
                PKTokenTypes.HTML_COMMENT -> {
                    addCommentFoldRegion(node, document, descriptors)
                }
            }
            true
        }

        addInterpolationFoldRegions(root, document, descriptors)
        addHtmlTagFoldRegions(root, document, descriptors)
        addRegionFoldRanges(document, descriptors)

        return descriptors.toTypedArray()
    }

    /**
     * Returns the placeholder text shown when a region is collapsed.
     *
     * @param node The AST node of the collapsed region.
     * @return The placeholder string to display.
     */
    override fun getPlaceholderText(node: ASTNode): String? {
        return when (node.elementType) {
            PKTokenTypes.TEMPLATE_BLOCK_ELEMENT -> PKConstants.FoldingPlaceholder.TEMPLATE
            PKTokenTypes.SCRIPT_BLOCK_ELEMENT -> getScriptPlaceholder(node)
            PKTokenTypes.STYLE_BLOCK_ELEMENT -> PKConstants.FoldingPlaceholder.STYLE
            PKTokenTypes.I18N_BLOCK_ELEMENT -> PKConstants.FoldingPlaceholder.I18N
            PKTokenTypes.HTML_COMMENT -> PKConstants.FoldingPlaceholder.COMMENT
            PKTokenTypes.INTERPOLATION_OPEN -> PKConstants.FoldingPlaceholder.INTERPOLATION
            PKTokenTypes.HTML_TAG_OPEN -> getHtmlTagPlaceholder(node)
            else -> PKConstants.FoldingPlaceholder.DEFAULT
        }
    }

    /**
     * Determines whether a region should be collapsed by default.
     *
     * @param node The AST node of the region.
     * @return false, so all regions are expanded by default.
     */
    override fun isCollapsedByDefault(node: ASTNode): Boolean = false

    /**
     * Adds a fold region for a block element if it spans multiple lines.
     *
     * @param node The AST node of the block element.
     * @param document The document containing the content.
     * @param descriptors The list to add the fold descriptor to.
     */
    private fun addBlockFoldRegion(
        node: ASTNode,
        document: Document,
        descriptors: MutableList<FoldingDescriptor>
    ) {
        val textRange = node.textRange
        val startLine = document.getLineNumber(textRange.startOffset)
        val endLine = document.getLineNumber(textRange.endOffset)

        if (endLine > startLine) {
            descriptors.add(FoldingDescriptor(node, textRange))
        }
    }

    /**
     * Adds a fold region for a comment if it spans multiple lines.
     *
     * @param node The AST node of the comment.
     * @param document The document containing the content.
     * @param descriptors The list to add the fold descriptor to.
     */
    private fun addCommentFoldRegion(
        node: ASTNode,
        document: Document,
        descriptors: MutableList<FoldingDescriptor>
    ) {
        val textRange = node.textRange
        val startLine = document.getLineNumber(textRange.startOffset)
        val endLine = document.getLineNumber(textRange.endOffset)

        if (endLine > startLine) {
            descriptors.add(FoldingDescriptor(node, textRange))
        }
    }

    /**
     * Scans the document for interpolation expressions and adds fold regions.
     *
     * @param root The root PSI element.
     * @param document The document containing the content.
     * @param descriptors The list to add fold descriptors to.
     */
    private fun addInterpolationFoldRegions(
        root: PsiElement,
        document: Document,
        descriptors: MutableList<FoldingDescriptor>
    ) {
        var node = root.node?.firstChildNode
        while (node != null) {
            processInterpolationsInNode(node, document, descriptors)
            node = node.treeNext
        }
    }

    /**
     * Recursively processes a node and its children for interpolations.
     *
     * @param node The current AST node to process.
     * @param document The document containing the content.
     * @param descriptors The list to add fold descriptors to.
     */
    private fun processInterpolationsInNode(
        node: ASTNode,
        document: Document,
        descriptors: MutableList<FoldingDescriptor>
    ) {
        var current: ASTNode? = node.firstChildNode
        while (current != null) {
            if (current.elementType == PKTokenTypes.INTERPOLATION_OPEN) {
                tryAddInterpolationFold(current, document, descriptors)
            }
            processInterpolationsInNode(current, document, descriptors)
            current = current.treeNext
        }
    }

    /**
     * Attempts to add a fold region for an interpolation expression.
     *
     * @param openNode The interpolation open token node.
     * @param document The document containing the content.
     * @param descriptors The list to add the fold descriptor to.
     */
    private fun tryAddInterpolationFold(
        openNode: ASTNode,
        document: Document,
        descriptors: MutableList<FoldingDescriptor>
    ) {
        val closeNode = findMatchingClose(
            openNode,
            PKTokenTypes.INTERPOLATION_OPEN,
            PKTokenTypes.INTERPOLATION_CLOSE
        ) ?: return

        val range = TextRange(openNode.startOffset, closeNode.startOffset + closeNode.textLength)
        if (isMultiLine(range, document)) {
            descriptors.add(FoldingDescriptor(openNode, range))
        }
    }

    /**
     * Checks if a text range spans multiple lines.
     *
     * @param range The text range to check.
     * @param document The document to get line numbers from.
     * @return true if the range spans more than one line.
     */
    private fun isMultiLine(range: TextRange, document: Document): Boolean {
        val startLine = document.getLineNumber(range.startOffset)
        val endLine = document.getLineNumber(range.endOffset)
        return endLine > startLine
    }

    /**
     * Scans the document for HTML tags and adds fold regions.
     *
     * @param root The root PSI element.
     * @param document The document containing the content.
     * @param descriptors The list to add fold descriptors to.
     */
    private fun addHtmlTagFoldRegions(
        root: PsiElement,
        document: Document,
        descriptors: MutableList<FoldingDescriptor>
    ) {
        var node = root.node?.firstChildNode
        while (node != null) {
            processHtmlTagsInNode(node, document, descriptors)
            node = node.treeNext
        }
    }

    /**
     * Recursively processes a node and its children for HTML tags.
     *
     * @param node The current AST node to process.
     * @param document The document containing the content.
     * @param descriptors The list to add fold descriptors to.
     */
    private fun processHtmlTagsInNode(
        node: ASTNode,
        document: Document,
        descriptors: MutableList<FoldingDescriptor>
    ) {
        var current: ASTNode? = node.firstChildNode
        while (current != null) {
            if (current.elementType == PKTokenTypes.HTML_TAG_OPEN) {
                tryAddHtmlTagFold(current, document, descriptors)
            }
            processHtmlTagsInNode(current, document, descriptors)
            current = current.treeNext
        }
    }

    /**
     * Attempts to add a fold region for an HTML tag.
     *
     * @param openNode The tag open token node.
     * @param document The document containing the content.
     * @param descriptors The list to add the fold descriptor to.
     */
    private fun tryAddHtmlTagFold(
        openNode: ASTNode,
        document: Document,
        descriptors: MutableList<FoldingDescriptor>
    ) {
        val tagName = findTagName(openNode) ?: return
        if (isSelfClosingTag(openNode)) return
        val closeNode = findMatchingEndTag(openNode, tagName) ?: return

        val range = TextRange(openNode.startOffset, closeNode.startOffset + closeNode.textLength)
        if (isMultiLine(range, document)) {
            descriptors.add(FoldingDescriptor(openNode, range))
        }
    }

    /**
     * Finds the matching close token for a given open token type.
     *
     * @param openNode The opening token node.
     * @param openType The element type of the open token.
     * @param closeType The element type of the close token.
     * @return The matching close node, or null if not found.
     */
    private fun findMatchingClose(
        openNode: ASTNode,
        openType: com.intellij.psi.tree.IElementType,
        closeType: com.intellij.psi.tree.IElementType
    ): ASTNode? = PKTreeUtils.findMatchingClose(openNode, openType, closeType)

    /**
     * Finds the tag name following a tag open node.
     *
     * @param tagOpenNode The tag open token node.
     * @return The tag name text, or null if not found.
     */
    private fun findTagName(tagOpenNode: ASTNode): String? = PKTreeUtils.findTagName(tagOpenNode)

    /**
     * Checks if a tag is self-closing.
     *
     * @param tagOpenNode The tag open token node.
     * @return true if the tag ends with a self-close token.
     */
    private fun isSelfClosingTag(tagOpenNode: ASTNode): Boolean = PKTreeUtils.isSelfClosingTag(tagOpenNode)

    /**
     * Finds the matching end tag for a given tag name.
     *
     * @param tagOpenNode The tag open token node.
     * @param tagName The tag name to match.
     * @return The close token node, or null if not found.
     */
    private fun findMatchingEndTag(tagOpenNode: ASTNode, tagName: String): ASTNode? {
        var depth = 1
        var current = tagOpenNode.treeNext
        while (current != null && depth > 0) {
            depth = updateDepthForTag(current, tagName, depth)
            if (depth == 0) {
                return findTagCloseToken(current)
            }
            current = current.treeNext
        }
        return null
    }

    /**
     * Updates the nesting depth based on tag open/close elements.
     *
     * @param node The current AST node being examined.
     * @param tagName The tag name being matched.
     * @param currentDepth The current nesting depth.
     * @return The updated depth after processing this node.
     */
    private fun updateDepthForTag(node: ASTNode, tagName: String, currentDepth: Int): Int {
        return when (node.elementType) {
            PKTokenTypes.HTML_TAG_OPEN -> {
                val name = findTagName(node)
                if (name == tagName && !isSelfClosingTag(node)) currentDepth + 1 else currentDepth
            }
            PKTokenTypes.HTML_END_TAG_OPEN -> {
                val closingTagName = findClosingTagName(node)
                if (closingTagName == tagName) currentDepth - 1 else currentDepth
            }
            else -> currentDepth
        }
    }

    /**
     * Finds the tag name in a closing tag.
     *
     * @param endTagOpenNode The end tag open token node.
     * @return The tag name text, or null if not found.
     */
    private fun findClosingTagName(endTagOpenNode: ASTNode): String? = PKTreeUtils.findTagName(endTagOpenNode)

    /**
     * Finds the close token following an end tag open.
     *
     * @param endTagOpenNode The end tag open token node.
     * @return The close token node, or the input node if not found.
     */
    private fun findTagCloseToken(endTagOpenNode: ASTNode): ASTNode? {
        return PKTreeUtils.findNextSiblingOfType(endTagOpenNode, PKTokenTypes.HTML_TAG_CLOSE)
            ?: endTagOpenNode
    }

    /**
     * Returns the placeholder text for a script block, including lang attribute.
     *
     * @param node The script block AST node.
     * @return The placeholder string with optional lang attribute.
     */
    private fun getScriptPlaceholder(node: ASTNode): String {
        var child = node.firstChildNode
        while (child != null) {
            if (child.elementType == PKTokenTypes.ATTR_VALUE) {
                val value = child.text.trim('"', '\'')
                if (PKConstants.ScriptLang.isJsLanguage(value)) {
                    return PKConstants.FoldingPlaceholder.scriptWithLang(value)
                }
            }
            child = child.treeNext
        }
        return PKConstants.FoldingPlaceholder.SCRIPT
    }

    /**
     * Returns the placeholder text for an HTML tag including the tag name.
     *
     * @param node The tag open AST node.
     * @return The placeholder string with the tag name.
     */
    private fun getHtmlTagPlaceholder(node: ASTNode): String {
        val tagName = findTagName(node) ?: return PKConstants.FoldingPlaceholder.TAG
        return PKConstants.FoldingPlaceholder.tagWithName(tagName)
    }

    /**
     * Scans for manual region markers and creates fold regions.
     *
     * Supports the pattern:
     * ```
     * <!-- #region [Optional Name] -->
     * ...content...
     * <!-- #endregion -->
     * ```
     *
     * @param document The document containing the content.
     * @param descriptors The list to add fold descriptors to.
     */
    private fun addRegionFoldRanges(
        document: Document,
        descriptors: MutableList<FoldingDescriptor>
    ) {
        val text = document.charsSequence
        val regionStack = mutableListOf<RegionStart>()

        for (line in 0 until document.lineCount) {
            processRegionLine(document, text, line, regionStack, descriptors)
        }
    }

    /**
     * Processes a single line for region markers.
     *
     * @param document The document being processed.
     * @param text The document text.
     * @param line The line number to process.
     * @param regionStack Stack of open region markers.
     * @param descriptors The list to add fold descriptors to.
     */
    private fun processRegionLine(
        document: Document,
        text: CharSequence,
        line: Int,
        regionStack: MutableList<RegionStart>,
        descriptors: MutableList<FoldingDescriptor>
    ) {
        val lineStart = document.getLineStartOffset(line)
        val lineEnd = document.getLineEndOffset(line)
        val lineText = text.subSequence(lineStart, lineEnd)

        if (tryParseRegionStart(lineText, lineStart, regionStack)) {
            return
        }

        tryParseRegionEnd(lineText, lineEnd, document, regionStack, descriptors)
    }

    /**
     * Attempts to parse a region start marker from a line.
     *
     * @param lineText The text of the line.
     * @param lineStart The offset of the line start.
     * @param regionStack Stack to push the region start onto.
     * @return true if a region start was found, false otherwise.
     */
    private fun tryParseRegionStart(
        lineText: CharSequence,
        lineStart: Int,
        regionStack: MutableList<RegionStart>
    ): Boolean {
        val matcher = REGION_START_PATTERN.matcher(lineText)
        if (!matcher.find()) {
            return false
        }

        val regionName = matcher.group(1).trim().ifEmpty { "region" }
        regionStack.add(RegionStart(lineStart, regionName))
        return true
    }

    /**
     * Attempts to parse a region end marker and create a fold descriptor.
     *
     * @param lineText The text of the line.
     * @param lineEnd The offset of the line end.
     * @param document The document being processed.
     * @param regionStack Stack of open region markers.
     * @param descriptors The list to add fold descriptors to.
     */
    private fun tryParseRegionEnd(
        lineText: CharSequence,
        lineEnd: Int,
        document: Document,
        regionStack: MutableList<RegionStart>,
        descriptors: MutableList<FoldingDescriptor>
    ) {
        val matcher = REGION_END_PATTERN.matcher(lineText)
        if (!matcher.find()) {
            return
        }
        if (regionStack.isEmpty()) {
            return
        }

        val regionStart = regionStack.removeLast()
        val range = TextRange(regionStart.offset, lineEnd)

        if (!isMultiLine(range, document)) {
            return
        }

        addRegionDescriptor(range, regionStart.name, descriptors)
    }

    /**
     * Creates and adds a folding descriptor for a region.
     *
     * @param range The text range of the region.
     * @param name The name of the region.
     * @param descriptors The list to add the descriptor to.
     */
    private fun addRegionDescriptor(
        range: TextRange,
        name: String,
        descriptors: MutableList<FoldingDescriptor>
    ) {
        regionPlaceholders[range] = "#region $name"
        descriptors.add(
            object : FoldingDescriptor(RegionLeafElement(range), range) {
                override fun getPlaceholderText(): String = regionPlaceholders[range] ?: "#region"
            }
        )
    }

    /**
     * Holds the start position and name of a region marker.
     */
    private data class RegionStart(val offset: Int, val name: String)

    /**
     * Minimal leaf element for region fold descriptors.
     */
    private class RegionLeafElement(
        private val range: TextRange
    ) : com.intellij.psi.impl.source.tree.LeafElement(PKTokenTypes.HTML_COMMENT, "") {
        override fun getTextRange(): TextRange = range
        override fun getStartOffset(): Int = range.startOffset
    }

    companion object {
        /** Pattern to match region start markers like `<!-- #region [name] -->` */
        private val REGION_START_PATTERN: Pattern =
            Pattern.compile("<!--\\s*#region\\s*(.*?)\\s*-->")

        /** Pattern to match region end markers `<!-- #endregion -->` */
        private val REGION_END_PATTERN: Pattern =
            Pattern.compile("<!--\\s*#endregion\\s*-->")

        /** Cache for region placeholder texts. */
        private val regionPlaceholders = mutableMapOf<TextRange, String>()

        /**
         * Returns the placeholder text for the given element type.
         *
         * @param elementType The element type of the folded region.
         * @return The placeholder string to display.
         */
        @JvmStatic
        fun getPlaceholderForElementType(
            elementType: com.intellij.psi.tree.IElementType
        ): String {
            return when (elementType) {
                PKTokenTypes.TEMPLATE_BLOCK_ELEMENT -> PKConstants.FoldingPlaceholder.TEMPLATE
                PKTokenTypes.SCRIPT_BLOCK_ELEMENT -> PKConstants.FoldingPlaceholder.SCRIPT
                PKTokenTypes.STYLE_BLOCK_ELEMENT -> PKConstants.FoldingPlaceholder.STYLE
                PKTokenTypes.I18N_BLOCK_ELEMENT -> PKConstants.FoldingPlaceholder.I18N
                PKTokenTypes.HTML_COMMENT -> PKConstants.FoldingPlaceholder.COMMENT
                PKTokenTypes.INTERPOLATION_OPEN -> PKConstants.FoldingPlaceholder.INTERPOLATION
                PKTokenTypes.HTML_TAG_OPEN -> PKConstants.FoldingPlaceholder.TAG
                else -> PKConstants.FoldingPlaceholder.DEFAULT
            }
        }
    }
}
