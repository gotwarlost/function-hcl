package crossplanecontrib.functionhcl

import com.intellij.openapi.fileTypes.LanguageFileType
import javax.swing.Icon

object HclFileType : LanguageFileType(HclLanguage) {
    override fun getName(): String = "FunctionHCL"
    override fun getDescription(): String = "Function HCL file"
    override fun getDefaultExtension(): String = "hcl"
    override fun getIcon(): Icon? = null
}
