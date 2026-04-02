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


package io.politepixels.piko.pk.service

import com.intellij.openapi.vfs.VirtualFile

/**
 * Contains parsed information from a go.mod file.
 *
 * Stores the module path, root directory, and any local replace directives
 * needed for Go import resolution in PK files.
 *
 * @property modulePath The Go module path from the module directive.
 * @property moduleRoot The directory containing the go.mod file.
 * @property replacements A map of replaced module paths to their local directories.
 */
data class PikoModuleInfo(
    val modulePath: String,
    val moduleRoot: VirtualFile,
    val replacements: Map<String, VirtualFile>
)
