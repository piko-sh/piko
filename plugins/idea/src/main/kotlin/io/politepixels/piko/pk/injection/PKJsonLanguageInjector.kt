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
import io.politepixels.piko.pk.psi.impl.PkI18nContentElementImpl

/** Logger for the JSON language injector. */
private val LOG = Logger.getInstance(PKJsonLanguageInjector::class.java)

/**
 * Injects JSON language support into i18n blocks.
 *
 * Enables IntelliJ's JSON plugin to provide full code intelligence including
 * validation, formatting, and navigation for translation data inside i18n blocks.
 */
class PKJsonLanguageInjector : MultiHostInjector {

    /**
     * Returns the PSI element types that can host JSON injection.
     *
     * @return A list containing only the i18n content element class.
     */
    override fun elementsToInjectIn(): List<Class<out PsiElement>> {
        return listOf(PkI18nContentElementImpl::class.java)
    }

    /**
     * Performs JSON language injection into the given context element.
     *
     * @param registrar The registrar for adding injection places.
     * @param context The PSI element to inject JSON into.
     */
    override fun getLanguagesToInject(registrar: MultiHostRegistrar, context: PsiElement) {
        if (context !is PkI18nContentElementImpl) return

        val jsonLanguage = Language.findLanguageByID("JSON")
        if (jsonLanguage == null) {
            LOG.warn("JSON Language plugin not found. Cannot inject JSON code.")
            return
        }

        registrar.startInjecting(jsonLanguage)
        registrar.addPlace("", "", context, TextRange(0, context.textLength))
        registrar.doneInjecting()
    }
}
