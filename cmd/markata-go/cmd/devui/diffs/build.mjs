import { cp, mkdir, mkdtemp, readdir, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';

const source = resolve(import.meta.dir, 'pierre-review.js');
const destination = resolve(import.meta.dir, '../../../../../pkg/servecontrol/pierre-diffs');
const temporary = await mkdtemp(join(tmpdir(), 'markata-pierre-'));

try {
  const build = Bun.spawnSync([
    'bun', 'build', source, `--outdir=${temporary}`, '--target=browser',
    '--minify', '--splitting',
  ]);
  if (build.exitCode !== 0) {
    throw new Error(new TextDecoder().decode(build.stderr));
  }

  const files = await readdir(temporary);
  const chosen = new Set(['pierre-review.js']);
  for (const name of files) {
    if (/^(markdown|pierre-light|pierre-dark|wasm)-[a-z0-9]+\.js$/.test(name)) {
      chosen.add(name);
    }
  }
  const pending = [...chosen];
  while (pending.length) {
    const name = pending.pop();
    const content = await readFile(join(temporary, name), 'utf8');
    for (const match of content.matchAll(/(?:from|import)\s*["']\.\/([^"']+)["']/g)) {
      if (!chosen.has(match[1])) {
        chosen.add(match[1]);
        pending.push(match[1]);
      }
    }
  }
  await rm(destination, { recursive: true, force: true });
  await mkdir(destination, { recursive: true });
  for (const name of [...chosen].sort()) {
    await cp(join(temporary, name), join(destination, name));
  }
  await cp(resolve(import.meta.dir, 'node_modules/@pierre/diffs/LICENSE.md'),
    join(destination, 'LICENSE.pierre.txt'));
  await cp(resolve(import.meta.dir, 'node_modules/shiki/LICENSE'),
    join(destination, 'LICENSE.shiki.txt'));
  const bytes = (await Promise.all([...chosen].map(async name =>
    (await Bun.file(join(destination, name)).size)))).reduce((a, b) => a + b, 0);
  console.log(`Bundled ${chosen.size} Pierre assets (${bytes} bytes) in ${destination}`);
} finally {
  await rm(temporary, { recursive: true, force: true });
}
