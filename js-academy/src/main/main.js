'use strict';

const fs = require('fs');
const path = require('path');
const { app, BrowserWindow, ipcMain, dialog, shell, Menu } = require('electron');
const store = require('./store');
const { packProject, toolsAvailable } = require('./packager');
const installer = require('./installer');

let win;
let installerWin;
const ICON = path.join(__dirname, '..', 'renderer', 'icon.png');

function createWindow() {
  win = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 900,
    minHeight: 600,
    title: 'JS Academy',
    backgroundColor: '#0f1420',
    icon: ICON,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
    },
  });
  Menu.setApplicationMenu(null);
  win.loadFile(path.join(__dirname, '..', 'renderer', 'index.html'));

  // Links in lessons open in the real browser, never inside the app.
  win.webContents.setWindowOpenHandler(({ url }) => {
    if (/^https?:/.test(url)) shell.openExternal(url);
    return { action: 'deny' };
  });
  win.webContents.on('will-navigate', (e, url) => {
    if (!url.startsWith('file:')) e.preventDefault();
  });
  win.webContents.on('before-input-event', (e, input) => {
    if (input.type === 'keyDown' && (input.key === 'F12' || (input.control && input.shift && input.key === 'I'))) {
      win.webContents.toggleDevTools();
    }
  });
}

// ── installing into the desktop ──

const installPrefsFile = () => path.join(app.getPath('userData'), 'install.json');

function readInstallPrefs() {
  try {
    return JSON.parse(require('fs').readFileSync(installPrefsFile(), 'utf8'));
  } catch {
    return {};
  }
}

function createInstallerWindow() {
  if (installerWin) return installerWin.focus();
  installerWin = new BrowserWindow({
    width: 580,
    height: 500,
    resizable: false,
    title: 'Install JS Academy',
    backgroundColor: '#0f1420',
    icon: ICON,
    webPreferences: {
      preload: path.join(__dirname, 'installer-preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
    },
  });
  installerWin.setMenu(null);
  installerWin.loadFile(path.join(__dirname, '..', 'installer', 'index.html'));
  installerWin.on('closed', () => {
    installerWin = null;
    if (!win) app.quit();
  });
}

ipcMain.handle('installer:status', () => installer.status());
ipcMain.handle('installer:install', (_e, options) => {
  try {
    const r = installer.install(options, ICON);
    fs.mkdirSync(app.getPath('userData'), { recursive: true });
    fs.writeFileSync(installPrefsFile(), JSON.stringify({ ...readInstallPrefs(), installed: Date.now() }));
    return r;
  } catch (err) {
    return { error: err.message };
  }
});
ipcMain.handle('installer:launch', () => {
  // Remember the choice, so "Just run it" is not asked again every launch.
  fs.mkdirSync(app.getPath('userData'), { recursive: true });
  fs.writeFileSync(installPrefsFile(), JSON.stringify({ ...readInstallPrefs(), asked: Date.now() }));
  if (!win) createWindow();
  if (installerWin) installerWin.close();
});
ipcMain.handle('installer:open', () => createInstallerWindow());

ipcMain.handle('progress:load', () => store.loadProgress());
ipcMain.handle('progress:save', (_e, data) => store.saveProgress(data));

ipcMain.handle('projects:list', () => store.listProjects());
ipcMain.handle('projects:load', (_e, id) => store.loadProject(id));
ipcMain.handle('projects:save', (_e, project) => store.saveProject(project));
ipcMain.handle('projects:delete', (_e, id) => store.deleteProject(id));

ipcMain.handle('projects:exportHtml', async (_e, { name, html }) => {
  const { canceled, filePath } = await dialog.showSaveDialog(win, {
    title: 'Save as web page',
    defaultPath: path.join(app.getPath('documents'), `${store.slug(name)}.html`),
    filters: [{ name: 'HTML', extensions: ['html'] }],
  });
  if (canceled || !filePath) return { canceled: true };
  fs.writeFileSync(filePath, html);
  return { filePath };
});

ipcMain.handle('pack:available', () => toolsAvailable());

ipcMain.handle('pack:appimage', async (event, project) => {
  const { canceled, filePath } = await dialog.showSaveDialog(win, {
    title: 'Pack project as AppImage',
    defaultPath: path.join(app.getPath('documents'), `${store.slug(project.name) || 'app'}-x86_64.AppImage`),
    filters: [{ name: 'AppImage', extensions: ['AppImage'] }],
  });
  if (canceled || !filePath) return { canceled: true };
  const log = (msg) => event.sender.send('pack:progress', msg);
  try {
    const out = await packProject(project, filePath, log);
    return { filePath: out };
  } catch (err) {
    return { error: err.message };
  }
});

ipcMain.handle('shell:showItem', (_e, p) => shell.showItemInFolder(p));
ipcMain.handle('shell:openExternal', (_e, url) => {
  if (/^https?:/.test(url)) shell.openExternal(url);
});

app.whenReady().then(async () => {
  // Command-line installing, e.g. ./JS-Academy.AppImage --uninstall
  if (process.argv.includes('--uninstall')) {
    const { removed } = installer.uninstall();
    const msg = removed.length ? 'JS Academy was removed from your desktop.\n\n' + removed.join('\n') : 'JS Academy was not installed.';
    console.log(msg);
    if (!process.stdout.isTTY) await dialog.showMessageBox({ type: 'info', title: 'JS Academy', message: msg });
    return app.quit();
  }
  if (process.argv.includes('--install')) {
    try {
      const { done } = installer.install({ menu: true, shortcut: true, copy: true }, ICON);
      console.log(done.join('\n'));
    } catch (err) {
      console.error(err.message);
      process.exitCode = 1;
    }
    return app.quit();
  }

  const st = installer.status();
  if (st.canInstall && !st.installed && !readInstallPrefs().asked) createInstallerWindow();
  else createWindow();
  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on('window-all-closed', () => app.quit());
