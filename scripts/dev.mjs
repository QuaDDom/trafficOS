import { spawn } from 'node:child_process';
import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
const root = resolve(import.meta.dirname, '..');
if (existsSync(resolve(root, '.env'))) {
 for (const line of readFileSync(resolve(root, '.env'), 'utf8').split(/\r?\n/)) {
  const match = line.match(/^([A-Z_]+)=(.*)$/); if (match && !process.env[match[1]]) process.env[match[1]] = match[2].replace(/^['"]|['"]$/g, '');
 }
}
if (process.argv.includes('--memory')) process.env.STORAGE_MODE = 'memory';
const portable = resolve(root, '.tools/go/bin/go.exe');
const go = existsSync(portable) ? portable : 'go';
const children = [spawn(go, ['run', './apps/api'], { cwd: root, stdio: 'inherit', windowsHide: true }), spawn(go, ['run', './apps/simulator'], { cwd: root, stdio: 'inherit', windowsHide: true }), spawn(process.execPath, [resolve(root, 'node_modules/next/dist/bin/next'), 'dev', 'apps/web'], { cwd: root, stdio: 'inherit', windowsHide: true })];
let stopping = false;
function stop(code = 0) { if (stopping) return; stopping = true; for (const child of children) child.kill(); process.exitCode = code; }
for (const child of children) { child.on('error', err => { console.error(err.message); stop(1); }); child.on('exit', code => { if (!stopping) stop(code ?? 1); }); }
process.on('SIGINT', () => stop()); process.on('SIGTERM', () => stop());
