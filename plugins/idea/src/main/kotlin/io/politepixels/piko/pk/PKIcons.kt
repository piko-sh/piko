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

import com.intellij.openapi.util.IconLoader
import javax.swing.Icon

/**
 * Holds icon resources for the Piko plugin.
 *
 * Icons are loaded from the plugin's resource directory and used throughout
 * the IDE for file type indicators, tool windows, and editor gutters.
 */
object PKIcons {

    /**
     * The primary file icon shown in the project tree and editor tabs for `.pk` files.
     */
    @JvmField
    val FILE: Icon = IconLoader.getIcon("/icons/panda.svg", PKIcons::class.java)
}
