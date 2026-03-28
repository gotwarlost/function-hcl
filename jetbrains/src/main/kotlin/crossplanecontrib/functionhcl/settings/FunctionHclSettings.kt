package crossplanecontrib.functionhcl.settings

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.components.PersistentStateComponent
import com.intellij.openapi.components.State
import com.intellij.openapi.components.Storage
import com.intellij.util.xmlb.XmlSerializerUtil

/**
 * Application-level settings for the Function HCL plugin.
 * Settings are persisted across IDE restarts in applicationConfig.xml.
 */
@State(
    name = "FunctionHclSettings",
    storages = [Storage("functionHclSettings.xml")]
)
class FunctionHclSettings : PersistentStateComponent<FunctionHclSettingsState> {
    private var myState = FunctionHclSettingsState()

    override fun getState(): FunctionHclSettingsState = myState

    override fun loadState(state: FunctionHclSettingsState) {
        XmlSerializerUtil.copyBean(state, myState)
    }

    companion object {
        fun getInstance(): FunctionHclSettings {
            return ApplicationManager.getApplication().getService(FunctionHclSettings::class.java)
        }
    }
}
