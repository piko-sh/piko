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

import com.intellij.execution.ExecutionException
import com.intellij.execution.configurations.GeneralCommandLine
import com.intellij.execution.process.OSProcessHandler
import com.intellij.execution.process.ProcessEvent
import com.intellij.execution.process.ProcessListener
import com.intellij.execution.process.ProcessOutputTypes
import com.intellij.notification.NotificationGroupManager
import com.intellij.notification.NotificationType
import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.project.Project
import com.intellij.openapi.util.Key
import com.intellij.openapi.vfs.VirtualFile
import com.intellij.util.EnvironmentUtil
import com.redhat.devtools.lsp4ij.LanguageServerEnablementSupport
import com.redhat.devtools.lsp4ij.LanguageServerFactory
import com.redhat.devtools.lsp4ij.client.LanguageClientImpl
import com.redhat.devtools.lsp4ij.server.CannotStartProcessException
import com.redhat.devtools.lsp4ij.server.LanguageServerLogErrorHandler
import com.redhat.devtools.lsp4ij.server.OSProcessStreamConnectionProvider
import com.redhat.devtools.lsp4ij.server.ProcessDataProvider
import com.redhat.devtools.lsp4ij.server.StreamConnectionProvider
import io.politepixels.piko.pk.PKFileType
import io.politepixels.piko.settings.GoPathResolver
import io.politepixels.piko.settings.PikoSettings
import java.io.File
import java.io.IOException
import java.io.InputStream
import java.io.OutputStream
import java.net.InetAddress
import java.net.InetSocketAddress
import java.net.ServerSocket
import java.net.Socket
import java.nio.charset.StandardCharsets
import java.nio.file.Files
import java.util.concurrent.CopyOnWriteArrayList

/**
 * Factory for creating Piko LSP server connections.
 *
 * Registered with LSP4IJ to launch and manage the Piko language server
 * for template block code intelligence. Also reports the enabled state to
 * LSP4IJ, so a disabled server is simply not started instead of failing.
 */
class PikoLanguageServerFactory : LanguageServerFactory, LanguageServerEnablementSupport {

    /**
     * Creates a connection provider for the Piko LSP.
     *
     * @param project The current project.
     * @return A stream connection provider for LSP communication.
     */
    override fun createConnectionProvider(project: Project): StreamConnectionProvider {
        return PikoLspConnectionProvider(project)
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

    /**
     * Reports whether the Piko LSP is enabled.
     *
     * @param project The current project.
     * @return True when LSP support is enabled in settings.
     */
    override fun isEnabled(project: Project): Boolean = PikoSettings.getInstance().lspEnabled

    /**
     * Records an enable or disable request made through LSP4IJ.
     *
     * @param enabled Whether the server should be enabled.
     * @param project The current project.
     */
    override fun setEnabled(enabled: Boolean, project: Project) {
        PikoSettings.getInstance().lspEnabled = enabled
    }
}

/**
 * Provides the connection to the Piko LSP server.
 *
 * Chooses the transport when LSP4IJ starts the server:
 * 1. An already-running external server, when one is configured and reachable.
 * 2. A child process over stdio (the default), launched through LSP4IJ's
 *    [OSProcessStreamConnectionProvider] so stderr reaches the LSP console.
 * 3. A child process over loopback TCP, when TCP transport is enabled for debugging.
 *
 * @param project The current project.
 */
class PikoLspConnectionProvider(private val project: Project) : StreamConnectionProvider, ProcessDataProvider {

    /** Logger for this connection provider. */
    private val log = Logger.getInstance(PikoLspConnectionProvider::class.java)

    /** Error handlers registered by LSP4IJ, forwarded to the active connection. */
    private val logErrorHandlers = CopyOnWriteArrayList<LanguageServerLogErrorHandler>()

    /** Unexpected-stop handlers registered by LSP4IJ, forwarded to the active connection. */
    private val unexpectedStopHandlers = CopyOnWriteArrayList<Runnable>()

    /** The connection chosen at start. */
    @Volatile
    private var delegate: StreamConnectionProvider? = null

    /**
     * Selects a transport and starts the connection.
     *
     * @throws CannotStartProcessException When the binary is missing or the server cannot be reached.
     */
    override fun start() {
        val settings = PikoSettings.getInstance()
        val connection = selectConnection(settings)
        logErrorHandlers.forEach { connection.addLogErrorHandler(it) }
        unexpectedStopHandlers.forEach { connection.addUnexpectedServerStopHandler(it) }
        delegate = connection
        try {
            connection.start()
        } catch (e: CannotStartProcessException) {
            showStartupErrorNotification(e.message ?: "unknown error")
            throw e
        }
    }

    /**
     * Chooses the connection for the current settings.
     *
     * @param settings The current plugin settings.
     * @return The connection to start.
     * @throws CannotStartProcessException When no binary can be found.
     */
    private fun selectConnection(settings: PikoSettings): StreamConnectionProvider {
        if (settings.useTcpMode) {
            val external = PikoExternalSocketConnection(settings.tcpHost, settings.tcpPort)
            if (external.tryConnect()) {
                log.info("Piko LSP connected to external server at ${settings.tcpHost}:${settings.tcpPort}")
                return external
            }
            log.info("No external Piko LSP at ${settings.tcpHost}:${settings.tcpPort}, launching one instead")
        }

        if (settings.useTcpTransport) {
            val transport = PikoLspTransport.Tcp(LOOPBACK_HOST, findFreePort())
            return PikoTcpProcessConnection(buildCommandLine(settings, transport), transport)
        }
        return PikoStdioProcessConnection(buildCommandLine(settings, PikoLspTransport.Stdio))
    }

    /**
     * Builds the command line for a server the plugin launches.
     *
     * @param settings The current plugin settings.
     * @param transport The transport to launch with.
     * @return The configured command line.
     * @throws CannotStartProcessException When no binary can be found.
     */
    private fun buildCommandLine(settings: PikoSettings, transport: PikoLspTransport): GeneralCommandLine {
        val binary = PikoLspBinaryLocator.locate(settings.lspPath)
        if (binary == null) {
            if (settings.showLspNotFoundNotification) {
                showLspNotFoundNotification()
            }
            throw CannotStartProcessException(
                "pikopls binary not found. Install it with: go install piko.sh/piko/cmd/pikopls@latest"
            )
        }

        val logFile = if (settings.fileLoggingEnabled) prepareLogFile() else null
        val options = PikoLspLaunchOptions(
            formatting = settings.formattingEnabled,
            logFile = logFile,
            pprof = settings.pprofEnabled,
            pprofPort = settings.pprofPort
        )

        val commandLine = GeneralCommandLine(listOf(binary.toString()) + PikoLspArguments.build(options, transport))
            .withParentEnvironmentType(GeneralCommandLine.ParentEnvironmentType.CONSOLE)
            .withCharset(StandardCharsets.UTF_8)
            .withEnvironment(DISABLE_CONSOLE_LOG_VARIABLE, "true")

        val goBinPath = GoPathResolver.resolve(project, settings)
        if (goBinPath != null) {
            val currentPath = EnvironmentUtil.getValue("PATH").orEmpty()
            commandLine.withEnvironment("PATH", "$goBinPath${File.pathSeparator}$currentPath")
            log.info("Set Go bin path in LSP environment: $goBinPath")
        }

        project.basePath?.let { commandLine.withWorkDirectory(it) }
        log.info("Piko LSP command line: ${commandLine.commandLineString}")
        return commandLine
    }

    /**
     * Ensures the log directory exists and returns the log file path.
     *
     * @return The log file path, or null when the directory cannot be created.
     */
    private fun prepareLogFile(): String? {
        val path = PikoLspLogFile.forProject(project)
        return try {
            Files.createDirectories(path.parent)
            path.toString()
        } catch (e: IOException) {
            log.warn("Cannot create Piko LSP log directory ${path.parent}: ${e.message}")
            null
        }
    }

    /**
     * Returns the input stream for reading LSP responses.
     *
     * @return The active connection's input stream, or null before start.
     */
    override fun getInputStream(): InputStream? = delegate?.inputStream

    /**
     * Returns the output stream for sending LSP requests.
     *
     * @return The active connection's output stream, or null before start.
     */
    override fun getOutputStream(): OutputStream? = delegate?.outputStream

    /**
     * Returns the server initialisation options.
     *
     * @param rootUri The workspace root, unused.
     * @return The options object, including `goBridge`.
     */
    override fun getInitializationOptions(rootUri: VirtualFile?): Any = PikoLspInitialisationOptions.build()

    /**
     * Registers a handler for server error output.
     *
     * @param handler The handler to register.
     */
    override fun addLogErrorHandler(handler: LanguageServerLogErrorHandler) {
        logErrorHandlers.add(handler)
        delegate?.addLogErrorHandler(handler)
    }

    /**
     * Registers a handler for unexpected server termination.
     *
     * @param handler The handler to register.
     */
    override fun addUnexpectedServerStopHandler(handler: Runnable) {
        unexpectedStopHandlers.add(handler)
        delegate?.addUnexpectedServerStopHandler(handler)
    }

    /**
     * Reports whether the connection is alive.
     *
     * @return True when the active connection is alive.
     */
    override fun isAlive(): Boolean = delegate?.isAlive ?: false

    /**
     * Throws when the connection has died.
     *
     * @throws CannotStartProcessException When the connection is not alive.
     */
    override fun ensureIsAlive() {
        val connection = delegate ?: throw CannotStartProcessException("Piko LSP has not been started")
        connection.ensureIsAlive()
    }

    /**
     * Returns the server process identifier.
     *
     * @return The process identifier, or null for an external server.
     */
    override fun getPid(): Long? = (delegate as? ProcessDataProvider)?.pid

    /**
     * Returns the server command line.
     *
     * @return The command and arguments, or an empty list for an external server.
     */
    override fun getCommands(): List<String> = (delegate as? ProcessDataProvider)?.commands ?: emptyList()

    /**
     * Stops the active connection and its process.
     */
    override fun stop() {
        delegate?.stop()
        delegate = null
    }

    /**
     * Shows a notification when the LSP binary is not found.
     */
    private fun showLspNotFoundNotification() {
        notify(
            "Piko LSP Not Found",
            """
            The pikopls binary could not be found.

            Template intelligence features will be unavailable.
            Go/CSS/TypeScript support via language injection will still work.

            To fix this:
            1. Install pikopls: go install piko.sh/piko/cmd/pikopls@latest
            2. Or set a custom path in Settings > Languages & Frameworks > Piko
            """.trimIndent(),
            NotificationType.WARNING
        )
    }

    /**
     * Shows an error notification when the LSP fails to start.
     *
     * @param error The error message to display.
     */
    private fun showStartupErrorNotification(error: String) {
        notify("Piko LSP Startup Failed", "Failed to start the Piko language server: $error", NotificationType.ERROR)
    }

    /**
     * Shows a balloon in the Piko notification group.
     *
     * @param title The notification title.
     * @param content The notification body.
     * @param type The notification type.
     */
    private fun notify(title: String, content: String, type: NotificationType) {
        try {
            NotificationGroupManager.getInstance()
                .getNotificationGroup("Piko")
                .createNotification(title, content, type)
                .notify(project)
        } catch (e: IllegalStateException) {
            log.warn("$title: $content (notification failed: ${e.message})")
        }
    }

    companion object {
        /** Loopback address used for a launched TCP server. */
        private const val LOOPBACK_HOST = "127.0.0.1"

        /** Environment variable that stops pikopls writing logs to its console. */
        private const val DISABLE_CONSOLE_LOG_VARIABLE = "PIKO_DISABLE_CONSOLE_LOG"

        /**
         * Finds a free loopback port by briefly opening a server socket.
         *
         * @return An available port number.
         * @throws CannotStartProcessException When no port can be reserved.
         */
        private fun findFreePort(): Int {
            return try {
                ServerSocket(0, 1, InetAddress.getByName(LOOPBACK_HOST)).use { serverSocket -> serverSocket.localPort }
            } catch (e: IOException) {
                throw CannotStartProcessException(IOException("Cannot reserve a loopback port for pikopls", e))
            }
        }
    }
}

/**
 * Runs pikopls as a child process over stdio.
 *
 * @param commandLine The command line to launch.
 */
internal class PikoStdioProcessConnection(commandLine: GeneralCommandLine) : OSProcessStreamConnectionProvider(commandLine)

/**
 * Runs pikopls as a child process listening on a loopback TCP port.
 *
 * @param commandLine The command line to launch.
 * @param transport The TCP endpoint the server listens on.
 */
internal class PikoTcpProcessConnection(
    private val commandLine: GeneralCommandLine,
    private val transport: PikoLspTransport.Tcp
) : StreamConnectionProvider, ProcessDataProvider {

    /** Logger for this connection. */
    private val log = Logger.getInstance(PikoTcpProcessConnection::class.java)

    /** Error handlers registered by LSP4IJ. */
    private val logErrorHandlers = CopyOnWriteArrayList<LanguageServerLogErrorHandler>()

    /** Unexpected-stop handlers registered by LSP4IJ. */
    private val unexpectedStopHandlers = CopyOnWriteArrayList<Runnable>()

    /** The running process handler. */
    @Volatile
    private var processHandler: OSProcessHandler? = null

    /** The socket connected to the server. */
    @Volatile
    private var socket: Socket? = null

    /** Whether [stop] has been called, so a later exit is expected. */
    @Volatile
    private var stopped = false

    /**
     * Launches the process and connects to it.
     *
     * @throws CannotStartProcessException When the process cannot be launched or reached.
     */
    override fun start() {
        val handler = try {
            OSProcessHandler(commandLine)
        } catch (e: ExecutionException) {
            throw CannotStartProcessException(e)
        }
        handler.addProcessListener(object : ProcessListener {
            override fun onTextAvailable(event: ProcessEvent, outputType: Key<*>) {
                if (outputType == ProcessOutputTypes.STDERR) {
                    logErrorHandlers.forEach { it.logError(event.text) }
                }
            }

            override fun processTerminated(event: ProcessEvent) {
                if (!stopped) {
                    unexpectedStopHandlers.forEach { it.run() }
                }
            }
        })
        handler.startNotify()
        processHandler = handler

        val connected = waitAndConnect(handler)
        if (connected == null) {
            val exitCode = handler.exitCode
            handler.destroyProcess()
            throw CannotStartProcessException(
                "Started pikopls but could not connect to ${transport.host}:${transport.port} (exit code: $exitCode)"
            )
        }
        socket = connected
    }

    /**
     * Polls the server port until it accepts a connection or the process exits.
     *
     * @param handler The running process handler.
     * @return The connected socket, or null on failure.
     */
    private fun waitAndConnect(handler: OSProcessHandler): Socket? {
        repeat(MAX_CONNECTION_ATTEMPTS) { attempt ->
            if (handler.isProcessTerminated) {
                log.warn("pikopls exited during startup")
                return null
            }
            connectSocket(transport.host, transport.port)?.let {
                log.info("Connected to pikopls on attempt ${attempt + 1}")
                return it
            }
            try {
                Thread.sleep(CONNECTION_RETRY_DELAY_MS)
            } catch (_: InterruptedException) {
                Thread.currentThread().interrupt()
                return null
            }
        }
        return null
    }

    /**
     * Returns the socket input stream.
     *
     * @return The input stream, or null before connection.
     */
    override fun getInputStream(): InputStream? = socket?.getInputStream()

    /**
     * Returns the socket output stream.
     *
     * @return The output stream, or null before connection.
     */
    override fun getOutputStream(): OutputStream? = socket?.getOutputStream()

    /**
     * Registers a handler for server error output.
     *
     * @param handler The handler to register.
     */
    override fun addLogErrorHandler(handler: LanguageServerLogErrorHandler) {
        logErrorHandlers.add(handler)
    }

    /**
     * Registers a handler for unexpected server termination.
     *
     * @param handler The handler to register.
     */
    override fun addUnexpectedServerStopHandler(handler: Runnable) {
        unexpectedStopHandlers.add(handler)
    }

    /**
     * Reports whether the process is running and the socket is open.
     *
     * @return True when alive.
     */
    override fun isAlive(): Boolean {
        val handler = processHandler ?: return false
        val connected = socket ?: return false
        return !handler.isProcessTerminated && !connected.isClosed
    }

    /**
     * Throws when the process has died.
     *
     * @throws CannotStartProcessException When the connection is not alive.
     */
    override fun ensureIsAlive() {
        if (!isAlive) {
            throw CannotStartProcessException("pikopls is not running: ${commandLine.commandLineString}")
        }
    }

    /**
     * Returns the server process identifier.
     *
     * @return The process identifier, or null before launch.
     */
    override fun getPid(): Long? = processHandler?.process?.pid()

    /**
     * Returns the server command line.
     *
     * @return The command and arguments.
     */
    override fun getCommands(): List<String> = commandLine.getCommandLineList(null)

    /**
     * Closes the socket and terminates the process.
     */
    override fun stop() {
        stopped = true
        try {
            socket?.close()
        } catch (e: IOException) {
            log.debug("Error closing pikopls socket: ${e.message}")
        }
        socket = null
        processHandler?.destroyProcess()
        processHandler = null
    }

    companion object {
        /** Maximum number of connection attempts before giving up. */
        private const val MAX_CONNECTION_ATTEMPTS = 50

        /** Delay in milliseconds between connection retry attempts. */
        private const val CONNECTION_RETRY_DELAY_MS = 100L
    }
}

/**
 * Connects to a pikopls server started outside the IDE, for debugging.
 *
 * @param host The server host.
 * @param port The server port.
 */
internal class PikoExternalSocketConnection(private val host: String, private val port: Int) : StreamConnectionProvider {

    /** The connected socket. */
    @Volatile
    private var socket: Socket? = null

    /**
     * Attempts to connect, bounded by a connect timeout.
     *
     * @return True when connected.
     */
    fun tryConnect(): Boolean {
        socket = connectSocket(host, port)
        return socket != null
    }

    /**
     * Connects when [tryConnect] has not already succeeded.
     *
     * @throws CannotStartProcessException When the server cannot be reached.
     */
    override fun start() {
        if (socket == null && !tryConnect()) {
            throw CannotStartProcessException("Cannot connect to Piko LSP at $host:$port")
        }
    }

    /**
     * Returns the socket input stream.
     *
     * @return The input stream, or null before connection.
     */
    override fun getInputStream(): InputStream? = socket?.getInputStream()

    /**
     * Returns the socket output stream.
     *
     * @return The output stream, or null before connection.
     */
    override fun getOutputStream(): OutputStream? = socket?.getOutputStream()

    /**
     * Reports whether the socket is open.
     *
     * @return True when connected.
     */
    override fun isAlive(): Boolean = socket?.let { it.isConnected && !it.isClosed } ?: false

    /**
     * Closes the socket. The external server keeps running.
     */
    override fun stop() {
        try {
            socket?.close()
        } catch (e: IOException) {
            socketLog.debug("Error closing external Piko LSP socket: ${e.message}")
        }
        socket = null
    }
}

/** Timeout in milliseconds for establishing a TCP connection to pikopls. */
private const val SOCKET_CONNECT_TIMEOUT_MS = 2000

/** Logger for socket helpers. */
private val socketLog = Logger.getInstance("io.politepixels.piko.pk.lsp.PikoLspSocket")

/**
 * Opens a TCP connection with a bounded connect timeout.
 *
 * @param host The host to connect to.
 * @param port The port to connect to.
 * @return The connected socket, or null when the connection fails.
 */
private fun connectSocket(host: String, port: Int): Socket? {
    val candidate = Socket()
    return try {
        candidate.connect(InetSocketAddress(host, port), SOCKET_CONNECT_TIMEOUT_MS)
        candidate.tcpNoDelay = true
        candidate
    } catch (e: IOException) {
        socketLog.debug("TCP connection to $host:$port failed: ${e.message}")
        candidate.close()
        null
    } catch (e: SecurityException) {
        socketLog.debug("TCP connection to $host:$port denied: ${e.message}")
        candidate.close()
        null
    } catch (e: IllegalArgumentException) {
        socketLog.debug("Invalid TCP endpoint $host:$port: ${e.message}")
        candidate.close()
        null
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
