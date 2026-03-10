/**
 * Identity fetching module for gitch VS Code extension.
 * Parses JSON output from gitch status --json and gitch list --json commands.
 */

import { runGitch } from './runner';

export interface GitchIdentity {
  name: string;
  git_name?: string;
  email: string;
  ssh_key_path?: string;
  gpg_key_id?: string;
  managed: boolean;
  partial_match?: boolean;
}

/**
 * Identity item from gitch list --json output.
 */
export interface GitchIdentityListItem {
  name: string;
  git_name?: string;
  email: string;
  ssh_key_path?: string;
  gpg_key_id?: string;
  is_active: boolean;
  is_default: boolean;
}

/**
 * Get current gitch identity by running gitch status --json.
 *
 * @param binaryPath - Absolute path to gitch binary
 * @param workspacePath - Optional workspace path used as the command cwd
 * @returns GitchIdentity or null if no identity configured
 */
export async function getCurrentIdentity(
  binaryPath: string,
  workspacePath?: string
): Promise<GitchIdentity | null> {
  try {
    const output = await runGitch(binaryPath, ['status', '--json'], { cwd: workspacePath });
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

/**
 * Explicitly auto-switch for a workspace using gitch rule matching.
 *
 * @param binaryPath - Absolute path to gitch binary
 * @param workspacePath - Path to workspace directory
 */
export async function autoSwitchIdentity(
  binaryPath: string,
  workspacePath: string
): Promise<void> {
  await runGitch(binaryPath, ['autoswitch', '--quiet'], { cwd: workspacePath });
}

/**
 * List all gitch identities by running gitch list --json.
 *
 * @param binaryPath - Absolute path to gitch binary
 * @param workspacePath - Optional workspace path used as the command cwd
 * @returns Array of identity items (empty if none or error)
 */
export async function listIdentities(
  binaryPath: string,
  workspacePath?: string
): Promise<GitchIdentityListItem[]> {
  try {
    const output = await runGitch(binaryPath, ['list', '--json'], { cwd: workspacePath });
    const identities = JSON.parse(output) as GitchIdentityListItem[];
    return identities || [];
  } catch (error) {
    console.error('[gitch] Failed to list identities:', error);
    return [];
  }
}

/**
 * Switch to a gitch identity by name.
 *
 * @param binaryPath - Absolute path to gitch binary
 * @param identityName - Name of identity to switch to
 * @param workspacePath - Optional workspace path used as the command cwd
 * @throws Error if switch fails
 */
export async function switchIdentity(
  binaryPath: string,
  identityName: string,
  workspacePath?: string
): Promise<void> {
  await runGitch(binaryPath, ['use', identityName], { cwd: workspacePath });
}
