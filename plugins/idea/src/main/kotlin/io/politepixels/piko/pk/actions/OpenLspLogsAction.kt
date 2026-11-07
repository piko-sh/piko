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

import com.intellij.notification.NotificationGroupManager
import com.intellij.notification.NotificationType
import com.intellij.openapi.actionSystem.ActionUpdateThread
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.fileEditor.OpenFileDescriptor
import com.intellij.openapi.project.DumbAware
import com.intellij.openapi.vfs.LocalFileSystem
import io.politepixels.piko.pk.lsp.PikoLspLogFile
import io.politepixels.piko.settings.PikoSettings
import java.nio.file.Files

/**
 * Action to open the Piko LSP log file.
 *
 * Opens the file the plugin passes to the server with `--log-file`, which
 * lives under the IDE log directory. File logging is off by default; the
 * notification explains how to enable it when no log exists yet.
 */
class OpenLspLogsAction : AnAction("Open Piko LSP Logs"), DumbAware {

    /** Logger for this action. */
    private val log = Logger.getInstance(OpenLspLogsAction::class.java)

    /**
     * Opens the LSP log file in the editor.
     *
     * Shows a notification if the log file does not exist.
     *
     * @param e The action event.
     */
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val logPath = PikoLspLogFile.forProject(project)

        if (!Files.isRegularFile(logPath)) {
            val hint = if (PikoSettings.getInstance().fileLoggingEnabled) {
                "The log file is created when the LSP server starts. Restart the server if it is already running."
            } else {
                "File logging is disabled. Enable it in Settings > Languages &amp; Frameworks > Piko, " +
                    "then restart the LSP server."
            }
            NotificationGroupManager.getInstance()
                .getNotificationGroup("Piko")
                .createNotification(
                    "Piko LSP Log Not Found",
                    "Log file not found at: $logPath<br><br>$hint",
                    NotificationType.WARNING
                )
                .notify(project)
            return
        }

        val virtualFile = LocalFileSystem.getInstance().refreshAndFindFileByNioFile(logPath)
        if (virtualFile == null) {
            log.warn("Piko LSP log exists but could not be opened: $logPath")
            return
        }
        OpenFileDescriptor(project, virtualFile).navigate(true)
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
