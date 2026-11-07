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

import com.intellij.openapi.actionSystem.ActionUpdateThread
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.project.DumbAware
import io.politepixels.piko.pk.lsp.PikoLspController

/**
 * Action to restart the Piko LSP server.
 *
 * Stops the current Piko language server instance and starts a new one.
 * Useful when the server becomes unresponsive or after configuration changes.
 * Registered from piko-lsp4ij.xml, so it only exists when LSP4IJ is installed.
 */
class RestartLspAction : AnAction("Restart Piko LSP"), DumbAware {

    /** Logger for this action. */
    private val log = Logger.getInstance(RestartLspAction::class.java)

    /**
     * Restarts the Piko language server.
     *
     * @param e The action event.
     */
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val controller = PikoLspController.getInstance()
        if (controller == null) {
            log.warn("Cannot restart the Piko LSP: LSP4IJ is not installed")
            return
        }
        ApplicationManager.getApplication().executeOnPooledThread {
            controller.restart(project)
        }
    }

    /**
     * Enables the action only when a project is open.
     *
     * @param e The action event.
     */
    override fun update(e: AnActionEvent) {
        e.presentation.isEnabledAndVisible = e.project != null
    }

    /**
     * Runs [update] on a background thread; it only reads the project from the event.
     *
     * @return [ActionUpdateThread.BGT].
     */
    override fun getActionUpdateThread(): ActionUpdateThread = ActionUpdateThread.BGT
}
