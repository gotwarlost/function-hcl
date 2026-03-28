package crossplanecontrib.functionhcl

import com.intellij.lang.Language
import com.intellij.psi.codeStyle.CommonCodeStyleSettings
import com.intellij.psi.codeStyle.LanguageCodeStyleSettingsProvider

class HclCodeStyleSettingsProvider : LanguageCodeStyleSettingsProvider() {
    override fun getLanguage(): Language = HclLanguage

    override fun customizeDefaults(
        commonSettings: CommonCodeStyleSettings,
        indentOptions: CommonCodeStyleSettings.IndentOptions
    ) {
        indentOptions.INDENT_SIZE = 2
        indentOptions.TAB_SIZE = 2
        indentOptions.USE_TAB_CHARACTER = false
    }

    override fun getCodeSample(settingsType: SettingsType): String =
        """
        resource "aws_instance" "example" {
          ami           = "ami-0c55b159cbfafe1f0"
          instance_type = "t2.micro"

          tags = {
            Name = "example"
          }
        }
        """.trimIndent()
}
