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
const tmp2 = fs.mkdtempSync(path.join(os.tmpdir(), 'specflow-test-'));
const tmp3 = fs.mkdtempSync(path.join(os.tmpdir(), 'specflow-test-'));
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

  check('managed files carry specflow region markers', () => {
    for (const f of ['AGENTS.md', 'specflow/procedures/claim-batch.md']) {
      const c = fs.readFileSync(path.join(tmp, f), 'utf8');
      assert.ok(c.includes('<!-- specflow:start -->') && c.includes('<!-- specflow:end -->'), f + ' lacks markers');
    }
  });

  check('stamp records managed-region baseline hashes', () => {
    const j = JSON.parse(fs.readFileSync(path.join(tmp, 'specflow/.spec-batch.json'), 'utf8'));
    assert.ok(j.managed && typeof j.managed['AGENTS.md'] === 'string', 'no managed baseline for AGENTS.md');
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

  // --- non-destructive upgrade (Batch U) ---
  run(tmp2, ['init', '--agents=claude']);
  const AG2 = path.join(tmp2, 'AGENTS.md');

  // 1. user text added OUTSIDE the markers survives an upgrade verbatim.
  fs.appendFileSync(AG2, '\n## Our team notes\nDeploy on Fridays only.\n');
  const up2 = run(tmp2, ['upgrade']);
  check('upgrade preserves user text outside the markers', () => {
    assert.strictEqual(up2.status, 0);
    assert.ok(fs.readFileSync(AG2, 'utf8').includes('Deploy on Fridays only.'), 'user text was lost');
  });

  // 2. an edit INSIDE the managed region is reported as drift, never overwritten.
  const edited = fs.readFileSync(AG2, 'utf8').replace('single source of truth', 'SINGLE SOURCE OF TRUTH (my edit)');
  fs.writeFileSync(AG2, edited);
  const up3 = run(tmp2, ['upgrade']);
  check('upgrade does not overwrite a hand-edited managed region', () => {
    assert.ok(fs.readFileSync(AG2, 'utf8').includes('SINGLE SOURCE OF TRUTH (my edit)'), 'edit was clobbered');
  });
  check('upgrade reports drift and writes a .specflow-new sidecar', () => {
    assert.ok(/edited|drift/i.test(up3.stdout), 'no drift warning printed');
    assert.ok(fs.existsSync(AG2 + '.specflow-new'), 'no .specflow-new sidecar');
  });

  // 3. a pre-marker install is migrated: backed up, then markers added — no data lost.
  run(tmp3, ['init', '--agents=claude']);
  const AG3 = path.join(tmp3, 'AGENTS.md');
  const stripped = fs.readFileSync(AG3, 'utf8').split('<!-- specflow:start -->').join('').split('<!-- specflow:end -->').join('');
  fs.writeFileSync(AG3, stripped);
  const up4 = run(tmp3, ['upgrade']);
  check('upgrade migrates a pre-marker file with a backup', () => {
    assert.strictEqual(up4.status, 0);
    assert.ok(fs.existsSync(AG3 + '.specflow-bak'), 'no .specflow-bak backup');
    assert.ok(fs.readFileSync(AG3, 'utf8').includes('<!-- specflow:start -->'), 'markers not added on migrate');
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
  for (const d of [tmp, tmp2, tmp3]) fs.rmSync(d, { recursive: true, force: true });
}

console.log(`\n${pass} passed, ${fail} failed`);
process.exit(fail ? 1 : 0);
