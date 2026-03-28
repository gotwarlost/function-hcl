package crossplanecontrib.functionhcl.settings

/**
 * State class for Function HCL plugin settings.
 * This class holds the configuration values that are persisted across IDE restarts.
 */
data class FunctionHclSettingsState(
    /**
     * Custom path to the language server binary.
     * If empty, the plugin will use the environment variable or the bundled binary.
     */
    var languageServerPath: String = ""
)
