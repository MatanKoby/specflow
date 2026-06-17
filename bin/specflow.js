#!/usr/bin/env node
// specflow — drop a spec-driven, batch, claim-before-work protocol into any repo.
// Zero runtime dependencies (Node >=18) so `npx` runs it with no install step.

import fs from 'node:fs';
import path from 'node:path';
import readline from 'node:readline';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '..');
const TEMPLATES = path.join(ROOT, 'templates');
const BASE = path.join(TEMPLATES, 'base');
const AGENTS_DIR = path.join(TEMPLATES, 'agents');
const VERSION = createRequire(import.meta.url)('../package.json').version;

// Paths specflow owns and will overwrite on `upgrade`. Everything else (queue, claims, spec,
// agent stubs) is written once at init and never clobbered.
const MANAGED = ['AGENTS.md', 'specflow/procedures'];

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

  // Overwrite only the managed mechanism files; never touch queue/claims/spec state.
  let count = 0;
  for (const rel of MANAGED) {
    const src = path.join(BASE, rel);
    const dest = path.join(targetDir, rel);
    if (!fs.existsSync(src)) continue;
    const stat = fs.statSync(src);
    if (stat.isDirectory()) {
      const res = copyTree(src, dest, 'overwrite');
      count += res.written.length;
    } else {
      fs.mkdirSync(path.dirname(dest), { recursive: true });
      fs.copyFileSync(src, dest);
      count += 1;
    }
  }
  stamp.kitVersion = VERSION;
  stamp.upgradedAt = new Date().toISOString().slice(0, 10);
  fs.writeFileSync(stampPath, JSON.stringify(stamp, null, 2) + '\n');

  console.log(C.green(`\n✓ Upgraded specflow ${from} → ${VERSION}`) + C.dim(` (${count} managed file(s) refreshed)`));
  console.log(C.dim('  AGENTS.md + specflow/procedures/ updated. Your queue, claims, and spec were left untouched.'));
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
