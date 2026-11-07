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

package io.politepixels.piko.pk.lsp

import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.project.Project
import com.redhat.devtools.lsp4ij.LanguageServerManager
import io.politepixels.piko.settings.PikoSettings

/**
 * LSP4IJ-backed implementation of [PikoLspController].
 */
class Lsp4ijPikoLspController : PikoLspController {

    /** Logger for this controller. */
    private val log = Logger.getInstance(Lsp4ijPikoLspController::class.java)

    /**
     * Restarts the server, or stops it when LSP support is disabled.
     *
     * @param project The project whose server should be restarted.
     */
    override fun restart(project: Project) {
        if (project.isDisposed) return

        val manager = LanguageServerManager.getInstance(project)
        if (!PikoSettings.getInstance().lspEnabled) {
            log.info("Stopping Piko LSP for project ${project.name} because it is disabled")
            manager.stop(PikoLspController.SERVER_ID)
            return
        }

        log.info("Restarting Piko LSP for project ${project.name}")
        manager.start(
            PikoLspController.SERVER_ID,
            LanguageServerManager.StartOptions().setForceRestart(true)
        )
    }
}
