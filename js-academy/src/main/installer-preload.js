'use strict';

const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('installer', {
  status: () => ipcRenderer.invoke('installer:status'),
  install: (options) => ipcRenderer.invoke('installer:install', options),
  launch: () => ipcRenderer.invoke('installer:launch'),
});
