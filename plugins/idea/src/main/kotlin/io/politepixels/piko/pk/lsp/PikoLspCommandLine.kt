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

import com.google.gson.JsonObject
import com.intellij.ide.plugins.PluginManagerCore
import com.intellij.openapi.extensions.PluginId

/**
 * The transport the plugin uses to talk to a server it launches.
 */
sealed interface PikoLspTransport {

    /** Standard input and output of the child process. This is the default. */
    data object Stdio : PikoLspTransport

    /**
     * A loopback TCP socket, intended for debugging.
     *
     * @property host The address the server binds to.
     * @property port The port the server listens on.
     */
    data class Tcp(val host: String, val port: Int) : PikoLspTransport
}

/**
 * Settings that shape the server command line, independent of transport.
 *
 * @property formatting Whether to enable document formatting.
 * @property logFile The debug log path, or null to disable file logging.
 * @property pprof Whether to start the pprof profiling server.
 * @property pprofPort The pprof HTTP port.
 */
data class PikoLspLaunchOptions(
    val formatting: Boolean,
    val logFile: String?,
    val pprof: Boolean,
    val pprofPort: Int
)

/**
 * Builds the pikopls argument list.
 *
 * One builder serves both transports, so settings such as formatting and
 * logging apply whichever transport is chosen.
 */
object PikoLspArguments {

    /**
     * Maximum concurrent connections for a TCP server the plugin launches.
     *
     * The plugin is the only client, so a second connection is never legitimate.
     */
    const val TCP_MAX_CONNECTIONS = 1

    /**
     * Builds the arguments that follow the binary path.
     *
     * @param options The transport-independent launch options.
     * @param transport The transport to launch with.
     * @return The ordered argument list.
     */
    @JvmStatic
    fun build(options: PikoLspLaunchOptions, transport: PikoLspTransport): List<String> {
        val arguments = mutableListOf<String>()
        if (transport is PikoLspTransport.Tcp) {
            arguments.add("--tcp")
            arguments.add("--host=${transport.host}")
            arguments.add("--port=${transport.port}")
            arguments.add("--max-connections=$TCP_MAX_CONNECTIONS")
        }
        if (options.formatting) {
            arguments.add("--formatting")
        }
        if (options.logFile != null) {
            arguments.add("--file-logging")
            arguments.add("--log-file=${options.logFile}")
        }
        if (options.pprof) {
            arguments.add("--pprof")
            arguments.add("--pprof-port=${options.pprofPort}")
        }
        return arguments
    }
}

/**
 * Builds the `initializationOptions` sent to the server.
 */
object PikoLspInitialisationOptions {

    /** Plugin identifier of the JetBrains Go plugin. */
    private const val GO_PLUGIN_ID = "org.jetbrains.plugins.go"

    /**
     * Builds the options for the current IDE.
     *
     * @return The JSON options object.
     */
    @JvmStatic
    fun build(): JsonObject = build(goPluginEnabled = isGoPluginEnabled())

    /**
     * Builds the options for a known Go plugin state.
     *
     * @param goPluginEnabled Whether the JetBrains Go plugin is active.
     * @return The JSON options object.
     */
    @JvmStatic
    fun build(goPluginEnabled: Boolean): JsonObject = JsonObject().apply {
        addProperty("goBridge", !goPluginEnabled)
    }

    /**
     * Checks whether the JetBrains Go plugin is installed and loaded.
     *
     * @return True when the Go plugin is active.
     */
    private fun isGoPluginEnabled(): Boolean = PluginManagerCore.isLoaded(PluginId.getId(GO_PLUGIN_ID))
}

/**
 * The operating system and architecture used to pick a bundled binary.
 *
 * @property os The Go operating system name, or null when unsupported.
 * @property arch The Go architecture name, or null when unsupported.
 */
data class PikoLspPlatform(val os: String?, val arch: String?) {

    /** Whether a bundled binary can exist for this platform. */
    val isSupported: Boolean get() = os != null && arch != null

    /** The bundled resource directory name, such as `linux-amd64`, or null when unsupported. */
    val directory: String? get() = if (isSupported) "$os-$arch" else null

    /** The executable file name for this platform. */
    val binaryName: String get() = if (os == "windows") "pikopls.exe" else "pikopls"

    companion object {
        /**
         * Detects the platform of the running JVM.
         *
         * @return The detected platform.
         */
        @JvmStatic
        fun current(): PikoLspPlatform =
            of(System.getProperty("os.name").orEmpty(), System.getProperty("os.arch").orEmpty())

        /**
         * Maps JVM `os.name` and `os.arch` values to Go names.
         *
         * Unknown values map to null rather than a guess, so an unsupported
         * machine never runs a binary built for a different architecture.
         *
         * @param osName The JVM operating system name.
         * @param osArch The JVM architecture name.
         * @return The mapped platform.
         */
        @JvmStatic
        fun of(osName: String, osArch: String): PikoLspPlatform {
            val osLower = osName.lowercase()
            val os = when {
                osLower.contains("linux") -> "linux"
                osLower.contains("mac") || osLower.contains("darwin") -> "darwin"
                osLower.contains("windows") -> "windows"
                else -> null
            }
            val arch = when (osArch.lowercase()) {
                "amd64", "x86_64", "x86-64", "x64" -> "amd64"
                "aarch64", "arm64" -> "arm64"
                else -> null
            }
            return PikoLspPlatform(os, arch)
        }
    }
}
