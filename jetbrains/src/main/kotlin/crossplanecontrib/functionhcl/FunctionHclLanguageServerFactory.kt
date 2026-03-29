package crossplanecontrib.functionhcl

import crossplanecontrib.functionhcl.binary.BinaryDownloader
import crossplanecontrib.functionhcl.binary.BinaryPathResolver
import crossplanecontrib.functionhcl.settings.FunctionHclConfigurable
import com.intellij.notification.Notification
import com.intellij.notification.NotificationAction
import com.intellij.notification.NotificationType
import com.intellij.ide.plugins.PluginManagerCore
import com.intellij.openapi.diagnostic.logger
import com.intellij.openapi.extensions.PluginId
import com.intellij.openapi.options.ShowSettingsUtil
import com.intellij.openapi.progress.ProcessCanceledException
import com.intellij.openapi.progress.ProgressManager
import com.intellij.openapi.progress.Task
import com.intellij.openapi.progress.ProgressIndicator
import com.intellij.openapi.project.Project
import com.redhat.devtools.lsp4ij.LanguageServerFactory
import com.redhat.devtools.lsp4ij.LanguageServerManager
import com.redhat.devtools.lsp4ij.client.LanguageClientImpl
import com.redhat.devtools.lsp4ij.client.features.LSPClientFeatures
import com.redhat.devtools.lsp4ij.server.ProcessStreamConnectionProvider
import com.redhat.devtools.lsp4ij.server.StreamConnectionProvider
import org.eclipse.lsp4j.ClientInfo
import org.eclipse.lsp4j.InitializeParams

class FunctionHclLanguageServerFactory : LanguageServerFactory {
    private val log = logger<FunctionHclLanguageServerFactory>()

    companion object {
        @Volatile
        private var downloadTriggered = false

        fun resetNotificationState() {
            downloadTriggered = false
        }
    }

    override fun createConnectionProvider(project: Project): StreamConnectionProvider {
        val binaryPath = BinaryPathResolver.resolve()
        if (binaryPath != null) {
            return FunctionHclStreamConnectionProvider(binaryPath)
        }

        // Binary not found — trigger background download if not already in progress
        if (!downloadTriggered) {
            downloadTriggered = true
            triggerBackgroundDownload(project)
        }

        throw ProcessCanceledException()
    }

    override fun createLanguageClient(project: Project): LanguageClientImpl {
        return LanguageClientImpl(project)
    }

    override fun createClientFeatures(): LSPClientFeatures {
        return object : LSPClientFeatures() {
            override fun initializeParams(initializeParams: InitializeParams) {
                val pluginVersion = PluginManagerCore.getPlugin(
                    PluginId.getId("crossplanecontrib.functionhcl")
                )?.version ?: "unknown"
                val lsp4ijVersion = PluginManagerCore.getPlugin(
                    PluginId.getId("com.redhat.devtools.lsp4ij")
                )?.version ?: "unknown"
                initializeParams.clientInfo = ClientInfo(
                    "function-hcl-intellij",
                    "$pluginVersion; lsp4ij $lsp4ijVersion"
                )
            }
        }
    }

    private fun triggerBackgroundDownload(project: Project) {
        val version = getPinnedVersion()
        log.info("Language server not found, triggering download (version: ${version.ifBlank { "latest" }})")

        ProgressManager.getInstance().run(
            object : Task.Backgroundable(
                project,
                FunctionHclBundle.message("download.progress.title"),
                true
            ) {
                override fun run(indicator: ProgressIndicator) {
                    try {
                        BinaryDownloader.download(version, indicator)

                        // Restart the language server now that the binary is available
                        downloadTriggered = false
                        LanguageServerManager.getInstance(project).start("functionHclLanguageServer")

                        Notification(
                            "Function HCL Language Server",
                            FunctionHclBundle.message("notification.download.success.title"),
                            FunctionHclBundle.message("notification.download.success"),
                            NotificationType.INFORMATION
                        ).notify(project)

                    } catch (e: Exception) {
                        log.warn("Failed to download language server", e)
                        downloadTriggered = false

                        val notification = Notification(
                            "Function HCL Language Server",
                            FunctionHclBundle.message("notification.download.failed.title"),
                            FunctionHclBundle.message("notification.download.failed", e.message ?: "Unknown error"),
                            NotificationType.ERROR
                        )
                        notification.addAction(
                            NotificationAction.createSimple(FunctionHclBundle.message("notification.binary.notFound.configure")) {
                                ShowSettingsUtil.getInstance().showSettingsDialog(project, FunctionHclConfigurable::class.java)
                            }
                        )
                        notification.notify(project)
                    }
                }
            }
        )
    }

    /**
     * Returns the pinned language server version for this plugin build.
     * For release builds, this is set at build time via the generated resource.
     * For local dev, returns blank which causes BinaryDownloader to fetch the latest.
     */
    private fun getPinnedVersion(): String {
        return try {
            val stream = javaClass.getResourceAsStream("/language-server-version.txt")
            stream?.bufferedReader()?.readText()?.trim() ?: ""
        } catch (_: Exception) {
            ""
        }
    }

    private class FunctionHclStreamConnectionProvider(
        binaryPath: String
    ) : ProcessStreamConnectionProvider() {
        init {
            setCommands(listOf(binaryPath, "serve", "--stdio"))
        }

        override fun toString(): String {
            return "Function HCL Language Server: ${super.toString()}"
        }
    }
}
