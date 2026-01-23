// Platform detection module - stub for Task 1 compilation
// Full implementation in Task 2

export type Platform = 'darwin' | 'linux' | 'windows';
export type Arch = 'amd64' | 'arm64';

export interface PlatformInfo {
  platform: Platform;
  arch: Arch;
}

export function detectPlatform(): PlatformInfo {
  // Stub - will be implemented in Task 2
  throw new Error('Not implemented');
}

export function getAssetName(_version: string): string {
  // Stub - will be implemented in Task 2
  throw new Error('Not implemented');
}

export function getBinaryName(): string {
  // Stub - will be implemented in Task 2
  throw new Error('Not implemented');
}
