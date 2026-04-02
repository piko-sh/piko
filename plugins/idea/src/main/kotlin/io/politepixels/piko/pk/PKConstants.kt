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

/**
 * Constants used throughout the PK plugin.
 *
 * Centralises magic strings and literal values to ensure consistency
 * and make them easier to maintain.
 */
object PKConstants {

    /**
     * Script language attribute values.
     */
    object ScriptLang {
        /** JavaScript shorthand lang value. */
        const val JS = "js"

        /** JavaScript full lang value. */
        const val JAVASCRIPT = "javascript"

        /** TypeScript lang value. */
        const val TYPESCRIPT = "typescript"

        /** All JavaScript-like language values. */
        val JS_LANGUAGES = setOf(JS, JAVASCRIPT, TYPESCRIPT)

        /**
         * Checks if a value represents a JavaScript-like language.
         *
         * @param value The lang attribute value.
         * @return true if the value is js, javascript, or typescript.
         */
        @JvmStatic
        fun isJsLanguage(value: String): Boolean = value in JS_LANGUAGES
    }

    /**
     * Display names for languages.
     */
    object LanguageDisplay {
        /** Go language display name. */
        const val GO = "Go"

        /** JavaScript language display name. */
        const val JAVASCRIPT = "JavaScript"

        /** TypeScript language display name. */
        const val TYPESCRIPT = "TypeScript"

        /** CSS language display name. */
        const val CSS = "CSS"

        /** JSON language display name. */
        const val JSON = "JSON"
    }

    /**
     * Placeholder text for folded regions.
     */
    object FoldingPlaceholder {
        /** Placeholder for template blocks. */
        const val TEMPLATE = "<template>..."

        /** Placeholder for script blocks. */
        const val SCRIPT = "<script>..."

        /** Placeholder for style blocks. */
        const val STYLE = "<style>..."

        /** Placeholder for i18n blocks. */
        const val I18N = "<i18n>..."

        /** Placeholder for HTML comments. */
        const val COMMENT = "<!--...-->"

        /** Placeholder for interpolation expressions. */
        const val INTERPOLATION = "{{...}}"

        /** Placeholder for generic HTML tags. */
        const val TAG = "<...>"

        /** Default placeholder. */
        const val DEFAULT = "..."

        /**
         * Returns the placeholder for a script block with a lang attribute.
         *
         * @param lang The language value.
         * @return The formatted script placeholder.
         */
        @JvmStatic
        fun scriptWithLang(lang: String): String = "<script lang=\"$lang\">..."

        /**
         * Returns the placeholder for an HTML tag.
         *
         * @param tagName The tag name.
         * @return The formatted tag placeholder.
         */
        @JvmStatic
        fun tagWithName(tagName: String): String = "<$tagName>..."
    }

    /**
     * Block tag names.
     */
    object BlockTags {
        /** Template block tag name. */
        const val TEMPLATE = "template"

        /** Script block tag name. */
        const val SCRIPT = "script"

        /** Style block tag name. */
        const val STYLE = "style"

        /** I18n block tag name. */
        const val I18N = "i18n"
    }

    /**
     * Structure view display names.
     */
    object StructureDisplay {
        /** Template block display. */
        const val TEMPLATE_TAG = "<template>"

        /** Script block display. */
        const val SCRIPT_TAG = "<script>"

        /** Style block display. */
        const val STYLE_TAG = "<style>"

        /** I18n block display. */
        const val I18N_TAG = "<i18n>"

        /** Template body content display. */
        const val CONTENT = "content"

        /** Go code display. */
        const val GO_CODE = "Go code"

        /** JavaScript code display. */
        const val JS_CODE = "JavaScript code"

        /** CSS rules display. */
        const val CSS_RULES = "CSS rules"

        /** Translations display. */
        const val TRANSLATIONS = "translations"

        /**
         * Returns the script tag display with lang attribute.
         *
         * @param lang The language value.
         * @return The formatted script tag display.
         */
        @JvmStatic
        fun scriptTagWithLang(lang: String): String = "<script lang=\"$lang\">"
    }

    /**
     * Breadcrumb display text.
     */
    object BreadcrumbDisplay {
        /** Template breadcrumb. */
        const val TEMPLATE = "template"

        /** Script breadcrumb. */
        const val SCRIPT = "script"

        /** Style breadcrumb. */
        const val STYLE = "style"

        /** I18n breadcrumb. */
        const val I18N = "i18n"

        /** Content breadcrumb. */
        const val CONTENT = "content"

        /** Script (Go) breadcrumb. */
        const val SCRIPT_GO = "script (Go)"

        /** Script (JS) breadcrumb. */
        const val SCRIPT_JS = "script (JS)"

        /** Script (TS) breadcrumb. */
        const val SCRIPT_TS = "script (TS)"
    }

    /**
     * Comment formats.
     */
    object Comments {
        /** HTML comment start. */
        const val HTML_START = "<!--"

        /** HTML comment end. */
        const val HTML_END = "-->"

        /** Expression comment start. */
        const val EXPR_START = "/*"

        /** Expression comment end. */
        const val EXPR_END = "*/"
    }
}
