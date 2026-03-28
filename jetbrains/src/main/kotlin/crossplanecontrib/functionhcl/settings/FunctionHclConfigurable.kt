package crossplanecontrib.functionhcl.settings

import crossplanecontrib.functionhcl.FunctionHclBundle
import crossplanecontrib.functionhcl.FunctionHclLanguageServerFactory
import crossplanecontrib.functionhcl.binary.BinaryPathResolver
import com.intellij.notification.Notification
import com.intellij.notification.NotificationType
import com.intellij.openapi.fileChooser.FileChooserDescriptorFactory
import com.intellij.openapi.options.BoundConfigurable
import com.intellij.openapi.project.ProjectManager
import com.intellij.openapi.ui.DialogPanel
import com.intellij.openapi.ui.ValidationInfo
import com.intellij.ui.dsl.builder.*
import com.redhat.devtools.lsp4ij.LanguageServerManager
import java.nio.file.Path
import kotlin.io.path.exists
import kotlin.io.path.isExecutable
import kotlin.io.path.isRegularFile

/**
 * Settings UI for the Function HCL plugin.
 * Allows users to configure a custom language server path override.
 */
class FunctionHclConfigurable : BoundConfigurable(FunctionHclBundle.message("settings.name")) {
    private val settings = FunctionHclSettings.getInstance()

    override fun createPanel(): DialogPanel = panel {
        group(FunctionHclBundle.message("settings.group.configuration")) {
            row(FunctionHclBundle.message("settings.languageServerPath.label")) {
                textFieldWithBrowseButton(
                    fileChooserDescriptor = FileChooserDescriptorFactory.createSingleFileDescriptor()
                )
                    .bindText(settings.state::languageServerPath)
                    .comment(FunctionHclBundle.message("settings.languageServerPath.comment"))
                    .validationOnApply {
                        validateServerPath(it.text)
                    }
                    .align(AlignX.FILL)
            }
        }

        group(FunctionHclBundle.message("settings.group.binaryStatus")) {
            row(FunctionHclBundle.message("settings.currentBinary.label")) {
                label(getCurrentBinaryPath())
            }

            row {
                button(FunctionHclBundle.message("settings.restartServer.button")) {
                    restartLanguageServer()
                }
            }
        }

        group(FunctionHclBundle.message("settings.group.priorityInfo")) {
            row {
                comment(FunctionHclBundle.message("settings.priorityInfo.text"))
            }
        }
    }

    private fun validateServerPath(pathString: String): ValidationInfo? {
        if (pathString.isBlank()) return null

        val path = Path.of(pathString)

        if (!path.exists()) {
            return ValidationInfo(FunctionHclBundle.message("settings.validation.fileNotFound"))
        }
        if (!path.isRegularFile()) {
            return ValidationInfo(FunctionHclBundle.message("settings.validation.fileNotFound"))
        }
        if (!path.isExecutable()) {
            return ValidationInfo(FunctionHclBundle.message("settings.validation.notExecutable")).asWarning()
        }

        return null
    }

    private fun getCurrentBinaryPath(): String {
        return BinaryPathResolver.resolve() ?: "Not found"
    }

    private fun restartLanguageServer() {
        FunctionHclLanguageServerFactory.resetNotificationState()

        val openProjects = ProjectManager.getInstance().openProjects
        for (project in openProjects) {
            if (!project.isDisposed) {
                LanguageServerManager.getInstance(project).stop("functionHclLanguageServer")
            }
        }

        val notification = Notification(
            "Function HCL Language Server",
            FunctionHclBundle.message("notification.restart.title"),
            FunctionHclBundle.message("notification.restart.success"),
            NotificationType.INFORMATION
        )
        notification.notify(null)
    }
}
