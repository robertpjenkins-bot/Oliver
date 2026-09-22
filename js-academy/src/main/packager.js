'use strict';

// Packs a playground project into a standalone AppImage.
//
// The AppImage needs a browser engine to show the project, and the one this
// app is already running on is the obvious choice: the packer copies the
// Electron runtime it was started from, swaps JS Academy's own code for a tiny
// app that opens the project's page, and hands the folder to appimagetool.
// Nothing is downloaded, so packing works offline.

const fs = require('fs');
const os = require('os');
const path = require('path');
const { spawn } = require('child_process');

const ARCH = 'x86_64';

function slug(name) {
  return String(name || '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 60) || 'my-app';
}

function isPackaged() {
  try {
    return require('electron').app.isPackaged;
  } catch {
    return false;
  }
}

// Where appimagetool and the AppImage runtime live: inside the installed
// app's resources, or in build-tools/ when running from a source checkout.
function toolsDir() {
  if (process.env.JSA_TOOLS_DIR) return process.env.JSA_TOOLS_DIR;
  if (isPackaged()) return path.join(process.resourcesPath, 'tools');
  return path.join(__dirname, '..', '..', 'build-tools');
}

function findTools(dir = toolsDir()) {
  const tool = path.join(dir, `appimagetool-${ARCH}.AppImage`);
  const runtime = path.join(dir, `runtime-${ARCH}`);
  return {
    tool: fs.existsSync(tool) ? tool : null,
    runtime: fs.existsSync(runtime) ? runtime : null,
  };
}

function toolsAvailable() {
  const { tool, runtime } = findTools();
  if (process.platform !== 'linux') return { ok: false, reason: 'AppImages can only be built on Linux.' };
  if (!tool || !runtime) {
    return { ok: false, reason: `appimagetool was not found in ${toolsDir()}. Run "npm run fetch-tools" in the js-academy folder.` };
  }
  return { ok: true };
}

// The folder holding the Electron binary this process runs on. Inside an
// AppImage that is the mounted image itself.
function electronRuntimeDir() {
  return path.dirname(process.execPath);
}

// Top-level entries belonging to JS Academy's own AppImage, or to Electron's
// default app, that must not end up in the user's AppImage.
function skipEntry(rel) {
  const parts = rel.split(path.sep);
  const top = parts[0];
  if (parts.length === 1) {
    if (top === 'AppRun' || top === '.DirIcon' || top.endsWith('.desktop')) return true;
    if (/\.(png|svg|xpm)$/.test(top)) return true;
  }
  if (top === 'usr' && parts[1] === 'share') return true;
  if (top === 'resources' && parts.length >= 2) {
    // Keep nothing from resources/ except what Electron itself ships
    // (locale .pak files live elsewhere); the new app goes in resources/app.
    return true;
  }
  return false;
}

function copyRuntime(src, dest, exeName, newExeName) {
  fs.mkdirSync(dest, { recursive: true });
  fs.cpSync(src, dest, {
    recursive: true,
    dereference: false,
    verbatimSymlinks: true,
    filter: (from) => {
      const rel = path.relative(src, from);
      return rel === '' || !skipEntry(rel);
    },
  });
  if (exeName !== newExeName) {
    fs.renameSync(path.join(dest, exeName), path.join(dest, newExeName));
  }
  fs.chmodSync(path.join(dest, newExeName), 0o755);
}

function escapeXml(s) {
  return String(s).replace(/[<>&"']/g, (c) => ({ '<': '&lt;', '>': '&gt;', '&': '&amp;', '"': '&quot;', "'": '&apos;' })[c]);
}

function iconSvg(name) {
  let hash = 0;
  for (const ch of name) hash = (hash * 31 + ch.codePointAt(0)) >>> 0;
  const hue = hash % 360;
  const letters = name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => [...w][0].toUpperCase())
    .join('') || 'JS';
  return `<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256" viewBox="0 0 256 256">
  <defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1">
    <stop offset="0" stop-color="hsl(${hue},70%,55%)"/><stop offset="1" stop-color="hsl(${(hue + 40) % 360},70%,40%)"/>
  </linearGradient></defs>
  <rect width="256" height="256" rx="48" fill="url(#g)"/>
  <text x="128" y="160" font-family="sans-serif" font-weight="700" font-size="104" text-anchor="middle" fill="#fff">${escapeXml(letters)}</text>
</svg>
`;
}

function appMain(title, width, height) {
  return `'use strict';
const path = require('path');
const { app, BrowserWindow, Menu, shell } = require('electron');

app.whenReady().then(() => {
  Menu.setApplicationMenu(null);
  const win = new BrowserWindow({
    width: ${Number(width) || 900},
    height: ${Number(height) || 700},
    title: ${JSON.stringify(title)},
    icon: path.join(__dirname, 'icon.png'),
    webPreferences: { contextIsolation: true, nodeIntegration: false, sandbox: true },
  });
  win.webContents.setWindowOpenHandler(({ url }) => {
    if (/^https?:/.test(url)) shell.openExternal(url);
    return { action: 'deny' };
  });
  win.loadFile(path.join(__dirname, 'index.html'));
  // Used by JS Academy's own tests to prove a packed app starts.
  if (process.env.JSA_SMOKE_TEST) {
    win.webContents.once('did-finish-load', () => {
      console.log('JSA_SMOKE_TITLE=' + win.webContents.getTitle());
      app.quit();
    });
  }
});

app.on('window-all-closed', () => app.quit());
`;
}

function appRun(exe) {
  return `#!/bin/sh
# Starts the packed app. The Chromium sandbox helper cannot be setuid inside
# a read-only AppImage mount, so the app runs without it, as most Electron
# AppImages do.
HERE="$(dirname "$(readlink -f "$0")")"
exec "$HERE/${exe}" --no-sandbox "$@"
`;
}

function desktopEntry(name, exe) {
  const clean = String(name).replace(/[\r\n]/g, ' ');
  return `[Desktop Entry]
Type=Application
Name=${clean}
Comment=Made with JS Academy
Exec=${exe}
Icon=${exe}
Categories=Utility;
Terminal=false
`;
}

function run(cmd, args, env, log) {
  return new Promise((resolve, reject) => {
    const child = spawn(cmd, args, { env });
    let output = '';
    const onData = (buf) => {
      const text = buf.toString();
      output += text;
      text.split('\n').filter((l) => l.trim()).forEach((l) => log(l));
    };
    child.stdout.on('data', onData);
    child.stderr.on('data', onData);
    child.on('error', reject);
    child.on('close', (code) => {
      if (code === 0) resolve(output);
      else reject(new Error(`appimagetool failed (exit ${code}):\n${output.split('\n').slice(-15).join('\n')}`));
    });
  });
}

// A clean environment for appimagetool: when JS Academy itself runs as an
// AppImage, variables describing *its* mount must not leak into the tool.
function toolEnv() {
  const env = { ...process.env, ARCH, APPIMAGE_EXTRACT_AND_RUN: '1' };
  for (const k of ['APPDIR', 'APPIMAGE', 'ARGV0', 'OWD', 'LD_LIBRARY_PATH', 'LD_PRELOAD']) delete env[k];
  return env;
}

/**
 * project: { name, document, width, height, iconPng? }
 *   document is the complete HTML page (markup with the CSS and JS inlined).
 * options.runtimeDir / options.toolsDir override where things are found (tests).
 */
async function packProject(project, outFile, log = () => {}, options = {}) {
  if (process.platform !== 'linux') throw new Error('AppImages can only be built on Linux.');
  const tools = findTools(options.toolsDir || toolsDir());
  if (!tools.tool || !tools.runtime) throw new Error(toolsAvailable().reason || 'appimagetool is missing');

  const name = String(project.name || 'My App').trim() || 'My App';
  const exe = slug(name);
  const runtimeDir = options.runtimeDir || electronRuntimeDir();
  const exeName = options.exeName || path.basename(process.execPath);
  if (!fs.existsSync(path.join(runtimeDir, exeName))) {
    throw new Error(`Could not find the Electron runtime at ${path.join(runtimeDir, exeName)}`);
  }

  const work = fs.mkdtempSync(path.join(os.tmpdir(), 'jsa-pack-'));
  const appDir = path.join(work, `${exe}.AppDir`);
  try {
    log(`Copying the app runtime from ${runtimeDir}…`);
    copyRuntime(runtimeDir, appDir, exeName, exe);

    log('Writing your project…');
    const appSrc = path.join(appDir, 'resources', 'app');
    fs.mkdirSync(appSrc, { recursive: true });
    fs.writeFileSync(
      path.join(appSrc, 'package.json'),
      JSON.stringify({ name: exe, productName: name, version: '1.0.0', main: 'main.js' }, null, 2),
    );
    fs.writeFileSync(path.join(appSrc, 'main.js'), appMain(name, project.width, project.height));
    fs.writeFileSync(path.join(appSrc, 'index.html'), project.document || '<!doctype html><title>Empty</title>');

    const svg = iconSvg(name);
    fs.writeFileSync(path.join(appDir, `${exe}.svg`), svg);
    if (project.iconPng) {
      fs.writeFileSync(path.join(appSrc, 'icon.png'), Buffer.from(project.iconPng, 'base64'));
    }
    fs.symlinkSync(`${exe}.svg`, path.join(appDir, '.DirIcon'));
    fs.writeFileSync(path.join(appDir, `${exe}.desktop`), desktopEntry(name, exe));
    fs.writeFileSync(path.join(appDir, 'AppRun'), appRun(exe), { mode: 0o755 });
    const iconDir = path.join(appDir, 'usr', 'share', 'icons', 'hicolor', 'scalable', 'apps');
    fs.mkdirSync(iconDir, { recursive: true });
    fs.writeFileSync(path.join(iconDir, `${exe}.svg`), svg);

    log('Compressing into an AppImage (this takes a little while)…');
    fs.mkdirSync(path.dirname(outFile), { recursive: true });
    await run(tools.tool, ['--no-appstream', '--runtime-file', tools.runtime, appDir, outFile], toolEnv(), log);
    fs.chmodSync(outFile, 0o755);
    const size = (fs.statSync(outFile).size / 1024 / 1024).toFixed(1);
    log(`Done: ${outFile} (${size} MB)`);
    return outFile;
  } finally {
    if (!options.keepWorkDir) fs.rmSync(work, { recursive: true, force: true });
  }
}

module.exports = { packProject, toolsAvailable, findTools, slug, iconSvg };
