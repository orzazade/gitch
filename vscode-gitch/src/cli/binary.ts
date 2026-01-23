/**
 * Binary management module for gitch VS Code extension.
 * Handles binary discovery (PATH, cache) and download from GitHub Releases.
 */

import * as vscode from 'vscode';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import { execFile } from 'child_process';
import { promisify } from 'util';
import { detectPlatform, getAssetName, getBinaryName } from './platform';

const execFileAsync = promisify(execFile);

const GITHUB_REPO = 'orzazade/gitch';
const GITHUB_API_URL = `https://api.github.com/repos/${GITHUB_REPO}/releases/latest`;
const DOWNLOAD_TIMEOUT = 30000; // 30 seconds

interface GitHubAsset {
  name: string;
  browser_download_url: string;
}

interface GitHubRelease {
  tag_name: string;
  assets: GitHubAsset[];
}

/**
 * Ensure gitch binary is available.
 * Cascade: config path -> PATH -> cache -> download
 *
 * @param context - VS Code extension context for storage access
 * @returns Absolute path to gitch binary
 */
export async function ensureBinary(context: vscode.ExtensionContext): Promise<string> {
  const config = vscode.workspace.getConfiguration('gitch');

  // 1. Check config gitch.binaryPath
  const configPath = config.get<string>('binaryPath');
  if (configPath && configPath.trim() !== '') {
    if (await fileExists(configPath)) {
      console.log(`[gitch] Using configured binary path: ${configPath}`);
      return configPath;
    }
    throw new Error(`Configured binary path does not exist: ${configPath}`);
  }

  // 2. Check PATH
  const pathBinary = await findInPath();
  if (pathBinary) {
    console.log(`[gitch] Found binary in PATH: ${pathBinary}`);
    return pathBinary;
  }

  // 3. Check globalStorageUri/bin/
  const cachedBinary = await getCachedBinaryPath(context);
  if (cachedBinary && (await fileExists(cachedBinary))) {
    console.log(`[gitch] Using cached binary: ${cachedBinary}`);
    return cachedBinary;
  }

  // 4. Check autoDownload setting
  const autoDownload = config.get<boolean>('autoDownload', true);
  if (!autoDownload) {
    throw new Error(
      'gitch binary not found and auto-download is disabled. ' +
        'Install gitch manually (https://github.com/orzazade/gitch) or enable gitch.autoDownload.',
    );
  }

  // 5. Download binary
  console.log('[gitch] Binary not found, downloading from GitHub Releases...');
  return await downloadBinary(context);
}

/**
 * Find gitch binary in system PATH.
 * @returns Absolute path to binary or null if not found
 */
async function findInPath(): Promise<string | null> {
  const { platform } = detectPlatform();
  const command = platform === 'windows' ? 'where' : 'which';
  const binaryName = getBinaryName();

  try {
    const { stdout } = await execFileAsync(command, [binaryName], {
      timeout: 5000,
    });
    const result = stdout.trim().split('\n')[0]; // First line for Windows 'where' command
    if (result && (await fileExists(result))) {
      return result;
    }
  } catch {
    // Binary not in PATH - this is expected, not an error
  }
  return null;
}

/**
 * Get path where cached binary should be stored.
 */
async function getCachedBinaryPath(context: vscode.ExtensionContext): Promise<string> {
  const binDir = path.join(context.globalStorageUri.fsPath, 'bin');
  return path.join(binDir, getBinaryName());
}

/**
 * Download gitch binary from GitHub Releases.
 *
 * @param context - VS Code extension context for storage access
 * @returns Absolute path to downloaded binary
 */
async function downloadBinary(context: vscode.ExtensionContext): Promise<string> {
  // Fetch latest release info
  const release = await fetchLatestRelease();
  const version = release.tag_name.replace(/^v/, ''); // Remove 'v' prefix
  const assetName = getAssetName(version);

  // Find matching asset
  const asset = release.assets.find((a) => a.name === assetName);
  if (!asset) {
    const { platform, arch } = detectPlatform();
    throw new Error(
      `No gitch release found for your platform (${platform} ${arch}). ` +
        `Expected asset: ${assetName}`,
    );
  }

  // Prepare directories
  const binDir = path.join(context.globalStorageUri.fsPath, 'bin');
  await fs.promises.mkdir(binDir, { recursive: true });

  const tempDir = path.join(os.tmpdir(), `gitch-download-${Date.now()}`);
  await fs.promises.mkdir(tempDir, { recursive: true });

  const archivePath = path.join(tempDir, assetName);
  const binaryPath = path.join(binDir, getBinaryName());

  try {
    // Download with progress
    await vscode.window.withProgress(
      {
        location: vscode.ProgressLocation.Notification,
        title: `Downloading gitch ${version}...`,
        cancellable: false,
      },
      async (progress) => {
        progress.report({ message: 'Fetching archive...' });
        await downloadFile(asset.browser_download_url, archivePath);

        progress.report({ message: 'Extracting binary...' });
        await extractBinary(archivePath, tempDir);

        // Find the extracted binary (could be in root or in a subfolder)
        const extractedBinary = await findExtractedBinary(tempDir);
        if (!extractedBinary) {
          throw new Error('Failed to find gitch binary in downloaded archive');
        }

        // Move to final location
        progress.report({ message: 'Installing...' });
        await fs.promises.copyFile(extractedBinary, binaryPath);

        // Make executable on Unix
        const { platform } = detectPlatform();
        if (platform !== 'windows') {
          await fs.promises.chmod(binaryPath, 0o755);
        }
      },
    );

    console.log(`[gitch] Successfully downloaded binary to: ${binaryPath}`);
    return binaryPath;
  } finally {
    // Cleanup temp directory
    try {
      await fs.promises.rm(tempDir, { recursive: true, force: true });
    } catch {
      // Ignore cleanup errors
    }
  }
}

/**
 * Fetch latest release information from GitHub API.
 */
async function fetchLatestRelease(): Promise<GitHubRelease> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), DOWNLOAD_TIMEOUT);

  try {
    const response = await fetch(GITHUB_API_URL, {
      signal: controller.signal,
      headers: {
        'User-Agent': 'gitch-vscode-extension',
        Accept: 'application/vnd.github.v3+json',
      },
    });

    if (!response.ok) {
      throw new Error(`GitHub API error: ${response.status} ${response.statusText}`);
    }

    return (await response.json()) as GitHubRelease;
  } catch (error) {
    if (error instanceof Error && error.name === 'AbortError') {
      throw new Error('Failed to fetch release info: request timed out');
    }
    throw new Error(`Failed to fetch release info: ${error}`);
  } finally {
    clearTimeout(timeoutId);
  }
}

/**
 * Download a file from URL to local path.
 */
async function downloadFile(url: string, destPath: string): Promise<void> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), DOWNLOAD_TIMEOUT);

  try {
    const response = await fetch(url, {
      signal: controller.signal,
      headers: {
        'User-Agent': 'gitch-vscode-extension',
      },
    });

    if (!response.ok) {
      throw new Error(`Download failed: ${response.status} ${response.statusText}`);
    }

    const buffer = await response.arrayBuffer();
    await fs.promises.writeFile(destPath, Buffer.from(buffer));
  } catch (error) {
    if (error instanceof Error && error.name === 'AbortError') {
      throw new Error('Download timed out. Please check your network connection and try again.');
    }
    throw new Error(`Failed to download gitch CLI: ${error}`);
  } finally {
    clearTimeout(timeoutId);
  }
}

/**
 * Extract binary from archive.
 */
async function extractBinary(archivePath: string, destDir: string): Promise<void> {
  const { platform } = detectPlatform();

  try {
    if (platform === 'windows') {
      // Use adm-zip for .zip files
      const AdmZip = require('adm-zip');
      const zip = new AdmZip(archivePath);
      zip.extractAllTo(destDir, true);
    } else {
      // Use tar for .tar.gz files
      const tar = require('tar');
      await tar.x({
        file: archivePath,
        cwd: destDir,
      });
    }
  } catch (error) {
    throw new Error(`Failed to extract gitch archive: ${error}`);
  }
}

/**
 * Find the extracted binary in the temp directory.
 * GoReleaser may place binary in root or in a subfolder.
 */
async function findExtractedBinary(dir: string): Promise<string | null> {
  const binaryName = getBinaryName();

  // Check root directory
  const rootPath = path.join(dir, binaryName);
  if (await fileExists(rootPath)) {
    return rootPath;
  }

  // Check subdirectories (GoReleaser sometimes creates a folder)
  const entries = await fs.promises.readdir(dir, { withFileTypes: true });
  for (const entry of entries) {
    if (entry.isDirectory()) {
      const subPath = path.join(dir, entry.name, binaryName);
      if (await fileExists(subPath)) {
        return subPath;
      }
    }
  }

  return null;
}

/**
 * Check if file exists at path.
 */
async function fileExists(filePath: string): Promise<boolean> {
  try {
    await fs.promises.access(filePath, fs.constants.F_OK);
    return true;
  } catch {
    return false;
  }
}
