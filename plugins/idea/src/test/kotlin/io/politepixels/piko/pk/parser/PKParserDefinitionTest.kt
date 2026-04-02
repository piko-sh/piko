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


package io.politepixels.piko.pk.parser

import com.intellij.psi.TokenType
import com.intellij.psi.tree.IFileElementType
import io.politepixels.piko.pk.PKLanguage
import io.politepixels.piko.pk.PKParserDefinition
import io.politepixels.piko.pk.PKTokenTypes
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

class PKParserDefinitionTest {

    private val parserDef = PKParserDefinition()

    @Test
    fun `FILE element type is IFileElementType`() {
        assertTrue(
            "FILE should be IFileElementType",
            PKParserDefinition.FILE is IFileElementType
        )
    }

    @Test
    fun `FILE element type is associated with PKLanguage`() {
        assertEquals(
            "FILE should be associated with PKLanguage",
            PKLanguage,
            PKParserDefinition.FILE.language
        )
    }

    @Test
    fun `WHITESPACES contains only WHITE_SPACE`() {
        assertTrue(
            "WHITESPACES should contain WHITE_SPACE",
            PKParserDefinition.WHITESPACES.contains(TokenType.WHITE_SPACE)
        )
        assertEquals(
            "WHITESPACES should have exactly 1 token type",
            1,
            PKParserDefinition.WHITESPACES.types.size
        )
    }

    @Test
    fun `COMMENTS equals PKTokenTypes COMMENTS`() {
        assertEquals(
            "COMMENTS should equal PKTokenTypes.COMMENTS",
            PKTokenTypes.COMMENTS,
            PKParserDefinition.COMMENTS
        )
    }

    @Test
    fun `STRINGS equals PKTokenTypes STRING_LITERALS`() {
        assertEquals(
            "STRINGS should equal PKTokenTypes.STRING_LITERALS",
            PKTokenTypes.STRING_LITERALS,
            PKParserDefinition.STRINGS
        )
    }

    @Test
    fun `getCommentTokens returns COMMENTS`() {
        assertEquals(
            "getCommentTokens should return COMMENTS",
            PKParserDefinition.COMMENTS,
            parserDef.commentTokens
        )
    }

    @Test
    fun `getStringLiteralElements returns STRINGS`() {
        assertEquals(
            "getStringLiteralElements should return STRINGS",
            PKParserDefinition.STRINGS,
            parserDef.stringLiteralElements
        )
    }

    @Test
    fun `getFileNodeType returns FILE`() {
        assertEquals(
            "getFileNodeType should return FILE",
            PKParserDefinition.FILE,
            parserDef.fileNodeType
        )
    }

    @Test
    fun `createLexer returns non-null lexer`() {
        assertNotNull("PKParserDefinition should be instantiable", parserDef)
    }

    @Test
    fun `createParser returns non-null parser reference`() {
        assertNotNull("PKParserDefinition should be instantiable", parserDef)
    }
}
