package crossplanecontrib.functionhcl.binary

import crossplanecontrib.functionhcl.settings.FunctionHclSettings
import com.intellij.openapi.diagnostic.logger
import java.nio.file.Path
import kotlin.io.path.exists
import kotlin.io.path.isExecutable
import kotlin.io.path.isRegularFile

/**
 * Resolves the path to the language server binary using a priority-based approach:
 * 1. Custom path from IntelliJ settings
 * 2. Environment variable FUNCTION_HCL_LS_PATH
 * 3. Cached binary downloaded from GitHub releases
 */
object BinaryPathResolver {
    private val log = logger<BinaryPathResolver>()

    /**
     * Resolves the path to the language server binary.
     *
     * @return The resolved path as a string, or null if no valid binary is found
     */
    fun resolve(): String? {
        // Priority 1: Check settings for custom path
        val settingsPath = FunctionHclSettings.getInstance().state.languageServerPath
        if (settingsPath.isNotBlank()) {
            val path = Path.of(settingsPath)
            if (validatePath(path, "settings")) {
                log.info("Using language server from settings: $settingsPath")
                return settingsPath
            }
            log.warn("Configured language server path is invalid: $settingsPath")
        }

        // Priority 2: Check environment variable
        val envPath = System.getenv("FUNCTION_HCL_LS_PATH")
        if (!envPath.isNullOrBlank()) {
            val path = Path.of(envPath)
            if (validatePath(path, "environment variable FUNCTION_HCL_LS_PATH")) {
                log.info("Using language server from FUNCTION_HCL_LS_PATH: $envPath")
                return envPath
            }
            log.warn("FUNCTION_HCL_LS_PATH is set but invalid: $envPath")
        }

        // Priority 3: Cached download
        val cachedPath = BinaryDownloader.resolveCache()
        if (cachedPath != null) {
            log.info("Using cached language server: $cachedPath")
            return cachedPath.toString()
        }

        log.info("No language server binary found")
        return null
    }

    private fun validatePath(path: Path, source: String): Boolean {
        if (!path.exists()) {
            log.warn("Path from $source does not exist: $path")
            return false
        }
        if (!path.isRegularFile()) {
            log.warn("Path from $source is not a regular file: $path")
            return false
        }
        if (!path.isExecutable()) {
            log.warn("Path from $source is not executable: $path")
            return false
        }
        return true
    }
}
