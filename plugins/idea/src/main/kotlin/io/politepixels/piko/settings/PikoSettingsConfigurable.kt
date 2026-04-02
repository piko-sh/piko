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
import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.fileChooser.FileChooserDescriptorFactory
import com.intellij.openapi.options.Configurable
import com.intellij.openapi.project.ProjectManager
import com.intellij.openapi.ui.DialogPanel
import com.intellij.ui.dsl.builder.AlignX
import com.intellij.ui.dsl.builder.bindIntText
import com.intellij.ui.dsl.builder.bindSelected
import com.intellij.ui.dsl.builder.bindText
import com.intellij.ui.dsl.builder.panel
import com.redhat.devtools.lsp4ij.LanguageServerManager
import javax.swing.JComponent

/**
 * Settings UI for the Piko plugin.
 *
 * Provides a configuration panel accessible via Settings > Languages & Frameworks > Piko.
 * Allows users to configure the LSP binary path, transport settings, and external server options.
 */
class PikoSettingsConfigurable : Configurable {

    /** Logger for this configurable. */
    private val log = Logger.getInstance(PikoSettingsConfigurable::class.java)

    /** The persistent settings instance. */
    private val settings = PikoSettings.getInstance()

    /** The UI panel, stored to delegate isModified/apply/reset to the UI DSL bindings. */
    private lateinit var settingsPanel: DialogPanel

    /** UI binding for the custom LSP binary path. */
    private var lspPath = settings.lspPath

    /** UI binding for whether LSP support is enabled. */
    private var lspEnabled = settings.lspEnabled

    /** UI binding for whether to show the LSP not found notification. */
    private var showLspNotFoundNotification = settings.showLspNotFoundNotification

    /** UI binding for whether to use stdio transport instead of TCP. */
    private var useStdioTransport = settings.useStdioTransport

    /** UI binding for whether to connect to an external LSP server. */
    private var useTcpMode = settings.useTcpMode

    /** UI binding for the external server host address. */
    private var tcpHost = settings.tcpHost

    /** UI binding for the external server port number. */
    private var tcpPort = settings.tcpPort

    /** UI binding for whether to enable the pprof profiling server. */
    private var pprofEnabled = settings.pprofEnabled

    /** UI binding for the pprof server port number. */
    private var pprofPort = settings.pprofPort

    /** UI binding for whether to enable document formatting. */
    private var formattingEnabled = settings.formattingEnabled

    /** UI binding for whether to enable file logging. */
    private var fileLoggingEnabled = settings.fileLoggingEnabled

    /** UI binding for custom Go binary directory path. */
    private var goBinPath = settings.goBinPath

    /** UI binding for whether to detect Go SDK from IDE. */
    private var detectGoFromIde = settings.detectGoFromIde

    /** UI binding for whether to search common Go installation locations. */
    private var searchGlobalGoLocations = settings.searchGlobalGoLocations

    /**
     * Returns the display name shown in the settings tree.
     *
     * @return The string "Piko".
     */
    override fun getDisplayName(): String = "Piko"

    /**
     * Creates the settings panel UI component.
     *
     * @return The root panel containing all settings controls.
     */
    override fun createComponent(): JComponent {
        settingsPanel = panel {
            languageServerGroup()
            lspBinaryGroup()
            goConfigurationGroup()
            transportGroup()
            externalServerGroup()
            profilingGroup()
            informationGroup()
        }
        return settingsPanel
    }

    /**
     * Creates the Language Server settings group.
     *
     * @return The settings group with LSP enable/disable options.
     */
    private fun com.intellij.ui.dsl.builder.Panel.languageServerGroup() = group("Language Server") {
        row { checkBox("Enable LSP support").bindSelected(::lspEnabled) }
        row { checkBox("Enable formatting").bindSelected(::formattingEnabled) }
        row { checkBox("Enable file logging").bindSelected(::fileLoggingEnabled).comment("Logs to /tmp/piko-lsp-&lt;pid&gt;.log") }
        row { checkBox("Show notification when LSP binary is not found").bindSelected(::showLspNotFoundNotification) }
    }

    /**
     * Creates the LSP Binary settings group.
     *
     * @return The settings group with custom binary path configuration.
     */
    private fun com.intellij.ui.dsl.builder.Panel.lspBinaryGroup() = group("LSP Binary") {
        row("Custom LSP path:") {
            textFieldWithBrowseButton(
                FileChooserDescriptorFactory.createSingleFileDescriptor().withTitle("Select Piko LSP Binary")
            ).bindText(::lspPath).align(AlignX.FILL).comment("Leave empty to use bundled binary or search standard locations.")
        }
    }

    /**
     * Creates the Go Configuration settings group.
     *
     * Allows configuring how the Go binary is located for the LSP server.
     * This is particularly useful on macOS where GUI apps don't inherit shell PATH.
     *
     * @return The settings group with Go binary path configuration.
     */
    private fun com.intellij.ui.dsl.builder.Panel.goConfigurationGroup() = group("Go Configuration") {
        row("Go binary directory:") {
            textFieldWithBrowseButton(
                FileChooserDescriptorFactory.createSingleFolderDescriptor().withTitle("Select Go Binary Directory")
            ).bindText(::goBinPath).align(AlignX.FILL)
                .comment("Path to directory containing the 'go' binary (e.g. /usr/local/go/bin). Leave empty for auto-detection.")
        }
        row {
            checkBox("Detect Go SDK from IDE").bindSelected(::detectGoFromIde)
                .comment("Use the Go SDK configured in GoLand or IntelliJ's Go plugin.")
        }
        row {
            checkBox("Search common installation locations").bindSelected(::searchGlobalGoLocations)
                .comment("Check standard paths like /usr/local/go/bin, Homebrew locations, and ~/sdk/go*/bin.")
        }
        row {
            text("<small>Resolution priority: Manual path → IDE detection → Common locations → System PATH</small>")
        }
    }

    /**
     * Creates the Transport settings group for advanced configuration.
     *
     * @return The settings group with stdio transport toggle.
     */
    private fun com.intellij.ui.dsl.builder.Panel.transportGroup() = group("Transport (Advanced)") {
        row {
            checkBox("Use stdio transport (experimental)").bindSelected(::useStdioTransport)
                .comment("<b>Warning:</b> May cause IDE freezes due to pipe buffer deadlocks.<br>Only enable for debugging purposes.")
        }
    }

    /**
     * Creates the External Server settings group for advanced configuration.
     *
     * @return The settings group with external server connection options.
     */
    private fun com.intellij.ui.dsl.builder.Panel.externalServerGroup() = group("External Server (Advanced)") {
        row {
            checkBox("Connect to external LSP server").bindSelected(::useTcpMode)
                .comment("Connect to an already-running LSP server instead of starting one automatically.")
        }
        row("Host:") { textField().bindText(::tcpHost).comment("Default: 127.0.0.1") }
        row("Port:") {
            intTextField(1..65535).bindIntText(::tcpPort)
                .comment("Default: 4389. Start external server with: piko-lsp --tcp --port=4389")
        }
    }

    /**
     * Creates the Profiling settings group for advanced configuration.
     *
     * @return The settings group with pprof server options.
     */
    private fun com.intellij.ui.dsl.builder.Panel.profilingGroup() = group("Profiling (Advanced)") {
        row {
            checkBox("Enable pprof profiling server").bindSelected(::pprofEnabled)
                .comment("Starts a pprof HTTP server for CPU/memory profiling. For debugging only.")
        }
        row("pprof Port:") {
            intTextField(1..65535).bindIntText(::pprofPort).comment("Default: 6060. Access at http://localhost:6060/debug/pprof/")
        }
    }

    /**
     * Creates the Information group with help text.
     *
     * @return The settings group with informational text.
     */
    private fun com.intellij.ui.dsl.builder.Panel.informationGroup() = group("Information") {
        row { text("The Piko LSP provides code intelligence for template blocks.") }
        row { text("Go/CSS/TypeScript support is provided by language injection.") }
    }

    /**
     * Checks if any settings have been modified.
     *
     * Delegates to the panel's built-in isModified, which compares the current UI
     * state against the bound property values. This is necessary because the UI DSL
     * defers writing to bound properties until [apply] is called.
     *
     * @return True if any setting differs from the stored value.
     */
    override fun isModified(): Boolean = settingsPanel.isModified()

    /**
     * Applies the modified settings to persistent storage.
     *
     * First applies the panel bindings to update the local properties from the UI,
     * then detects whether settings that affect the LSP connection have changed
     * and automatically restarts the LSP server if needed.
     */
    override fun apply() {
        val oldLspPath = settings.lspPath
        val oldLspEnabled = settings.lspEnabled
        val oldUseStdioTransport = settings.useStdioTransport
        val oldUseTcpMode = settings.useTcpMode
        val oldTcpHost = settings.tcpHost
        val oldTcpPort = settings.tcpPort
        val oldPprofEnabled = settings.pprofEnabled
        val oldPprofPort = settings.pprofPort
        val oldFormattingEnabled = settings.formattingEnabled
        val oldFileLoggingEnabled = settings.fileLoggingEnabled
        val oldGoBinPath = settings.goBinPath
        val oldDetectGoFromIde = settings.detectGoFromIde
        val oldSearchGlobalGoLocations = settings.searchGlobalGoLocations

        settingsPanel.apply()

        settings.lspPath = lspPath
        settings.lspEnabled = lspEnabled
        settings.showLspNotFoundNotification = showLspNotFoundNotification
        settings.useStdioTransport = useStdioTransport
        settings.useTcpMode = useTcpMode
        settings.tcpHost = tcpHost
        settings.tcpPort = tcpPort
        settings.pprofEnabled = pprofEnabled
        settings.pprofPort = pprofPort
        settings.formattingEnabled = formattingEnabled
        settings.fileLoggingEnabled = fileLoggingEnabled
        settings.goBinPath = goBinPath
        settings.detectGoFromIde = detectGoFromIde
        settings.searchGlobalGoLocations = searchGlobalGoLocations

        val needsRestart = lspPath != oldLspPath ||
                lspEnabled != oldLspEnabled ||
                useStdioTransport != oldUseStdioTransport ||
                useTcpMode != oldUseTcpMode ||
                tcpHost != oldTcpHost ||
                tcpPort != oldTcpPort ||
                pprofEnabled != oldPprofEnabled ||
                pprofPort != oldPprofPort ||
                formattingEnabled != oldFormattingEnabled ||
                fileLoggingEnabled != oldFileLoggingEnabled ||
                goBinPath != oldGoBinPath ||
                detectGoFromIde != oldDetectGoFromIde ||
                searchGlobalGoLocations != oldSearchGlobalGoLocations

        if (needsRestart) {
            restartLspForAllProjects()
        }
    }

    /**
     * Restarts the Piko LSP server for all open projects.
     *
     * Stops the existing server connection (which properly cleans up the process)
     * and starts a new one with the updated configuration.
     */
    private fun restartLspForAllProjects() {
        log.info("Settings changed, restarting Piko LSP for all projects...")

        ApplicationManager.getApplication().executeOnPooledThread {
            val projects = ProjectManager.getInstance().openProjects
            for (project in projects) {
                if (project.isDisposed) continue

                try {
                    val manager = LanguageServerManager.getInstance(project)
                    log.info("Stopping Piko LSP for project: ${project.name}")
                    manager.stop("pikoLsp")

                    Thread.sleep(LSP_RESTART_DELAY_MS)

                    log.info("Starting Piko LSP for project: ${project.name}")
                    manager.start("pikoLsp")
                } catch (e: Exception) {
                    log.warn("Failed to restart Piko LSP for project ${project.name}: ${e.message}", e)
                }
            }
            log.info("Piko LSP restart complete for all projects")
        }
    }

    /**
     * Resets the UI to match the currently stored settings.
     *
     * Updates the local backing properties from persistent settings first,
     * then resets the panel so the UI reflects the current property values.
     */
    override fun reset() {
        lspPath = settings.lspPath
        lspEnabled = settings.lspEnabled
        showLspNotFoundNotification = settings.showLspNotFoundNotification
        useStdioTransport = settings.useStdioTransport
        useTcpMode = settings.useTcpMode
        tcpHost = settings.tcpHost
        tcpPort = settings.tcpPort
        pprofEnabled = settings.pprofEnabled
        pprofPort = settings.pprofPort
        formattingEnabled = settings.formattingEnabled
        fileLoggingEnabled = settings.fileLoggingEnabled
        goBinPath = settings.goBinPath
        detectGoFromIde = settings.detectGoFromIde
        searchGlobalGoLocations = settings.searchGlobalGoLocations

        settingsPanel.reset()
    }

    companion object {
        /** Delay in milliseconds between stopping and starting the LSP server during restart. */
        private const val LSP_RESTART_DELAY_MS = 100L
    }
}
