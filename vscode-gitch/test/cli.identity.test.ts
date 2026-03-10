import assert from 'node:assert/strict';
import { chmod, mkdir, mkdtemp, readFile, realpath, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { test } from 'node:test';

import {
  autoSwitchIdentity,
  getCurrentIdentity,
  listIdentities,
  switchIdentity,
} from '../src/cli/identity';
import { runGitch } from '../src/cli/runner';

async function createFakeGitch(scriptBody: string): Promise<string> {
  const dir = await mkdtemp(path.join(tmpdir(), 'gitch-ext-'));
  const scriptPath = path.join(dir, 'fake-gitch');
  const contents = `#!/bin/sh
set -eu
${scriptBody}
`;

  await writeFile(scriptPath, contents, 'utf8');
  await chmod(scriptPath, 0o755);

  return scriptPath;
}

function shQuote(value: string): string {
  return `'${value.replace(/'/g, `'\"'\"'`)}'`;
}

test('runGitch trims stdout output', async () => {
  const binaryPath = await createFakeGitch(`printf ' hello \\n'`);

  const output = await runGitch(binaryPath, []);

  assert.equal(output, 'hello');
});

test('getCurrentIdentity parses gitch status JSON', async () => {
  const dir = await mkdtemp(path.join(tmpdir(), 'gitch-ext-status-'));
  const workspacePath = path.join(dir, 'workspace');
  const cwdPath = path.join(dir, 'cwd.txt');
  await mkdir(workspacePath, { recursive: true });

  const binaryPath = await createFakeGitch(`
if [ "$1" = "status" ] && [ "$2" = "--json" ]; then
  pwd > ${shQuote(cwdPath)}
  printf '%s' '{"name":"work","git_name":"Jane Doe","email":"jane@example.com","managed":true}'
  exit 0
fi
echo "unexpected args: $*" >&2
exit 1
`);

  const identity = await getCurrentIdentity(binaryPath, workspacePath);
  const cwd = (await readFile(cwdPath, 'utf8')).trim();

  assert.deepEqual(identity, {
    name: 'work',
    git_name: 'Jane Doe',
    email: 'jane@example.com',
    managed: true,
  });
  assert.equal(cwd, await realpath(workspacePath));
});

test('listIdentities parses gitch list JSON', async () => {
  const dir = await mkdtemp(path.join(tmpdir(), 'gitch-ext-list-'));
  const workspacePath = path.join(dir, 'workspace');
  const cwdPath = path.join(dir, 'cwd.txt');
  await mkdir(workspacePath, { recursive: true });

  const binaryPath = await createFakeGitch(`
if [ "$1" = "list" ] && [ "$2" = "--json" ]; then
  pwd > ${shQuote(cwdPath)}
  printf '%s' '[{"name":"work","git_name":"Jane Doe","email":"jane@example.com","is_active":true,"is_default":false}]'
  exit 0
fi
echo "unexpected args: $*" >&2
exit 1
`);

  const identities = await listIdentities(binaryPath, workspacePath);
  const cwd = (await readFile(cwdPath, 'utf8')).trim();

  assert.deepEqual(identities, [
    {
      name: 'work',
      git_name: 'Jane Doe',
      email: 'jane@example.com',
      is_active: true,
      is_default: false,
    },
  ]);
  assert.equal(cwd, await realpath(workspacePath));
});

test('autoSwitchIdentity runs explicit autoswitch command in workspace', async () => {
  const dir = await mkdtemp(path.join(tmpdir(), 'gitch-ext-workspace-'));
  const workspacePath = path.join(dir, 'workspace');
  const argsPath = path.join(dir, 'args.txt');
  const cwdPath = path.join(dir, 'cwd.txt');

  await mkdir(workspacePath, { recursive: true });

  const binaryPath = await createFakeGitch(`
printf '%s\\n' "$@" > ${shQuote(argsPath)}
pwd > ${shQuote(cwdPath)}
`);

  await autoSwitchIdentity(binaryPath, workspacePath);

  const args = (await readFile(argsPath, 'utf8'))
    .trim()
    .split('\n')
    .filter(Boolean);
  const cwd = (await readFile(cwdPath, 'utf8')).trim();
  const resolvedWorkspacePath = await realpath(workspacePath);

  assert.deepEqual(args, ['autoswitch', '--quiet']);
  assert.equal(cwd, resolvedWorkspacePath);
});

test('switchIdentity runs gitch use with the selected profile', async () => {
  const dir = await mkdtemp(path.join(tmpdir(), 'gitch-ext-switch-'));
  const workspacePath = path.join(dir, 'workspace');
  const argsPath = path.join(dir, 'args.txt');
  const cwdPath = path.join(dir, 'cwd.txt');
  await mkdir(workspacePath, { recursive: true });

  const binaryPath = await createFakeGitch(`
printf '%s\\n' "$@" > ${shQuote(argsPath)}
pwd > ${shQuote(cwdPath)}
`);

  await switchIdentity(binaryPath, 'work', workspacePath);

  const args = (await readFile(argsPath, 'utf8'))
    .trim()
    .split('\n')
    .filter(Boolean);
  const cwd = (await readFile(cwdPath, 'utf8')).trim();

  assert.deepEqual(args, ['use', 'work']);
  assert.equal(cwd, await realpath(workspacePath));
});
