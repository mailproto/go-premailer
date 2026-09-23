// Generates golden files from juice, the reference implementation.
//
// This is the ONLY thing that may write testdata/out. The Go test suite never
// regenerates goldens: doing so would certify the implementation against itself
// and destroy the oracle.
//
//   node generate.mjs            regenerate all goldens
//   node generate.mjs --check    verify committed goldens match pinned juice
import fs from 'node:fs/promises';
import path from 'node:path';
import juice from 'juice';

const ROOT = path.resolve(import.meta.dirname, '..');
const IN = path.join(ROOT, 'in');
const OUT = path.join(ROOT, 'out');
const STAMP = `juice ${await version()}\nnode ${process.versions.node}\n`;

async function version() {
  const p = path.join(import.meta.dirname, 'node_modules', 'juice', 'package.json');
  return JSON.parse(await fs.readFile(p, 'utf8')).version;
}

// ignoredPseudos, excludedProperties, widthElements, styleToAttribute and
// codeBlocks are module-global on the juice client rather than per-call
// options. Without a reset between fixtures, one fixture's `client` overrides
// leak into every later golden.
const DEFAULTS = {
  ignoredPseudos: [...juice.ignoredPseudos],
  widthElements: [...juice.widthElements],
  heightElements: [...juice.heightElements],
  tableElements: [...juice.tableElements],
  nonVisualElements: [...juice.nonVisualElements],
  styleToAttribute: { ...juice.styleToAttribute },
  excludedProperties: [...juice.excludedProperties],
  codeBlocks: structuredClone(juice.codeBlocks),
};

async function* fixtures(dir) {
  let entries;
  try {
    entries = await fs.readdir(dir, { withFileTypes: true });
  } catch (e) {
    if (e.code === 'ENOENT') return;
    throw e;
  }
  for (const e of entries.sort((a, b) => a.name.localeCompare(b.name))) {
    const p = path.join(dir, e.name);
    if (e.isDirectory()) yield* fixtures(p);
    else if (e.name.endsWith('.html')) yield p;
  }
}

const check = process.argv.includes('--check');
const drift = [];
let count = 0;

for await (const file of fixtures(IN)) {
  const rel = path.relative(IN, file);
  let cfg = {};
  try {
    cfg = JSON.parse(await fs.readFile(file.replace(/\.html$/, '.json'), 'utf8'));
  } catch (e) {
    if (e.code !== 'ENOENT') throw new Error(`${rel}: bad options sidecar: ${e.message}`);
  }

  Object.assign(juice, structuredClone(DEFAULTS), cfg.client ?? {});

  const html = await fs.readFile(file, 'utf8');
  let body, outPath;
  try {
    body = juice(html, cfg.options ?? {});
    outPath = path.join(OUT, rel);
  } catch (err) {
    body = `${err.name}: ${err.message}\n`;
    outPath = path.join(OUT, rel.replace(/\.html$/, '.err'));
  }
  count++;

  if (check) {
    const prev = await fs.readFile(outPath, 'utf8').catch(() => null);
    if (prev !== body) drift.push(rel);
  } else {
    await fs.mkdir(path.dirname(outPath), { recursive: true });
    await fs.writeFile(outPath, body);
  }
}

if (check) {
  const prev = await fs.readFile(path.join(OUT, 'VERSION'), 'utf8').catch(() => null);
  if (prev !== STAMP) drift.push(`VERSION (have ${JSON.stringify(prev)}, want ${JSON.stringify(STAMP)})`);
  if (drift.length) {
    console.error(`goldens out of date (${drift.length}):\n  ${drift.join('\n  ')}`);
    console.error('\nrun `make goldens` and review the diff');
    process.exit(1);
  }
  console.log(`goldens up to date (${count} fixtures, ${STAMP.trim().replace('\n', ', ')})`);
} else {
  await fs.mkdir(OUT, { recursive: true });
  await fs.writeFile(path.join(OUT, 'VERSION'), STAMP);
  console.log(`wrote ${count} goldens with ${STAMP.trim().replace('\n', ', ')}`);
}
