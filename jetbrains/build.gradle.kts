import java.net.URL
import org.jetbrains.changelog.Changelog
import org.jetbrains.changelog.markdownToHTML
import org.jetbrains.intellij.platform.gradle.TestFrameworkType

plugins {
    id("java") // Java support
    alias(libs.plugins.kotlin) // Kotlin support
    alias(libs.plugins.intelliJPlatform) // IntelliJ Platform Gradle Plugin
    alias(libs.plugins.changelog) // Gradle Changelog Plugin
    alias(libs.plugins.qodana) // Gradle Qodana Plugin
    alias(libs.plugins.kover) // Gradle Kover Plugin
    id("org.jetbrains.grammarkit") version "2022.3.2.2"
}

group = providers.gradleProperty("pluginGroup").get()
version = providers.gradleProperty("pluginVersion").get()

// Set the JVM language level used to build the project.
kotlin {
    jvmToolchain(21)
}

// Configure project's dependencies
repositories {
    mavenCentral()

    // IntelliJ Platform Gradle Plugin Repositories Extension - read more: https://plugins.jetbrains.com/docs/intellij/tools-intellij-platform-gradle-plugin-repositories-extension.html
    intellijPlatform {
        defaultRepositories()
    }
}

grammarKit {
    jflexRelease.set("1.7.0-1")
}

// Dependencies are managed with Gradle version catalog - read more: https://docs.gradle.org/current/userguide/version_catalogs.html
dependencies {

    testImplementation(libs.junit)
    testImplementation(libs.opentest4j)

    // IntelliJ Platform Gradle Plugin Dependencies Extension - read more: https://plugins.jetbrains.com/docs/intellij/tools-intellij-platform-gradle-plugin-dependencies-extension.html
    intellijPlatform {
        intellijIdea(providers.gradleProperty("platformVersion"))

        // Plugin Dependencies. Uses `platformBundledPlugins` property from the gradle.properties file for bundled IntelliJ Platform plugins.
        bundledPlugins(providers.gradleProperty("platformBundledPlugins").map { it.split(',') })

        // Plugin Dependencies. Uses `platformPlugins` property from the gradle.properties file for plugin from JetBrains Marketplace.
        plugins(providers.gradleProperty("platformPlugins").map { it.split(',') })

        // Module Dependencies. Uses `platformBundledModules` property from the gradle.properties file for bundled IntelliJ Platform modules.
        bundledModules(providers.gradleProperty("platformBundledModules").map { it.split(',') })

        plugin("com.redhat.devtools.lsp4ij:0.19.1")
        testFramework(TestFrameworkType.Platform)
    }
}

// Configure IntelliJ Platform Gradle Plugin - read more: https://plugins.jetbrains.com/docs/intellij/tools-intellij-platform-gradle-plugin-extension.html
intellijPlatform {
    pluginConfiguration {
        name = providers.gradleProperty("pluginName")
        version = providers.gradleProperty("pluginVersion")
        // Extract the <!-- Plugin description --> section from README.md and provide for the plugin's manifest
        description = providers.fileContents(layout.projectDirectory.file("README.md")).asText.map {
            val start = "<!-- Plugin description -->"
            val end = "<!-- Plugin description end -->"

            with(it.lines()) {
                if (!containsAll(listOf(start, end))) {
                    throw GradleException("Plugin description section not found in README.md:\n$start ... $end")
                }
                subList(indexOf(start) + 1, indexOf(end)).joinToString("\n").let(::markdownToHTML)
            }
        }

        val changelog = project.changelog // local variable for configuration cache compatibility
        // Get the latest available change notes from the changelog file
        changeNotes = providers.gradleProperty("pluginVersion").map { pluginVersion ->
            with(changelog) {
                renderItem(
                    (getOrNull(pluginVersion) ?: getUnreleased())
                        .withHeader(false)
                        .withEmptySections(false),
                    Changelog.OutputType.HTML,
                )
            }
        }
        ideaVersion {
            sinceBuild = providers.gradleProperty("pluginSinceBuild")
        }
    }

    signing {
        certificateChain = providers.environmentVariable("CERTIFICATE_CHAIN")
        privateKey = providers.environmentVariable("PRIVATE_KEY")
        password = providers.environmentVariable("PRIVATE_KEY_PASSWORD")
    }

    publishing {
        token = providers.environmentVariable("PUBLISH_TOKEN")
        // The pluginVersion is based on the SemVer (https://semver.org) and supports pre-release labels, like 2.1.7-alpha.3
        // Specify pre-release label to publish the plugin in a custom Release Channel automatically. Read more:
        // https://plugins.jetbrains.com/docs/intellij/publishing-plugin.html#specifying-a-release-channel
        channels = providers.gradleProperty("pluginVersion").map { listOf(it.substringAfter('-', "").substringBefore('.').ifEmpty { "default" }) }
    }

    pluginVerification {
        ides {
            recommended()
        }
    }
}

// Configure Gradle Changelog Plugin - read more: https://github.com/JetBrains/gradle-changelog-plugin
changelog {
    groups.empty()
    repositoryUrl = providers.gradleProperty("pluginRepositoryUrl")
}

// Configure Gradle Kover Plugin - read more: https://kotlin.github.io/kotlinx-kover/gradle-plugin/#configuration-details
kover {
    reports {
        total {
            xml {
                onCheck = true
            }
        }
    }
}

// Downloads the language server binary from GitHub releases for the current platform.
// Pure Gradle — no shell scripts — works on macOS, Linux, and Windows.
abstract class DownloadLanguageServerTask @javax.inject.Inject constructor(
    private val fs: FileSystemOperations,
    private val archiveOps: ArchiveOperations,
) : DefaultTask() {
    @get:Input
    abstract val binaryName: Property<String>

    @get:Input
    abstract val githubRepo: Property<String>

    @get:OutputDirectory
    abstract val binDir: DirectoryProperty

    @TaskAction
    fun download() {
        val name = binaryName.get()
        val repo = githubRepo.get()
        val outDir = binDir.get().asFile
        val isWindows = org.gradle.internal.os.OperatingSystem.current().isWindows
        val exeName = if (isWindows) "$name.exe" else name
        val binaryFile = File(outDir, exeName)

        if (binaryFile.exists()) {
            logger.lifecycle("Language server already exists at ${binaryFile.absolutePath}, skipping.")
            return
        }

        val os = when {
            org.gradle.internal.os.OperatingSystem.current().isMacOsX -> "darwin"
            org.gradle.internal.os.OperatingSystem.current().isLinux -> "linux"
            isWindows -> "windows"
            else -> error("Unsupported OS")
        }
        val arch = when (System.getProperty("os.arch").lowercase()) {
            "x86_64", "amd64", "x64" -> "amd64"
            "aarch64", "arm64" -> "arm64"
            else -> error("Unsupported architecture: ${System.getProperty("os.arch")}")
        }

        // Fetch latest release version from GitHub API
        val releaseUrl = URL("https://api.github.com/repos/$repo/releases/latest")
        val releaseJson = releaseUrl.readText()
        val version = Regex(""""tag_name"\s*:\s*"v([^"]+)"""").find(releaseJson)
            ?.groupValues?.get(1) ?: error("Could not determine latest release version")

        val assetName = "$name-$os-$arch.tar.gz"
        val downloadUrl = "https://github.com/$repo/releases/download/v$version/$assetName"
        val tarball = File(outDir, assetName)

        logger.lifecycle("Downloading $name v$version for $os/$arch...")
        outDir.mkdirs()
        URL(downloadUrl).openStream().use { input ->
            tarball.outputStream().use { output -> input.copyTo(output) }
        }

        // Extract the binary from the tarball
        fs.copy {
            from(archiveOps.tarTree(tarball))
            include(exeName)
            into(outDir)
        }
        tarball.delete()

        if (!isWindows) {
            binaryFile.setExecutable(true, false)
        }

        logger.lifecycle("Language server ready at ${binaryFile.absolutePath}")
    }
}

val downloadLanguageServer by tasks.registering(DownloadLanguageServerTask::class) {
    description = "Downloads the function-hcl-ls binary for the current platform"
    group = "build setup"
    binaryName.set("function-hcl-ls")
    githubRepo.set("crossplane-contrib/function-hcl")
    binDir.set(layout.projectDirectory.dir("bin"))
}

tasks {
    wrapper {
        gradleVersion = providers.gradleProperty("gradleVersion").get()
    }
    publishPlugin {
        dependsOn(patchChangelog)
    }
    clean {
        delete(layout.projectDirectory.dir("bin"))
    }
    // Copy the language server binary into the plugin sandbox so it's available during runIde
    prepareSandbox {
        dependsOn(downloadLanguageServer)
        from(layout.projectDirectory.dir("bin")) {
            into("${intellijPlatform.projectName.get()}/bin")
        }
    }
}

intellijPlatformTesting {
    runIde {
        register("runIdeForUiTests") {
            task {
                jvmArgumentProviders += CommandLineArgumentProvider {
                    listOf(
                        "-Drobot-server.port=8082",
                        "-Dide.mac.message.dialogs.as.sheets=false",
                        "-Djb.privacy.policy.text=<!--999.999-->",
                        "-Djb.consents.confirmation.enabled=false",
                    )
                }
            }

            plugins {
                robotServerPlugin()
            }
        }
    }
}
