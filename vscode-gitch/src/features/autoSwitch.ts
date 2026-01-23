/**
 * Auto-switch identity based on gitch rules when workspace changes.
 */

import * as vscode from 'vscode';
import { checkIdentityRule } from '../cli/identity';

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
    // checkIdentityRule runs gitch status which returns identity info
    const result = await checkIdentityRule(binaryPath, workspacePath);

    if (result.has_mismatch && result.current_identity) {
      // Identity is set but not managed by gitch - warn user
      const action = await vscode.window.showWarningMessage(
        `Git identity (${result.current_identity.email}) is not managed by gitch. ` +
          `Click the status bar to switch to a gitch identity.`,
        'Switch Identity'
      );

      if (action === 'Switch Identity') {
        vscode.commands.executeCommand('gitch.switchIdentity');
      }
    }
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
