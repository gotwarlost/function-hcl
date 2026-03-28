package crossplanecontrib.functionhcl

import crossplanecontrib.functionhcl.binary.BinaryPathResolver
import crossplanecontrib.functionhcl.settings.FunctionHclConfigurable
import com.intellij.notification.Notification
import com.intellij.notification.NotificationAction
import com.intellij.notification.NotificationType
import com.intellij.ide.plugins.PluginManagerCore
import com.intellij.openapi.extensions.PluginId
import com.intellij.openapi.options.ShowSettingsUtil
import com.intellij.openapi.progress.ProcessCanceledException
import com.intellij.openapi.project.Project
import com.redhat.devtools.lsp4ij.LanguageServerFactory
import com.redhat.devtools.lsp4ij.client.LanguageClientImpl
import com.redhat.devtools.lsp4ij.client.features.LSPClientFeatures
import com.redhat.devtools.lsp4ij.server.ProcessStreamConnectionProvider
import com.redhat.devtools.lsp4ij.server.StreamConnectionProvider
import org.eclipse.lsp4j.ClientInfo
import org.eclipse.lsp4j.InitializeParams

class FunctionHclLanguageServerFactory : LanguageServerFactory {

    companion object {
        @Volatile
        private var notificationShown = false

        fun resetNotificationState() {
            notificationShown = false
        }
    }

    override fun createConnectionProvider(project: Project): StreamConnectionProvider {
        val binaryPath = BinaryPathResolver.resolve()
        if (binaryPath != null) {
            return FunctionHclStreamConnectionProvider(binaryPath)
        }

        showBinaryNotFoundNotification(project)
        throw ProcessCanceledException()
    }

    override fun createLanguageClient(project: Project): LanguageClientImpl {
        return LanguageClientImpl(project)
    }

    override fun createClientFeatures(): LSPClientFeatures {
        return object : LSPClientFeatures() {
            override fun initializeParams(initializeParams: InitializeParams) {
                val lsp4ijVersion = PluginManagerCore.getPlugin(
                    PluginId.getId("com.redhat.devtools.lsp4ij")
                )?.version ?: "unknown"
                initializeParams.clientInfo = ClientInfo(
                    "function-hcl-intellij",
                    "0.0.1; lsp4ij $lsp4ijVersion"
                )
            }
        }
    }

    private fun showBinaryNotFoundNotification(project: Project) {
        if (notificationShown) return
        notificationShown = true

        val notification = Notification(
            "Function HCL Language Server",
            FunctionHclBundle.message("notification.binary.notFound.title"),
            FunctionHclBundle.message("notification.binary.notFound.message"),
            NotificationType.WARNING
        )
        notification.addAction(
            NotificationAction.createSimple(FunctionHclBundle.message("notification.binary.notFound.configure")) {
                ShowSettingsUtil.getInstance().showSettingsDialog(project, FunctionHclConfigurable::class.java)
            }
        )
        notification.notify(project)
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
