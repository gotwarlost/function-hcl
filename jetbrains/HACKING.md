## Building and running locally

```sh
./gradlew runIde
```

This downloads the language server binary for your platform into `bin/`,
copies it into the IntelliJ sandbox, builds the plugin, and launches a
development instance of IntelliJ IDEA with the plugin loaded.

The `bin/function-hcl-ls` binary is cached — the download is a no-op if it
already exists. To force a re-download, delete the `bin/` directory first:

```sh
rm -rf bin/
./gradlew runIde
```

To use a locally-built language server instead of downloading one, either:

- Set the `FUNCTION_HCL_LS_PATH` environment variable:
  ```sh
  FUNCTION_HCL_LS_PATH=/path/to/function-hcl-ls ./gradlew runIde
  ```
- Or configure a custom path after launch in **Settings → Tools → Function HCL**.

## How the language server binary is resolved

The plugin uses the same priority order as the VS Code extension:

1. Custom path from **Settings → Tools → Function HCL**
2. `FUNCTION_HCL_LS_PATH` environment variable
3. Bundled binary shipped with the plugin (at `{pluginPath}/bin/function-hcl-ls`)

During local development, `prepareSandbox` copies `bin/function-hcl-ls` into
the IntelliJ sandbox so it appears as the bundled binary.

## Restarting the language server

To restart the language server without restarting the IDE, go to
**Settings → Tools → Function HCL** and click **Restart Language Server**.
This stops the server for all open projects; it restarts automatically when
the next `.hcl` file is opened.

## Producing a distributable plugin

```sh
./gradlew buildPlugin
```

This creates a plugin ZIP under `build/distributions/` that can be installed
via **Settings → Plugins → ⚙ → Install Plugin from Disk**.

## Gradle tasks

| Task                       | Description                                                                                                      |
|----------------------------|------------------------------------------------------------------------------------------------------------------|
| `./gradlew runIde`         | Builds the plugin and launches a sandboxed IntelliJ instance with it loaded                                      |
| `./gradlew buildPlugin`    | Produces a distributable plugin ZIP in `build/distributions/`                                                    |
| `./gradlew downloadLanguageServer` | Downloads the language server binary for your platform into `bin/`. Runs automatically before `runIde`   |
| `./gradlew prepareSandbox` | Assembles the plugin in the sandbox directory, including the language server binary from `bin/`                   |
| `./gradlew compileKotlin`  | Compiles Kotlin sources without building the full plugin                                                         |
| `./gradlew test`           | Runs the test suite                                                                                              |
| `./gradlew clean`          | Removes the `build/` and `bin/` directories, including the downloaded language server binary                      |

## File reference

| File | Purpose |
|------|---------|
| `src/.../FunctionHclLanguageServerFactory.kt` | LSP4IJ factory — resolves the binary path and creates the stdio connection to the language server |
| `src/.../binary/BinaryPathResolver.kt` | Priority-based binary path resolution (settings → env var → bundled) |
| `src/.../settings/FunctionHclConfigurable.kt` | Settings UI panel under **Settings → Tools → Function HCL** |
| `src/.../settings/FunctionHclSettings.kt` | Persistent settings service |
| `src/.../settings/FunctionHclSettingsState.kt` | Settings state data class |
| `src/.../HclLanguage.kt` | Language definition for `FunctionHCL` |
| `src/.../HclFileType.kt` | File type registration for `.hcl` files |
| `src/.../HclParserDefinition.kt` | Minimal parser and PSI file implementation (required by LSP4IJ) |
| `src/.../HclCodeStyleSettingsProvider.kt` | Code style settings (tab size, etc.) |
| `src/.../FunctionHclBundle.kt` | Message bundle for i18n strings |
| `src/main/resources/META-INF/plugin.xml` | Plugin descriptor — extensions, services, dependencies |
| `src/main/resources/META-INF/terraform-support.xml` | Optional Terraform plugin integration for HCL syntax highlighting |
| `src/main/resources/messages/FunctionHclBundle.properties` | User-facing strings (settings labels, notifications, errors) |
| `build.gradle.kts` | Build configuration — dependencies, IntelliJ Platform config, sandbox wiring |
| `gradle.properties` | Plugin version, platform version, build settings |
| `build.gradle.kts` (`downloadLanguageServer` task) | Downloads the language server binary from GitHub releases for the current platform |
