package crossplanecontrib.functionhcl.binary

import com.intellij.openapi.application.PathManager
import com.intellij.openapi.diagnostic.logger
import com.intellij.openapi.progress.ProgressIndicator
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URI
import java.net.URL
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardCopyOption
import java.nio.file.attribute.PosixFilePermission
import kotlin.io.path.deleteIfExists
import kotlin.io.path.exists

private const val BINARY_NAME = "function-hcl-ls"
private const val GITHUB_REPO = "crossplane-contrib/function-hcl"
private const val CACHE_DIR = "function-hcl-ls"

/**
 * Downloads the language server binary from GitHub releases.
 * The binary is cached in the IDE's plugin data directory.
 */
object BinaryDownloader {
    private val log = logger<BinaryDownloader>()

    /**
     * Returns the path where the cached binary lives (or would live).
     */
    fun getCachedBinaryPath(): Path {
        val cacheDir = Path.of(PathManager.getPluginsPath(), CACHE_DIR)
        val isWindows = System.getProperty("os.name").lowercase().startsWith("win")
        val exeName = if (isWindows) "$BINARY_NAME.exe" else BINARY_NAME
        return cacheDir.resolve(exeName)
    }

    /**
     * Returns the cached binary path if it exists and is executable, null otherwise.
     */
    fun resolveCache(): Path? {
        val path = getCachedBinaryPath()
        if (path.exists() && Files.isRegularFile(path) && Files.isExecutable(path)) {
            return path
        }
        return null
    }

    /**
     * Downloads the language server binary for the current platform.
     *
     * @param version The version to download. If blank, fetches the latest release.
     * @param indicator Optional progress indicator for reporting download progress.
     * @return The path to the downloaded binary.
     * @throws IOException if the download fails.
     */
    fun download(version: String, indicator: ProgressIndicator? = null): Path {
        val targetPath = getCachedBinaryPath()
        val parentDir = targetPath.parent
        if (!parentDir.exists()) {
            Files.createDirectories(parentDir)
        }

        val osName = System.getProperty("os.name").lowercase()
        val isWindows = osName.startsWith("win")
        val os = when {
            osName.contains("mac") || osName.contains("darwin") -> "darwin"
            osName.contains("linux") -> "linux"
            isWindows -> "windows"
            else -> throw IOException("Unsupported OS: $osName")
        }
        val arch = when (val a = System.getProperty("os.arch").lowercase()) {
            "x86_64", "amd64", "x64" -> "amd64"
            "aarch64", "arm64" -> "arm64"
            else -> throw IOException("Unsupported architecture: $a")
        }

        val resolvedVersion = version.ifBlank { fetchLatestVersion() }
        val exeName = if (isWindows) "$BINARY_NAME.exe" else BINARY_NAME
        val assetName = "$BINARY_NAME-$os-$arch.tar.gz"
        val downloadUrl = "https://github.com/$GITHUB_REPO/releases/download/v$resolvedVersion/$assetName"

        log.info("Downloading language server v$resolvedVersion for $os/$arch from $downloadUrl")
        indicator?.text = "Downloading function-hcl language server v$resolvedVersion..."
        indicator?.isIndeterminate = true

        val tempFile = Files.createTempFile("function-hcl-ls-", ".tar.gz")
        try {
            downloadFile(URI(downloadUrl).toURL(), tempFile, indicator)

            // Extract the binary from the tarball using ProcessBuilder (tar is available on all platforms)
            val extractDir = Files.createTempDirectory("function-hcl-ls-extract-")
            try {
                val process = ProcessBuilder("tar", "-xzf", tempFile.toString(), "-C", extractDir.toString(), exeName)
                    .redirectErrorStream(true)
                    .start()
                val output = process.inputStream.bufferedReader().readText()
                val exitCode = process.waitFor()
                if (exitCode != 0) {
                    throw IOException("Failed to extract tarball (exit code $exitCode): $output")
                }

                val extractedBinary = extractDir.resolve(exeName)
                if (!extractedBinary.exists()) {
                    throw IOException("Binary '$exeName' not found in archive")
                }

                Files.move(extractedBinary, targetPath, StandardCopyOption.REPLACE_EXISTING)
            } finally {
                extractDir.toFile().deleteRecursively()
            }
        } finally {
            tempFile.deleteIfExists()
        }

        // Make executable on non-Windows
        if (!isWindows) {
            try {
                Files.setPosixFilePermissions(targetPath, setOf(
                    PosixFilePermission.OWNER_READ, PosixFilePermission.OWNER_WRITE, PosixFilePermission.OWNER_EXECUTE,
                    PosixFilePermission.GROUP_READ, PosixFilePermission.GROUP_EXECUTE,
                    PosixFilePermission.OTHERS_READ, PosixFilePermission.OTHERS_EXECUTE,
                ))
            } catch (_: UnsupportedOperationException) {
                // POSIX permissions not supported
            }
        }

        // Remove macOS quarantine attribute
        if (os == "darwin") {
            try {
                ProcessBuilder("xattr", "-d", "com.apple.quarantine", targetPath.toString())
                    .redirectErrorStream(true).start().waitFor()
            } catch (_: Exception) {
                // Not critical
            }
        }

        log.info("Language server downloaded to: $targetPath")
        return targetPath
    }

    /**
     * Deletes the cached binary.
     */
    fun deleteCache(): Boolean {
        return getCachedBinaryPath().deleteIfExists()
    }

    private fun fetchLatestVersion(): String {
        val url = URI("https://api.github.com/repos/$GITHUB_REPO/releases/latest").toURL()
        val conn = (url.openConnection() as HttpURLConnection).apply {
            requestMethod = "GET"
            connectTimeout = 15_000
            readTimeout = 30_000
            setRequestProperty("Accept", "application/vnd.github.v3+json")
        }
        try {
            val code = conn.responseCode
            if (code != 200) {
                val body = conn.errorStream?.bufferedReader()?.readText() ?: ""
                throw IOException("GitHub API returned HTTP $code: $body")
            }
            val json = conn.inputStream.bufferedReader().readText()
            return Regex(""""tag_name"\s*:\s*"v([^"]+)"""").find(json)
                ?.groupValues?.get(1) ?: throw IOException("Could not find tag_name in GitHub API response")
        } finally {
            conn.disconnect()
        }
    }

    private fun downloadFile(url: URL, target: Path, indicator: ProgressIndicator?) {
        val conn = (url.openConnection() as HttpURLConnection).apply {
            connectTimeout = 15_000
            readTimeout = 60_000
            instanceFollowRedirects = true
        }
        try {
            val code = conn.responseCode
            if (code != 200) {
                val body = conn.errorStream?.bufferedReader()?.readText() ?: ""
                throw IOException("Download failed with HTTP $code for $url: $body")
            }
            val contentLength = conn.contentLengthLong
            if (contentLength > 0) {
                indicator?.isIndeterminate = false
            }
            conn.inputStream.use { input ->
                Files.newOutputStream(target).use { output ->
                    val buffer = ByteArray(8192)
                    var totalRead = 0L
                    var bytesRead: Int
                    while (input.read(buffer).also { bytesRead = it } != -1) {
                        if (indicator?.isCanceled == true) {
                            throw IOException("Download cancelled")
                        }
                        output.write(buffer, 0, bytesRead)
                        totalRead += bytesRead
                        if (contentLength > 0) {
                            indicator?.fraction = totalRead.toDouble() / contentLength.toDouble()
                        }
                    }
                }
            }
        } finally {
            conn.disconnect()
        }
    }
}
