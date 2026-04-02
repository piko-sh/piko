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

import com.intellij.codeInsight.template.TemplateActionContext
import com.intellij.codeInsight.template.TemplateContextType

/**
 * Defines the context for Piko live templates.
 *
 * This context type enables Piko-specific live templates to be available
 * when editing .pk files.
 */
class PKTemplateContextType : TemplateContextType("Piko") {

    /**
     * Determines if this template context is applicable.
     *
     * @param templateActionContext The context containing the file and caret position.
     * @return true if the file is a PK file, false otherwise.
     */
    override fun isInContext(templateActionContext: TemplateActionContext): Boolean {
        val file = templateActionContext.file
        return file.language == PKLanguage || file.fileType == PKFileType
    }
}
