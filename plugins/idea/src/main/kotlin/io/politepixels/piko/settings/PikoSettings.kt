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


package io.politepixels.piko.settings

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.components.PersistentStateComponent
import com.intellij.openapi.components.State
import com.intellij.openapi.components.Storage
import com.intellij.openapi.diagnostic.Logger
import com.intellij.util.xmlb.XmlSerializerUtil

/**
 * Persistent settings for the Piko plugin.
 *
 * Stores user preferences including the LSP binary path, transport settings,
 * and external server configuration. Settings are persisted to piko.xml in
 * the IDE configuration directory.
 */
@State(
    name = "PikoSettings",
    storages = [Storage("piko.xml")]
)
class PikoSettings : PersistentStateComponent<PikoSettings.State> {

    /** The current settings state. */
    private var myState = State()

    /**
     * Holds all persistent settings values.
     *
     * @property lspPath Custom path to the pikopls binary. Empty uses standard locations.
     * @property lspEnabled Whether LSP support is enabled.
     * @property showLspNotFoundNotification Whether to show a notification when LSP is missing.
     * @property useTcpTransport Whether to launch the server over loopback TCP instead of stdio. For debugging.
     * @property useTcpMode Whether to connect to an external LSP server.
     * @property tcpHost The host address for external server connection.
     * @property tcpPort The port number for external server connection.
     * @property pprofEnabled Whether to enable the pprof profiling server.
     * @property pprofPort The port number for the pprof HTTP server.
     * @property formattingEnabled Whether to enable document formatting.
     * @property fileLoggingEnabled Whether the server writes a debug log under the IDE log directory.
     * @property goBinPath Custom path to directory containing the Go binary. Empty uses auto-detection.
     * @property detectGoFromIde Whether to detect Go SDK from IntelliJ/GoLand configuration.
     * @property searchGlobalGoLocations Whether to search common Go installation locations.
     * @property showLsp4ijNotification Whether to show a notification recommending LSP4IJ installation.
     */
    data class State(
        var lspPath: String = "",
        var lspEnabled: Boolean = true,
        var showLspNotFoundNotification: Boolean = true,
        var useTcpTransport: Boolean = false,
        var useTcpMode: Boolean = false,
        var tcpHost: String = DEFAULT_TCP_HOST,
        var tcpPort: Int = DEFAULT_TCP_PORT,
        var pprofEnabled: Boolean = false,
        var pprofPort: Int = DEFAULT_PPROF_PORT,
        var formattingEnabled: Boolean = false,
        var fileLoggingEnabled: Boolean = false,
        var goBinPath: String = "",
        var detectGoFromIde: Boolean = true,
        var searchGlobalGoLocations: Boolean = true,
        var showLsp4ijNotification: Boolean = true
    )

    /**
     * Returns the current settings state for serialisation.
     *
     * @return The current state object.
     */
    override fun getState(): State = myState

    /**
     * Loads a settings state from storage.
     *
     * @param state The state to load.
     */
    override fun loadState(state: State) {
        XmlSerializerUtil.copyBean(state, myState)
        sanitise(myState)
    }

    /** Custom path to the pikopls binary. Empty uses standard locations. */
    var lspPath: String
        get() = myState.lspPath
        set(value) {
            myState.lspPath = value
        }

    /** Whether LSP support is enabled. */
    var lspEnabled: Boolean
        get() = myState.lspEnabled
        set(value) {
            myState.lspEnabled = value
        }

    /** Whether to show a notification when the LSP binary is not found. */
    var showLspNotFoundNotification: Boolean
        get() = myState.showLspNotFoundNotification
        set(value) {
            myState.showLspNotFoundNotification = value
        }

    /** Whether to launch the server over loopback TCP instead of stdio. Intended for debugging. */
    var useTcpTransport: Boolean
        get() = myState.useTcpTransport
        set(value) {
            myState.useTcpTransport = value
        }

    /** Whether to connect to an external LSP server instead of starting one. */
    var useTcpMode: Boolean
        get() = myState.useTcpMode
        set(value) {
            myState.useTcpMode = value
        }

    /** The host address for external server connection. */
    var tcpHost: String
        get() = myState.tcpHost
        set(value) {
            myState.tcpHost = value
        }

    /** The port number for external server connection. */
    var tcpPort: Int
        get() = myState.tcpPort
        set(value) {
            myState.tcpPort = value
        }

    /** Whether to enable the pprof profiling server. */
    var pprofEnabled: Boolean
        get() = myState.pprofEnabled
        set(value) {
            myState.pprofEnabled = value
        }

    /** The port number for the pprof HTTP server. */
    var pprofPort: Int
        get() = myState.pprofPort
        set(value) {
            myState.pprofPort = value
        }

    /** Whether to enable document formatting from the LSP server. */
    var formattingEnabled: Boolean
        get() = myState.formattingEnabled
        set(value) {
            myState.formattingEnabled = value
        }

    /** Whether the server writes a debug log under the IDE log directory. */
    var fileLoggingEnabled: Boolean
        get() = myState.fileLoggingEnabled
        set(value) {
            myState.fileLoggingEnabled = value
        }

    /** Custom path to directory containing the Go binary. Empty uses auto-detection. */
    var goBinPath: String
        get() = myState.goBinPath
        set(value) {
            myState.goBinPath = value
        }

    /** Whether to detect Go SDK from IntelliJ/GoLand configuration. */
    var detectGoFromIde: Boolean
        get() = myState.detectGoFromIde
        set(value) {
            myState.detectGoFromIde = value
        }

    /** Whether to search common Go installation locations. */
    var searchGlobalGoLocations: Boolean
        get() = myState.searchGlobalGoLocations
        set(value) {
            myState.searchGlobalGoLocations = value
        }

    /** Whether to show a notification recommending LSP4IJ installation. */
    var showLsp4ijNotification: Boolean
        get() = myState.showLsp4ijNotification
        set(value) {
            myState.showLsp4ijNotification = value
        }

    companion object {
        /** Default host for the LSP TCP server. */
        const val DEFAULT_TCP_HOST = "127.0.0.1"

        /** Default port for the LSP TCP server. */
        const val DEFAULT_TCP_PORT = 4389

        /** Default port for the pprof HTTP server. */
        const val DEFAULT_PPROF_PORT = 6060

        /** The range of valid TCP port numbers. */
        val VALID_PORTS = 1..65535

        /** Logger for settings validation. */
        private val log = Logger.getInstance(PikoSettings::class.java)

        /**
         * Checks whether a host name or IP address is usable.
         *
         * @param host The host to check.
         * @return True when the host is not blank and has no whitespace.
         */
        @JvmStatic
        fun isValidHost(host: String): Boolean {
            val trimmed = host.trim()
            return trimmed.isNotEmpty() && trimmed.none { it.isWhitespace() }
        }

        /**
         * Replaces stored values that are out of range with their defaults.
         *
         * A hand-edited or old settings file must not start the server on an
         * invalid port or host.
         *
         * @param state The state to correct in place.
         */
        internal fun sanitise(state: PikoSettings.State) {
            state.lspPath = state.lspPath.trim()
            state.goBinPath = state.goBinPath.trim()
            if (!isValidHost(state.tcpHost)) {
                log.warn("Stored Piko LSP host '${state.tcpHost}' is invalid, using $DEFAULT_TCP_HOST")
                state.tcpHost = DEFAULT_TCP_HOST
            }
            state.tcpHost = state.tcpHost.trim()
            if (state.tcpPort !in VALID_PORTS) {
                log.warn("Stored Piko LSP port ${state.tcpPort} is out of range, using $DEFAULT_TCP_PORT")
                state.tcpPort = DEFAULT_TCP_PORT
            }
            if (state.pprofPort !in VALID_PORTS) {
                log.warn("Stored pprof port ${state.pprofPort} is out of range, using $DEFAULT_PPROF_PORT")
                state.pprofPort = DEFAULT_PPROF_PORT
            }
        }

        /**
         * Returns the application-wide settings instance.
         *
         * @return The PikoSettings service instance.
         */
        @JvmStatic
        fun getInstance(): PikoSettings {
            return ApplicationManager.getApplication().getService(PikoSettings::class.java)
        }
    }
}
