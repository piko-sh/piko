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

package io.politepixels.piko.pk.injection

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class PkComponentImportsTest {

    @Test
    fun `only pk imports are component imports`() {
        assertTrue(PkComponentImports.isComponentImport("example.com/app/partials/card.pk"))
        assertFalse(PkComponentImports.isComponentImport("example.com/app/partials"))
    }

    @Test
    fun `module alias resolves against the current module`() {
        assertEquals(
            PkImportTarget(null, "partials/card.pk"),
            PkComponentImports.target("@/partials/card.pk", "example.com/app", emptyList())
        )
    }

    @Test
    fun `full module path resolves against the current module`() {
        assertEquals(
            PkImportTarget(null, "partials/card.pk"),
            PkComponentImports.target("example.com/app/partials/card.pk", "example.com/app", emptyList())
        )
    }

    @Test
    fun `replaced module paths resolve against the replacement`() {
        assertEquals(
            PkImportTarget("example.com/ui", "components/button.pk"),
            PkComponentImports.target("example.com/ui/components/button.pk", "example.com/app", listOf("example.com/ui"))
        )
    }

    @Test
    fun `the longest replacement wins`() {
        assertEquals(
            PkImportTarget("example.com/ui/v2", "button.pk"),
            PkComponentImports.target(
                "example.com/ui/v2/button.pk",
                "example.com/app",
                listOf("example.com/ui", "example.com/ui/v2")
            )
        )
    }

    @Test
    fun `module path prefix must end at a path separator`() {
        assertNull(PkComponentImports.target("example.com/application/card.pk", "example.com/app", emptyList()))
    }

    @Test
    fun `imports outside every module are unresolved`() {
        assertNull(PkComponentImports.target("other.org/x/card.pk", "example.com/app", emptyList()))
        assertNull(PkComponentImports.target("example.com/app/pkg", "example.com/app", emptyList()))
    }

    @Test
    fun `paths that leave the module root are unresolved`() {
        assertNull(PkComponentImports.target("@/../secret/card.pk", "example.com/app", emptyList()))
        assertNull(PkComponentImports.target("example.com/app/partials/../../card.pk", "example.com/app", emptyList()))
        assertNull(PkComponentImports.target("@//card.pk", "example.com/app", emptyList()))
    }
}
