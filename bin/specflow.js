#!/usr/bin/env node
// specflow — drop a spec-driven, batch, claim-before-work protocol into any repo.
// Zero runtime dependencies (Node >=18) so `npx` runs it with no install step.

import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import readline from 'node:readline';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '..');
const TEMPLATES = path.join(ROOT, 'templates');
const BASE = path.join(TEMPLATES, 'base');
const AGENTS_DIR = path.join(TEMPLATES, 'agents');
const VERSION = createRequire(import.meta.url)('../package.json').version;

// Files/dirs whose specflow-managed *region* `upgrade` refreshes. Each managed file wraps its
// generated content in `START … END` markers; `upgrade` replaces only what's between them and
// preserves everything outside. Everything else (queue, claims, spec, agent stubs) is written
// once at init and never touched.
const MANAGED = ['AGENTS.md', 'specflow/procedures'];

// Markers are matched by their `specflow:start` / `specflow:end` token, not an exact string, so the
// human-readable note inside a marker can evolve without breaking parsing or forcing a migration.
const START_RE = /<!--\s*specflow:start\b.*?-->/s;
const END_RE = /<!--\s*specflow:end\b.*?-->/s;

// Expand MANAGED into concrete relpaths by walking the template tree (the authoritative set).
function managedFileList() {
  const out = [];
  for (const rel of MANAGED) {
    const src = path.join(BASE, rel);
    if (!fs.existsSync(src)) continue;
    if (fs.statSync(src).isDirectory()) {
      for (const f of listFiles(src)) out.push(path.relative(BASE, f).split(path.sep).join('/'));
    } else {
      out.push(rel);
    }
  }
  return out;
}

// Split a managed file around its single specflow region. Returns null if the markers are
// absent or malformed (a pre-marker install, or hand-mangled). `startMarker`/`endMarker` carry the
// matched marker text verbatim so the template's wording can be re-applied on a clean refresh.
function extractRegion(content) {
  const sm = content.match(START_RE);
  const em = content.match(END_RE);
  if (!sm || !em) return null;
  const sEnd = sm.index + sm[0].length;
  if (em.index < sEnd) return null;
  return {
    before: content.slice(0, sm.index),
    startMarker: sm[0],
    region: content.slice(sEnd, em.index),
    endMarker: em[0],
    after: content.slice(em.index + em[0].length),
  };
}

function hashRegion(region) {
  return crypto.createHash('sha256').update(region).digest('hex');
}

// Baseline hash of each managed file's region, as currently on disk. Stored in the stamp so a
// later `upgrade` can tell a pristine region (safe to refresh) from a hand-edited one (drift).
function computeManaged(targetDir) {
  const map = {};
  for (const rel of managedFileList()) {
    const dest = path.join(targetDir, rel);
    if (!fs.existsSync(dest)) continue;
    const parts = extractRegion(fs.readFileSync(dest, 'utf8'));
    if (parts) map[rel] = hashRegion(parts.region);
  }
  return map;
}

const AGENT_CHOICES = [
  { key: 'claude', label: 'Claude Code', detail: 'CLAUDE.md + auto-triggering skills' },
  { key: 'cursor', label: 'Cursor', detail: '.cursor/rules/specflow.mdc' },
  { key: 'copilot', label: 'GitHub Copilot', detail: '.github/copilot-instructions.md' },
  { key: 'bob', label: 'IBM Bob', detail: '.bob/rules/ (also reads AGENTS.md)' },
  { key: 'antigravity', label: 'Google Antigravity', detail: '.agents/rules/ (also reads AGENTS.md)' },
];

const C = {
  bold: (s) => `\x1b[1m${s}\x1b[0m`,
  green: (s) => `\x1b[32m${s}\x1b[0m`,
  yellow: (s) => `\x1b[33m${s}\x1b[0m`,
  dim: (s) => `\x1b[2m${s}\x1b[0m`,
  cyan: (s) => `\x1b[36m${s}\x1b[0m`,
};

function isGitRepo(dir) {
  let cur = dir;
  for (;;) {
    if (fs.existsSync(path.join(cur, '.git'))) return true;
    const parent = path.dirname(cur);
    if (parent === cur) return false;
    cur = parent;
  }
}

function listFiles(dir) {
  const out = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) out.push(...listFiles(full));
    else out.push(full);
  }
  return out;
}

// Copy every file under `srcRoot` into `destRoot`, preserving structure.
// onExisting: 'skip' (init) or 'overwrite' (upgrade). Returns { written, skipped }.
function copyTree(srcRoot, destRoot, onExisting) {
  const written = [];
  const skipped = [];
  for (const src of listFiles(srcRoot)) {
    const rel = path.relative(srcRoot, src);
    const dest = path.join(destRoot, rel);
    if (fs.existsSync(dest) && onExisting === 'skip') {
      skipped.push(rel);
      continue;
    }
    fs.mkdirSync(path.dirname(dest), { recursive: true });
    fs.copyFileSync(src, dest);
    written.push(rel);
  }
  return { written, skipped };
}

function fillStamp(targetDir, agentKeys) {
  const stampPath = path.join(targetDir, 'specflow', '.spec-batch.json');
  if (!fs.existsSync(stampPath)) return;
  const filled = fs
    .readFileSync(stampPath, 'utf8')
    .replace('{{VERSION}}', VERSION)
    .replace('{{INIT_DATE}}', new Date().toISOString().slice(0, 10))
    .replace('{{AGENTS}}', agentKeys.join(','));
  fs.writeFileSync(stampPath, filled);
}

// Read the stamp, attach/refresh the managed-region baseline map, write it back.
function recordManaged(targetDir, extra) {
  const stampPath = path.join(targetDir, 'specflow', '.spec-batch.json');
  if (!fs.existsSync(stampPath)) return;
  const stamp = JSON.parse(fs.readFileSync(stampPath, 'utf8'));
  stamp.managed = computeManaged(targetDir);
  Object.assign(stamp, extra);
  fs.writeFileSync(stampPath, JSON.stringify(stamp, null, 2) + '\n');
}

function ask(question) {
  const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
  return new Promise((resolve) => rl.question(question, (a) => { rl.close(); resolve(a.trim()); }));
}

async function pickAgents(preset) {
  if (preset) {
    const keys = preset.split(',').map((s) => s.trim()).filter(Boolean);
    const valid = keys.filter((k) => AGENT_CHOICES.some((c) => c.key === k));
    const bad = keys.filter((k) => !valid.includes(k));
    if (bad.length) console.log(C.yellow(`  ignoring unknown agent(s): ${bad.join(', ')}`));
    return valid;
  }
  console.log(C.bold('\nWhich agents will work in this repo?') + C.dim('  (AGENTS.md is always written as the universal base)'));
  AGENT_CHOICES.forEach((c, i) => {
    console.log(`  ${C.cyan(String(i + 1))}. ${c.label}  ${C.dim(c.detail)}`);
  });
  const answer = await ask(
    `\nEnter numbers (comma-separated), ${C.cyan('a')} for all, or Enter for ${C.cyan('Claude Code')}: `
  );
  if (answer === '') return ['claude'];
  if (answer.toLowerCase() === 'a') return AGENT_CHOICES.map((c) => c.key);
  const idx = answer.split(',').map((s) => parseInt(s.trim(), 10) - 1);
  return idx.filter((i) => i >= 0 && i < AGENT_CHOICES.length).map((i) => AGENT_CHOICES[i].key);
}

async function cmdInit(args) {
  const targetDir = process.cwd();
  const preset = (args.find((a) => a.startsWith('--agents=')) || '').split('=')[1] || (args.includes('--all') ? AGENT_CHOICES.map((c) => c.key).join(',') : '');

  if (fs.existsSync(path.join(targetDir, 'specflow', '.spec-batch.json'))) {
    console.log(C.yellow('\nThis repo already has specflow installed.') + ` Run ${C.cyan('specflow upgrade')} to update it.\n`);
    return;
  }

  if (!isGitRepo(targetDir)) {
    console.log(
      C.yellow('\n⚠ Not a git repository.') +
        C.dim(" specflow's claim/commit workflow assumes git — run `git init` for the full flow.")
    );
  }

  const agentKeys = await pickAgents(preset);
  if (!agentKeys.length) {
    console.log(C.yellow('\nNo agents selected. Writing the universal AGENTS.md base only.\n'));
  }

  console.log(C.bold(`\nspecflow ${VERSION}`) + ` → ${C.dim(targetDir)}`);

  const base = copyTree(BASE, targetDir, 'skip');
  let skipped = [...base.skipped];
  for (const key of agentKeys) {
    const dir = path.join(AGENTS_DIR, key);
    if (!fs.existsSync(dir)) continue;
    const res = copyTree(dir, targetDir, 'skip');
    skipped = skipped.concat(res.skipped);
  }
  fillStamp(targetDir, agentKeys);
  recordManaged(targetDir);

  console.log(C.green('\n✓ specflow installed.'));
  console.log('  Base protocol:   AGENTS.md, BUILD_QUEUE.md, CLAIMS.md, spec/, specflow/');
  if (agentKeys.length) console.log('  Agent adapters:  ' + agentKeys.join(', '));
  if (skipped.length) {
    console.log(C.yellow(`\n  Left ${skipped.length} existing file(s) untouched (review/merge manually):`));
    skipped.forEach((f) => console.log(C.dim('    · ' + f)));
  }
  console.log(C.bold('\nNext steps:'));
  console.log('  1. Fill in ' + C.cyan('spec/README.md') + ' with what this project is.');
  console.log('  2. Replace the example batch in ' + C.cyan('BUILD_QUEUE.md') + ' with real work.');
  console.log('  3. Point your agent at ' + C.cyan('AGENTS.md') + ' and let it claim a batch.\n');
}

function cmdUpgrade() {
  const targetDir = process.cwd();
  const stampPath = path.join(targetDir, 'specflow', '.spec-batch.json');
  if (!fs.existsSync(stampPath)) {
    console.log(C.yellow('\nNo specflow install found here.') + ` Run ${C.cyan('specflow init')} first.\n`);
    return;
  }
  const stamp = JSON.parse(fs.readFileSync(stampPath, 'utf8'));
  const from = stamp.kitVersion;
  const baseline = stamp.managed || {};
  const nextManaged = { ...baseline };

  // Refresh only specflow's own marked region in each managed file; never touch text outside
  // the markers, and never clobber a region that's been hand-edited since install.
  const refreshed = [];
  const added = [];
  const migrated = [];
  const drifted = [];

  for (const rel of managedFileList()) {
    const srcContent = fs.readFileSync(path.join(BASE, rel), 'utf8');
    const srcParts = extractRegion(srcContent);
    if (!srcParts) continue; // template lacks markers — nothing to manage (shouldn't happen)
    const dest = path.join(targetDir, rel);

    // Managed file introduced by a newer kit version: write it whole.
    if (!fs.existsSync(dest)) {
      fs.mkdirSync(path.dirname(dest), { recursive: true });
      fs.writeFileSync(dest, srcContent);
      nextManaged[rel] = hashRegion(srcParts.region);
      added.push(rel);
      continue;
    }

    const destContent = fs.readFileSync(dest, 'utf8');
    const destParts = extractRegion(destContent);

    // Pre-marker install: no region to target. Back up verbatim, then write the marked template.
    if (!destParts) {
      fs.writeFileSync(dest + '.specflow-bak', destContent);
      fs.writeFileSync(dest, srcContent);
      nextManaged[rel] = hashRegion(srcParts.region);
      migrated.push(rel);
      continue;
    }

    // Region hand-edited since install → never overwrite. Leave it, drop the new version
    // alongside for manual reconciliation, and keep flagging it (baseline hash unchanged).
    const baseHash = baseline[rel];
    if (baseHash && hashRegion(destParts.region) !== baseHash) {
      fs.writeFileSync(dest + '.specflow-new', srcContent);
      drifted.push(rel);
      continue;
    }

    // Clean: swap in the fresh region (and the template's current marker wording), preserve
    // everything outside the markers verbatim.
    const updated = destParts.before + srcParts.startMarker + srcParts.region + srcParts.endMarker + destParts.after;
    if (updated !== destContent) {
      fs.writeFileSync(dest, updated);
      refreshed.push(rel);
    }
    nextManaged[rel] = hashRegion(srcParts.region);
  }

  stamp.kitVersion = VERSION;
  stamp.upgradedAt = new Date().toISOString().slice(0, 10);
  stamp.managed = nextManaged;
  fs.writeFileSync(stampPath, JSON.stringify(stamp, null, 2) + '\n');

  console.log(C.green(`\n✓ Upgraded specflow ${from} → ${VERSION}`));
  if (refreshed.length) console.log(C.dim(`  refreshed: ${refreshed.join(', ')}`));
  if (added.length) console.log(C.dim(`  added:     ${added.join(', ')}`));
  if (migrated.length) {
    console.log(C.yellow(`  migrated to managed-region format (previous saved as *.specflow-bak): ${migrated.join(', ')}`));
  }
  if (drifted.length) {
    console.log(C.yellow(`\n  ⚠ ${drifted.length} managed region(s) edited since install — left untouched:`));
    drifted.forEach((f) => console.log(C.dim(`    · ${f}  → new version written to ${f}.specflow-new (reconcile, then re-run upgrade)`)));
  }
  if (!refreshed.length && !added.length && !migrated.length && !drifted.length) {
    console.log(C.dim('  Already current — nothing to refresh.'));
  }
  console.log(C.dim('\n  Your queue, claims, and spec were left untouched.'));
  if (stamp.schemaVersion !== 1) console.log(C.yellow('  Note: schema changed — review state-file format.'));
  console.log('');
}

function usage() {
  console.log(`
${C.bold('specflow')} ${C.dim(VERSION)} — spec-driven batch/claim protocol for AI coding agents

${C.bold('Usage:')}
  npx specflow init [--agents=claude,cursor] [--all]   ${C.dim('scaffold into the current repo')}
  npx specflow upgrade                                 ${C.dim('refresh the managed protocol files')}
  npx specflow --version                               ${C.dim('print the installed version')}
  npx specflow --help

${C.bold('Agents:')} ${AGENT_CHOICES.map((c) => c.key).join(', ')}
`);
}

const [, , command, ...args] = process.argv;
(async () => {
  try {
    switch (command) {
      case 'init': await cmdInit(args); break;
      case 'upgrade': cmdUpgrade(); break;
      case '--version':
      case '-v':
      case 'version': console.log(VERSION); break;
      case undefined:
      case '--help':
      case '-h':
      case 'help': usage(); break;
      default:
        console.log(C.yellow(`Unknown command: ${command}`));
        usage();
        process.exit(1);
    }
  } catch (err) {
    console.error(C.yellow('\nspecflow error: ') + (err && err.message ? err.message : String(err)));
    process.exit(1);
  }
})();
