'use strict';

// Every lesson file calls registerModule(). Works in the app (browser global)
// and in the Node validator (globalThis).
(function (root) {
  const modules = (root.JSA_MODULES = root.JSA_MODULES || []);
  root.registerModule = function registerModule(mod) {
    mod.lessons.forEach((lesson, i) => {
      lesson.id = mod.id + '-' + (i + 1);
      lesson.module = mod.id;
    });
    modules.push(mod);
  };
})(typeof window !== 'undefined' ? window : globalThis);
