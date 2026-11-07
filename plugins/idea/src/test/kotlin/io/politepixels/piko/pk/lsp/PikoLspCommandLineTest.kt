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

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.ByteArrayInputStream

class PikoLspCommandLineTest {

    private val defaults = PikoLspLaunchOptions(formatting = false, logFile = null, pprof = false, pprofPort = 6060)

    @Test
    fun `stdio launch has no transport flags`() {
        assertEquals(emptyList<String>(), PikoLspArguments.build(defaults, PikoLspTransport.Stdio))
    }

    @Test
    fun `tcp launch binds loopback and allows one connection`() {
        val arguments = PikoLspArguments.build(defaults, PikoLspTransport.Tcp("127.0.0.1", 4567))
        assertEquals(listOf("--tcp", "--host=127.0.0.1", "--port=4567", "--max-connections=1"), arguments)
    }

    @Test
    fun `settings flags apply to both transports`() {
        val options = PikoLspLaunchOptions(formatting = true, logFile = "/logs/pikopls.log", pprof = true, pprofPort = 7070)
        val expected = listOf("--formatting", "--file-logging", "--log-file=/logs/pikopls.log", "--pprof", "--pprof-port=7070")

        assertEquals(expected, PikoLspArguments.build(options, PikoLspTransport.Stdio))
        assertEquals(expected, PikoLspArguments.build(options, PikoLspTransport.Tcp("127.0.0.1", 1)).drop(4))
    }

    @Test
    fun `no log file means no file logging flags`() {
        val arguments = PikoLspArguments.build(defaults.copy(formatting = true), PikoLspTransport.Stdio)
        assertFalse(arguments.any { it.startsWith("--file-logging") || it.startsWith("--log-file") })
    }

    @Test
    fun `go bridge is enabled only without the Go plugin`() {
        assertTrue(PikoLspInitialisationOptions.build(goPluginEnabled = false).get("goBridge").asBoolean)
        assertFalse(PikoLspInitialisationOptions.build(goPluginEnabled = true).get("goBridge").asBoolean)
    }

    @Test
    fun `known platforms map to Go names`() {
        assertEquals("linux-amd64", PikoLspPlatform.of("Linux", "amd64").directory)
        assertEquals("linux-amd64", PikoLspPlatform.of("Linux", "x86_64").directory)
        assertEquals("darwin-arm64", PikoLspPlatform.of("Mac OS X", "aarch64").directory)
        assertEquals("windows-amd64", PikoLspPlatform.of("Windows 11", "amd64").directory)
        assertEquals("pikopls.exe", PikoLspPlatform.of("Windows 11", "amd64").binaryName)
        assertEquals("pikopls", PikoLspPlatform.of("Linux", "aarch64").binaryName)
    }

    @Test
    fun `unknown architectures are unsupported rather than amd64`() {
        val platform = PikoLspPlatform.of("Linux", "riscv64")
        assertFalse(platform.isSupported)
        assertNull(platform.directory)
        assertNull(PikoLspPlatform.of("Linux", "x86").arch)
    }

    @Test
    fun `unknown operating systems are unsupported`() {
        assertFalse(PikoLspPlatform.of("FreeBSD", "amd64").isSupported)
    }

    @Test
    fun `version directory names are sanitised`() {
        assertEquals("1.2.3", PikoLspBundledBinary.versionDirectoryName("1.2.3"))
        assertEquals("1.2.3-rc_1_.._x", PikoLspBundledBinary.versionDirectoryName("1.2.3-rc/1/../x"))
        assertEquals("unknown", PikoLspBundledBinary.versionDirectoryName(""))
    }

    @Test
    fun `sha256 matches a known digest`() {
        val digest = PikoLspBundledBinary.sha256(ByteArrayInputStream("abc".toByteArray()))
        assertEquals("ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", digest)
    }
}
