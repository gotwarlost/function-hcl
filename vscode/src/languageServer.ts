import * as fs from 'fs';
import * as path from 'path';
import * as https from 'https';
import * as http from 'http';
import { ExtensionContext, ProgressLocation, window } from 'vscode';
import { x as extract } from 'tar';

const BINARY_NAME = process.platform === 'win32' ? 'function-hcl-ls.exe' : 'function-hcl-ls';
const GITHUB_REPO = 'crossplane-contrib/function-hcl';

const PLATFORM_MAP: Record<string, string> = { darwin: 'darwin', linux: 'linux', win32: 'windows' };
const ARCH_MAP: Record<string, string> = { x64: 'amd64', arm64: 'arm64' };

/**
 * Returns the pinned language server version baked in at build time,
 * or empty string for local dev (which fetches latest).
 */
function getPinnedVersion(): string {
    try {
        const versionFile = path.join(__dirname, '..', 'language-server-version.txt');
        if (fs.existsSync(versionFile)) {
            return fs.readFileSync(versionFile, 'utf-8').trim();
        }
    } catch {
        // ignore
    }
    return '';
}

/**
 * Returns the path to the cached language server binary in globalStorageUri.
 * Downloads it if not already cached.
 */
export async function ensureServerBinary(context: ExtensionContext): Promise<string> {
    const cacheDir = context.globalStorageUri.fsPath;
    const binaryPath = path.join(cacheDir, BINARY_NAME);

    if (fs.existsSync(binaryPath)) {
        return binaryPath;
    }

    // Download with progress
    await window.withProgress(
        {
            location: ProgressLocation.Notification,
            title: 'Downloading function-hcl language server...',
            cancellable: true,
        },
        async (progress, token) => {
            const os = PLATFORM_MAP[process.platform];
            const arch = ARCH_MAP[process.arch];
            if (!os || !arch) {
                throw new Error(`Unsupported platform: ${process.platform}/${process.arch}`);
            }

            let version = getPinnedVersion();
            if (!version) {
                version = await fetchLatestVersion();
            }

            const assetName = `function-hcl-ls-${os}-${arch}.tar.gz`;
            const downloadUrl = `https://github.com/${GITHUB_REPO}/releases/download/v${version}/${assetName}`;

            progress.report({ message: `v${version} for ${os}/${arch}` });

            fs.mkdirSync(cacheDir, { recursive: true });
            const tarballPath = path.join(cacheDir, assetName);

            try {
                await downloadFile(downloadUrl, tarballPath, token);

                if (token.isCancellationRequested) {
                    throw new Error('Download cancelled');
                }

                await extract({
                    file: tarballPath,
                    cwd: cacheDir,
                    filter: (entryPath: string) => entryPath.endsWith(BINARY_NAME),
                });
            } finally {
                if (fs.existsSync(tarballPath)) {
                    fs.unlinkSync(tarballPath);
                }
            }

            if (!fs.existsSync(binaryPath)) {
                throw new Error(`Binary '${BINARY_NAME}' not found in archive`);
            }

            if (process.platform !== 'win32') {
                fs.chmodSync(binaryPath, 0o755);
            }
        }
    );

    return binaryPath;
}

async function fetchLatestVersion(): Promise<string> {
    const url = `https://api.github.com/repos/${GITHUB_REPO}/releases/latest`;
    const text = await httpGet(url);
    const match = text.match(/"tag_name"\s*:\s*"v([^"]+)"/);
    if (!match) {
        throw new Error('Could not determine latest release version');
    }
    return match[1];
}

function downloadFile(url: string, dest: string, token?: { isCancellationRequested: boolean }): Promise<void> {
    return new Promise((resolve, reject) => {
        const follow = (url: string, redirects = 0) => {
            if (redirects > 10) {
                return reject(new Error('Too many redirects'));
            }
            const get = url.startsWith('https') ? https.get : http.get;
            get(url, { headers: { 'User-Agent': 'function-hcl-vscode' } }, (res) => {
                if (res.statusCode && res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
                    return follow(res.headers.location, redirects + 1);
                }
                if (res.statusCode !== 200) {
                    return reject(new Error(`Download failed: HTTP ${res.statusCode}`));
                }
                const file = fs.createWriteStream(dest);
                res.pipe(file);
                res.on('error', reject);
                file.on('finish', () => file.close(() => resolve()));
                file.on('error', reject);

                if (token) {
                    const interval = setInterval(() => {
                        if (token.isCancellationRequested) {
                            res.destroy();
                            file.close();
                            clearInterval(interval);
                            reject(new Error('Download cancelled'));
                        }
                    }, 500);
                    file.on('close', () => clearInterval(interval));
                }
            }).on('error', reject);
        };
        follow(url);
    });
}

function httpGet(url: string): Promise<string> {
    return new Promise((resolve, reject) => {
        const follow = (url: string, redirects = 0) => {
            if (redirects > 10) {
                return reject(new Error('Too many redirects'));
            }
            const get = url.startsWith('https') ? https.get : http.get;
            get(url, { headers: { 'User-Agent': 'function-hcl-vscode', 'Accept': 'application/vnd.github.v3+json' } }, (res) => {
                if (res.statusCode && res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
                    return follow(res.headers.location, redirects + 1);
                }
                if (res.statusCode !== 200) {
                    return reject(new Error(`HTTP ${res.statusCode}`));
                }
                let data = '';
                res.on('data', (chunk: string) => { data += chunk; });
                res.on('end', () => resolve(data));
                res.on('error', reject);
            }).on('error', reject);
        };
        follow(url);
    });
}
