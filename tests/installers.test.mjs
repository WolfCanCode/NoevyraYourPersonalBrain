import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {readFileSync} from 'node:fs';
import test from 'node:test';

const shell = readFileSync(new URL('../install.sh', import.meta.url), 'utf8');
const powershell = readFileSync(new URL('../install.ps1', import.meta.url), 'utf8');

test('POSIX installer is valid shell and verifies release checksums', () => {
  const syntax = spawnSync('sh', ['-n'], {input: shell, encoding: 'utf8'});
  assert.equal(syntax.status, 0, syntax.stderr);
  assert.match(shell, /SHA256SUMS/);
  assert.match(shell, /sha256sum|shasum/);
  assert.match(shell, /releases\/download/);
  assert.doesNotMatch(shell, /\bsudo\b/);
});

test('PowerShell installer verifies the Windows release before installation', () => {
  assert.match(powershell, /Get-FileHash -Algorithm SHA256/);
  assert.match(powershell, /brain_\$\{Version\}_windows_amd64\.zip/);
  assert.match(powershell, /releases\/download/);
});

