import * as vscode from 'vscode';
import { ensureBinary } from './cli/binary';

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  try {
    const binaryPath = await ensureBinary(context);
    console.log(`[gitch] Binary found at: ${binaryPath}`);

    // Store binary path for later use by commands
    context.workspaceState.update('gitch.binaryPath', binaryPath);

    vscode.window.showInformationMessage(`gitch extension activated. Binary: ${binaryPath}`);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    console.error(`[gitch] Failed to acquire binary: ${message}`);
    vscode.window.showErrorMessage(`gitch: Failed to acquire CLI binary. ${message}`);
  }
}

export function deactivate(): void {
  console.log('[gitch] Extension deactivated');
}
