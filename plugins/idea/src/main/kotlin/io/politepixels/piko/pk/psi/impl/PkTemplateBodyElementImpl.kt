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


package io.politepixels.piko.pk.psi.impl

import com.intellij.lang.ASTNode
import com.intellij.openapi.util.TextRange
import com.intellij.psi.LiteralTextEscaper
import com.intellij.psi.PsiLanguageInjectionHost

/**
 * PSI element for template body content.
 *
 * Implements PsiLanguageInjectionHost to allow the Piko LSP to provide
 * code intelligence for the HTML-like template markup inside template blocks.
 *
 * @param node The AST node containing the template body content.
 */
class PkTemplateBodyElementImpl(node: ASTNode) :
    PkPsiElementImpl(node), PsiLanguageInjectionHost {

    /**
     * Indicates whether this element can host injected language content.
     *
     * @return Always true, as template bodies always support injection.
     */
    override fun isValidHost(): Boolean = true

    /**
     * Updates the text content of this injection host.
     *
     * @param text The new text content.
     * @return This method always throws as text updates are not supported.
     * @throws UnsupportedOperationException Always thrown.
     */
    override fun updateText(text: String): PsiLanguageInjectionHost {
        throw UnsupportedOperationException("updateText() not supported in PK template body")
    }

    /**
     * Creates a text escaper for mapping positions between host and injected content.
     *
     * @return A literal text escaper that performs no escape transformations.
     */
    override fun createLiteralTextEscaper(): LiteralTextEscaper<out PsiLanguageInjectionHost> {
        return object : LiteralTextEscaper<PkTemplateBodyElementImpl>(this) {

            /**
             * Decodes the host text within the given range into the output buffer.
             *
             * @param rangeInsideHost The text range within the host element.
             * @param outChars        The buffer to append decoded characters to.
             * @return Always true, as no escape transformations are needed.
             */
            override fun decode(rangeInsideHost: TextRange, outChars: StringBuilder): Boolean {
                val subText = myHost.text.substring(rangeInsideHost.startOffset, rangeInsideHost.endOffset)
                outChars.append(subText)
                return true
            }

            /**
             * Maps an offset in the decoded text back to the host element.
             *
             * @param offsetInDecoded The character offset in the decoded content.
             * @param rangeInsideHost The text range within the host element.
             * @return The corresponding offset in the host, or -1 if out of range.
             */
            override fun getOffsetInHost(offsetInDecoded: Int, rangeInsideHost: TextRange): Int {
                val result = rangeInsideHost.startOffset + offsetInDecoded
                return if (result <= rangeInsideHost.endOffset) result else -1
            }

            /**
             * Indicates whether the injected content is restricted to a single line.
             *
             * @return Always false, as template content can span multiple lines.
             */
            override fun isOneLine(): Boolean = false
        }
    }
}
