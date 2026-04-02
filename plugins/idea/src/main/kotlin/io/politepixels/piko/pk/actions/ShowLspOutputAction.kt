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


package io.politepixels.piko.pk.actions

import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.project.DumbAware
import com.intellij.openapi.wm.ToolWindowManager

/**
 * Action to show the Piko LSP output panel.
 *
 * Opens the LSP4IJ console tool window where LSP server output and
 * communication logs can be viewed.
 */
class ShowLspOutputAction : AnAction("Show Piko LSP Output"), DumbAware {

    companion object {
        /** LSP4IJ console tool window ID. */
        private const val LSP_CONSOLE_TOOL_WINDOW_ID = "Language Servers"
    }

    /**
     * Shows the LSP console tool window.
     *
     * @param e The action event.
     */
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val toolWindow = ToolWindowManager.getInstance(project)
            .getToolWindow(LSP_CONSOLE_TOOL_WINDOW_ID)
        toolWindow?.show()
    }

    /**
     * Enables the action only when a project is open.
     *
     * @param e The action event.
     */
    override fun update(e: AnActionEvent) {
        e.presentation.isEnabledAndVisible = e.project != null
    }
}
