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

import com.intellij.openapi.fileTypes.LanguageFileType
import javax.swing.Icon

/**
 * File type definition for Piko template files.
 *
 * This object tells IntelliJ how to recognise and handle `.pk` files,
 * linking them to the PK language for parsing and syntax highlighting.
 */
object PKFileType : LanguageFileType(PKLanguage) {

    /**
     * Returns the internal identifier used by IntelliJ for this file type.
     *
     * @return The file type name "PK File".
     */
    override fun getName(): String = "PK File"

    /**
     * Returns the description shown in file type settings and dialogs.
     *
     * @return The description "Piko Template File".
     */
    override fun getDescription(): String = "Piko Template File"

    /**
     * Returns the default file extension without the leading dot.
     *
     * @return The extension "pk".
     */
    override fun getDefaultExtension(): String = "pk"

    /**
     * Returns the icon displayed in the project tree and editor tabs.
     *
     * @return The Piko file icon, or null if unavailable.
     */
    override fun getIcon(): Icon? = PKIcons.FILE
}
