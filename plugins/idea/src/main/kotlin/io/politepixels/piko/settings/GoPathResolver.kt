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

import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.project.Project
import java.io.File

/**
 * Resolves the Go binary path using a priority chain.
 *
 * Resolution order:
 * 1. Manual override from settings (goBinPath)
 * 2. IDE detection (GoLand/IntelliJ Go plugin SDK)
 * 3. Global fallback locations (common installation paths)
 * 4. System PATH (return null, let the process inherit PATH)
 */
object GoPathResolver {

    /** Logger for this resolver. */
    private val log = Logger.getInstance(GoPathResolver::class.java)

    /** Common Go installation locations on Linux and macOS. */
    private val LINUX_MACOS_LOCATIONS = listOf(
        "/usr/local/go/bin",
        "/opt/go/bin",
        "${System.getProperty("user.home")}/go/bin"
    )

    /** Additional Homebrew locations on macOS. */
    private val MACOS_HOMEBREW_LOCATIONS = listOf(
        "/opt/homebrew/opt/go/bin",
        "/usr/local/opt/go/bin"
    )

    /** Common Go installation locations on Windows. */
    private val WINDOWS_LOCATIONS = listOf(
        "C:\\Go\\bin",
        "C:\\Program Files\\Go\\bin",
        "${System.getProperty("user.home")}\\go\\bin"
    )

    /**
     * Resolves the Go bin directory path.
     *
     * @param project The current project (for IDE SDK detection).
     * @param settings The plugin settings.
     * @return The resolved Go bin path, or null if not found (use system PATH).
     */
    fun resolve(project: Project?, settings: PikoSettings): String? {
        settings.goBinPath.takeIf { it.isNotBlank() }?.let { path ->
            if (isValidGoBinDirectory(path)) {
                log.info("Using manual Go bin path: $path")
                return path
            }
            log.warn("Manual Go bin path is invalid or does not contain Go binary: $path")
        }

        if (settings.detectGoFromIde && project != null) {
            detectFromIde(project)?.let { path ->
                log.info("Detected Go bin path from IDE: $path")
                return path
            }
        }

        if (settings.searchGlobalGoLocations) {
            findInGlobalLocations()?.let { path ->
                log.info("Found Go bin in global location: $path")
                return path
            }
        }

        log.debug("No specific Go bin path found, using system PATH")
        return null
    }

    /**
     * Detects the Go SDK path from IntelliJ/GoLand's Go plugin.
     *
     * Uses reflection to avoid a hard dependency on the Go plugin,
     * which may not be installed in all IntelliJ variants.
     *
     * @param project The current project.
     * @return The Go bin directory path, or null if not available.
     */
    private fun detectFromIde(project: Project): String? {
        return try {
            val goSdkServiceClass = Class.forName("com.goide.sdk.GoSdkService")
            val getInstanceMethod = goSdkServiceClass.getMethod("getInstance", Project::class.java)
            val service = getInstanceMethod.invoke(null, project)

            val getSdkMethod = goSdkServiceClass.getMethod("getSdk", Class.forName("com.intellij.openapi.module.Module"))
            val sdk = getSdkMethod.invoke(service, null as Any?) ?: return null

            val getHomePathMethod = sdk.javaClass.getMethod("getHomePath")
            val goRoot = getHomePathMethod.invoke(sdk) as? String ?: return null

            val binPath = "$goRoot/bin"
            if (isValidGoBinDirectory(binPath)) binPath else null
        } catch (e: ClassNotFoundException) {
            log.debug("Go plugin not installed: ${e.message}")
            null
        } catch (e: NoSuchMethodException) {
            log.debug("Go plugin API changed: ${e.message}")
            null
        } catch (e: Exception) {
            log.debug("Could not detect Go SDK from IDE: ${e.message}")
            null
        }
    }

    /**
     * Searches common global installation locations for Go.
     *
     * @return The first valid Go bin directory found, or null if none found.
     */
    private fun findInGlobalLocations(): String? {
        val osName = System.getProperty("os.name").lowercase()

        val locations = when {
            osName.contains("windows") -> WINDOWS_LOCATIONS
            osName.contains("mac") || osName.contains("darwin") ->
                LINUX_MACOS_LOCATIONS + MACOS_HOMEBREW_LOCATIONS
            else -> LINUX_MACOS_LOCATIONS
        }

        val goSdkLocations = findGoSdkManagerInstalls()

        return (locations + goSdkLocations).firstOrNull { isValidGoBinDirectory(it) }
    }

    /**
     * Finds Go installations from the Go SDK manager (~/sdk/go*).
     *
     * @return A list of bin directories, sorted by version (newest first).
     */
    private fun findGoSdkManagerInstalls(): List<String> {
        val home = System.getProperty("user.home")
        val sdkDir = File(home, "sdk")

        if (!sdkDir.exists() || !sdkDir.isDirectory) {
            return emptyList()
        }

        return try {
            sdkDir.listFiles { file ->
                file.isDirectory && file.name.startsWith("go")
            }?.sortedByDescending { it.name }
                ?.map { "${it.absolutePath}/bin" }
                ?: emptyList()
        } catch (e: SecurityException) {
            log.debug("Cannot access SDK directory: ${e.message}")
            emptyList()
        }
    }

    /**
     * Checks if a directory contains a valid Go binary.
     *
     * @param path The directory path to check.
     * @return True if the directory contains an executable 'go' binary.
     */
    private fun isValidGoBinDirectory(path: String): Boolean {
        val dir = File(path)
        if (!dir.exists() || !dir.isDirectory) {
            return false
        }

        val osName = System.getProperty("os.name").lowercase()
        val goBinary = if (osName.contains("windows")) {
            File(dir, "go.exe")
        } else {
            File(dir, "go")
        }

        return goBinary.exists() && goBinary.canExecute()
    }
}
