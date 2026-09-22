'use strict';

const path = require('path');
const { app, BrowserWindow, ipcMain, dialog, shell, Menu } = require('electron');
const store = require('./store');
const { packProject, toolsAvailable } = require('./packager');

let win;

function createWindow() {
  win = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 900,
    minHeight: 600,
    title: 'JS Academy',
    backgroundColor: '#0f1420',
    icon: path.join(__dirname, '..', 'renderer', 'icon.png'),
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
  require('fs').writeFileSync(filePath, html);
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

app.whenReady().then(() => {
  createWindow();
  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on('window-all-closed', () => app.quit());
