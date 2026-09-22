'use strict';

// Packs a small project with the Electron runtime from node_modules and
// checks the result is a working AppImage that contains the project.

const fs = require('fs');
const os = require('os');
const path = require('path');
const { execFileSync } = require('child_process');
const { packProject } = require('../src/main/packager');

function hasCmd(cmd) {
  try {
    execFileSync('sh', ['-c', `command -v ${cmd}`], { stdio: 'ignore' });
    return true;
  } catch {
    return false;
  }
}

(async () => {
  const electronDist = path.join(__dirname, '..', 'node_modules', 'electron', 'dist');
  const out = path.join(os.tmpdir(), `jsa-test-${process.pid}.AppImage`);
  const doc = '<!doctype html><title>Hi</title><h1 id="t">Packed!</h1><script>document.title="ok"</script>';
  await packProject({ name: 'Test Packed App', document: doc, width: 400, height: 300 }, out, (m) => console.log('  ' + m), {
    runtimeDir: electronDist,
    exeName: 'electron',
  });
  // Launch it for real when a display (or xvfb-run) is available.
  const env = { ...process.env, APPIMAGE_EXTRACT_AND_RUN: '1', JSA_SMOKE_TEST: '1' };
  delete env.NODE_OPTIONS;
  let launch = null;
  if (process.env.DISPLAY) launch = [out, []];
  else if (hasCmd('xvfb-run')) launch = ['xvfb-run', ['-a', out]];
  if (launch) {
    const outText = execFileSync(launch[0], launch[1], { env, stdio: ['ignore', 'pipe', 'ignore'], timeout: 60000 }).toString();
    if (!outText.includes('JSA_SMOKE_TITLE=ok')) throw new Error(`packed app did not run the project:\n${outText}`);
    console.log('  packed app launched and ran the project script');
  } else {
    console.log('  (no display; skipped launching the packed app)');
  }
  const extractDir = fs.mkdtempSync(path.join(os.tmpdir(), 'jsa-x-'));
  execFileSync(out, ['--appimage-extract'], { cwd: extractDir });
  const root = path.join(extractDir, 'squashfs-root');
  const html = fs.readFileSync(path.join(root, 'resources', 'app', 'index.html'), 'utf8');
  if (!html.includes('Packed!')) throw new Error('project page missing from AppImage');
  for (const f of ['AppRun', 'test-packed-app', 'test-packed-app.desktop', 'resources/app/main.js']) {
    if (!fs.existsSync(path.join(root, f))) throw new Error(`${f} missing from AppImage`);
  }
  if (fs.existsSync(path.join(root, 'resources', 'default_app.asar'))) throw new Error('default_app.asar leaked into AppImage');
  console.log(`✓ packer produced ${out} (${(fs.statSync(out).size / 1048576).toFixed(1)} MB)`);
  fs.rmSync(extractDir, { recursive: true, force: true });
  fs.rmSync(out, { force: true });
})().catch((err) => {
  console.error(`✗ ${err.stack || err}`);
  process.exit(1);
});
