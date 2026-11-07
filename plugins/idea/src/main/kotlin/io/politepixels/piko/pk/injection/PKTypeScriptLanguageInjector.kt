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
import com.intellij.openapi.progress.ProcessCanceledException
import com.intellij.openapi.util.TextRange
import com.intellij.psi.PsiElement
import io.politepixels.piko.pk.PKPRefExtractor
import io.politepixels.piko.pk.psi.impl.PkJsScriptContentElementImpl

/** Logger for the TypeScript language injector. */
private val LOG = Logger.getInstance(PKTypeScriptLanguageInjector::class.java)

/**
 * Injects TypeScript language support into JavaScript script blocks.
 *
 * Used for script blocks with `lang="js"` or `lang="ts"`. Falls back to
 * ECMAScript 6 or JavaScript if TypeScript is not available. Enables
 * IntelliJ's JavaScript plugin to provide code intelligence.
 *
 * The `piko` namespace and action declarations come from `piko-ide.d.ts`
 * and `piko-actions.d.ts`, which [io.politepixels.piko.pk.js.PikoTypeDefinitionsLibraryProvider]
 * exposes as a library, so they are indexed once rather than re-parsed into
 * every fragment. Only the small per-file `pk`/`pkc` declaration, which
 * depends on the file's `p-ref` names, is injected as a suffix.
 */
class PKTypeScriptLanguageInjector : MultiHostInjector {

    companion object {
        /**
         * TypeScript lib reference directives to enable modern JavaScript features.
         * - esnext: Enables Set, Map, Promise, async/await, etc.
         * - dom: Enables DOM APIs (document, window, HTMLElement, etc.)
         */
        private const val LIB_REFERENCES = """/// <reference lib="esnext" />
/// <reference lib="dom" />
"""
    }

    /**
     * Returns the PSI element types that can host TypeScript injection.
     *
     * @return A list containing only the JS script content element class.
     */
    override fun elementsToInjectIn(): List<Class<out PsiElement>> {
        return listOf(PkJsScriptContentElementImpl::class.java)
    }

    /**
     * Performs TypeScript or JavaScript language injection into the given context.
     *
     * Tries TypeScript first, then ECMAScript 6, then JavaScript as fallbacks.
     * Lib reference directives go in the prefix (must precede code for the
     * TypeScript compiler to process them). The per-file context declaration
     * goes in the suffix so that user code element offsets stay small.
     *
     * @param registrar The registrar for adding injection places.
     * @param context The PSI element to inject TypeScript into.
     */
    override fun getLanguagesToInject(registrar: MultiHostRegistrar, context: PsiElement) {
        if (context !is PkJsScriptContentElementImpl) return

        val language = Language.findLanguageByID("TypeScript")
            ?: Language.findLanguageByID("ECMAScript 6")
            ?: Language.findLanguageByID("JavaScript")

        if (language == null) {
            LOG.warn("TypeScript/JavaScript Language plugin not found. Cannot inject JS code.")
            return
        }

        val refsDeclaration = getRefsDeclaration(context)

        registrar.startInjecting(language)
        registrar.addPlace(LIB_REFERENCES, refsDeclaration, context, TextRange(0, context.textLength))
        registrar.doneInjecting()
    }

    /**
     * Gets a per-file context declaration by extracting p-ref names from the
     * template block.
     *
     * For `.pkc` files, generates a `declare const pkc: HTMLElement & { ... }`
     * declaration with component instance methods. For `.pk` files, generates
     * a `declare const pk: { ... }` declaration with lifecycle methods.
     *
     * @param context The PSI element context to extract refs from.
     * @return The context type declaration, or empty string if extraction fails.
     */
    private fun getRefsDeclaration(context: PsiElement): String {
        if (context !is PkJsScriptContentElementImpl) return ""
        return try {
            val refNames = PKPRefExtractor.extractRefNames(context)
            val fileName = context.containingFile?.name ?: ""
            if (fileName.endsWith(".pkc")) {
                PKPRefExtractor.generatePKCDeclaration(refNames)
            } else {
                PKPRefExtractor.generatePKDeclaration(refNames)
            }
        } catch (e: ProcessCanceledException) {
            throw e
        } catch (e: Exception) {
            LOG.warn("Failed to extract p-ref names: ${e.message}", e)
            ""
        }
    }
}
