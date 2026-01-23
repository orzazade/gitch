/**
 * Command to switch gitch identity via Quick Pick.
 */

import * as vscode from 'vscode';
import { listIdentities, switchIdentity as cliSwitchIdentity } from '../cli/identity';
import { showIdentityQuickPick } from '../ui/quickPick';

/**
 * Register the switch identity command.
 *
 * @param context - Extension context
 * @param binaryPath - Path to gitch binary
 * @param onIdentityChanged - Callback to refresh UI after switch
 * @returns Disposable for the command
 */
export function registerSwitchIdentityCommand(
  context: vscode.ExtensionContext,
  binaryPath: string,
  onIdentityChanged: () => void
): vscode.Disposable {
  return vscode.commands.registerCommand('gitch.switchIdentity', async () => {
    try {
      // Get all identities
      const identities = await listIdentities(binaryPath);

      // Show Quick Pick
      const selectedName = await showIdentityQuickPick(identities);
      if (!selectedName) {
        return; // User cancelled
      }

      // Check if already active
      const activeIdentity = identities.find((id) => id.is_active);
      if (activeIdentity?.name === selectedName) {
        vscode.window.showInformationMessage(`Already using identity: ${selectedName}`);
        return;
      }

      // Switch identity
      await cliSwitchIdentity(binaryPath, selectedName);

      // Notify success and refresh
      vscode.window.showInformationMessage(`Switched to identity: ${selectedName}`);
      onIdentityChanged();
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      vscode.window.showErrorMessage(`Failed to switch identity: ${message}`);
    }
  });
}
