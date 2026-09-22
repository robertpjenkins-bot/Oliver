'use strict';

// Downloads appimagetool and the AppImage runtime into build-tools/. They are
// bundled into JS Academy so it can pack playground projects offline.

const fs = require('fs');
const path = require('path');
const https = require('https');

const dir = path.join(__dirname, '..', 'build-tools');
const files = {
  'appimagetool-x86_64.AppImage': 'https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage',
  'runtime-x86_64': 'https://github.com/AppImage/type2-runtime/releases/download/continuous/runtime-x86_64',
};

function download(url, dest, redirects = 0) {
  return new Promise((resolve, reject) => {
    https
      .get(url, { headers: { 'User-Agent': 'js-academy' } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          res.resume();
          if (redirects > 10) return reject(new Error('too many redirects'));
          return resolve(download(new URL(res.headers.location, url).toString(), dest, redirects + 1));
        }
        if (res.statusCode !== 200) {
          res.resume();
          return reject(new Error(`${url}: HTTP ${res.statusCode}`));
        }
        const tmp = `${dest}.part`;
        const out = fs.createWriteStream(tmp);
        res.pipe(out);
        out.on('finish', () => out.close(() => (fs.renameSync(tmp, dest), resolve())));
        out.on('error', reject);
      })
      .on('error', reject);
  });
}

(async () => {
  fs.mkdirSync(dir, { recursive: true });
  for (const [name, url] of Object.entries(files)) {
    const dest = path.join(dir, name);
    if (fs.existsSync(dest) && fs.statSync(dest).size > 0) {
      console.log(`✓ ${name} (already present)`);
      continue;
    }
    console.log(`↓ ${name}`);
    await download(url, dest);
    fs.chmodSync(dest, 0o755);
  }
  console.log(`tools ready in ${dir}`);
})().catch((err) => {
  console.error(`could not download tools: ${err.message}`);
  process.exit(1);
});
