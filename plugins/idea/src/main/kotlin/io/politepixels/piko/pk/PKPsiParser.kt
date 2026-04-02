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
import com.intellij.lang.PsiBuilder
import com.intellij.lang.PsiParser
import com.intellij.psi.TokenType
import com.intellij.psi.tree.IElementType

/**
 * Builds PSI trees for PK template files.
 *
 * Creates structural elements for template, script, style, and i18n blocks.
 * Each block contains a body element that serves as an injection host for
 * embedded language content. TextMate handles syntax highlighting while the
 * LSP provides semantic features.
 */
class PKPsiParser : PsiParser {

    /**
     * Parses the token stream and builds the complete AST.
     *
     * @param root The root element type for the file.
     * @param builder The PSI builder providing tokens and tree construction.
     * @return The root AST node of the parsed file.
     */
    override fun parse(root: IElementType, builder: PsiBuilder): ASTNode {
        val rootMarker = builder.mark()
        parseFile(builder)
        rootMarker.done(root)
        return builder.treeBuilt
    }

    /**
     * Parses top-level blocks until end of file.
     *
     * @param builder The PSI builder to consume tokens from.
     */
    private fun parseFile(builder: PsiBuilder) {
        while (!builder.eof()) {
            when (builder.tokenType) {
                PKTokenTypes.TEMPLATE_TAG_START -> parseTemplateBlock(builder)
                PKTokenTypes.SCRIPT_TAG_START -> parseScriptBlock(builder)
                PKTokenTypes.STYLE_TAG_START -> parseStyleBlock(builder)
                PKTokenTypes.I18N_TAG_START -> parseI18nBlock(builder)
                PKTokenTypes.HTML_COMMENT,
                TokenType.WHITE_SPACE -> builder.advanceLexer()
                else -> builder.advanceLexer()
            }
        }
    }

    /**
     * Parses a template block including its opening tag, body, and closing tag.
     *
     * @param builder The PSI builder positioned at the template opening tag.
     */
    private fun parseTemplateBlock(builder: PsiBuilder) {
        val blockMarker = builder.mark()

        skipToTagEnd(builder)

        val bodyMarker = builder.mark()
        while (!builder.eof() && builder.tokenType != PKTokenTypes.TEMPLATE_TAG_END) {
            builder.advanceLexer()
        }
        bodyMarker.done(PKTokenTypes.TEMPLATE_BODY_ELEMENT)

        skipClosingTag(builder)

        blockMarker.done(PKTokenTypes.TEMPLATE_BLOCK_ELEMENT)
    }

    /**
     * Parses a script block, detecting whether it contains Go or JavaScript.
     *
     * @param builder The PSI builder positioned at the script opening tag.
     */
    private fun parseScriptBlock(builder: PsiBuilder) {
        val blockMarker = builder.mark()

        skipToTagEnd(builder)

        val isJsScript = builder.tokenType == PKTokenTypes.JS_SCRIPT_CONTENT

        val bodyMarker = builder.mark()
        while (!builder.eof() && builder.tokenType != PKTokenTypes.SCRIPT_TAG_END) {
            builder.advanceLexer()
        }

        if (isJsScript) {
            bodyMarker.done(PKTokenTypes.JS_SCRIPT_BODY_ELEMENT)
        } else {
            bodyMarker.done(PKTokenTypes.GO_SCRIPT_BODY_ELEMENT)
        }

        skipClosingTag(builder)

        blockMarker.done(PKTokenTypes.SCRIPT_BLOCK_ELEMENT)
    }

    /**
     * Parses a style block containing CSS content.
     *
     * @param builder The PSI builder positioned at the style opening tag.
     */
    private fun parseStyleBlock(builder: PsiBuilder) {
        val blockMarker = builder.mark()

        skipToTagEnd(builder)

        val bodyMarker = builder.mark()
        while (!builder.eof() && builder.tokenType != PKTokenTypes.STYLE_TAG_END) {
            builder.advanceLexer()
        }
        bodyMarker.done(PKTokenTypes.CSS_STYLE_BODY_ELEMENT)

        skipClosingTag(builder)

        blockMarker.done(PKTokenTypes.STYLE_BLOCK_ELEMENT)
    }

    /**
     * Parses an i18n block containing JSON translation content.
     *
     * @param builder The PSI builder positioned at the i18n opening tag.
     */
    private fun parseI18nBlock(builder: PsiBuilder) {
        val blockMarker = builder.mark()

        skipToTagEnd(builder)

        val bodyMarker = builder.mark()
        while (!builder.eof() && builder.tokenType != PKTokenTypes.I18N_TAG_END) {
            builder.advanceLexer()
        }
        bodyMarker.done(PKTokenTypes.I18N_BODY_ELEMENT)

        skipClosingTag(builder)

        blockMarker.done(PKTokenTypes.I18N_BLOCK_ELEMENT)
    }

    /**
     * Advances past the opening tag and its attributes to the closing angle bracket.
     *
     * @param builder The PSI builder to consume tokens from.
     */
    private fun skipToTagEnd(builder: PsiBuilder) {
        while (!builder.eof() && builder.tokenType != PKTokenTypes.TAG_END_GT) {
            builder.advanceLexer()
        }
        if (builder.tokenType == PKTokenTypes.TAG_END_GT) {
            builder.advanceLexer()
        }
    }

    /**
     * Advances past a closing tag token and its trailing angle bracket.
     *
     * @param builder The PSI builder to consume tokens from.
     */
    private fun skipClosingTag(builder: PsiBuilder) {
        if (isBlockClosingTag(builder.tokenType)) {
            builder.advanceLexer()
        }
        if (builder.tokenType == PKTokenTypes.TAG_END_GT) {
            builder.advanceLexer()
        }
    }

    /**
     * Checks if the token type is a block-closing tag.
     *
     * @param tokenType The token type to check.
     * @return True if the token closes a template, script, style, or i18n block.
     */
    private fun isBlockClosingTag(tokenType: IElementType?): Boolean {
        return tokenType == PKTokenTypes.TEMPLATE_TAG_END ||
            tokenType == PKTokenTypes.SCRIPT_TAG_END ||
            tokenType == PKTokenTypes.STYLE_TAG_END ||
            tokenType == PKTokenTypes.I18N_TAG_END
    }
}
