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


package io.politepixels.piko.pk.css

import com.intellij.codeInsight.daemon.ImplicitUsageProvider
import com.intellij.lang.injection.InjectedLanguageManager
import com.intellij.psi.PsiElement
import com.intellij.psi.css.CssClass
import com.intellij.psi.css.CssIdSelector
import io.politepixels.piko.pk.psi.impl.PkCssStyleContentElementImpl

/**
 * Suppresses "unused CSS selector" warnings for selectors inside PK style blocks.
 *
 * IntelliJ's CSS plugin marks selectors as unused when it cannot find references
 * in the same file. Since PK template bodies are flat token elements without
 * composite HTML PSI nodes, the CSS plugin cannot detect class references in
 * templates. This provider blanket-suppresses unused warnings for all CSS
 * selectors within PK files.
 *
 * Blanket suppression is necessary because dynamic class bindings (`p-class`,
 * `:class` with runtime expressions) make it impossible to statically determine
 * all used classes.
 */
class PKCssImplicitUsageProvider : ImplicitUsageProvider {

    /**
     * Checks whether the given element should be considered implicitly used.
     *
     * Returns true for CSS class and ID selectors that live inside a PK
     * file's style block, suppressing "unused" inspections.
     *
     * @param element The PSI element to check.
     * @return True if the element is a CSS class or ID inside a PK style block.
     */
    override fun isImplicitUsage(element: PsiElement): Boolean {
        if (element !is CssClass && element !is CssIdSelector) return false
        return isInsidePkStyleBlock(element)
    }

    /**
     * Checks whether the given element should be considered implicitly read.
     *
     * @param element The PSI element to check.
     * @return Always false; read detection is not needed for CSS selectors.
     */
    override fun isImplicitRead(element: PsiElement): Boolean = false

    /**
     * Checks whether the given element should be considered implicitly written.
     *
     * @param element The PSI element to check.
     * @return Always false; write detection is not needed for CSS selectors.
     */
    override fun isImplicitWrite(element: PsiElement): Boolean = false

    /**
     * Determines whether a CSS element resides inside a PK style block.
     *
     * CSS elements in PK files live in injected PSI files. This method
     * retrieves the injection host and checks if it is a PK CSS content element.
     *
     * @param element The CSS PSI element to check.
     * @return True if the element's injection host is a PkCssStyleContentElementImpl.
     */
    private fun isInsidePkStyleBlock(element: PsiElement): Boolean {
        val project = element.project
        val injectionManager = InjectedLanguageManager.getInstance(project)
        val host = injectionManager.getInjectionHost(element)
        return host is PkCssStyleContentElementImpl
    }
}
