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


package io.politepixels.piko.pk.injection

import com.intellij.lang.Language
import com.intellij.lang.injection.MultiHostInjector
import com.intellij.lang.injection.MultiHostRegistrar
import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.util.TextRange
import com.intellij.psi.PsiElement
import io.politepixels.piko.pk.psi.impl.PkGoScriptContentElementImpl

/** Logger for the Go language injector. */
private val LOG = Logger.getInstance(PKGoLanguageInjector::class.java)

/**
 * Injects Go language support into script blocks.
 *
 * Enables IntelliJ's Go plugin to provide full code intelligence including
 * autocompletion, type checking, and navigation for Go code inside script blocks.
 * Automatically prepends a package declaration if one is not present.
 */
class PKGoLanguageInjector : MultiHostInjector {

    companion object {
        /** The prefix added when no package declaration is present. */
        const val DEFAULT_PACKAGE_PREFIX = "package main\n\n"

        /**
         * Checks if the content starts with a package declaration.
         *
         * @param content The Go script content to check.
         * @return True if content starts with "package " after trimming.
         */
        fun hasPackageDeclaration(content: String): Boolean =
            content.trim().startsWith("package ")

        /**
         * Computes the prefix to prepend for Go injection.
         *
         * @param content The Go script content.
         * @return "package main\n\n" if no package declaration, null otherwise.
         */
        fun computePrefix(content: String): String? =
            if (hasPackageDeclaration(content)) null else DEFAULT_PACKAGE_PREFIX
    }

    /**
     * Returns the PSI element types that can host Go injection.
     *
     * @return A list containing only the Go script content element class.
     */
    override fun elementsToInjectIn(): List<Class<out PsiElement>> {
        return listOf(PkGoScriptContentElementImpl::class.java)
    }

    /**
     * Performs Go language injection into the given context element.
     *
     * If the script does not start with a package declaration, this method
     * prepends "package main" to enable proper Go language features.
     *
     * @param registrar The registrar for adding injection places.
     * @param context The PSI element to inject Go into.
     */
    override fun getLanguagesToInject(registrar: MultiHostRegistrar, context: PsiElement) {
        if (context !is PkGoScriptContentElementImpl) return

        val goLanguage = Language.findLanguageByID("go")
        if (goLanguage == null) {
            LOG.warn("Go Language plugin not found. Cannot inject Go code.")
            return
        }

        val prefix = computePrefix(context.text)

        registrar.startInjecting(goLanguage)
        registrar.addPlace(
            prefix,
            null,
            context,
            TextRange.create(0, context.textLength)
        )
        registrar.doneInjecting()
    }
}
