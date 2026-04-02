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
import com.intellij.psi.PsiElement
import com.intellij.psi.tree.IElementType
import java.util.Collections

/**
 * Utility functions for traversing PSI and AST trees in PK files.
 *
 * Provides common sibling traversal patterns used across folding, selection,
 * and other editor features.
 */
object PKTreeUtils {

    /**
     * Finds the first following sibling that matches any of the target types.
     *
     * @param node The starting AST node.
     * @param targetTypes The element types to search for.
     * @return The matching sibling node, or null if not found.
     */
    @JvmStatic
    fun findNextSiblingOfType(node: ASTNode, vararg targetTypes: IElementType): ASTNode? {
        var sibling = node.treeNext
        while (sibling != null) {
            if (sibling.elementType in targetTypes) {
                return sibling
            }
            sibling = sibling.treeNext
        }
        return null
    }

    /**
     * Finds the first following sibling that matches any of the target types,
     * stopping at any of the boundary types.
     *
     * @param node The starting AST node.
     * @param targetTypes The element types to search for.
     * @param boundaryTypes The element types that stop the search.
     * @return The matching sibling node, or null if not found or boundary reached.
     */
    @JvmStatic
    fun findNextSiblingOfTypeUntil(
        node: ASTNode,
        targetTypes: Set<IElementType>,
        boundaryTypes: Set<IElementType>
    ): ASTNode? {
        var sibling = node.treeNext
        while (sibling != null) {
            if (sibling.elementType in targetTypes) {
                return sibling
            }
            if (sibling.elementType in boundaryTypes) {
                return null
            }
            sibling = sibling.treeNext
        }
        return null
    }

    /**
     * Finds the first preceding sibling that matches any of the target types.
     *
     * @param element The starting PSI element.
     * @param targetTypes The element types to search for.
     * @return The matching sibling element, or null if not found.
     */
    @JvmStatic
    fun findPrevSiblingOfType(element: PsiElement, vararg targetTypes: IElementType): PsiElement? {
        var sibling = element.prevSibling
        while (sibling != null) {
            val elementType = sibling.node?.elementType
            if (elementType in targetTypes) {
                return sibling
            }
            sibling = sibling.prevSibling
        }
        return null
    }

    /**
     * Finds the first following sibling that matches any of the target types.
     *
     * @param element The starting PSI element.
     * @param targetTypes The element types to search for.
     * @return The matching sibling element, or null if not found.
     */
    @JvmStatic
    fun findNextSiblingOfType(element: PsiElement, vararg targetTypes: IElementType): PsiElement? {
        var sibling = element.nextSibling
        while (sibling != null) {
            val elementType = sibling.node?.elementType
            if (elementType in targetTypes) {
                return sibling
            }
            sibling = sibling.nextSibling
        }
        return null
    }

    /**
     * Finds the first following sibling that matches any of the target types,
     * stopping at any of the boundary types.
     *
     * @param element The starting PSI element.
     * @param targetTypes The element types to search for.
     * @param boundaryTypes The element types that stop the search.
     * @return The matching sibling element, or null if not found or boundary reached.
     */
    @JvmStatic
    fun findNextSiblingOfTypeUntil(
        element: PsiElement,
        targetTypes: Set<IElementType>,
        boundaryTypes: Set<IElementType>
    ): PsiElement? {
        var sibling = element.nextSibling
        while (sibling != null) {
            val elementType = sibling.node?.elementType
            if (elementType in targetTypes) {
                return sibling
            }
            if (elementType in boundaryTypes) {
                return null
            }
            sibling = sibling.nextSibling
        }
        return null
    }

    /**
     * Finds the tag name following a tag open node.
     *
     * Searches for HTML_TAG_NAME or PIKO_TAG_NAME until reaching a tag close token.
     *
     * @param tagOpenNode The tag open AST node.
     * @return The tag name text, or null if not found.
     */
    @JvmStatic
    fun findTagName(tagOpenNode: ASTNode): String? {
        val tagNameNode = findNextSiblingOfTypeUntil(
            tagOpenNode,
            TAG_NAME_TYPES,
            TAG_CLOSE_TYPES
        )
        return tagNameNode?.text
    }

    /**
     * Finds the tag name following a tag open element.
     *
     * @param tagOpenElement The tag open PSI element.
     * @return The tag name text, or null if not found.
     */
    @JvmStatic
    fun findTagNameFromElement(tagOpenElement: PsiElement): String? {
        val tagNameElement = findNextSiblingOfTypeUntil(
            tagOpenElement,
            TAG_NAME_TYPES,
            TAG_CLOSE_TYPES
        )
        return tagNameElement?.text
    }

    /**
     * Checks if a tag is self-closing by looking for HTML_TAG_SELF_CLOSE.
     *
     * @param tagOpenNode The tag open AST node.
     * @return true if the tag ends with a self-close token.
     */
    @JvmStatic
    fun isSelfClosingTag(tagOpenNode: ASTNode): Boolean {
        var sibling = tagOpenNode.treeNext
        while (sibling != null) {
            when (sibling.elementType) {
                PKTokenTypes.HTML_TAG_SELF_CLOSE -> return true
                PKTokenTypes.HTML_TAG_CLOSE -> return false
            }
            sibling = sibling.treeNext
        }
        return false
    }

    /**
     * Checks if a tag element is self-closing.
     *
     * @param tagOpenElement The tag open PSI element.
     * @return true if the tag ends with a self-close token.
     */
    @JvmStatic
    fun isSelfClosingElement(tagOpenElement: PsiElement): Boolean {
        var sibling = tagOpenElement.nextSibling
        while (sibling != null) {
            when (sibling.node?.elementType) {
                PKTokenTypes.HTML_TAG_SELF_CLOSE -> return true
                PKTokenTypes.HTML_TAG_CLOSE -> return false
            }
            sibling = sibling.nextSibling
        }
        return false
    }

    /**
     * Finds the matching close token for a paired open/close token.
     *
     * Handles nesting by tracking depth.
     *
     * @param openNode The opening token node.
     * @param openType The element type of open tokens.
     * @param closeType The element type of close tokens.
     * @return The matching close node, or null if not found.
     */
    @JvmStatic
    fun findMatchingClose(
        openNode: ASTNode,
        openType: IElementType,
        closeType: IElementType
    ): ASTNode? {
        var depth = 1
        var current = openNode.treeNext
        while (current != null && depth > 0) {
            when (current.elementType) {
                openType -> depth++
                closeType -> {
                    depth--
                    if (depth == 0) return current
                }
            }
            current = current.treeNext
        }
        return null
    }

    /** Token types that represent tag names. */
    val TAG_NAME_TYPES: Set<IElementType> = Collections.unmodifiableSet(
        setOf(
            PKTokenTypes.HTML_TAG_NAME,
            PKTokenTypes.PIKO_TAG_NAME
        )
    )

    /** Token types that close a tag declaration. */
    val TAG_CLOSE_TYPES: Set<IElementType> = Collections.unmodifiableSet(
        setOf(
            PKTokenTypes.HTML_TAG_CLOSE,
            PKTokenTypes.HTML_TAG_SELF_CLOSE
        )
    )

    /** Token types for attribute names. */
    val ATTR_NAME_TYPES: Set<IElementType> = Collections.unmodifiableSet(
        setOf(
            PKTokenTypes.HTML_ATTR_NAME,
            PKTokenTypes.ATTR_NAME,
            PKTokenTypes.DIRECTIVE_NAME,
            PKTokenTypes.DIRECTIVE_BIND,
            PKTokenTypes.DIRECTIVE_EVENT
        )
    )
}
