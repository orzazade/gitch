/**
 * Identity fetching module for gitch VS Code extension.
 * Parses JSON output from gitch status --json and gitch list --json commands.
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
 * Identity item from gitch list --json output.
 */
export interface GitchIdentityListItem {
  name: string;
  email: string;
  ssh_key_path?: string;
  gpg_key_id?: string;
  is_active: boolean;
  is_default: boolean;
}

/**
 * Result of checking identity rule match for a workspace.
 */
export interface GitchRuleCheck {
  current_identity: GitchIdentity | null;
  expected_identity: string | null;
  has_mismatch: boolean;
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

/**
 * List all gitch identities by running gitch list --json.
 *
 * @param binaryPath - Absolute path to gitch binary
 * @returns Array of identity items (empty if none or error)
 */
export async function listIdentities(binaryPath: string): Promise<GitchIdentityListItem[]> {
  try {
    const output = await runGitch(binaryPath, ['list', '--json']);
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
 * @throws Error if switch fails
 */
export async function switchIdentity(binaryPath: string, identityName: string): Promise<void> {
  await runGitch(binaryPath, ['use', identityName]);
}

/**
 * Check if current workspace matches a gitch rule and if identity is correct.
 * Uses gitch status --json to get current identity state.
 *
 * @param binaryPath - Absolute path to gitch binary
 * @param workspacePath - Path to workspace directory
 * @returns Rule check result with mismatch info
 */
export async function checkIdentityRule(
  binaryPath: string,
  workspacePath: string
): Promise<GitchRuleCheck> {
  try {
    // Run gitch status in workspace directory
    const output = await runGitch(binaryPath, ['status', '--json'], { cwd: workspacePath });
    const identity = JSON.parse(output) as GitchIdentity;

    // Status command returns identity info
    // Mismatch = identity exists but not managed by gitch
    return {
      current_identity: identity.name || identity.email ? identity : null,
      expected_identity: identity.managed ? identity.name : null,
      has_mismatch: !identity.managed && !!identity.email,
    };
  } catch (error) {
    console.error('[gitch] Failed to check identity rule:', error);
    return {
      current_identity: null,
      expected_identity: null,
      has_mismatch: false,
    };
  }
}
