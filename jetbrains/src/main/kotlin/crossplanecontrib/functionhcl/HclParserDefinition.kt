package crossplanecontrib.functionhcl

import com.intellij.extapi.psi.PsiFileBase
import com.intellij.lang.ASTNode
import com.intellij.lang.ParserDefinition
import com.intellij.lang.PsiBuilder
import com.intellij.lang.PsiParser
import com.intellij.lexer.LexerBase
import com.intellij.lexer.Lexer
import com.intellij.openapi.fileTypes.FileType
import com.intellij.openapi.project.Project
import com.intellij.psi.FileViewProvider
import com.intellij.psi.PsiElement
import com.intellij.psi.PsiFile
import com.intellij.psi.TokenType
import com.intellij.psi.impl.source.tree.LeafPsiElement
import com.intellij.psi.tree.IElementType
import com.intellij.psi.tree.IFileElementType
import com.intellij.psi.tree.TokenSet

private val HCL_TOKEN = IElementType("HCL_TOKEN", HclLanguage)

class HclParserDefinition : ParserDefinition {

    companion object {
        private val FILE_ELEMENT_TYPE = IFileElementType("FunctionHCL", HclLanguage)
    }

    override fun createLexer(project: Project): Lexer = HclWordLexer()

    override fun createParser(project: Project): PsiParser = HclParser()

    override fun getFileNodeType(): IFileElementType = FILE_ELEMENT_TYPE

    override fun getCommentTokens(): TokenSet = TokenSet.EMPTY

    override fun getStringLiteralElements(): TokenSet = TokenSet.EMPTY

    override fun createElement(node: ASTNode): PsiElement = LeafPsiElement(node.elementType, node.text)

    override fun createFile(viewProvider: FileViewProvider): PsiFile = HclPsiFile(viewProvider)

    override fun spaceExistenceTypeBetweenTokens(
        left: ASTNode,
        right: ASTNode
    ): ParserDefinition.SpaceRequirements = ParserDefinition.SpaceRequirements.MAY
}

/**
 * Minimal lexer that splits text into alternating whitespace and non-whitespace runs.
 * This gives word-level PSI leaf elements so that findElementAt() returns a small
 * element rather than the whole file when there is no semantic token at the offset.
 */
private class HclWordLexer : LexerBase() {
    private var buffer: CharSequence = ""
    private var bufferEnd = 0
    private var tokenStart = 0
    private var tokenEnd = 0
    private var tokenType: IElementType? = null

    override fun start(buffer: CharSequence, startOffset: Int, endOffset: Int, initialState: Int) {
        this.buffer = buffer
        this.bufferEnd = endOffset
        this.tokenStart = startOffset
        this.tokenEnd = startOffset
        advance()
    }

    override fun getState(): Int = 0
    override fun getTokenType(): IElementType? = tokenType
    override fun getTokenStart(): Int = tokenStart
    override fun getTokenEnd(): Int = tokenEnd
    override fun getBufferSequence(): CharSequence = buffer
    override fun getBufferEnd(): Int = bufferEnd

    override fun advance() {
        tokenStart = tokenEnd
        if (tokenStart >= bufferEnd) {
            tokenType = null
            return
        }
        val isWhitespace = buffer[tokenStart].isWhitespace()
        tokenEnd = tokenStart + 1
        while (tokenEnd < bufferEnd && buffer[tokenEnd].isWhitespace() == isWhitespace) {
            tokenEnd++
        }
        tokenType = if (isWhitespace) TokenType.WHITE_SPACE else HCL_TOKEN
    }
}

private class HclParser : PsiParser {
    override fun parse(root: IElementType, builder: PsiBuilder): ASTNode {
        val mark = builder.mark()
        while (!builder.eof()) {
            builder.advanceLexer()
        }
        mark.done(root)
        return builder.treeBuilt
    }
}

class HclPsiFile(viewProvider: FileViewProvider) : PsiFileBase(viewProvider, HclLanguage) {
    override fun getFileType(): FileType = HclFileType
}
