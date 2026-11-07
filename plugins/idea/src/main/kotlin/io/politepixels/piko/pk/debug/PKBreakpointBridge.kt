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

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.project.Project
import com.intellij.xdebugger.breakpoints.XBreakpoint
import com.intellij.xdebugger.breakpoints.XBreakpointListener
import com.intellij.xdebugger.breakpoints.XLineBreakpoint

/**
 * Turns each new Piko gutter shim breakpoint into a Delve breakpoint.
 *
 * @property project The project whose breakpoints are watched.
 */
class PKBreakpointBridge(private val project: Project) : XBreakpointListener<XBreakpoint<*>> {

    /**
     * Schedules conversion of a newly added shim breakpoint.
     *
     * @param breakpoint The breakpoint that was added.
     */
    override fun breakpointAdded(breakpoint: XBreakpoint<*>) {
        if (breakpoint.type !is PKLineBreakpointType) return
        val shim = breakpoint as? XLineBreakpoint<*> ?: return
        ApplicationManager.getApplication().invokeLater(
            { PKDelveBreakpoints.convert(project, shim, toggle = true) },
            project.disposed,
        )
    }
}
