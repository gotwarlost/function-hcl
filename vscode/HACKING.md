## Building and running locally

```sh
npm run setup
```

This installs dependencies and builds the extension. The language server binary
is **not** bundled — it is downloaded automatically from GitHub releases the
first time you open an `.hcl` file.

To produce a `.vsix` package for local installation:

```sh
npm run package
```

The resulting `.vsix` does not contain a pinned language server version, so when
installed it will download the **latest release** from GitHub on first activation.
To pin a specific version:

```sh
echo "0.2.0-rc6" > language-server-version.txt
npm run package
```

To skip downloading entirely, set the `FUNCTION_HCL_LS_PATH` environment variable
or the `function-hcl.languageServerPath` VS Code setting to a local build of the
language server.

## Local testing

To test the extension in a development instance of VS Code:

1. Open this directory in VS Code:
   ```sh
   code .
   ```

2. Press `F5` or go to **Run > Start Debugging**.

This launches a second VS Code window (the Extension Development Host) with the
extension loaded from source. The language server binary will be downloaded
automatically on first activation. To use a local build instead, set either:

- The `function-hcl.languageServerPath` setting in VS Code
- The `FUNCTION_HCL_LS_PATH` environment variable

The setup that makes debugging work is in `.vscode/`:

- **`launch.json`** — defines a launch configuration of type `extensionHost` that
  starts VS Code with `--extensionDevelopmentPath` pointing to this directory.
  It sets a pre-launch task that runs `npm run watch`.
- **`tasks.json`** — defines the `watch` task (`node esbuild.js --watch`) as a background build
  that rebundles the extension on every save, so changes are picked up automatically
  in the debug session.

To pick up changes, save the file and reload the Extension Development Host
window (`Ctrl+Shift+P` / `Cmd+Shift+P` → "Developer: Reload Window").

## How the language server binary is resolved

The extension uses the same priority order as the JetBrains plugin:

1. Custom path from VS Code settings (`function-hcl.languageServerPath`)
2. `FUNCTION_HCL_LS_PATH` environment variable
3. Automatically downloaded binary (cached in the extension's global storage)

For release builds, the language server version is pinned to match the extension
version (baked in via `language-server-version.txt`). For local dev, the latest
release is downloaded.

## Publishing to the VS Code Marketplace

The extension is published automatically by the release workflow (`.github/workflows/release.yaml`)
when a `v*` tag is pushed. The workflow builds a single lightweight `.vsix` package
(no bundled binary) with the language server version pinned to the tag. Users' editors
download the language server on first activation.

### Dependencies for publishing

- **`VSCE_PAT` secret** — a Personal Access Token from Azure DevOps with the
  Marketplace > Manage scope, stored as a GitHub Actions secret in the repository.
- **`publisher` in `package.json`** — currently set to `function-hcl-authors`.
  This must match the publisher ID registered at https://marketplace.visualstudio.com/manage.
- To create a PAT you need to go to this URL (which cannot be found from the Azure portal, as far as I could tell),
  https://dev.azure.com/, go to User Settings and create one.

## npm scripts

| Script              | Command                                                                                                             |
|---------------------|---------------------------------------------------------------------------------------------------------------------|
| `npm run compile`   | Bundles the extension into `dist/extension.js` using esbuild                                                        |
| `npm run watch`     | Runs esbuild in watch mode, rebuilding `dist/extension.js` on file changes. Used by the debug launch configuration  |
| `npm run package`   | Produces a `.vsix` package for local installation                                                                   |
| `npm run test`      | Type-checks with `tsc` and runs the test suite via `vscode-test`                                                    |
| `npm run clean`     | Removes all generated directories: `dist/`, `out/`, and `node_modules/`                                             |
| `npm run setup`     | First-time setup: installs dependencies and builds the extension                                                    |
| `vscode:prepublish` | Runs automatically during `vsce package` — builds the extension in production mode. Not intended to be run directly |

## File reference

| File                          | Purpose                                                                                                                                              |
|-------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------|
| `src/extension.ts`            | Extension entry point — activates the language client, resolves the server binary path                                                               |
| `src/languageServer.ts`       | Downloads and caches the language server binary from GitHub releases                                                                                 |
| `src/test/extension.test.ts`  | Extension test suite                                                                                                                                 |
| `package.json`                | Extension manifest — metadata, dependencies, scripts, VS Code contribution points                                                                    |
| `esbuild.js`                  | Bundles the TypeScript source into a single `dist/extension.js` using esbuild                                                                        |
| `tsconfig.json`               | TypeScript compiler configuration                                                                                                                    |
| `eslint.config.mjs`           | ESLint configuration                                                                                                                                 |
| `language-configuration.json` | HCL language behavior in the editor — comments, brackets, folding regions                                                                            |
| `syntaxes/hcl.tmGrammar.json` | TextMate grammar for HCL syntax highlighting                                                                                                         |
| `.vscodeignore`               | Files excluded from the `.vsix` package                                                                                                              |
| `.vscode/launch.json`         | Debug launch configuration for the Extension Development Host                                                                                        |
| `.vscode/tasks.json`          | Build tasks (compile and watch)                                                                                                                      |
| `.vscode/settings.json`       | Workspace settings for this directory                                                                                                                |
| `.vscode/extensions.json`     | Recommended extensions for contributors                                                                                                              |
| `.vscode-test.mjs`            | Test runner configuration for `@vscode/test-cli`                                                                                                     |
