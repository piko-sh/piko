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

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PKPRefExtractorTest {

    @Test
    fun `generatePKDeclaration declares pk with empty refs for empty list`() {
        val result = PKPRefExtractor.generatePKDeclaration(emptyList())
        assertTrue(
            "Should contain declare namespace pk",
            result.contains("declare namespace pk")
        )
        assertTrue(
            "Should contain empty refs",
            result.contains("const refs: {}")
        )
    }

    @Test
    fun `generatePKDeclaration includes lifecycle methods for empty list`() {
        val result = PKPRefExtractor.generatePKDeclaration(emptyList())
        assertTrue(result.contains("onConnected(cb: () => void): void"))
        assertTrue(result.contains("onDisconnected(cb: () => void): void"))
        assertTrue(result.contains("onBeforeRender(cb: () => void): void"))
        assertTrue(result.contains("onAfterRender(cb: () => void): void"))
        assertTrue(result.contains("onUpdated(cb: (context?: unknown) => void): void"))
        assertTrue(result.contains("onCleanup(fn: () => void): void"))
    }

    @Test
    fun `generatePKDeclaration generates pk with single ref`() {
        val result = PKPRefExtractor.generatePKDeclaration(listOf("emailInput"))
        assertTrue(
            "Should contain declare namespace pk",
            result.contains("declare namespace pk")
        )
        assertTrue(
            "Should contain readonly ref field",
            result.contains("readonly emailInput: HTMLElement | null")
        )
    }

    @Test
    fun `generatePKDeclaration generates pk with multiple refs`() {
        val result = PKPRefExtractor.generatePKDeclaration(listOf("emailInput", "submitBtn"))
        assertTrue(result.contains("readonly emailInput: HTMLElement | null"))
        assertTrue(result.contains("readonly submitBtn: HTMLElement | null"))
    }

    @Test
    fun `generatePKDeclaration preserves field order`() {
        val result = PKPRefExtractor.generatePKDeclaration(listOf("zeta", "alpha", "middle"))
        assertTrue("zeta should come before alpha", result.indexOf("zeta") < result.indexOf("alpha"))
        assertTrue("alpha should come before middle", result.indexOf("alpha") < result.indexOf("middle"))
    }

    @Test
    fun `generatePKDeclaration wraps refs in const refs`() {
        val result = PKPRefExtractor.generatePKDeclaration(listOf("myRef"))
        assertTrue(
            "Should contain const refs",
            result.contains("const refs: { readonly myRef: HTMLElement | null }")
        )
    }

    @Test
    fun `generatePKDeclaration includes lifecycle methods alongside refs`() {
        val result = PKPRefExtractor.generatePKDeclaration(listOf("myRef"))
        assertTrue(result.contains("onConnected(cb: () => void): void"))
        assertTrue(result.contains("onCleanup(fn: () => void): void"))
    }

    @Test
    fun `generatePKDeclaration handles many refs`() {
        val names = (1..10).map { "ref$it" }
        val result = PKPRefExtractor.generatePKDeclaration(names)
        for (name in names) {
            assertTrue(
                "Should contain field $name",
                result.contains("$name: HTMLElement | null")
            )
        }
    }

    @Test
    fun `generatePKDeclaration handles underscore-prefixed names`() {
        val result = PKPRefExtractor.generatePKDeclaration(listOf("_internal"))
        assertTrue(result.contains("readonly _internal: HTMLElement | null"))
    }

    @Test
    fun `generatePKDeclaration handles dollar-prefixed names`() {
        val result = PKPRefExtractor.generatePKDeclaration(listOf("\$el"))
        assertTrue(result.contains("readonly \$el: HTMLElement | null"))
    }

    @Test
    fun `generatePKCDeclaration declares pkc as interface extending HTMLElement`() {
        val result = PKPRefExtractor.generatePKCDeclaration(emptyList())
        assertTrue(
            "Should contain interface extending HTMLElement",
            result.contains("interface _PikoComponent extends HTMLElement")
        )
        assertTrue(
            "Should declare pkc as _PikoComponent",
            result.contains("declare const pkc: _PikoComponent")
        )
        assertTrue(
            "Should contain empty refs",
            result.contains("readonly refs: {}")
        )
    }

    @Test
    fun `generatePKCDeclaration includes component instance methods`() {
        val result = PKPRefExtractor.generatePKCDeclaration(emptyList())
        assertTrue(result.contains("setState(partialState: Record<string, unknown>): void"))
        assertTrue(result.contains("render(): void"))
        assertTrue(result.contains("scheduleRender(): void"))
        assertTrue(result.contains("state: Record<string, unknown> | undefined"))
    }

    @Test
    fun `generatePKCDeclaration includes lifecycle registration methods`() {
        val result = PKPRefExtractor.generatePKCDeclaration(emptyList())
        assertTrue(result.contains("onConnected(cb: () => void): void"))
        assertTrue(result.contains("onDisconnected(cb: () => void): void"))
        assertTrue(result.contains("onBeforeRender(cb: () => void): void"))
        assertTrue(result.contains("onAfterRender(cb: () => void): void"))
        assertTrue(result.contains("onUpdated(cb: (changedProperties: Set<string>) => void): void"))
    }

    @Test
    fun `generatePKCDeclaration includes onCleanup`() {
        val result = PKPRefExtractor.generatePKCDeclaration(emptyList())
        assertTrue(
            "Should contain onCleanup",
            result.contains("onCleanup(cb: () => void): void")
        )
    }

    @Test
    fun `generatePKCDeclaration includes typed refs`() {
        val result = PKPRefExtractor.generatePKCDeclaration(listOf("myInput", "submitBtn"))
        assertTrue(result.contains("readonly myInput: HTMLElement | null"))
        assertTrue(result.contains("readonly submitBtn: HTMLElement | null"))
    }

    @Test
    fun `generatePKCDeclaration includes slot management methods`() {
        val result = PKPRefExtractor.generatePKCDeclaration(emptyList())
        assertTrue(
            "Should contain attachSlotListener",
            result.contains("attachSlotListener(slotName: string, callback: (elements: Element[]) => void): void")
        )
        assertTrue(
            "Should contain getSlottedElements",
            result.contains("getSlottedElements(slotName?: string): Element[]")
        )
        assertTrue(
            "Should contain hasSlotContent",
            result.contains("hasSlotContent(slotName?: string): boolean")
        )
    }
}
