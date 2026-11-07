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

package io.politepixels.piko.pk.debug

import com.goide.dlv.breakpoint.DlvBreakpointType
import com.intellij.openapi.application.WriteAction
import com.intellij.openapi.vfs.VirtualFile
import com.intellij.testFramework.PlatformTestUtil
import com.intellij.testFramework.fixtures.BasePlatformTestCase
import com.intellij.xdebugger.XDebuggerManager
import com.intellij.xdebugger.XDebuggerUtil
import com.intellij.xdebugger.breakpoints.XBreakpointManager
import com.intellij.xdebugger.breakpoints.XLineBreakpointType

/**
 * Platform tests for `.pk` gutter breakpoints and their conversion to Delve breakpoints.
 */
class PKLineBreakpointTypeTest : BasePlatformTestCase() {

    private val shimType: PKLineBreakpointType
        get() = XDebuggerUtil.getInstance().findBreakpointType(PKLineBreakpointType::class.java)

    private val delveType: DlvBreakpointType
        get() = XDebuggerUtil.getInstance().findBreakpointType(DlvBreakpointType::class.java)

    private val manager: XBreakpointManager
        get() = XDebuggerManager.getInstance(project).breakpointManager

    override fun tearDown() {
        try {
            WriteAction.runAndWait<RuntimeException> {
                manager.allBreakpoints.filter { it.type is XLineBreakpointType<*> }.forEach(manager::removeBreakpoint)
            }
        } finally {
            super.tearDown()
        }
    }

    fun testCanPutAtAcceptsGoAndTemplateLinesOnly() {
        val file = configurePage()
        val expected = mapOf(
            0 to false,
            1 to true,
            2 to true,
            3 to true,
            4 to false,
            5 to false,
            6 to false,
            7 to false,
            8 to false,
            9 to false,
            10 to false,
            11 to false,
            12 to true,
            13 to false,
            14 to true,
            15 to true,
            16 to true,
            17 to false,
            18 to false,
            19 to false,
            20 to false,
            21 to false,
            22 to false,
            23 to false,
            24 to false,
        )
        for ((line, allowed) in expected) {
            assertEquals("line $line", allowed, shimType.canPutAt(file, line, project))
        }
    }

    fun testCanPutAtRefusesPkcComponents() {
        val file = myFixture.configureByText(
            "widget.pkc",
            """
            <template><p>{{ state.Title }}</p></template>

            <script type="application/x-go">
            package main

            func Render() {
                x := 1
            }
            </script>
            """.trimIndent()
        ).virtualFile
        for (line in 0..7) {
            assertFalse("line $line", shimType.canPutAt(file, line, project))
        }
    }

    fun testGoLinesAfterLessThanUseTheWholeScript() {
        val file = myFixture.configureByText(
            "loop.pk",
            """
            <script type="application/x-go">
            package main

            func Render() {
                if a < b {
                    // note
                    c := 1
                }
            }
            </script>
            """.trimIndent()
        ).virtualFile
        val expected = listOf(false, false, false, true, true, false, true, true, true, false)
        for ((line, allowed) in expected.withIndex()) {
            assertEquals("line $line", allowed, shimType.canPutAt(file, line, project))
        }
    }

    fun testDelveTypeRefusesPkLinesSoTheShimIsNeeded() {
        val file = configurePage()
        assertFalse(delveType.canPutAt(file, 12, project))
    }

    fun testShimBecomesDelveBreakpointAndTogglesOff() {
        val file = configurePage()

        addShim(file, 12)
        PlatformTestUtil.dispatchAllInvocationEventsInIdeEventQueue()

        assertEmpty(manager.getBreakpoints(PKLineBreakpointType::class.java))
        val delve = manager.findBreakpointsAtLine(delveType, file, 12)
        assertEquals(1, delve.size)
        assertEquals(file.url, delve.single().fileUrl)
        assertTrue("A line with a Delve breakpoint stays clickable", shimType.canPutAt(file, 12, project))

        addShim(file, 12)
        PlatformTestUtil.dispatchAllInvocationEventsInIdeEventQueue()

        assertEmpty(manager.getBreakpoints(PKLineBreakpointType::class.java))
        assertEmpty(manager.findBreakpointsAtLine(delveType, file, 12))
    }

    fun testMigrationKeepsSettingsAndDoesNotToggle() {
        val file = configurePage()
        WriteAction.runAndWait<RuntimeException> {
            manager.addLineBreakpoint(delveType, file.url, 14, delveType.createBreakpointProperties(file, 14))
        }

        val shim = WriteAction.computeAndWait<_, RuntimeException> {
            manager.addLineBreakpoint(shimType, file.url, 2, PKBreakpointProperties())
        }
        shim.isEnabled = false
        val duplicate = WriteAction.computeAndWait<_, RuntimeException> {
            manager.addLineBreakpoint(shimType, file.url, 14, PKBreakpointProperties())
        }
        assertNotNull(duplicate)

        WriteAction.runAndWait<RuntimeException> { PKDelveBreakpoints.migrateAll(project) }
        PlatformTestUtil.dispatchAllInvocationEventsInIdeEventQueue()

        assertEmpty(manager.getBreakpoints(PKLineBreakpointType::class.java))
        val migrated = manager.findBreakpointsAtLine(delveType, file, 2)
        assertEquals(1, migrated.size)
        assertFalse(migrated.single().isEnabled)
        assertEquals("Migration must not toggle an existing Delve breakpoint away", 1, manager.findBreakpointsAtLine(delveType, file, 14).size)
    }

    private fun addShim(file: VirtualFile, line: Int) {
        WriteAction.runAndWait<RuntimeException> {
            manager.addLineBreakpoint(shimType, file.url, line, PKBreakpointProperties())
        }
    }

    private fun configurePage(): VirtualFile =
        myFixture.configureByText(
            "page.pk",
            """
            <template>
              <div p-if="state.Show"
                   :title="state.Title">
                <p>{{ state.Title }}</p>
              </div>
            </template>

            <script type="application/x-go">
            package main

            import "fmt"

            func Render() Response {
                // explain
                x := fmt.Sprint(1)
                return Response{X: x}
            }
            </script>

            <script lang="ts">
            const greeting: string = "hello";
            console.log(greeting);
            </script>

            <style>
            .a { color: red; }
            </style>
            """.trimIndent()
        ).virtualFile
}
