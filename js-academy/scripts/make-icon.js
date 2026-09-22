'use strict';

// Renders build/icon.svg to the PNG icons the app and its AppImage use.
// Run with: xvfb-run -a npx electron scripts/make-icon.js

const fs = require('fs');
const path = require('path');
const { app, BrowserWindow } = require('electron');

app.whenReady().then(async () => {
  const svg = fs.readFileSync(path.join(__dirname, '..', 'build', 'icon.svg'), 'utf8');
  const win = new BrowserWindow({ width: 512, height: 512, show: false, frame: false, transparent: true, webPreferences: { offscreen: true } });
  const html = `<html><body style="margin:0;background:transparent">${svg}</body></html>`;
  await win.loadURL('data:text/html;charset=utf-8,' + encodeURIComponent(html));
  await new Promise((r) => setTimeout(r, 300));
  const image = await win.webContents.capturePage({ x: 0, y: 0, width: 512, height: 512 });
  const png = image.resize({ width: 512, height: 512 }).toPNG();
  fs.writeFileSync(path.join(__dirname, '..', 'build', 'icon.png'), png);
  fs.writeFileSync(path.join(__dirname, '..', 'src', 'renderer', 'icon.png'), image.resize({ width: 256, height: 256 }).toPNG());
  console.log('wrote build/icon.png and src/renderer/icon.png');
  app.quit();
});
