/**
 * CLI runner module for gitch VS Code extension.
 * Executes gitch commands and returns output.
 */

import { execFile } from 'child_process';
import { promisify } from 'util';

const execFileAsync = promisify(execFile);

const DEFAULT_TIMEOUT = 10000; // 10 seconds
const MAX_BUFFER = 1024 * 1024; // 1MB

/**
 * Execute gitch CLI with arguments.
 *
 * @param binaryPath - Absolute path to gitch binary
 * @param args - Command line arguments
 * @returns stdout output trimmed
 * @throws Error with stderr message on failure
 */
export async function runGitch(binaryPath: string, args: string[]): Promise<string> {
  try {
    const { stdout } = await execFileAsync(binaryPath, args, {
      timeout: DEFAULT_TIMEOUT,
      maxBuffer: MAX_BUFFER,
    });
    return stdout.trim();
  } catch (error) {
    if (error instanceof Error && 'stderr' in error) {
      const stderr = (error as { stderr: string }).stderr;
      if (stderr) {
        throw new Error(`gitch command failed: ${stderr.trim()}`);
      }
    }
    throw new Error(`gitch command failed: ${error}`);
  }
}
