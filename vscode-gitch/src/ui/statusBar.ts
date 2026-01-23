/**
 * Status bar management for gitch VS Code extension.
 * Shows current identity with hover tooltip.
 */

import * as vscode from 'vscode';
import { GitchIdentity, getCurrentIdentity } from '../cli/identity';

const DEBOUNCE_MS = 500;

export class StatusBarManager implements vscode.Disposable {
  private statusBarItem: vscode.StatusBarItem;
  private binaryPath: string;
  private updateTimer: NodeJS.Timeout | undefined;
  private disposed = false;

  constructor(binaryPath: string) {
    this.binaryPath = binaryPath;

    // Create status bar item on left side, high priority (shows near git branch)
    this.statusBarItem = vscode.window.createStatusBarItem(
      vscode.StatusBarAlignment.Left,
      100
    );

    // Command to switch identity via Quick Pick
    this.statusBarItem.command = 'gitch.switchIdentity';
  }

  /**
   * Initialize status bar and start watching.
   * Call this after construction.
   */
  async initialize(): Promise<void> {
    await this.refresh();
    this.statusBarItem.show();
  }

  /**
   * Refresh identity display (debounced).
   */
  scheduleRefresh(): void {
    if (this.disposed) return;

    if (this.updateTimer) {
      clearTimeout(this.updateTimer);
    }

    this.updateTimer = setTimeout(() => {
      this.refresh();
    }, DEBOUNCE_MS);
  }

  /**
   * Immediately refresh identity display.
   */
  async refresh(): Promise<void> {
    if (this.disposed) return;

    const identity = await getCurrentIdentity(this.binaryPath);
    this.updateDisplay(identity);
  }

  /**
   * Update status bar text and tooltip.
   */
  private updateDisplay(identity: GitchIdentity | null): void {
    if (!identity) {
      this.statusBarItem.text = '$(person) No Identity';
      this.statusBarItem.tooltip = 'No gitch identity configured';
      this.statusBarItem.backgroundColor = new vscode.ThemeColor(
        'statusBarItem.warningBackground'
      );
      return;
    }

    // Text: $(person) identity-name
    this.statusBarItem.text = `$(person) ${identity.name}`;

    // Tooltip: Rich markdown with details
    const tooltip = new vscode.MarkdownString();
    tooltip.appendMarkdown(`**gitch Identity**\n\n`);
    tooltip.appendMarkdown(`**Name:** ${identity.name}\n\n`);
    tooltip.appendMarkdown(`**Email:** ${identity.email}\n\n`);

    if (identity.ssh_key_path) {
      tooltip.appendMarkdown(`**SSH Key:** \`${identity.ssh_key_path}\`\n\n`);
    }

    if (identity.gpg_key_id) {
      tooltip.appendMarkdown(`**GPG Key:** \`${identity.gpg_key_id}\` $(verified)\n\n`);
    } else {
      tooltip.appendMarkdown(`**GPG:** Not configured\n\n`);
    }

    if (!identity.managed) {
      tooltip.appendMarkdown(`\n---\n*Identity not managed by gitch*`);
    }

    this.statusBarItem.tooltip = tooltip;
    this.statusBarItem.backgroundColor = undefined; // Reset warning color
  }

  dispose(): void {
    this.disposed = true;
    if (this.updateTimer) {
      clearTimeout(this.updateTimer);
    }
    this.statusBarItem.dispose();
  }
}
