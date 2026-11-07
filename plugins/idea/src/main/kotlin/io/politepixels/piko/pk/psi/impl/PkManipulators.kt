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

import com.intellij.openapi.util.TextRange
import com.intellij.psi.AbstractElementManipulator
import com.intellij.psi.PsiLanguageInjectionHost
import com.intellij.psi.PsiFileFactory
import com.intellij.psi.util.PsiTreeUtil
import com.intellij.util.IncorrectOperationException
import io.politepixels.piko.pk.PKLanguage

/**
 * Edits the text of a PK leaf token, for example when a reference on a
 * template identifier or class name is renamed.
 */
class PkLeafManipulator : AbstractElementManipulator<PkLeafPsiElement>() {

    /**
     * Replaces part of the leaf's text.
     *
     * @param element The leaf to edit.
     * @param range The range inside the leaf to replace.
     * @param newContent The replacement text.
     * @return The new leaf.
     * @throws IncorrectOperationException When the replacement is not a PK leaf.
     */
    override fun handleContentChange(element: PkLeafPsiElement, range: TextRange, newContent: String): PkLeafPsiElement {
        val newText = range.replace(element.text, newContent)
        return element.replaceWithText(newText) as? PkLeafPsiElement
            ?: throw IncorrectOperationException("Replacement for ${element.elementType} is not a PK leaf")
    }
}

/**
 * Edits the text of a PK injection host (template body, Go, JS, CSS or i18n
 * block content), for example when an injected-language refactoring writes
 * back into the `.pk` file.
 */
class PkInjectionHostManipulator : AbstractElementManipulator<PsiLanguageInjectionHost>() {

    /**
     * Replaces part of the host's text.
     *
     * @param element The host to edit.
     * @param range The range inside the host to replace.
     * @param newContent The replacement text.
     * @return The new host.
     */
    override fun handleContentChange(
        element: PsiLanguageInjectionHost,
        range: TextRange,
        newContent: String
    ): PsiLanguageInjectionHost = element.updateText(range.replace(element.text, newContent))
}

/**
 * Implements [PsiLanguageInjectionHost.updateText] for PK hosts.
 */
object PkInjectionHostText {

    /**
     * Replaces a host's entire text.
     *
     * @param host The host to update.
     * @param newText The new host text.
     * @return The replacement host now in the tree.
     * @throws IncorrectOperationException When the new text would change the block structure.
     */
    @JvmStatic
    fun <T : PsiLanguageInjectionHost> replace(host: T, newText: String): T {
        val file = host.containingFile
            ?: throw IncorrectOperationException("Host ${host.javaClass.simpleName} has no containing file")
        val range = host.textRange
        val fileText = file.text
        val newFileText = fileText.substring(0, range.startOffset) + newText + fileText.substring(range.endOffset)

        val copy = PsiFileFactory.getInstance(host.project)
            .createFileFromText(file.name, PKLanguage, newFileText)
        val hostClass = host.javaClass
        val replacement = PsiTreeUtil.findChildrenOfType(copy, hostClass)
            .firstOrNull { it.textRange.startOffset == range.startOffset }
        if (replacement == null || replacement.textLength != newText.length) {
            throw IncorrectOperationException("New text would change the structure of ${file.name}")
        }
        return hostClass.cast(host.replace(replacement))
    }
}
