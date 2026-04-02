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

import com.intellij.notification.NotificationGroupManager
import com.intellij.notification.NotificationType
import com.intellij.openapi.application.PathManager
import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.VirtualFile
import com.redhat.devtools.lsp4ij.LanguageServerFactory
import com.redhat.devtools.lsp4ij.client.LanguageClientImpl
import com.redhat.devtools.lsp4ij.server.StreamConnectionProvider
import io.politepixels.piko.pk.PKFileType
import io.politepixels.piko.settings.GoPathResolver
import io.politepixels.piko.settings.PikoSettings
import java.io.File
import java.io.IOException
import java.io.InputStream
import java.io.OutputStream
import java.net.ServerSocket
import java.net.Socket
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardCopyOption

/**
 * Factory for creating Piko LSP server connections.
 *
 * Registered with LSP4IJ to launch and manage the Piko language server
 * for template block code intelligence.
 */
class PikoLanguageServerFactory : LanguageServerFactory {

    /**
     * Creates a connection provider for the Piko LSP.
     *
     * @param project The current project.
     * @return A stream connection provider for LSP communication.
     */
    override fun createConnectionProvider(project: Project): StreamConnectionProvider {
        return PikoLspStreamConnectionProvider(project)
    }

    /**
     * Creates the language client for handling LSP notifications.
     *
     * @param project The current project.
     * @return A language client implementation.
     */
    override fun createLanguageClient(project: Project): LanguageClientImpl {
        return LanguageClientImpl(project)
    }
}

/**
 * Provides the connection to the Piko LSP server.
 *
 * By default, starts the LSP binary in TCP mode on a random port to avoid
 * stdio buffering issues that can cause IDE freezes. Falls back to connecting
 * to a manually-started LSP server if configured.
 *
 * Binary search order:
 * 1. Custom path from settings
 * 2. Bundled binary (extracted from plugin JAR)
 * 3. Project-local paths
 * 4. Home directory paths (~/.piko/bin, ~/go/bin)
 * 5. System paths (/usr/local/bin, /usr/bin)
 * 6. PATH lookup
 *
 * @param project The current project.
 */
class PikoLspStreamConnectionProvider(private val project: Project) : StreamConnectionProvider {

    /** Logger for this connection provider. */
    private val log = Logger.getInstance(PikoLspStreamConnectionProvider::class.java)

    /** The LSP server process when started locally. */
    private var process: Process? = null

    /** The TCP socket connection to the LSP server. */
    private var socket: Socket? = null

    /** Whether the LSP binary could not be found. */
    private var lspNotFound = false

    companion object {
        /** Maximum number of connection attempts before giving up. */
        private const val MAX_CONNECTION_ATTEMPTS = 50

        /** Delay in milliseconds between connection retry attempts. */
        private const val CONNECTION_RETRY_DELAY_MS = 100L
    }

    /**
     * Starts the LSP server connection.
     *
     * Checks settings to determine whether to use TCP or stdio transport,
     * and whether to connect to an external server or start a new instance.
     */
    override fun start() {
        val settings = PikoSettings.getInstance()

        if (!settings.lspEnabled) {
            log.info("Piko LSP is disabled in settings")
            lspNotFound = true
            return
        }

        if (settings.useTcpMode) {
            if (tryTcpConnection(settings.tcpHost, settings.tcpPort)) {
                log.info("Piko LSP connected to existing server via TCP on ${settings.tcpHost}:${settings.tcpPort}")
                return
            }
            log.info("Could not connect to existing LSP server at ${settings.tcpHost}:${settings.tcpPort}, will start new instance")
        }

        if (settings.useStdioTransport) {
            log.warn("Using experimental stdio transport - may cause IDE freezes")
            startStdioMode(settings)
            return
        }

        startTcpMode(settings)
    }

    /**
     * Finds a free port by briefly opening a server socket.
     *
     * @return An available port number.
     */
    private fun findFreePort(): Int {
        ServerSocket(0).use { serverSocket ->
            return serverSocket.localPort
        }
    }

    /**
     * Ensures the LSP binary is available.
     *
     * Attempts to find the binary and handles the not-found case by setting
     * the lspNotFound flag and optionally showing a notification.
     *
     * @param settings The current plugin settings.
     * @return The path to the LSP binary, or null if not found.
     */
    private fun ensureLspBinary(settings: PikoSettings): String? {
        val lspPath = findPikoLspBinary()
        if (lspPath != null) return lspPath

        lspNotFound = true
        log.warn("piko-lsp binary not found")
        if (settings.showLspNotFoundNotification) {
            showLspNotFoundNotification()
        }
        return null
    }

    /**
     * Builds the command-line arguments for the LSP binary.
     *
     * @param settings The current plugin settings.
     * @param port The TCP port to use.
     * @return The list of command arguments.
     */
    private fun buildLspCommand(lspPath: String, port: Int, settings: PikoSettings): List<String> {
        val command = mutableListOf(lspPath, "--tcp", "--port=$port")
        if (settings.pprofEnabled) {
            command.add("--pprof")
            command.add("--pprof-port=${settings.pprofPort}")
        }
        if (settings.formattingEnabled) {
            command.add("--formatting")
        }
        if (settings.fileLoggingEnabled) {
            command.add("--file-logging")
        }
        return command
    }

    /**
     * Starts the LSP binary in TCP mode on a random port and connects to it.
     *
     * @param settings The current plugin settings.
     */
    private fun startTcpMode(settings: PikoSettings) {
        val lspPath = ensureLspBinary(settings) ?: return

        val port = findFreePort()
        val command = buildLspCommand(lspPath, port, settings)
        log.info("Starting Piko LSP with command: ${command.joinToString(" ")}")

        try {
            val processBuilder = ProcessBuilder(command)
            processBuilder.environment()["PIKO_DISABLE_CONSOLE_LOG"] = "true"

            val goBinPath = GoPathResolver.resolve(project, settings)
            if (goBinPath != null) {
                val currentPath = processBuilder.environment()["PATH"] ?: ""
                processBuilder.environment()["PATH"] = "$goBinPath${File.pathSeparator}$currentPath"
                log.info("Set Go bin path in LSP environment: $goBinPath")
            }

            project.basePath?.let { processBuilder.directory(File(it)) }

            processBuilder.redirectErrorStream(false)
            process = processBuilder.start()

            process?.errorStream?.let { errorStream ->
                Thread {
                    errorStream.bufferedReader().useLines { lines ->
                        lines.forEach { line ->
                            log.warn("Piko LSP stderr: $line")
                        }
                    }
                }.start()
            }

            log.info("Piko LSP process started (PID check: isAlive=${process?.isAlive})")

            if (!waitAndConnect("127.0.0.1", port)) {
                val exitCode = try { process?.exitValue() } catch (_: IllegalThreadStateException) { null }
                log.error("Failed to connect to Piko LSP after starting it. Exit code: $exitCode")
                lspNotFound = true
                process?.destroyForcibly()
                process = null
                showStartupErrorNotification("LSP server started but connection failed (exit code: $exitCode)")
            }
        } catch (e: IOException) {
            log.error("Failed to start Piko LSP: I/O error", e)
            lspNotFound = true
            showStartupErrorNotification(e.message ?: "I/O error")
        } catch (e: SecurityException) {
            log.error("Failed to start Piko LSP: permission denied", e)
            lspNotFound = true
            showStartupErrorNotification("Permission denied: ${e.message}")
        }
    }

    /**
     * Starts the LSP binary in stdio mode.
     *
     * This is experimental and may cause IDE freezes due to pipe buffer deadlocks.
     *
     * @param settings The current plugin settings.
     */
    private fun startStdioMode(settings: PikoSettings) {
        val lspPath = ensureLspBinary(settings) ?: return

        val command = listOf(lspPath)
        log.info("Starting Piko LSP in stdio mode (experimental): ${command.joinToString(" ")}")

        try {
            val processBuilder = ProcessBuilder(command)
            processBuilder.environment()["PIKO_DISABLE_CONSOLE_LOG"] = "true"

            val goBinPath = GoPathResolver.resolve(project, settings)
            if (goBinPath != null) {
                val currentPath = processBuilder.environment()["PATH"] ?: ""
                processBuilder.environment()["PATH"] = "$goBinPath${File.pathSeparator}$currentPath"
                log.info("Set Go bin path in LSP environment: $goBinPath")
            }

            project.basePath?.let { processBuilder.directory(File(it)) }

            process = processBuilder.start()
            log.info("Piko LSP process started in stdio mode (PID check: isAlive=${process?.isAlive})")

            NotificationGroupManager.getInstance()
                .getNotificationGroup("Piko")
                .createNotification(
                    "Piko LSP: Experimental Mode",
                    "Using stdio transport which may cause IDE freezes. " +
                        "Disable 'Use stdio transport' in Settings > Languages & Frameworks > Piko if you experience problems.",
                    NotificationType.WARNING
                )
                .notify(project)

        } catch (e: IOException) {
            log.error("Failed to start Piko LSP in stdio mode: I/O error", e)
            lspNotFound = true
            showStartupErrorNotification(e.message ?: "I/O error")
        } catch (e: SecurityException) {
            log.error("Failed to start Piko LSP in stdio mode: permission denied", e)
            lspNotFound = true
            showStartupErrorNotification("Permission denied: ${e.message}")
        }
    }

    /**
     * Waits for the LSP server to be ready and establishes a TCP connection.
     *
     * @param host The host to connect to.
     * @param port The port to connect to.
     * @return True if connection succeeded, false otherwise.
     */
    private fun waitAndConnect(host: String, port: Int): Boolean {
        for (attempt in 1..MAX_CONNECTION_ATTEMPTS) {
            process?.let {
                if (!it.isAlive) {
                    log.error("LSP process died during startup")
                    return false
                }
            }

            if (tryTcpConnection(host, port)) {
                log.info("Connected to Piko LSP on attempt $attempt")
                return true
            }

            Thread.sleep(CONNECTION_RETRY_DELAY_MS)
        }
        return false
    }

    /**
     * Attempts to establish a TCP connection to the LSP server.
     *
     * @param host The host to connect to.
     * @param port The port to connect to.
     * @return True if connection succeeded, false otherwise.
     */
    private fun tryTcpConnection(host: String, port: Int): Boolean {
        return try {
            val sock = Socket(host, port)
            sock.tcpNoDelay = true
            socket = sock
            true
        } catch (e: IOException) {
            log.debug("TCP connection to $host:$port failed: ${e.message}")
            false
        } catch (e: SecurityException) {
            log.debug("TCP connection to $host:$port denied: ${e.message}")
            false
        }
    }

    /**
     * Returns the input stream for reading LSP responses.
     *
     * @return The input stream from socket or process stdout, or null if unavailable.
     */
    override fun getInputStream(): InputStream? {
        if (lspNotFound) return null
        return socket?.getInputStream() ?: process?.inputStream
    }

    /**
     * Returns the output stream for sending LSP requests.
     *
     * @return The output stream to socket or process stdin, or null if unavailable.
     */
    override fun getOutputStream(): OutputStream? {
        if (lspNotFound) return null
        return socket?.getOutputStream() ?: process?.outputStream
    }

    /**
     * Stops the LSP server connection and cleans up resources.
     */
    override fun stop() {
        socket?.close()
        socket = null
        process?.destroyForcibly()
        process = null
    }

    /**
     * Platform information for binary selection.
     *
     * @property platform The OS platform (linux, darwin, windows).
     * @property goArch The Go architecture (amd64, arm64).
     * @property binaryName The platform-specific binary name.
     * @property platformDir The combined platform directory name.
     */
    private data class PlatformInfo(
        val platform: String,
        val goArch: String,
        val binaryName: String,
        val platformDir: String
    )

    /**
     * Searches for the piko-lsp binary in standard locations.
     *
     * Checks custom path, bundled binary, file system paths, and system PATH.
     *
     * @return The path to the binary, or null if not found.
     */
    private fun findPikoLspBinary(): String? {
        val settings = PikoSettings.getInstance()
        val platformInfo = detectPlatformInfo()

        tryCustomPath(settings.lspPath)?.let { return it }

        extractBundledLsp(platformInfo.platformDir, platformInfo.binaryName)?.let {
            log.info("Using bundled LSP: $it")
            return it
        }

        findInSearchPaths(buildSearchPaths(platformInfo))?.let { return it }

        return lookupInSystemPath(platformInfo.platform)
    }

    /**
     * Detects the current platform and architecture.
     *
     * @return Platform information for binary selection.
     */
    private fun detectPlatformInfo(): PlatformInfo {
        val os = System.getProperty("os.name").lowercase()
        val arch = System.getProperty("os.arch").lowercase()

        val platform = when {
            os.contains("linux") -> "linux"
            os.contains("mac") || os.contains("darwin") -> "darwin"
            os.contains("windows") -> "windows"
            else -> "linux"
        }
        val goArch = when {
            arch.contains("aarch64") || arch.contains("arm64") -> "arm64"
            else -> "amd64"
        }
        val binaryName = if (platform == "windows") "piko-lsp.exe" else "piko-lsp"

        return PlatformInfo(
            platform = platform,
            goArch = goArch,
            binaryName = binaryName,
            platformDir = "$platform-$goArch"
        )
    }

    /**
     * Attempts to use a custom LSP path from settings.
     *
     * @param customPath The custom path to check.
     * @return The path if valid and executable, or null otherwise.
     */
    private fun tryCustomPath(customPath: String): String? {
        if (customPath.isBlank()) return null

        val customFile = File(customPath)
        if (customFile.exists() && customFile.canExecute()) {
            log.info("Using custom LSP path: $customPath")
            return customPath
        }
        log.warn("Custom LSP path not found or not executable: $customPath")
        return null
    }

    /**
     * Builds the list of paths to search for the LSP binary.
     *
     * @param platformInfo The current platform information.
     * @return A list of potential paths to check.
     */
    private fun buildSearchPaths(platformInfo: PlatformInfo): List<String> {
        val paths = mutableListOf<String>()
        val home = System.getProperty("user.home")

        project.basePath?.let { basePath ->
            paths.add("$basePath/plugins/vscode/bin/${platformInfo.platformDir}/${platformInfo.binaryName}")
            paths.add("$basePath/bin/${platformInfo.binaryName}")
            paths.add("$basePath/.piko/bin/${platformInfo.binaryName}")
        }

        paths.add("$home/.piko/bin/${platformInfo.binaryName}")
        paths.add("$home/go/bin/${platformInfo.binaryName}")
        paths.add("/usr/local/bin/${platformInfo.binaryName}")
        paths.add("/usr/bin/${platformInfo.binaryName}")

        return paths
    }

    /**
     * Searches for the LSP binary in a list of paths.
     *
     * @param paths The paths to search.
     * @return The first valid executable path, or null if none found.
     */
    private fun findInSearchPaths(paths: List<String>): String? {
        for (path in paths) {
            val file = File(path)
            if (file.exists() && file.canExecute()) {
                log.info("Found LSP at: $path")
                return path
            }
        }
        return null
    }

    /**
     * Looks up piko-lsp in the system PATH.
     *
     * @param platform The current platform for command selection.
     * @return The command name if found in PATH, or null otherwise.
     */
    private fun lookupInSystemPath(platform: String): String? {
        val whichCmd = if (platform == "windows") "where" else "which"
        return try {
            val proc = ProcessBuilder(whichCmd, "piko-lsp")
                .redirectErrorStream(true)
                .start()
            if (proc.waitFor() == 0) {
                log.info("Found piko-lsp in PATH")
                "piko-lsp"
            } else {
                null
            }
        } catch (e: IOException) {
            log.debug("Failed to check PATH for piko-lsp: I/O error", e)
            null
        } catch (e: InterruptedException) {
            log.debug("PATH check for piko-lsp interrupted", e)
            Thread.currentThread().interrupt()
            null
        } catch (e: SecurityException) {
            log.debug("Failed to check PATH for piko-lsp: permission denied", e)
            null
        }
    }

    /**
     * Extracts the bundled LSP binary from plugin resources.
     *
     * Compares a version marker file with the current plugin version to determine
     * if re-extraction is needed after a plugin upgrade.
     *
     * @param platformDir The platform directory name (e.g. linux-amd64).
     * @param binaryName The binary file name.
     * @return The path to the extracted binary, or null if not bundled.
     */
    private fun extractBundledLsp(platformDir: String, binaryName: String): String? {
        val dataDir = Path.of(PathManager.getPluginsPath(), "piko-lsp-bin")
        val binaryFile = dataDir.resolve(platformDir).resolve(binaryName)
        val versionFile = dataDir.resolve(".version")
        val currentVersion = getPluginVersion()

        if (Files.exists(binaryFile) && binaryFile.toFile().canExecute()
            && isVersionCurrent(versionFile, currentVersion, binaryFile)) {
            log.debug("Bundled LSP is up to date (version: $currentVersion)")
            return binaryFile.toString()
        }

        val resourcePath = "/bin/$platformDir/$binaryName"
        val resourceStream = javaClass.getResourceAsStream(resourcePath)
        if (resourceStream == null) {
            log.debug("Bundled LSP not found in resources: $resourcePath")
            return null
        }

        try {
            Files.createDirectories(binaryFile.parent)
            resourceStream.use { input ->
                Files.copy(input, binaryFile, StandardCopyOption.REPLACE_EXISTING)
            }

            if (!System.getProperty("os.name").lowercase().contains("windows")) {
                binaryFile.toFile().setExecutable(true)
            }

            val extractedSize = Files.size(binaryFile)
            Files.writeString(versionFile, "$currentVersion:$extractedSize")

            log.info("Extracted bundled LSP to: $binaryFile (version: $currentVersion, size: $extractedSize)")
            return binaryFile.toString()
        } catch (e: IOException) {
            log.warn("Failed to extract bundled LSP binary: I/O error", e)
            return null
        } catch (e: SecurityException) {
            log.warn("Failed to extract bundled LSP binary: permission denied", e)
            return null
        }
    }

    /**
     * Gets the current plugin version.
     *
     * @return The plugin version string, or "unknown" if not available.
     */
    private fun getPluginVersion(): String {
        return try {
            val pluginId = com.intellij.openapi.extensions.PluginId.getId("io.politepixels.piko")
            com.intellij.ide.plugins.PluginManagerCore.getPlugin(pluginId)?.version ?: "unknown"
        } catch (e: Exception) {
            log.debug("Could not determine plugin version: ${e.message}")
            "unknown"
        }
    }

    /**
     * Checks if the extracted binary version and file size match expectations.
     *
     * The version file stores "version:filesize" (e.g. "1.2.3:57803264").
     * Both must match for the binary to be considered current. The size check
     * catches partial writes from interrupted extractions.
     *
     * Falls back to version-only comparison for legacy version files that
     * don't contain a size component.
     *
     * @param versionFile The path to the version marker file.
     * @param currentVersion The current plugin version.
     * @param binaryFile The path to the extracted binary.
     * @return True if version and size both match, false otherwise.
     */
    private fun isVersionCurrent(versionFile: Path, currentVersion: String, binaryFile: Path): Boolean {
        if (!Files.exists(versionFile)) return false
        return try {
            val stored = Files.readString(versionFile).trim()
            val parts = stored.split(":", limit = 2)
            val storedVersion = parts[0]

            if (storedVersion != currentVersion) return false

            if (parts.size < 2) {
                log.debug("Version file missing size component, forcing re-extraction")
                return false
            }

            val expectedSize = parts[1].toLongOrNull()
            if (expectedSize == null) {
                log.debug("Version file has invalid size component: ${parts[1]}")
                return false
            }

            val actualSize = Files.size(binaryFile)
            if (actualSize != expectedSize) {
                log.warn("Binary size mismatch (expected: $expectedSize, actual: $actualSize), forcing re-extraction")
                return false
            }

            true
        } catch (e: IOException) {
            log.debug("Could not read version file: ${e.message}")
            false
        }
    }

    /**
     * Shows a notification when the LSP binary is not found.
     */
    private fun showLspNotFoundNotification() {
        try {
            NotificationGroupManager.getInstance()
                .getNotificationGroup("Piko")
                .createNotification(
                    "Piko LSP Not Found",
                    """
                    The piko-lsp binary could not be found.

                    Template intelligence features will be unavailable.
                    Go/CSS/TypeScript support via language injection will still work.

                    To fix this:
                    1. Install piko-lsp: go install piko.sh/piko/cmd/lsp@latest
                    2. Or set a custom path in Settings > Languages & Frameworks > Piko
                    """.trimIndent(),
                    NotificationType.WARNING
                )
                .notify(project)
        } catch (e: IllegalStateException) {
            log.warn("piko-lsp binary not found (notification failed: ${e.message}). " +
                "Install with: go install piko.sh/piko/cmd/lsp@latest")
        }
    }

    /**
     * Shows an error notification when the LSP fails to start.
     *
     * @param error The error message to display.
     */
    private fun showStartupErrorNotification(error: String) {
        try {
            NotificationGroupManager.getInstance()
                .getNotificationGroup("Piko")
                .createNotification(
                    "Piko LSP Startup Failed",
                    "Failed to start the Piko language server: $error",
                    NotificationType.ERROR
                )
                .notify(project)
        } catch (e: IllegalStateException) {
            log.error("Failed to start Piko LSP: $error (notification failed: ${e.message})")
        }
    }
}

/**
 * Matches files that should use the Piko LSP.
 *
 * Used by LSP4IJ to determine which files should be handled by this language server.
 */
object PikoFileMatcher {

    /**
     * Checks if a file should use the Piko LSP.
     *
     * @param file The file to check.
     * @return True if the file is a PK or PKC file.
     */
    fun matches(file: VirtualFile): Boolean {
        return file.extension == PKFileType.defaultExtension ||
               file.extension == "pkc"
    }
}
