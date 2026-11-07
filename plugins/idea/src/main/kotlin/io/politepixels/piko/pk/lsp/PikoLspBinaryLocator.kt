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

import com.intellij.execution.configurations.PathEnvironmentVariableUtil
import com.intellij.openapi.application.PathManager
import com.intellij.openapi.diagnostic.Logger
import com.intellij.util.EnvironmentUtil
import java.io.File
import java.io.IOException
import java.io.InputStream
import java.net.URL
import java.nio.file.AtomicMoveNotSupportedException
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardCopyOption
import java.security.MessageDigest
import java.util.Properties

/**
 * Finds the pikopls executable.
 *
 * Search order:
 * 1. The custom path from settings.
 * 2. The binary bundled in the plugin, extracted to the IDE system directory.
 * 3. `GOBIN`, then each `GOPATH` entry's `bin` directory (default `~/go/bin`).
 * 4. `PATH`.
 */
object PikoLspBinaryLocator {

    /** Logger for binary resolution. */
    private val log = Logger.getInstance(PikoLspBinaryLocator::class.java)

    /**
     * Locates the pikopls executable.
     *
     * @param customPath The custom path from settings, or blank for none.
     * @param platform The platform to locate a binary for.
     * @return The executable path, or null when none is found.
     */
    @JvmStatic
    fun locate(customPath: String, platform: PikoLspPlatform = PikoLspPlatform.current()): Path? {
        resolveCustomPath(customPath)?.let { return it }

        PikoLspBundledBinary.extract(platform)?.let {
            log.info("Using bundled pikopls: $it")
            return it
        }

        if (!platform.isSupported) {
            log.warn("No bundled pikopls for os.name=${System.getProperty("os.name")} os.arch=${System.getProperty("os.arch")}")
        }

        findInGoBinDirectories(platform.binaryName)?.let { return it }

        return PathEnvironmentVariableUtil.findInPath(platform.binaryName)?.toPath()?.also {
            log.info("Found pikopls on PATH: $it")
        }
    }

    /**
     * Validates the custom binary path from settings.
     *
     * @param customPath The configured path.
     * @return The path when it exists and is executable, or null.
     */
    private fun resolveCustomPath(customPath: String): Path? {
        if (customPath.isBlank()) return null
        val file = File(customPath)
        if (file.isFile && file.canExecute()) {
            log.info("Using custom pikopls path: $customPath")
            return file.toPath()
        }
        log.warn("Custom pikopls path not found or not executable: $customPath")
        return null
    }

    /**
     * Searches `GOBIN` and each `GOPATH/bin` directory.
     *
     * @param binaryName The executable file name.
     * @return The first executable found, or null.
     */
    private fun findInGoBinDirectories(binaryName: String): Path? {
        return goBinDirectories()
            .map { File(it, binaryName) }
            .firstOrNull { it.isFile && it.canExecute() }
            ?.toPath()
            ?.also { log.info("Found pikopls in Go bin directory: $it") }
    }

    /**
     * Lists Go binary directories in the order `go install` would use.
     *
     * @return The candidate directories.
     */
    private fun goBinDirectories(): List<String> {
        val directories = mutableListOf<String>()
        EnvironmentUtil.getValue("GOBIN")?.takeIf { it.isNotBlank() }?.let { directories.add(it) }

        val goPath = EnvironmentUtil.getValue("GOPATH")?.takeIf { it.isNotBlank() }
        if (goPath != null) {
            goPath.split(File.pathSeparator)
                .filter { it.isNotBlank() }
                .forEach { directories.add(File(it, "bin").path) }
        } else {
            directories.add(File(System.getProperty("user.home"), "go${File.separator}bin").path)
        }
        return directories
    }
}

/**
 * Extracts the pikopls binary bundled in the plugin JAR.
 */
object PikoLspBundledBinary {

    /** Logger for extraction. */
    private val log = Logger.getInstance(PikoLspBundledBinary::class.java)

    /** Serialises extraction across every project in this JVM. */
    private val extractionLock = Any()

    /** Directory name under the IDE system directory. */
    private const val EXTRACTION_DIRECTORY = "pikopls"

    /** Directory name used by plugin versions that extracted into the plugins directory. */
    private const val LEGACY_EXTRACTION_DIRECTORY = "pikopls-bin"

    /** Resource holding the plugin version, generated at build time. */
    private const val VERSION_RESOURCE = "/piko-plugin-version.properties"

    /** Buffer size for hashing and copying. */
    private const val BUFFER_SIZE = 64 * 1024

    /** Characters allowed in the version directory name. */
    private val UNSAFE_DIRECTORY_CHARACTERS = Regex("[^A-Za-z0-9._-]")

    /**
     * Extracts the bundled binary for a platform when one is bundled.
     *
     * @param platform The platform to extract for.
     * @return The extracted executable, or null when nothing is bundled or extraction fails.
     */
    @JvmStatic
    fun extract(platform: PikoLspPlatform): Path? {
        val platformDirectory = platform.directory ?: return null
        val resource = PikoLspBundledBinary::class.java.getResource("/bin/$platformDirectory/${platform.binaryName}")
        if (resource == null) {
            log.debug("No bundled pikopls for $platformDirectory")
            return null
        }

        val baseDirectory = Path.of(PathManager.getSystemPath(), EXTRACTION_DIRECTORY)
        val versionDirectory = baseDirectory.resolve(versionDirectoryName(pluginVersion()))
        val target = versionDirectory.resolve(platformDirectory).resolve(platform.binaryName)

        return try {
            synchronized(extractionLock) {
                extractIfStale(resource, target)
                removeStaleVersions(baseDirectory, versionDirectory)
            }
            target
        } catch (e: IOException) {
            log.warn("Failed to extract bundled pikopls to $target", e)
            if (Files.isRegularFile(target) && Files.isExecutable(target)) target else null
        } catch (e: SecurityException) {
            log.warn("Permission denied extracting bundled pikopls to $target", e)
            null
        }
    }

    /**
     * Converts a plugin version to a safe directory name.
     *
     * @param version The plugin version.
     * @return The directory name.
     */
    @JvmStatic
    fun versionDirectoryName(version: String): String =
        version.replace(UNSAFE_DIRECTORY_CHARACTERS, "_").ifEmpty { "unknown" }

    /**
     * Computes the hex SHA-256 of a stream.
     *
     * @param input The stream to hash; it is read to the end but not closed.
     * @return The lowercase hex digest.
     */
    @JvmStatic
    fun sha256(input: InputStream): String {
        val digest = MessageDigest.getInstance("SHA-256")
        val buffer = ByteArray(BUFFER_SIZE)
        while (true) {
            val read = input.read(buffer)
            if (read < 0) break
            digest.update(buffer, 0, read)
        }
        return digest.digest().joinToString("") { "%02x".format(it) }
    }

    /**
     * Writes the bundled resource to the target unless an identical file is already there.
     *
     * @param resource The bundled binary resource.
     * @param target The extraction target.
     */
    private fun extractIfStale(resource: URL, target: Path) {
        val bundledHash = resource.openStream().use { sha256(it) }
        if (Files.isRegularFile(target) && Files.isExecutable(target)) {
            val existingHash = Files.newInputStream(target).use { sha256(it) }
            if (existingHash == bundledHash) {
                log.debug("Bundled pikopls already extracted at $target")
                return
            }
            log.info("Extracted pikopls at $target does not match the bundled binary, replacing it")
        }

        Files.createDirectories(target.parent)
        val temporary = Files.createTempFile(target.parent, target.fileName.toString(), ".tmp")
        try {
            resource.openStream().use { input ->
                Files.copy(input, temporary, StandardCopyOption.REPLACE_EXISTING)
            }
            val writtenHash = Files.newInputStream(temporary).use { sha256(it) }
            if (writtenHash != bundledHash) {
                throw IOException("Extracted pikopls hash $writtenHash does not match bundled hash $bundledHash")
            }
            if (!temporary.toFile().setExecutable(true, false)) {
                throw IOException("Cannot mark extracted pikopls at $temporary as executable")
            }
            moveIntoPlace(temporary, target)
            log.info("Extracted bundled pikopls to $target")
        } finally {
            Files.deleteIfExists(temporary)
        }
    }

    /**
     * Moves a file into place atomically where the file system allows it.
     *
     * @param source The fully written temporary file.
     * @param target The final path.
     */
    private fun moveIntoPlace(source: Path, target: Path) {
        try {
            Files.move(source, target, StandardCopyOption.ATOMIC_MOVE, StandardCopyOption.REPLACE_EXISTING)
        } catch (_: AtomicMoveNotSupportedException) {
            Files.move(source, target, StandardCopyOption.REPLACE_EXISTING)
        }
    }

    /**
     * Deletes extraction directories left by other plugin versions.
     *
     * @param baseDirectory The extraction root.
     * @param keep The directory of the current version.
     */
    private fun removeStaleVersions(baseDirectory: Path, keep: Path) {
        val legacy = Path.of(PathManager.getPluginsPath(), LEGACY_EXTRACTION_DIRECTORY)
        deleteQuietly(legacy)

        val children = try {
            Files.list(baseDirectory).use { stream -> stream.toList() }
        } catch (e: IOException) {
            log.debug("Could not list $baseDirectory: ${e.message}")
            return
        }
        children.filter { Files.isDirectory(it) && it != keep }.forEach { deleteQuietly(it) }
    }

    /**
     * Recursively deletes a path, logging rather than failing when it cannot.
     *
     * @param path The path to delete.
     */
    private fun deleteQuietly(path: Path) {
        if (!Files.exists(path)) return
        try {
            if (!path.toFile().deleteRecursively()) {
                log.debug("Could not fully delete $path; it may still be in use")
            }
        } catch (e: SecurityException) {
            log.debug("Could not delete $path: ${e.message}")
        }
    }

    /**
     * Reads the plugin version generated at build time.
     *
     * @return The plugin version, or "unknown".
     */
    private fun pluginVersion(): String {
        return try {
            PikoLspBundledBinary::class.java.getResourceAsStream(VERSION_RESOURCE)?.use { stream ->
                Properties().apply { load(stream) }.getProperty("version")
            } ?: "unknown"
        } catch (e: IOException) {
            log.debug("Could not determine plugin version: ${e.message}")
            "unknown"
        }
    }
}
