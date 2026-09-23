'use strict';

// Installs a running AppImage into the user's desktop: a permanent copy in
// ~/Applications, an entry in the applications menu, and optionally a
// desktop shortcut. Everything lives in the user's home folder, so no
// administrator password is needed, and uninstalling removes exactly what
// was added (lessons progress and projects are kept).

const fs = require('fs');
const os = require('os');
const path = require('path');
const { execFileSync } = require('child_process');

const ID = 'js-academy';
const home = () => os.homedir();
const dataHome = () => process.env.XDG_DATA_HOME || path.join(home(), '.local', 'share');
const menuFile = () => path.join(dataHome(), 'applications', `${ID}.desktop`);
const iconFile = () => path.join(dataHome(), 'icons', 'hicolor', '256x256', 'apps', `${ID}.png`);
const appsDir = () => path.join(home(), 'Applications');
const installedAppImage = () => path.join(appsDir(), 'JS-Academy.AppImage');

// The AppImage runtime tells us where the file we were started from is.
const runningAppImage = () => process.env.APPIMAGE || null;

function desktopDir() {
  try {
    const dir = execFileSync('xdg-user-dir', ['DESKTOP'], { encoding: 'utf8' }).trim();
    if (dir && dir !== home()) return dir;
  } catch {
    // xdg-user-dir is not everywhere
  }
  return path.join(home(), 'Desktop');
}
const shortcutFile = () => path.join(desktopDir(), `${ID}.desktop`);

function readExec(file) {
  try {
    const m = fs.readFileSync(file, 'utf8').match(/^Exec="?([^"\n]+?)"?(?: %U)?$/m);
    return m ? m[1] : null;
  } catch {
    return null;
  }
}

function status() {
  const exec = readExec(menuFile());
  const installed = !!exec && fs.existsSync(exec);
  return {
    canInstall: process.platform === 'linux' && !!runningAppImage(),
    installed,
    installedPath: installed ? exec : null,
    runningInstalled: installed && runningAppImage() === exec,
    runningFrom: runningAppImage(),
    shortcut: fs.existsSync(shortcutFile()),
    appsDir: appsDir(),
    desktopDir: desktopDir(),
  };
}

function desktopEntry(exec) {
  return `[Desktop Entry]
Type=Application
Name=JS Academy
GenericName=Learn JavaScript
Comment=Learn JavaScript with 500 lessons, and build your own apps
Exec="${exec}" %U
Icon=${ID}
Terminal=false
Categories=Education;Development;
Keywords=javascript;coding;programming;learn;
StartupWMClass=${ID}
`;
}

function refreshMenus() {
  for (const [cmd, args] of [
    ['update-desktop-database', [path.dirname(menuFile())]],
    ['gtk-update-icon-cache', ['-q', '-t', path.join(dataHome(), 'icons', 'hicolor')]],
  ]) {
    try {
      execFileSync(cmd, args, { stdio: 'ignore' });
    } catch {
      // optional helpers
    }
  }
}

/**
 * options: { copy: boolean, menu: boolean, shortcut: boolean }
 * iconPng: path to the PNG icon inside the running app
 */
function install(options, iconPng) {
  const source = runningAppImage();
  if (!source) throw new Error('JS Academy is not running from an AppImage, so there is nothing to install.');
  const done = [];

  let target = source;
  if (options.copy && source !== installedAppImage()) {
    fs.mkdirSync(appsDir(), { recursive: true });
    const tmp = installedAppImage() + '.part';
    fs.copyFileSync(source, tmp);
    fs.chmodSync(tmp, 0o755);
    fs.renameSync(tmp, installedAppImage());
    target = installedAppImage();
    done.push(`Copied the app to ${target}`);
  }

  fs.mkdirSync(path.dirname(iconFile()), { recursive: true });
  fs.copyFileSync(iconPng, iconFile());

  if (options.menu !== false) {
    fs.mkdirSync(path.dirname(menuFile()), { recursive: true });
    fs.writeFileSync(menuFile(), desktopEntry(target), { mode: 0o755 });
    done.push('Added JS Academy to your applications menu');
  }

  if (options.shortcut) {
    fs.mkdirSync(desktopDir(), { recursive: true });
    fs.writeFileSync(shortcutFile(), desktopEntry(target), { mode: 0o755 });
    try {
      // GNOME only launches desktop files it has been told to trust.
      execFileSync('gio', ['set', shortcutFile(), 'metadata::trusted', 'true'], { stdio: 'ignore' });
    } catch {
      // not GNOME
    }
    done.push('Put a shortcut on your desktop');
  }

  refreshMenus();
  return { target, done };
}

function uninstall() {
  const removed = [];
  const st = status();
  for (const file of [menuFile(), shortcutFile(), iconFile()]) {
    if (fs.existsSync(file)) {
      fs.rmSync(file, { force: true });
      removed.push(file);
    }
  }
  // Only remove the copy we made ourselves, never the file the user downloaded.
  if (st.installedPath === installedAppImage() && fs.existsSync(installedAppImage())) {
    fs.rmSync(installedAppImage(), { force: true });
    removed.push(installedAppImage());
  }
  refreshMenus();
  return { removed };
}

module.exports = { status, install, uninstall, installedAppImage };
