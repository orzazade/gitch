/**
 * Platform detection module for gitch VS Code extension.
 * Detects user platform and architecture to select correct binary from GitHub Releases.
 */

export type Platform = 'darwin' | 'linux' | 'windows';
export type Arch = 'amd64' | 'arm64';

export interface PlatformInfo {
  platform: Platform;
  arch: Arch;
}

/**
 * Detect current platform and architecture.
 * Maps Node.js process values to GoReleaser naming conventions.
 */
export function detectPlatform(): PlatformInfo {
  let platform: Platform;
  let arch: Arch;

  // Map process.platform to GoReleaser platform names
  switch (process.platform) {
    case 'darwin':
      platform = 'darwin';
      break;
    case 'linux':
      platform = 'linux';
      break;
    case 'win32':
      platform = 'windows'; // GoReleaser uses 'windows' not 'win32'
      break;
    default:
      throw new Error(`Unsupported platform: ${process.platform}`);
  }

  // Map process.arch to GoReleaser architecture names
  switch (process.arch) {
    case 'arm64':
      arch = 'arm64';
      break;
    case 'x64':
      arch = 'amd64'; // GoReleaser uses 'amd64' not 'x64'
      break;
    default:
      throw new Error(`Unsupported architecture: ${process.arch}`);
  }

  return { platform, arch };
}

/**
 * Get the asset filename for a specific version.
 * Matches GoReleaser naming convention: gitch_VERSION_PLATFORM_ARCH.EXT
 *
 * @param version - Version string without 'v' prefix (e.g., "1.0.0")
 * @returns Asset filename (e.g., "gitch_1.0.0_darwin_arm64.tar.gz")
 */
export function getAssetName(version: string): string {
  const { platform, arch } = detectPlatform();
  const ext = platform === 'windows' ? 'zip' : 'tar.gz';
  return `gitch_${version}_${platform}_${arch}.${ext}`;
}

/**
 * Get the binary filename for current platform.
 * @returns "gitch.exe" on Windows, "gitch" elsewhere
 */
export function getBinaryName(): string {
  const { platform } = detectPlatform();
  return platform === 'windows' ? 'gitch.exe' : 'gitch';
}
