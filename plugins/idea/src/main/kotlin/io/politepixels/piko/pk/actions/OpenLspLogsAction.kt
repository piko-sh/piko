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
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.fileEditor.FileEditorManager
import com.intellij.openapi.project.DumbAware
import com.intellij.openapi.vfs.LocalFileSystem
import java.io.File

/**
 * Action to open the Piko LSP log file.
 *
 * Opens the LSP server log file in the editor for debugging purposes.
 * The log file location is platform-specific.
 */
class OpenLspLogsAction : AnAction("Open Piko LSP Logs"), DumbAware {

    companion object {
        /** Standard log file path on Unix-like systems. */
        private const val UNIX_LOG_PATH = "/tmp/piko-lsp-main.log"

        /** Log file name for Windows, located in user's temp directory. */
        private const val WINDOWS_LOG_NAME = "piko-lsp-main.log"
    }

    /**
     * Opens the LSP log file in the editor.
     *
     * Shows a notification if the log file does not exist.
     *
     * @param e The action event.
     */
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val logPath = getLogFilePath()
        val logFile = File(logPath)

        if (!logFile.exists()) {
            NotificationGroupManager.getInstance()
                .getNotificationGroup("Piko")
                .createNotification(
                    "Piko LSP Log Not Found",
                    "Log file not found at: $logPath\n\n" +
                        "The log file is created when the LSP server starts.",
                    NotificationType.WARNING
                )
                .notify(project)
            return
        }

        val virtualFile = LocalFileSystem.getInstance().findFileByIoFile(logFile)
        if (virtualFile != null) {
            FileEditorManager.getInstance(project).openFile(virtualFile, true)
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
     * Returns the platform-specific log file path.
     *
     * @return The path to the LSP log file.
     */
    private fun getLogFilePath(): String {
        val os = System.getProperty("os.name").lowercase()
        return if (os.contains("windows")) {
            File(System.getenv("TEMP") ?: System.getProperty("java.io.tmpdir"), WINDOWS_LOG_NAME).absolutePath
        } else {
            UNIX_LOG_PATH
        }
    }
}
