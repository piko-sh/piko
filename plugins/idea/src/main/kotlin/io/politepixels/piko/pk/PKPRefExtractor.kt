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
import com.intellij.psi.PsiRecursiveElementVisitor
import com.intellij.psi.TokenType
import io.politepixels.piko.pk.psi.impl.PkJsScriptContentElementImpl

/**
 * Extracts `p-ref` directive names from PK template blocks.
 *
 * Walks the PSI tree of a `.pk` file to find all `p-ref="refName"` directives
 * in the template and generates a TypeScript declaration for the `pk` context
 * object. This enables autocompletion for `pk.refs.refName` and lifecycle
 * methods like `pk.onConnected()` in script blocks.
 */
object PKPRefExtractor {

    /** The directive name token text that identifies a p-ref directive. */
    private const val P_REF_DIRECTIVE = "p-ref"

    /** Matches names that can be used as bare TypeScript property keys. */
    private val IDENTIFIER_PATTERN = Regex("^[A-Za-z_$][A-Za-z0-9_$]*$")

    /** Namespace member declarations for the `pk` context object. */
    private val PK_NAMESPACE_MEMBERS = listOf(
        "/** Register a callback invoked when the element connects to the DOM. */ function onConnected(cb: () => void): void",
        "/** Register a callback invoked when the element disconnects from the DOM. */ function onDisconnected(cb: () => void): void",
        "/** Register a callback invoked before the template is rendered. */ function onBeforeRender(cb: () => void): void",
        "/** Register a callback invoked after the template is rendered. */ function onAfterRender(cb: () => void): void",
        "/** Register a callback invoked when the partial is updated. */ function onUpdated(cb: (context?: unknown) => void): void",
        "/** Register a cleanup function to run when the element is removed. */ function onCleanup(fn: () => void): void",
    )

    /** Interface member declarations for the PKC component instance (extends HTMLElement). */
    private val PKC_INTERFACE_MEMBERS = listOf(
        "/** Reactive state object for the component. */ state: Record<string, unknown> | undefined",
        "/** Merge partial state and schedule a re-render. */ setState(partialState: Record<string, unknown>): void",
        "/** Immediately re-render the component. */ render(): void",
        "/** Schedule a re-render on the next microtask. */ scheduleRender(): void",
        "/** Register a callback invoked when the component connects to the DOM. */ onConnected(cb: () => void): void",
        "/** Register a callback invoked when the component disconnects from the DOM. */ onDisconnected(cb: () => void): void",
        "/** Register a callback invoked before the component renders. */ onBeforeRender(cb: () => void): void",
        "/** Register a callback invoked after the component renders. */ onAfterRender(cb: () => void): void",
        "/** Register a callback invoked when observed attributes change. */ onUpdated(cb: (changedProperties: Set<string>) => void): void",
        "/** Register a cleanup function to run when the component is removed. */ onCleanup(cb: () => void): void",
        "/** Attaches a listener for slot content changes. The callback is invoked immediately with initial content. */ attachSlotListener(slotName: string, callback: (elements: Element[]) => void): void",
        "/** Returns elements assigned to a named slot. */ getSlottedElements(slotName?: string): Element[]",
        "/** Checks whether a slot has any assigned content. */ hasSlotContent(slotName?: string): boolean",
    )

    /**
     * Extracts p-ref names from the template block of the PK file containing
     * the given script content element.
     *
     * Walks the file's PSI tree looking for `DIRECTIVE_NAME` tokens with text
     * `"p-ref"`, then reads the raw attribute value between the quotes, so
     * names containing hyphens (`p-ref="submit-button"`) are kept whole.
     *
     * @param scriptContent The JS/TS script content element.
     * @return Ordered list of unique p-ref names, or empty list if none found.
     */
    @JvmStatic
    fun extractRefNames(scriptContent: PkJsScriptContentElementImpl): List<String> {
        val pkFile = scriptContent.containingFile ?: return emptyList()
        val refNames = mutableListOf<String>()
        val seen = mutableSetOf<String>()

        pkFile.accept(object : PsiRecursiveElementVisitor() {
            override fun visitElement(element: PsiElement) {
                val elementType = element.node?.elementType
                if (elementType == PKTokenTypes.DIRECTIVE_NAME && element.text == P_REF_DIRECTIVE) {
                    val refName = findRefValue(element)
                    if (refName != null && seen.add(refName)) {
                        refNames.add(refName)
                    }
                }
                super.visitElement(element)
            }
        })

        return refNames
    }

    /**
     * Generates a TypeScript `pk` namespace declaration from extracted ref names.
     *
     * Produces a `declare namespace pk` with a `refs` const containing each ref
     * name typed as `HTMLElement | null`, plus lifecycle function declarations.
     * Using `declare namespace` rather than `declare const` gives IntelliJ
     * proper autocomplete on `pk.` in injected TypeScript fragments.
     *
     * @param refNames The list of ref names to include.
     * @return The TypeScript declaration string.
     */
    @JvmStatic
    fun generatePKDeclaration(refNames: List<String>): String {
        val refsType = buildRefsType(refNames)
        val members = PK_NAMESPACE_MEMBERS.joinToString("; ")
        return "\ndeclare namespace pk { const refs: $refsType; $members; }\n"
    }

    /**
     * Generates a TypeScript declaration for the `pkc` component instance.
     *
     * Produces an interface extending HTMLElement with per-file refs and component
     * instance methods, then declares `pkc` as an instance of that interface.
     * This enables autocompletion for both custom members (`pkc.refs.refName`,
     * `pkc.setState()`) and inherited HTMLElement methods (`pkc.addEventListener()`,
     * `pkc.removeEventListener()`, etc.) in `.pkc` script blocks.
     *
     * @param refNames The list of ref names to include.
     * @return The TypeScript declaration string.
     */
    @JvmStatic
    fun generatePKCDeclaration(refNames: List<String>): String {
        val refsType = buildRefsType(refNames)
        val members = PKC_INTERFACE_MEMBERS.joinToString("; ")
        return "\ninterface _PikoComponent extends HTMLElement { readonly refs: $refsType; $members; } declare const pkc: _PikoComponent;\n"
    }

    /**
     * Builds the refs type object from extracted ref names.
     *
     * @param refNames The list of ref names to include.
     * @return The refs type string (e.g. `{ readonly foo: HTMLElement | null }` or `{}`).
     */
    private fun buildRefsType(refNames: List<String>): String {
        if (refNames.isEmpty()) return "{}"
        val fields = refNames.joinToString("; ") { "readonly ${propertyKey(it)}: HTMLElement | null" }
        return "{ $fields }"
    }

    /**
     * Formats a ref name as a TypeScript property key.
     *
     * @param name The ref name.
     * @return The bare name when it is a valid identifier, otherwise a quoted string.
     */
    private fun propertyKey(name: String): String {
        if (IDENTIFIER_PATTERN.matches(name)) return name
        val escaped = name.replace("\\", "\\\\").replace("\"", "\\\"")
        return "\"$escaped\""
    }

    /**
     * Starting from a `DIRECTIVE_NAME` element (`"p-ref"`), reads the raw text
     * of its quoted value.
     *
     * Token sequence: `DIRECTIVE_NAME` -> `HTML_ATTR_EQ` -> `HTML_ATTR_QUOTE`
     * -> value tokens -> `HTML_ATTR_QUOTE`. The value tokens are concatenated
     * whatever the lexer classified them as, because `p-ref` values are plain
     * names rather than expressions.
     *
     * @param directiveElement The DIRECTIVE_NAME PSI element for "p-ref".
     * @return The trimmed ref name, or null if the directive has no value.
     */
    private fun findRefValue(directiveElement: PsiElement): String? {
        val equals = nextNonWhitespaceSibling(directiveElement)
        if (equals?.node?.elementType != PKTokenTypes.HTML_ATTR_EQ) return null
        val openingQuote = nextNonWhitespaceSibling(equals)
        if (openingQuote?.node?.elementType != PKTokenTypes.HTML_ATTR_QUOTE) return null

        val value = StringBuilder()
        var current = openingQuote.nextSibling
        while (current != null) {
            val type = current.node?.elementType
            if (type == PKTokenTypes.HTML_ATTR_QUOTE || type in PKTreeUtils.TAG_CLOSE_TYPES) break
            value.append(current.text)
            current = current.nextSibling
        }
        return value.toString().trim().ifEmpty { null }
    }

    /**
     * Returns the next sibling that is not whitespace.
     *
     * @param element The element to start after.
     * @return The next non-whitespace sibling, or null.
     */
    private fun nextNonWhitespaceSibling(element: PsiElement): PsiElement? {
        var sibling = element.nextSibling
        while (sibling != null && sibling.node?.elementType == TokenType.WHITE_SPACE) {
            sibling = sibling.nextSibling
        }
        return sibling
    }
}
