/**
 * Identity fetching module for gitch VS Code extension.
 * Parses JSON output from gitch status --json command.
 */

import { runGitch } from './runner';

export interface GitchIdentity {
  name: string;
  email: string;
  ssh_key_path?: string;
  gpg_key_id?: string;
  managed: boolean;
}

/**
 * Get current gitch identity by running gitch status --json.
 *
 * @param binaryPath - Absolute path to gitch binary
 * @returns GitchIdentity or null if no identity configured
 */
export async function getCurrentIdentity(binaryPath: string): Promise<GitchIdentity | null> {
  try {
    const output = await runGitch(binaryPath, ['status', '--json']);
    const identity = JSON.parse(output) as GitchIdentity;

    // Empty name means no identity configured
    if (!identity.name && !identity.email) {
      return null;
    }

    return identity;
  } catch (error) {
    console.error('[gitch] Failed to get identity:', error);
    return null;
  }
}
