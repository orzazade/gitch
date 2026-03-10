/**
 * Auto-switch identity based on gitch rules when workspace changes.
 */

import * as vscode from 'vscode';
import { autoSwitchIdentity } from '../cli/identity';

/**
 * Check if auto-switch is needed for the current workspace.
 * Shows warning if identity mismatch detected.
 *
 * @param binaryPath - Path to gitch binary
 * @param workspacePath - Path to workspace directory
 */
export async function checkAutoSwitch(
  binaryPath: string,
  workspacePath: string
): Promise<void> {
  try {
    await autoSwitchIdentity(binaryPath, workspacePath);
  } catch (error) {
    console.error('[gitch] Auto-switch check failed:', error);
  }
}

/**
 * Register auto-switch behavior for workspace changes.
 *
 * @param context - Extension context
 * @param binaryPath - Path to gitch binary
 * @param onIdentityChanged - Callback to refresh UI after switch
 * @returns Disposable for the watcher
 */
export function registerAutoSwitch(
  context: vscode.ExtensionContext,
  binaryPath: string,
  onIdentityChanged: () => void
): vscode.Disposable {
  // Check on workspace folder change
  return vscode.workspace.onDidChangeWorkspaceFolders(async () => {
    // Get first workspace folder (primary workspace)
    const folder = vscode.workspace.workspaceFolders?.[0];
    if (!folder) return;

    console.log('[gitch] Workspace changed, checking auto-switch');
    await checkAutoSwitch(binaryPath, folder.uri.fsPath);
    onIdentityChanged();
  });
}
