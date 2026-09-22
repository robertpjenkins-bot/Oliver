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

Learn.init();
Playground.init();
