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

package io.politepixels.piko.pk.platform

import com.goide.psi.impl.imports.GoImportResolver
import com.intellij.lang.injection.InjectedLanguageManager
import com.intellij.lang.injection.MultiHostInjector
import com.intellij.openapi.actionSystem.ActionManager
import com.intellij.openapi.application.PathManager
import com.intellij.openapi.command.WriteCommandAction
import com.intellij.openapi.roots.AdditionalLibraryRootsProvider
import com.intellij.psi.ElementManipulators
import com.intellij.psi.PsiElement
import com.intellij.psi.PsiLanguageInjectionHost
import com.intellij.psi.util.PsiTreeUtil
import com.intellij.testFramework.fixtures.BasePlatformTestCase
import io.politepixels.piko.pk.PKTokenTypes
import io.politepixels.piko.pk.injection.PKCssLanguageInjector
import io.politepixels.piko.pk.injection.PKGoLanguageInjector
import io.politepixels.piko.pk.injection.PKJsonLanguageInjector
import io.politepixels.piko.pk.injection.PKTypeScriptLanguageInjector
import io.politepixels.piko.pk.injection.PkGoImportResolver
import io.politepixels.piko.pk.js.PikoTypeDefinitionsLibraryProvider
import io.politepixels.piko.pk.lsp.PikoLanguageServerFactory
import io.politepixels.piko.pk.lsp.PikoLspArguments
import io.politepixels.piko.pk.lsp.PikoLspController
import io.politepixels.piko.pk.lsp.PikoLspInitialisationOptions
import io.politepixels.piko.pk.lsp.PikoLspLaunchOptions
import io.politepixels.piko.pk.lsp.PikoLspLogFile
import io.politepixels.piko.pk.lsp.PikoLspTransport
import io.politepixels.piko.pk.psi.impl.PkCssStyleContentElementImpl
import io.politepixels.piko.pk.psi.impl.PkGoScriptContentElementImpl
import io.politepixels.piko.pk.psi.impl.PkI18nContentElementImpl
import io.politepixels.piko.pk.psi.impl.PkJsScriptContentElementImpl
import io.politepixels.piko.pk.psi.impl.PkLeafPsiElement
import io.politepixels.piko.settings.PikoSettings

/**
 * Platform tests that load the real plugin descriptors into a light IDE.
 */
class PikoPluginPlatformTest : BasePlatformTestCase() {

    fun testOptionalDescriptorsAreLoaded() {
        assertNotNull(
            "piko-lsp4ij.xml should register the LSP controller",
            PikoLspController.getInstance()
        )
        val actionManager = ActionManager.getInstance()
        assertNotNull("piko-lsp4ij.xml should register the restart action", actionManager.getAction("Piko.RestartLsp"))
        assertNotNull(actionManager.getAction("Piko.ShowLspOutput"))
        assertNotNull(actionManager.getAction("Piko.OpenLspLogs"))
        assertNotNull(actionManager.getAction("Piko.NewFile"))

        val injectors = MultiHostInjector.MULTIHOST_INJECTOR_EP_NAME.getExtensions(project)
        assertTrue("piko-go.xml", injectors.any { it is PKGoLanguageInjector })
        assertTrue("piko-javascript.xml", injectors.any { it is PKTypeScriptLanguageInjector })
        assertTrue("piko-css.xml", injectors.any { it is PKCssLanguageInjector })
        assertTrue("plugin.xml", injectors.any { it is PKJsonLanguageInjector })

        assertTrue(GoImportResolver.EP_NAME.extensionList.any { it is PkGoImportResolver })
        assertTrue(AdditionalLibraryRootsProvider.EP_NAME.extensionList.any { it is PikoTypeDefinitionsLibraryProvider })
    }

    fun testEachBlockIsInjected() {
        val file = myFixture.configureByText(
            "page.pk",
            """
            <template><p>{{ state.Title }}</p></template>

            <script type="application/x-go">
            package main

            func Render() {}
            </script>

            <script lang="ts">
            const greeting: string = "hello";
            </script>

            <style>
            .title { color: red; }
            </style>

            <i18n lang="json">
            { "en": { "title": "Title" } }
            </i18n>
            """.trimIndent()
        )

        assertEquals("go", injectedLanguage(PsiTreeUtil.findChildOfType(file, PkGoScriptContentElementImpl::class.java)))
        assertTrue(
            injectedLanguage(PsiTreeUtil.findChildOfType(file, PkJsScriptContentElementImpl::class.java))
                in setOf("TypeScript", "ECMAScript 6", "JavaScript")
        )
        assertEquals("CSS", injectedLanguage(PsiTreeUtil.findChildOfType(file, PkCssStyleContentElementImpl::class.java)))
        assertEquals("JSON", injectedLanguage(PsiTreeUtil.findChildOfType(file, PkI18nContentElementImpl::class.java)))
    }

    fun testUnknownScriptLangIsNotInjected() {
        val file = myFixture.configureByText("page.pk", "<script lang=\"python\">print('x')</script>")
        assertNull(PsiTreeUtil.findChildOfType(file, PkGoScriptContentElementImpl::class.java))
        assertNull(PsiTreeUtil.findChildOfType(file, PkJsScriptContentElementImpl::class.java))
    }

    fun testNestedTemplateParsesAsOneBlock() {
        val file = myFixture.configureByText(
            "page.pk",
            """
            <template>
              <ul>
                <template p-for="item in state.Items"><li>{{ item }}</li></template>
              </ul>
            </template>

            <script type="application/x-go">
            package main
            </script>
            """.trimIndent()
        )

        val templateBlocks = file.node.getChildren(null).count { it.elementType == PKTokenTypes.TEMPLATE_BLOCK_ELEMENT }
        assertEquals(1, templateBlocks)
        assertNotNull(
            "The script after a nested template should still be a Go block",
            PsiTreeUtil.findChildOfType(file, PkGoScriptContentElementImpl::class.java)
        )
    }

    fun testLeafManipulatorRenamesTemplateIdentifier() {
        val file = myFixture.configureByText("page.pk", "<template>{{ formatDate(x) }}</template>")
        val leaf = PsiTreeUtil.findChildrenOfType(file, PkLeafPsiElement::class.java)
            .first { it.elementType == PKTokenTypes.EXPR_FUNCTION_NAME }

        WriteCommandAction.runWriteCommandAction(project) {
            ElementManipulators.handleContentChange(leaf, "formatTime")
        }
        assertEquals("<template>{{ formatTime(x) }}</template>", file.text)
    }

    fun testInjectionHostUpdateText() {
        val file = myFixture.configureByText("page.pk", "<style>.a { color: red; }</style>")
        val host = PsiTreeUtil.findChildOfType(file, PkCssStyleContentElementImpl::class.java)!!

        WriteCommandAction.runWriteCommandAction(project) {
            host.updateText(".b { color: blue; }")
        }
        assertEquals("<style>.b { color: blue; }</style>", file.text)
    }

    fun testLspDescriptorCommandLineAndEnablement() {
        val settings = PikoSettings.getInstance()
        val factory = PikoLanguageServerFactory()
        val originalEnabled = settings.lspEnabled
        try {
            factory.setEnabled(false, project)
            assertFalse(factory.isEnabled(project))
            factory.setEnabled(true, project)
            assertTrue(factory.isEnabled(project))
        } finally {
            settings.lspEnabled = originalEnabled
        }

        assertFalse("Stdio is the default transport", settings.useTcpTransport)
        assertFalse("File logging is off by default", settings.fileLoggingEnabled)

        val logFile = PikoLspLogFile.forProject(project)
        assertTrue(logFile.startsWith(PathManager.getLogPath()))

        val arguments = PikoLspArguments.build(
            PikoLspLaunchOptions(formatting = false, logFile = logFile.toString(), pprof = false, pprofPort = 0),
            PikoLspTransport.Stdio
        )
        assertEquals(listOf("--file-logging", "--log-file=$logFile"), arguments)

        assertFalse(
            "The Go plugin is present in tests, so the gopls bridge stays off",
            PikoLspInitialisationOptions.build().get("goBridge").asBoolean
        )
    }

    private fun injectedLanguage(host: PsiLanguageInjectionHost?): String? {
        assertNotNull(host)
        val injected: List<com.intellij.openapi.util.Pair<PsiElement, com.intellij.openapi.util.TextRange>>? =
            InjectedLanguageManager.getInstance(project).getInjectedPsiFiles(host!!)
        return injected?.firstOrNull()?.first?.language?.id
    }
}
