'use strict';

// Everything the user makes lives under the app's data folder
// (~/.config/JS Academy): lesson progress in one JSON file and each
// playground project in its own folder.

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

function dataDir() {
  // Required lazily so the packager tests can use this module outside Electron.
  const { app } = require('electron');
  return app.getPath('userData');
}

const progressFile = () => path.join(dataDir(), 'progress.json');
const projectsDir = () => path.join(dataDir(), 'projects');

function readJson(file, fallback) {
  try {
    return JSON.parse(fs.readFileSync(file, 'utf8'));
  } catch {
    return fallback;
  }
}

// Write to a temporary file first so a crash mid-write never leaves a
// half-written progress file behind.
function writeJson(file, data) {
  fs.mkdirSync(path.dirname(file), { recursive: true });
  const tmp = `${file}.${process.pid}.tmp`;
  fs.writeFileSync(tmp, JSON.stringify(data, null, 2));
  fs.renameSync(tmp, file);
}

function slug(name) {
  return String(name || '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 60);
}

function loadProgress() {
  return readJson(progressFile(), { completed: {}, code: {}, current: null });
}

function saveProgress(data) {
  writeJson(progressFile(), data);
  return true;
}

function safeId(id) {
  if (!/^[a-z0-9-]+$/.test(id || '')) throw new Error('invalid project id');
  return id;
}

function listProjects() {
  const dir = projectsDir();
  if (!fs.existsSync(dir)) return [];
  return fs
    .readdirSync(dir)
    .map((id) => readJson(path.join(dir, id, 'project.json'), null))
    .filter(Boolean)
    .map(({ id, name, updated, template }) => ({ id, name, updated, template }))
    .sort((a, b) => (b.updated || 0) - (a.updated || 0));
}

function loadProject(id) {
  const dir = path.join(projectsDir(), safeId(id));
  const meta = readJson(path.join(dir, 'project.json'), null);
  if (!meta) return null;
  const read = (f) => {
    try {
      return fs.readFileSync(path.join(dir, f), 'utf8');
    } catch {
      return '';
    }
  };
  return { ...meta, html: read('index.html'), css: read('style.css'), js: read('script.js') };
}

function saveProject(project) {
  const id = project.id ? safeId(project.id) : `${slug(project.name) || 'project'}-${crypto.randomBytes(3).toString('hex')}`;
  const dir = path.join(projectsDir(), id);
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(path.join(dir, 'index.html'), project.html || '');
  fs.writeFileSync(path.join(dir, 'style.css'), project.css || '');
  fs.writeFileSync(path.join(dir, 'script.js'), project.js || '');
  const meta = {
    id,
    name: project.name || 'Untitled',
    template: project.template || 'blank',
    width: project.width || 900,
    height: project.height || 700,
    created: project.created || Date.now(),
    updated: Date.now(),
  };
  writeJson(path.join(dir, 'project.json'), meta);
  return meta;
}

function deleteProject(id) {
  fs.rmSync(path.join(projectsDir(), safeId(id)), { recursive: true, force: true });
  return true;
}

module.exports = { loadProgress, saveProgress, listProjects, loadProject, saveProject, deleteProject, slug };
