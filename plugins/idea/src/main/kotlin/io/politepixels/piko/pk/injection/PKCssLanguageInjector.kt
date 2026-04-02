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
import io.politepixels.piko.pk.psi.impl.PkCssStyleContentElementImpl

/** Logger for the CSS language injector. */
private val LOG = Logger.getInstance(PKCssLanguageInjector::class.java)

/**
 * Injects CSS language support into style blocks.
 *
 * Enables IntelliJ's CSS plugin to provide full code intelligence including
 * autocompletion, validation, and refactoring for styles inside style blocks.
 */
class PKCssLanguageInjector : MultiHostInjector {

    /**
     * Returns the PSI element types that can host CSS injection.
     *
     * @return A list containing only the CSS style content element class.
     */
    override fun elementsToInjectIn(): List<Class<out PsiElement>> {
        return listOf(PkCssStyleContentElementImpl::class.java)
    }

    /**
     * Performs CSS language injection into the given context element.
     *
     * @param registrar The registrar for adding injection places.
     * @param context The PSI element to inject CSS into.
     */
    override fun getLanguagesToInject(registrar: MultiHostRegistrar, context: PsiElement) {
        if (context !is PkCssStyleContentElementImpl) return

        val cssLanguage = Language.findLanguageByID("CSS")
        if (cssLanguage == null) {
            LOG.warn("CSS Language plugin not found. Cannot inject CSS code.")
            return
        }

        registrar.startInjecting(cssLanguage)
        registrar.addPlace("", "", context, TextRange(0, context.textLength))
        registrar.doneInjecting()
    }
}
