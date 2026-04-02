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

import com.intellij.ide.actions.CreateFileFromTemplateAction
import com.intellij.ide.actions.CreateFileFromTemplateDialog
import com.intellij.openapi.project.DumbAware
import com.intellij.openapi.project.Project
import com.intellij.psi.PsiDirectory
import io.politepixels.piko.pk.PKIcons

/**
 * Action for creating new Piko template files.
 *
 * Adds a "Piko File" option to the New menu in the Project view.
 * Users can choose between a basic component template or a full page template.
 */
class CreatePikoFileAction : CreateFileFromTemplateAction(
    "Piko File",
    "Create a new Piko template file",
    PKIcons.FILE
), DumbAware {

    /**
     * Builds the creation dialog with available template options.
     *
     * @param project The current project.
     * @param directory The directory where the file will be created.
     * @param builder The dialog builder to configure.
     */
    override fun buildDialog(
        project: Project,
        directory: PsiDirectory,
        builder: CreateFileFromTemplateDialog.Builder
    ) {
        builder
            .setTitle("New Piko File")
            .addKind("Page", PKIcons.FILE, "Piko Page")
            .addKind("Partial", PKIcons.FILE, "Piko Partial")
    }

    /**
     * Returns the action name shown in the IDE for undo/redo.
     *
     * @param directory The target directory, or null.
     * @param newName The name of the file being created.
     * @param templateName The template being used, or null.
     * @return The action name string.
     */
    override fun getActionName(
        directory: PsiDirectory?,
        newName: String,
        templateName: String?
    ): String = "Create Piko File $newName"
}
