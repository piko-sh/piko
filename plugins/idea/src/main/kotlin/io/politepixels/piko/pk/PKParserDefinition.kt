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
import com.intellij.lang.ParserDefinition
import com.intellij.lang.PsiParser
import com.intellij.lexer.Lexer
import com.intellij.openapi.project.Project
import com.intellij.psi.FileViewProvider
import com.intellij.psi.PsiElement
import com.intellij.psi.PsiFile
import com.intellij.psi.TokenType
import com.intellij.psi.tree.IFileElementType
import com.intellij.psi.tree.TokenSet
import io.politepixels.piko.pk.psi.impl.*

/**
 * Defines how IntelliJ parses PK template files into PSI trees.
 *
 * Creates structural elements for template, script, style, and i18n blocks.
 * These elements serve as injection hosts for embedded languages like Go, CSS,
 * TypeScript, and JSON.
 */
class PKParserDefinition : ParserDefinition {

    companion object {
        /** The root file element type for PK files. */
        val FILE: IFileElementType = IFileElementType(PKLanguage)

        /** Token set for whitespace handling during parsing. */
        val WHITESPACES: TokenSet = TokenSet.create(TokenType.WHITE_SPACE)

        /** Token set for comment tokens, used by brace matching and folding. */
        val COMMENTS: TokenSet = PKTokenTypes.COMMENTS

        /** Token set for string literal tokens, used by syntax highlighting. */
        val STRINGS: TokenSet = PKTokenTypes.STRING_LITERALS
    }

    /**
     * Creates a new lexer for tokenising PK file content.
     *
     * @param project The current project context.
     * @return A fresh lexer adapter wrapping the generated JFlex lexer.
     */
    override fun createLexer(project: Project): Lexer = PKLexerAdapter()

    /**
     * Creates a new parser for building PSI trees from tokens.
     *
     * @param project The current project context.
     * @return A parser that builds structural PSI elements for language injection.
     */
    override fun createParser(project: Project): PsiParser = PKPsiParser()

    /**
     * Returns the root file element type for PK files.
     *
     * @return The file-level IFileElementType used as the PSI tree root.
     */
    override fun getFileNodeType(): IFileElementType = FILE

    /**
     * Returns the token set containing comment token types.
     *
     * @return The comment tokens for brace matching and folding.
     */
    override fun getCommentTokens(): TokenSet = COMMENTS

    /**
     * Returns the token set containing string literal token types.
     *
     * @return The string tokens for syntax highlighting.
     */
    override fun getStringLiteralElements(): TokenSet = STRINGS

    /**
     * Creates a PSI element for the given AST node.
     *
     * Maps each element type to its corresponding PSI implementation class.
     * Block elements wrap entire sections, while body elements serve as
     * injection hosts for embedded language content.
     *
     * @param node The AST node to wrap in a PSI element.
     * @return The appropriate PSI element implementation.
     */
    override fun createElement(node: ASTNode): PsiElement {
        return when (node.elementType) {
            PKTokenTypes.TEMPLATE_BLOCK_ELEMENT -> PkTemplateBlockElementImpl(node)
            PKTokenTypes.SCRIPT_BLOCK_ELEMENT -> PkScriptBlockElementImpl(node)
            PKTokenTypes.STYLE_BLOCK_ELEMENT -> PkStyleBlockElementImpl(node)
            PKTokenTypes.I18N_BLOCK_ELEMENT -> PkPsiElementImpl(node)
            PKTokenTypes.TEMPLATE_BODY_ELEMENT -> PkTemplateBodyElementImpl(node)
            PKTokenTypes.GO_SCRIPT_BODY_ELEMENT -> PkGoScriptContentElementImpl(node)
            PKTokenTypes.JS_SCRIPT_BODY_ELEMENT -> PkJsScriptContentElementImpl(node)
            PKTokenTypes.CSS_STYLE_BODY_ELEMENT -> PkCssStyleContentElementImpl(node)
            PKTokenTypes.I18N_BODY_ELEMENT -> PkI18nContentElementImpl(node)
            else -> PkPsiElementImpl(node)
        }
    }

    /**
     * Creates a PsiFile instance for the given file view provider.
     *
     * @param viewProvider The view provider supplying file content.
     * @return A new PKPsiFile representing the parsed file.
     */
    override fun createFile(viewProvider: FileViewProvider): PsiFile = PKPsiFile(viewProvider)

    /**
     * Determines whitespace requirements between adjacent tokens.
     *
     * @param left The left AST node.
     * @param right The right AST node.
     * @return MAY, indicating whitespace is optional between tokens.
     */
    override fun spaceExistenceTypeBetweenTokens(
        left: ASTNode,
        right: ASTNode
    ): ParserDefinition.SpaceRequirements = ParserDefinition.SpaceRequirements.MAY
}
