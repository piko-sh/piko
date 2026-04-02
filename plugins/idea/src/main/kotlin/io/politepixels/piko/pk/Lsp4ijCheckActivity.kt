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

import com.intellij.ide.BrowserUtil
import com.intellij.ide.plugins.PluginManagerCore
import com.intellij.notification.NotificationAction
import com.intellij.notification.NotificationGroupManager
import com.intellij.notification.NotificationType
import com.intellij.openapi.extensions.PluginId
import com.intellij.openapi.project.Project
import com.intellij.openapi.startup.ProjectActivity
import io.politepixels.piko.settings.PikoSettings

private const val LSP4IJ_PLUGIN_ID = "com.redhat.devtools.lsp4ij"
private const val LSP4IJ_MARKETPLACE_URL = "https://plugins.jetbrains.com/plugin/23257-lsp4ij"

/**
 * Post-startup activity that checks whether LSP4IJ is installed.
 *
 * If LSP4IJ is not installed and the user has not dismissed the notification,
 * shows a one-time balloon recommending installation. The notification provides
 * an action to open the JetBrains Marketplace page and a dismiss action that
 * persists the preference so the notification never appears again.
 */
class Lsp4ijCheckActivity : ProjectActivity {

    override suspend fun execute(project: Project) {
        val settings = PikoSettings.getInstance()

        if (!settings.showLsp4ijNotification) {
            return
        }

        if (PluginManagerCore.isPluginInstalled(PluginId.getId(LSP4IJ_PLUGIN_ID))) {
            return
        }

        val notification = NotificationGroupManager.getInstance()
            .getNotificationGroup("Piko")
            .createNotification(
                "LSP4IJ Plugin Recommended",
                "Install the LSP4IJ plugin for template block intelligence in .pk files.",
                NotificationType.INFORMATION
            )

        notification.addAction(NotificationAction.createSimpleExpiring("Install LSP4IJ") {
            BrowserUtil.browse(LSP4IJ_MARKETPLACE_URL)
        })

        notification.addAction(NotificationAction.createSimpleExpiring("Don't Show Again") {
            settings.showLsp4ijNotification = false
        })

        notification.notify(project)
    }
}
