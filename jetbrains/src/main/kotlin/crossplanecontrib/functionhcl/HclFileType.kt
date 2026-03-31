package crossplanecontrib.functionhcl

import com.intellij.openapi.fileTypes.LanguageFileType
import com.intellij.openapi.util.IconLoader
import javax.swing.Icon

object HclFileType : LanguageFileType(HclLanguage) {
    private val FILE_ICON = IconLoader.getIcon("/icons/hcl-file.svg", HclFileType::class.java)

    override fun getName(): String = "FunctionHCL"
    override fun getDescription(): String = "Function HCL file"
    override fun getDefaultExtension(): String = "hcl"
    override fun getIcon(): Icon = FILE_ICON
}
