'use strict';

/* global Learn, Playground */

document.querySelectorAll('.tab').forEach((tab) => {
  tab.onclick = () => {
    document.querySelectorAll('.tab').forEach((t) => t.classList.toggle('active', t === tab));
    document.querySelectorAll('.view').forEach((v) => v.classList.toggle('active', v.id === 'view-' + tab.dataset.view));
    (tab.dataset.view === 'learn' ? Learn : Playground).refresh();
  };
});

// Links inside lesson text open in the system browser.
document.addEventListener('click', (e) => {
  const a = e.target.closest('a[href^="http"]');
  if (a) {
    e.preventDefault();
    window.api.openExternal(a.href);
  }
});

// Offer to install when running from an AppImage that is not installed yet.
const installBtn = document.getElementById('btn-install');
installBtn.onclick = () => window.api.openInstaller();
const checkInstalled = () =>
  window.api.installStatus().then((st) => {
    installBtn.hidden = !(st.canInstall && !st.installed);
  });
checkInstalled();
window.addEventListener('focus', checkInstalled);

Learn.init();
Playground.init();
