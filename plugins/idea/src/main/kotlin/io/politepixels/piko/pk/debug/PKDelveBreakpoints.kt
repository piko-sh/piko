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

import com.goide.dlv.breakpoint.DlvBreakpointProperties
import com.goide.dlv.breakpoint.DlvBreakpointType
import com.intellij.openapi.application.WriteAction
import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.VirtualFile
import com.intellij.openapi.vfs.VirtualFileManager
import com.intellij.xdebugger.XDebuggerManager
import com.intellij.xdebugger.XDebuggerUtil
import com.intellij.xdebugger.breakpoints.XBreakpointManager
import com.intellij.xdebugger.breakpoints.XLineBreakpoint

/**
 * Converts Piko gutter shim breakpoints into GoLand Delve line breakpoints.
 *
 * All mutating functions must run on the event dispatch thread.
 */
object PKDelveBreakpoints {

    /**
     * Finds the Delve line breakpoints on a file line.
     *
     * @param project The project whose breakpoints to search.
     * @param file The file to search.
     * @param line The zero-based line number.
     * @return The Delve breakpoints at that line, empty when there are none.
     */
    fun findAtLine(project: Project, file: VirtualFile, line: Int): Collection<XLineBreakpoint<DlvBreakpointProperties>> {
        val delveType = delveType() ?: return emptyList()
        return breakpointManager(project).findBreakpointsAtLine(delveType, file, line)
    }

    /**
     * Replaces a shim breakpoint with the matching Delve breakpoint state.
     *
     * @param project The project that owns the breakpoint.
     * @param shim The shim breakpoint to convert.
     * @param toggle Whether an existing Delve breakpoint at the line should be removed.
     * @return The new Delve breakpoint, or null when none was created.
     */
    fun convert(
        project: Project,
        shim: XLineBreakpoint<*>,
        toggle: Boolean,
    ): XLineBreakpoint<DlvBreakpointProperties>? {
        val manager = breakpointManager(project)
        if (manager.getBreakpoints(PKLineBreakpointType::class.java).none { it === shim }) return null

        val delveType = delveType()
        val file = VirtualFileManager.getInstance().findFileByUrl(shim.fileUrl)

        return WriteAction.compute<XLineBreakpoint<DlvBreakpointProperties>?, RuntimeException> {
            manager.removeBreakpoint(shim)
            if (delveType == null) return@compute null

            val existing = if (file != null) manager.findBreakpointsAtLine(delveType, file, shim.line) else emptyList()
            when {
                existing.isNotEmpty() && toggle -> {
                    existing.forEach(manager::removeBreakpoint)
                    null
                }
                existing.isNotEmpty() -> null
                else -> addDelveBreakpoint(manager, delveType, file, shim)
            }
        }
    }

    /**
     * Converts every persisted shim breakpoint in a project.
     *
     * @param project The project to migrate.
     * @return The number of shim breakpoints converted.
     */
    fun migrateAll(project: Project): Int {
        val shims = breakpointManager(project).getBreakpoints(PKLineBreakpointType::class.java).toList()
        shims.forEach { convert(project, it, toggle = false) }
        return shims.size
    }

    private fun addDelveBreakpoint(
        manager: XBreakpointManager,
        delveType: DlvBreakpointType,
        file: VirtualFile?,
        shim: XLineBreakpoint<*>,
    ): XLineBreakpoint<DlvBreakpointProperties> {
        val properties = if (file != null) delveType.createBreakpointProperties(file, shim.line) else DlvBreakpointProperties()
        val delve = manager.addLineBreakpoint(delveType, shim.fileUrl, shim.line, properties, shim.isTemporary)
        delve.isEnabled = shim.isEnabled
        delve.suspendPolicy = shim.suspendPolicy
        delve.isLogMessage = shim.isLogMessage
        delve.isLogStack = shim.isLogStack
        delve.conditionExpression = shim.conditionExpression
        delve.logExpressionObject = shim.logExpressionObject
        return delve
    }

    private fun delveType(): DlvBreakpointType? =
        XDebuggerUtil.getInstance().findBreakpointType(DlvBreakpointType::class.java)

    private fun breakpointManager(project: Project): XBreakpointManager =
        XDebuggerManager.getInstance(project).breakpointManager
}
