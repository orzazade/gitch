import * as vscode from 'vscode';
import { ensureBinary } from './cli/binary';
import { StatusBarManager } from './ui/statusBar';
import { getCurrentIdentity } from './cli/identity';
import { registerSwitchIdentityCommand } from './commands/switchIdentity';
import { registerAutoSwitch, checkAutoSwitch } from './features/autoSwitch';
import { getPreferredWorkspacePath } from './workspace';

let statusBarManager: StatusBarManager | undefined;

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  try {
    const binaryPath = await ensureBinary(context);
    console.log(`[gitch] Binary found at: ${binaryPath}`);

    // Store binary path for later use by commands
    context.workspaceState.update('gitch.binaryPath', binaryPath);

    // Initialize status bar
    statusBarManager = new StatusBarManager(binaryPath, getPreferredWorkspacePath);
    context.subscriptions.push(statusBarManager);
    await statusBarManager.initialize();

    // Register refresh command
    const refreshCmd = vscode.commands.registerCommand('gitch.refreshIdentity', async () => {
      await statusBarManager?.refresh();
      await updateScmPlaceholder(binaryPath);
    });
    context.subscriptions.push(refreshCmd);

    // Register switch identity command (clicking status bar)
    const switchCmd = registerSwitchIdentityCommand(
      context,
      binaryPath,
      getPreferredWorkspacePath,
      async () => {
        await statusBarManager?.refresh();
        await updateScmPlaceholder(binaryPath);
      }
    );
    context.subscriptions.push(switchCmd);

    // Register auto-switch on workspace change
    const autoSwitchWatcher = registerAutoSwitch(
      context,
      binaryPath,
      async () => {
        await statusBarManager?.refresh();
        await updateScmPlaceholder(binaryPath);
      }
    );
    context.subscriptions.push(autoSwitchWatcher);

    // Initial auto-switch check for current workspace
    const initialFolder = vscode.workspace.workspaceFolders?.[0];
    if (initialFolder) {
      await checkAutoSwitch(binaryPath, initialFolder.uri.fsPath);
    }

    // Watch for workspace folder changes
    const workspaceWatcher = vscode.workspace.onDidChangeWorkspaceFolders(() => {
      console.log('[gitch] Workspace folders changed, refreshing identity');
      statusBarManager?.scheduleRefresh();
      updateScmPlaceholder(binaryPath);
    });
    context.subscriptions.push(workspaceWatcher);

    // Watch for git extension state changes (identity might change on branch switch)
    const gitExtension = vscode.extensions.getExtension('vscode.git')?.exports;
    if (gitExtension) {
      const git = gitExtension.getAPI(1);
      if (git && git.repositories) {
        for (const repo of git.repositories) {
          const stateListener = repo.state.onDidChange(() => {
            statusBarManager?.scheduleRefresh();
            updateScmPlaceholder(binaryPath);
          });
          context.subscriptions.push(stateListener);
        }

        // Also watch for new repositories
        const repoListener = git.onDidOpenRepository((repo: { state: { onDidChange: (cb: () => void) => vscode.Disposable } }) => {
          const stateListener = repo.state.onDidChange(() => {
            statusBarManager?.scheduleRefresh();
            updateScmPlaceholder(binaryPath);
          });
          context.subscriptions.push(stateListener);
        });
        context.subscriptions.push(repoListener);
      }
    }

    // Initial SCM placeholder update
    await updateScmPlaceholder(binaryPath);

    console.log('[gitch] Extension activated with status bar');
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    console.error(`[gitch] Failed to activate: ${message}`);
    vscode.window.showErrorMessage(`gitch: Failed to activate. ${message}`);
  }
}

/**
 * Update SCM input box placeholder to show current identity.
 */
async function updateScmPlaceholder(binaryPath: string): Promise<void> {
  const gitExtension = vscode.extensions.getExtension('vscode.git')?.exports;
  if (!gitExtension) return;

  const git = gitExtension.getAPI(1);
  if (!git || !git.repositories) return;

  for (const repo of git.repositories) {
    if (repo.inputBox) {
      const workspacePath = repo.rootUri?.fsPath ?? getPreferredWorkspacePath();
      const identity = await getCurrentIdentity(binaryPath, workspacePath);
      if (!identity) {
        repo.inputBox.placeholder = '';
        continue;
      }

      const authorName = identity.git_name || identity.name;
      repo.inputBox.placeholder = `Committing as: ${authorName} <${identity.email}>`;
    }
  }
}

export function deactivate(): void {
  console.log('[gitch] Extension deactivated');
  statusBarManager = undefined;
}
