'use strict';

const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('api', {
  loadProgress: () => ipcRenderer.invoke('progress:load'),
  saveProgress: (data) => ipcRenderer.invoke('progress:save', data),

  listProjects: () => ipcRenderer.invoke('projects:list'),
  loadProject: (id) => ipcRenderer.invoke('projects:load', id),
  saveProject: (project) => ipcRenderer.invoke('projects:save', project),
  deleteProject: (id) => ipcRenderer.invoke('projects:delete', id),
  exportHtml: (name, html) => ipcRenderer.invoke('projects:exportHtml', { name, html }),

  packAvailable: () => ipcRenderer.invoke('pack:available'),
  packAppImage: (project) => ipcRenderer.invoke('pack:appimage', project),
  onPackProgress: (fn) => {
    const listener = (_e, msg) => fn(msg);
    ipcRenderer.on('pack:progress', listener);
    return () => ipcRenderer.removeListener('pack:progress', listener);
  },

  installStatus: () => ipcRenderer.invoke('installer:status'),
  openInstaller: () => ipcRenderer.invoke('installer:open'),

  showItem: (p) => ipcRenderer.invoke('shell:showItem', p),
  openExternal: (url) => ipcRenderer.invoke('shell:openExternal', url),
});
