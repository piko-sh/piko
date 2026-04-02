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


package io.politepixels.piko.pk.debug

import com.intellij.xdebugger.breakpoints.XBreakpointProperties

/**
 * Properties for Piko line breakpoints.
 *
 * Currently carries no additional state beyond the base class.
 * The breakpoint location (file + line) is stored by the framework.
 */
class PKBreakpointProperties : XBreakpointProperties<PKBreakpointProperties>() {

    override fun getState(): PKBreakpointProperties = this

    override fun loadState(state: PKBreakpointProperties) {}
}
