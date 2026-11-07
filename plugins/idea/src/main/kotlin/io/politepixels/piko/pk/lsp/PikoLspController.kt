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

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.project.Project

/**
 * Controls the lifecycle of the Piko language server.
 */
interface PikoLspController {

    /**
     * Restarts the Piko language server for a project, or stops it when the
     * server has been disabled in settings.
     *
     * @param project The project whose server should be restarted.
     */
    fun restart(project: Project)

    companion object {
        /** The LSP4IJ server identifier declared in piko-lsp4ij.xml. */
        const val SERVER_ID = "pikoLsp"

        /**
         * Returns the controller when LSP4IJ is installed.
         *
         * @return The registered controller, or null when LSP4IJ is absent.
         */
        @JvmStatic
        fun getInstance(): PikoLspController? =
            ApplicationManager.getApplication().getService(PikoLspController::class.java)
    }
}
