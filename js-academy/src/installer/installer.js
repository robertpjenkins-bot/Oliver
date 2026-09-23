'use strict';

const $ = (id) => document.getElementById(id);

(async () => {
  const st = await window.installer.status();
  $('shortcut-where').textContent = `A JS Academy icon in ${st.desktopDir}.`;
  $('copy-where').textContent = `Saved as ${st.appsDir}/JS-Academy.AppImage, so you can delete the file you downloaded.`;
})();

$('skip').onclick = () => window.installer.launch();
$('open').onclick = () => window.installer.launch();

$('install').onclick = async () => {
  $('install').disabled = true;
  $('skip').disabled = true;
  $('error').hidden = true;
  const r = await window.installer.install({
    menu: $('opt-menu').checked,
    shortcut: $('opt-shortcut').checked,
    copy: $('opt-copy').checked,
  });
  if (r.error) {
    $('error').textContent = 'Could not install: ' + r.error;
    $('error').hidden = false;
    $('install').disabled = false;
    $('skip').disabled = false;
    return;
  }
  $('title').textContent = 'JS Academy is installed 🎉';
  $('subtitle').textContent = 'Everything is ready.';
  $('done-list').replaceChildren(...(r.done.length ? r.done : ['Nothing to change']).map((t) => Object.assign(document.createElement('li'), { textContent: t })));
  $('choose').hidden = true;
  $('finished').hidden = false;
  $('install').hidden = true;
  $('skip').hidden = true;
  $('open').hidden = false;
  $('open').focus();
};
