/**
 * Quick Pick UI for identity selection.
 */

import * as vscode from 'vscode';
import { GitchIdentityListItem } from '../cli/identity';

interface IdentityQuickPickItem extends vscode.QuickPickItem {
  identityName: string;
}

/**
 * Show Quick Pick dropdown for identity selection.
 * Returns selected identity name or undefined if cancelled.
 *
 * @param identities - List of available identities
 * @returns Selected identity name or undefined
 */
export async function showIdentityQuickPick(
  identities: GitchIdentityListItem[]
): Promise<string | undefined> {
  if (identities.length === 0) {
    vscode.window.showWarningMessage(
      'No gitch identities configured. Run "gitch add" in terminal to create one.'
    );
    return undefined;
  }

  const items: IdentityQuickPickItem[] = identities.map((id) => ({
    label: id.is_active ? `$(check) ${id.name}` : id.name,
    description: id.email,
    detail: id.is_default ? '(default)' : undefined,
    identityName: id.name,
  }));

  const selected = await vscode.window.showQuickPick(items, {
    placeHolder: 'Select gitch identity',
    title: 'Switch Git Identity',
  });

  return selected?.identityName;
}
