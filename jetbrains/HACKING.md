## Building and running locally

```sh
./gradlew runIde
```

This builds the plugin and launches a development instance of IntelliJ IDEA
with the plugin loaded. The language server binary is **not** bundled — it is
downloaded automatically from GitHub releases the first time you open an `.hcl`
file.

The resulting `.zip` does not contain a pinned language server version, so when
installed it will download the **latest release** from GitHub on first activation.
To pin a specific version:

```sh
./gradlew buildPlugin -PlanguageServerVersion=0.2.0-rc6
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
3. Automatically downloaded binary (cached in the IDE's plugins directory)

For release builds, the language server version is pinned to match the plugin
version (baked in at build time via `-PlanguageServerVersion=X.Y.Z`).
For local dev, the latest release is downloaded.

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

## Publishing to the JetBrains Marketplace

The plugin is published automatically by the release workflow (`.github/workflows/release.yaml`)
when a `v*` tag is pushed. The workflow builds a lightweight plugin ZIP (no bundled binary)
with the language server version pinned to the tag. Users' IDEs download the
language server on first activation.

### First-time setup

1. Create an account at https://plugins.jetbrains.com/author/me
2. Generate a Marketplace token at https://plugins.jetbrains.com/author/me/tokens with "Plugin upload" scope
3. Upload the initial version manually via https://plugins.jetbrains.com/plugin/add (required for the first release)
4. Store the token as the `PUBLISH_TOKEN` GitHub Actions secret for subsequent automated releases

## Gradle tasks

| Task                       | Description                                                                                                      |
|----------------------------|------------------------------------------------------------------------------------------------------------------|
| `./gradlew runIde`         | Builds the plugin and launches a sandboxed IntelliJ instance with it loaded                                      |
| `./gradlew buildPlugin`    | Produces a distributable plugin ZIP in `build/distributions/`                                                    |
| `./gradlew compileKotlin`  | Compiles Kotlin sources without building the full plugin                                                         |
| `./gradlew test`           | Runs the test suite                                                                                              |
| `./gradlew clean`          | Removes the `build/` directory                                                                                   |

## File reference

| File | Purpose |
|------|---------|
| `src/.../FunctionHclLanguageServerFactory.kt` | LSP4IJ factory — resolves the binary path, triggers download, creates the stdio connection |
| `src/.../binary/BinaryPathResolver.kt` | Priority-based binary path resolution (settings → env var → cached download) |
| `src/.../binary/BinaryDownloader.kt` | Downloads the language server binary from GitHub releases with progress reporting |
| `src/.../settings/FunctionHclConfigurable.kt` | Settings UI panel under **Settings → Tools → Function HCL** |
| `src/.../settings/FunctionHclSettings.kt` | Persistent settings service |
| `src/.../settings/FunctionHclSettingsState.kt` | Settings state data class |
| `src/.../HclLanguage.kt` | Language definition for `FunctionHCL` |
| `src/.../HclFileType.kt` | File type registration for `.hcl` files |
| `src/.../HclParserDefinition.kt` | Minimal parser and PSI file implementation (required by LSP4IJ) |
| `src/.../HclCodeStyleSettingsProvider.kt` | Code style settings (tab size, etc.) |
| `src/.../FunctionHclBundle.kt` | Message bundle for i18n strings |
| `src/main/resources/META-INF/plugin.xml` | Plugin descriptor — extensions, services, dependencies |
| `src/main/resources/messages/FunctionHclBundle.properties` | User-facing strings (settings labels, notifications, errors) |
| `build.gradle.kts` | Build configuration — dependencies, IntelliJ Platform config, version resource generation |
| `gradle.properties` | Plugin version, platform version, build settings |
