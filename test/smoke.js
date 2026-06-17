// Smoke test for the specflow CLI. Zero dependencies — `node test/smoke.js`.
// Spawns the real bin against temp dirs and asserts the observable behavior.

import assert from 'node:assert';
import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const BIN = path.join(__dirname, '..', 'bin', 'specflow.js');

let pass = 0;
let fail = 0;
function check(name, fn) {
  try {
    fn();
    console.log('  \x1b[32m✓\x1b[0m ' + name);
    pass++;
  } catch (e) {
    console.log('  \x1b[31m✗\x1b[0m ' + name + ' — ' + e.message);
    fail++;
  }
}

function run(cwd, args) {
  return spawnSync('node', [BIN, ...args], { cwd, encoding: 'utf8' });
}

const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'specflow-test-'));
try {
  // --- init ---
  const init = run(tmp, ['init', '--agents=claude,cursor']);
  check('init exits 0', () => assert.strictEqual(init.status, 0));

  const expected = [
    'AGENTS.md', 'BUILD_QUEUE.md', 'BUILD_QUEUE_DONE.md', 'CLAIMS.md', 'CLAIMS_DONE.md',
    'spec/README.md', 'specflow/.spec-batch.json',
    'specflow/procedures/claim-batch.md', 'specflow/procedures/finish-batch.md', 'specflow/procedures/spec-edit.md',
    'CLAUDE.md', '.claude/skills/claim-batch/SKILL.md', '.cursor/rules/specflow.mdc',
  ];
  check('writes all base + selected-agent files', () => {
    for (const f of expected) assert.ok(fs.existsSync(path.join(tmp, f)), 'missing ' + f);
  });

  check('does not write unselected adapters', () => {
    assert.ok(!fs.existsSync(path.join(tmp, '.github/copilot-instructions.md')), 'copilot leaked in');
  });

  check('fills the version stamp (no placeholders left)', () => {
    const raw = fs.readFileSync(path.join(tmp, 'specflow/.spec-batch.json'), 'utf8');
    assert.ok(!raw.includes('{{'), 'unfilled placeholder remains');
    const j = JSON.parse(raw);
    assert.ok(/^\d+\.\d+\.\d+/.test(j.kitVersion), 'kitVersion not a version');
    assert.strictEqual(j.agents, 'claude,cursor');
  });

  // --- re-init guard ---
  const reinit = run(tmp, ['init', '--agents=claude']);
  check('re-init is guarded (exit 0, no clobber)', () => {
    assert.strictEqual(reinit.status, 0);
    assert.ok(/already/i.test(reinit.stdout), 'no "already installed" notice');
  });

  // --- upgrade ---
  const claimsBefore = fs.readFileSync(path.join(tmp, 'CLAIMS.md'), 'utf8');
  const up = run(tmp, ['upgrade']);
  check('upgrade exits 0', () => assert.strictEqual(up.status, 0));
  check('upgrade preserves state files (CLAIMS.md untouched)', () => {
    assert.strictEqual(fs.readFileSync(path.join(tmp, 'CLAIMS.md'), 'utf8'), claimsBefore);
  });
  check('upgrade records upgradedAt in the stamp', () => {
    const j = JSON.parse(fs.readFileSync(path.join(tmp, 'specflow/.spec-batch.json'), 'utf8'));
    assert.ok(j.upgradedAt, 'no upgradedAt');
  });

  // --- version ---
  for (const flag of ['--version', '-v']) {
    const v = run(tmp, [flag]);
    check(`${flag} prints a version`, () => {
      assert.strictEqual(v.status, 0);
      assert.ok(/^\d+\.\d+\.\d+/.test(v.stdout.trim()), 'not a version: ' + v.stdout.trim());
    });
  }

  // --- unknown command ---
  const bad = run(tmp, ['frobnicate']);
  check('unknown command exits non-zero', () => assert.notStrictEqual(bad.status, 0));

  // --- non-git warning ---
  check('init warns when not in a git repo', () => {
    assert.ok(/not a git repository/i.test(init.stdout), 'no git warning');
  });
} finally {
  fs.rmSync(tmp, { recursive: true, force: true });
}

console.log(`\n${pass} passed, ${fail} failed`);
process.exit(fail ? 1 : 0);
