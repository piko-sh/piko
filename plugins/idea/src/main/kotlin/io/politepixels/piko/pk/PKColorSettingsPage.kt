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

import com.intellij.openapi.editor.colors.TextAttributesKey
import com.intellij.openapi.fileTypes.SyntaxHighlighter
import com.intellij.openapi.options.colors.AttributesDescriptor
import com.intellij.openapi.options.colors.ColorDescriptor
import com.intellij.openapi.options.colors.ColorSettingsPage
import javax.swing.Icon

/**
 * Colour settings page for Piko syntax highlighting.
 *
 * Allows users to customise colours in Settings > Editor > Color Scheme > Piko.
 * Displays a demo template showing all highlighting categories.
 */
class PKColorSettingsPage : ColorSettingsPage {

    companion object {
        /** Colour categories shown in the settings page. */
        private val DESCRIPTORS = arrayOf(
            AttributesDescriptor("Structural tags//Template, script, style tags", PKSyntaxHighlighter.PK_STRUCTURAL_TAG),
            AttributesDescriptor("Structural tags//Tag brackets", PKSyntaxHighlighter.PK_TAG_BRACKET),
            AttributesDescriptor("Piko components//Component tags (piko:*)", PKSyntaxHighlighter.PK_PIKO_COMPONENT),
            AttributesDescriptor("HTML//Tag name", PKSyntaxHighlighter.PK_HTML_TAG_NAME),
            AttributesDescriptor("HTML//Tag brackets", PKSyntaxHighlighter.PK_HTML_TAG_BRACKET),
            AttributesDescriptor("Attributes//Attribute name", PKSyntaxHighlighter.PK_ATTRIBUTE),
            AttributesDescriptor("Attributes//Attribute value", PKSyntaxHighlighter.PK_ATTRIBUTE_VALUE),
            AttributesDescriptor("Attributes//Attribute quotes", PKSyntaxHighlighter.PK_ATTRIBUTE_QUOTE),
            AttributesDescriptor("Directives//Directive name (p-if, p-for)", PKSyntaxHighlighter.PK_DIRECTIVE),
            AttributesDescriptor("Directives//Bind shorthand (:attr)", PKSyntaxHighlighter.PK_DIRECTIVE_BIND),
            AttributesDescriptor("Directives//Event shorthand (@event)", PKSyntaxHighlighter.PK_DIRECTIVE_EVENT),
            AttributesDescriptor("Interpolation//Brackets {{ }}", PKSyntaxHighlighter.PK_INTERPOLATION_BRACKET),
            AttributesDescriptor("Expressions//Boolean literals", PKSyntaxHighlighter.PK_EXPR_BOOLEAN),
            AttributesDescriptor("Expressions//Number literals", PKSyntaxHighlighter.PK_EXPR_NUMBER),
            AttributesDescriptor("Expressions//String literals", PKSyntaxHighlighter.PK_EXPR_STRING),
            AttributesDescriptor("Expressions//String quotes", PKSyntaxHighlighter.PK_EXPR_STRING_QUOTE),
            AttributesDescriptor("Expressions//Escape sequences", PKSyntaxHighlighter.PK_EXPR_ESCAPE),
            AttributesDescriptor("Expressions//Operators", PKSyntaxHighlighter.PK_EXPR_OPERATOR),
            AttributesDescriptor("Expressions//Dot accessor", PKSyntaxHighlighter.PK_EXPR_DOT),
            AttributesDescriptor("Expressions//Parentheses", PKSyntaxHighlighter.PK_EXPR_PAREN),
            AttributesDescriptor("Expressions//Brackets", PKSyntaxHighlighter.PK_EXPR_BRACKET),
            AttributesDescriptor("Expressions//Braces", PKSyntaxHighlighter.PK_EXPR_BRACE),
            AttributesDescriptor("Expressions//Comma", PKSyntaxHighlighter.PK_EXPR_COMMA),
            AttributesDescriptor("Expressions//Colon", PKSyntaxHighlighter.PK_EXPR_COLON),
            AttributesDescriptor("Expressions//Built-in functions", PKSyntaxHighlighter.PK_EXPR_BUILTIN),
            AttributesDescriptor("Expressions//Context variables (props, state)", PKSyntaxHighlighter.PK_EXPR_CONTEXT_VAR),
            AttributesDescriptor("Expressions//Function calls", PKSyntaxHighlighter.PK_EXPR_FUNCTION),
            AttributesDescriptor("Expressions//Identifiers", PKSyntaxHighlighter.PK_EXPR_IDENTIFIER),
            AttributesDescriptor("Template literals//Interpolation \${}", PKSyntaxHighlighter.PK_TEMPLATE_INTERP),
            AttributesDescriptor("Text content", PKSyntaxHighlighter.PK_TEXT),
            AttributesDescriptor("Comments", PKSyntaxHighlighter.PK_COMMENT),
        )

        /** Custom tags used in the demo text to show specific highlighting. */
        private val ADDITIONAL_TAGS = mapOf(
            "piko" to PKSyntaxHighlighter.PK_PIKO_COMPONENT,
            "directive" to PKSyntaxHighlighter.PK_DIRECTIVE,
            "bind" to PKSyntaxHighlighter.PK_DIRECTIVE_BIND,
            "event" to PKSyntaxHighlighter.PK_DIRECTIVE_EVENT,
            "interp" to PKSyntaxHighlighter.PK_INTERPOLATION_BRACKET,
            "ctx" to PKSyntaxHighlighter.PK_EXPR_CONTEXT_VAR,
            "func" to PKSyntaxHighlighter.PK_EXPR_FUNCTION,
            "bool" to PKSyntaxHighlighter.PK_EXPR_BOOLEAN,
            "num" to PKSyntaxHighlighter.PK_EXPR_NUMBER,
            "str" to PKSyntaxHighlighter.PK_EXPR_STRING,
            "op" to PKSyntaxHighlighter.PK_EXPR_OPERATOR,
        )
    }

    /**
     * Returns the list of attribute descriptors for the settings page.
     *
     * @return An array of descriptors defining customisable colour categories.
     */
    override fun getAttributeDescriptors(): Array<AttributesDescriptor> = DESCRIPTORS

    /**
     * Returns colour descriptors for background and foreground settings.
     *
     * @return An empty array as no custom colour descriptors are defined.
     */
    override fun getColorDescriptors(): Array<ColorDescriptor> = ColorDescriptor.EMPTY_ARRAY

    /**
     * Returns the display name shown in the settings tree.
     *
     * @return The string "Piko".
     */
    override fun getDisplayName(): String = "Piko"

    /**
     * Returns the icon shown in the settings tree.
     *
     * @return The Piko file type icon.
     */
    override fun getIcon(): Icon? = PKFileType.icon

    /**
     * Returns the syntax highlighter used for the demo text.
     *
     * @return A new PKSyntaxHighlighter instance.
     */
    override fun getHighlighter(): SyntaxHighlighter = PKSyntaxHighlighter()

    /**
     * Returns the demo text displayed in the colour settings preview.
     *
     * @return A sample template demonstrating all highlighting categories.
     */
    override fun getDemoText(): String = """
<template>
  <!-- This is a comment -->
  <<piko>piko:card</piko> <bind>:title</bind>="<ctx>props</ctx>.Title">
    <div class="container" <directive>p-if</directive>="<ctx>props</ctx>.Visible">
      <h1><interp>{{</interp> <ctx>props</ctx>.<func>Heading</func>() <interp>}}</interp></h1>
      <p <directive>p-text</directive>="<ctx>state</ctx>.Message"></p>
      <button <event>@click</event>="<func>handleClick</func>">
        <interp>{{</interp> <ctx>state</ctx>.Count <op>></op> <num>0</num> <op>?</op> <ctx>state</ctx>.Count <op>:</op> <str>'None'</str> <interp>}}</interp>
      </button>
      <span <directive>p-for</directive>="item in <ctx>props</ctx>.Items">
        <interp>{{</interp> item.Name <interp>}}</interp>
      </span>
    </div>
  </<piko>piko:card</piko>>
</template>

<script>
// Go code here (injected)
</script>

<style>
/* CSS code here (injected) */
</style>
    """.trimIndent()

    /**
     * Returns additional highlighting tags used in the demo text.
     *
     * @return A map of tag names to their corresponding text attribute keys.
     */
    override fun getAdditionalHighlightingTagToDescriptorMap(): Map<String, TextAttributesKey> = ADDITIONAL_TAGS
}
