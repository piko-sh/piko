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

import com.intellij.lang.BracePair
import com.intellij.lang.PairedBraceMatcher
import com.intellij.psi.PsiFile
import com.intellij.psi.TokenType
import com.intellij.psi.tree.IElementType
import com.intellij.psi.tree.TokenSet

/**
 * Provides brace matching for PK template files.
 *
 * Matches the following brace pairs:
 * - {{ }} for template interpolation
 * - ( ) for expression parentheses
 * - [ ] for expression brackets
 * - { } for expression braces
 * - ${ } for interpolations inside backtick strings
 */
class PKBraceMatcher : PairedBraceMatcher {

    /**
     * Returns the array of brace pairs supported by this matcher.
     *
     * @return Array of brace pairs for interpolation, parentheses, brackets, and braces.
     */
    override fun getPairs(): Array<BracePair> = PAIRS

    /**
     * Determines if a closing brace may be auto-inserted before the given token.
     *
     * @param lbraceType The type of the left brace.
     * @param contextType The element type following the brace, or null.
     * @return true when a matching closing brace may be inserted.
     */
    override fun isPairedBracesAllowedBeforeType(
        lbraceType: IElementType,
        contextType: IElementType?
    ): Boolean = contextType == null || contextType in ALLOWED_BEFORE

    /**
     * Returns the start offset of the code construct containing the brace.
     *
     * @param file The PSI file containing the brace.
     * @param openingBraceOffset The offset of the opening brace.
     * @return The same offset, indicating no special construct handling.
     */
    override fun getCodeConstructStart(file: PsiFile?, openingBraceOffset: Int): Int =
        openingBraceOffset

    companion object {
        /** The array of supported brace pairs for matching. */
        private val PAIRS = arrayOf(
            BracePair(PKTokenTypes.INTERPOLATION_OPEN, PKTokenTypes.INTERPOLATION_CLOSE, false),
            BracePair(PKTokenTypes.TEMPLATE_INTERP_OPEN, PKTokenTypes.TEMPLATE_INTERP_CLOSE, false),
            BracePair(PKTokenTypes.EXPR_PAREN_OPEN, PKTokenTypes.EXPR_PAREN_CLOSE, false),
            BracePair(PKTokenTypes.EXPR_BRACKET_OPEN, PKTokenTypes.EXPR_BRACKET_CLOSE, false),
            BracePair(PKTokenTypes.EXPR_BRACE_OPEN, PKTokenTypes.EXPR_BRACE_CLOSE, false)
        )

        /** Tokens before which a closing brace may be auto-inserted. */
        private val ALLOWED_BEFORE = TokenSet.create(
            TokenType.WHITE_SPACE,
            PKTokenTypes.INTERPOLATION_CLOSE,
            PKTokenTypes.TEMPLATE_INTERP_CLOSE,
            PKTokenTypes.EXPR_PAREN_CLOSE,
            PKTokenTypes.EXPR_BRACKET_CLOSE,
            PKTokenTypes.EXPR_BRACE_CLOSE,
            PKTokenTypes.EXPR_COMMA,
            PKTokenTypes.EXPR_COLON,
            PKTokenTypes.HTML_ATTR_QUOTE,
            PKTokenTypes.EXPR_STRING_QUOTE
        )
    }
}
