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

import com.intellij.codeInsight.editorActions.SimpleTokenSetQuoteHandler
import com.intellij.psi.tree.TokenSet

/**
 * Provides quote auto-pairing for PK template files.
 *
 * Handles auto-pairing for double quotes, single quotes, and backticks
 * in attribute values and expressions.
 */
class PKQuoteHandler : SimpleTokenSetQuoteHandler(QUOTE_TOKENS) {

    companion object {
        /** Token set containing all quote-related token types for auto-pairing. */
        private val QUOTE_TOKENS = TokenSet.create(
            PKTokenTypes.HTML_ATTR_QUOTE,
            PKTokenTypes.EXPR_STRING_QUOTE,
            PKTokenTypes.HTML_ATTR_VALUE,
            PKTokenTypes.EXPR_STRING,
            PKTokenTypes.ATTR_VALUE
        )
    }
}
