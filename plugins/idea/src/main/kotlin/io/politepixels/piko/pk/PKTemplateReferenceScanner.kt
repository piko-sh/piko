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

import com.intellij.psi.PsiElement
import com.intellij.psi.PsiFile
import com.intellij.psi.PsiRecursiveElementWalkingVisitor
import com.intellij.psi.PsiReference
import com.intellij.psi.TokenType
import com.intellij.psi.tree.IElementType
import com.intellij.util.Processor

/**
 * Walks a PK file's template tokens on behalf of reference-search contributors.
 *
 * Stops as soon as the search consumer asks it to, which Find Usages relies
 * on when the user cancels or a single result is enough.
 */
object PKTemplateReferenceScanner {

    /** Expression tokens that can name a script function or variable. */
    @JvmField
    val IDENTIFIER_TYPES: Set<IElementType> = setOf(
        PKTokenTypes.EXPR_FUNCTION_NAME,
        PKTokenTypes.EXPR_IDENTIFIER
    )

    /**
     * Visits every token of the given types and offers its references to the consumer.
     *
     * @param file The PK file to scan.
     * @param tokenTypes The token types to consider.
     * @param consumer The search consumer.
     * @param referencesFor Builds the references for a candidate token; an empty list skips it.
     * @return False when the consumer stopped the search.
     */
    @JvmStatic
    fun scan(
        file: PsiFile,
        tokenTypes: Set<IElementType>,
        consumer: Processor<in PsiReference>,
        referencesFor: (PsiElement) -> List<PsiReference>
    ): Boolean {
        var completed = true
        file.accept(object : PsiRecursiveElementWalkingVisitor() {
            override fun visitElement(element: PsiElement) {
                if (element.node?.elementType in tokenTypes) {
                    for (reference in referencesFor(element)) {
                        if (!consumer.process(reference)) {
                            completed = false
                            stopWalking()
                            return
                        }
                    }
                }
                super.visitElement(element)
            }
        })
        return completed
    }

    /**
     * Reports whether an expression identifier is accessed as a member, as in `state.name`.
     *
     * @param element The identifier token.
     * @return True when the previous non-whitespace token is a dot.
     */
    @JvmStatic
    fun isMemberAccess(element: PsiElement): Boolean {
        var previous = element.prevSibling
        while (previous != null && previous.node?.elementType == TokenType.WHITE_SPACE) {
            previous = previous.prevSibling
        }
        return previous?.node?.elementType == PKTokenTypes.EXPR_OP_DOT
    }
}
