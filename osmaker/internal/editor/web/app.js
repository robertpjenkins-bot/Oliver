"use strict";
// OS Maker 2.0 designer. The design object here is saved as a .osmaker file
// and is exactly what the Go side turns into an Arch Linux ISO.

const token = location.hash.slice(1);
const $ = (s, el = document) => el.querySelector(s);
const $$ = (s, el = document) => [...el.querySelectorAll(s)];
const h = (tag, attrs = {}, ...kids) => {
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (k === "class") el.className = v;
    else if (k === "style") Object.assign(el.style, v);
    else if (k.startsWith("on")) el.addEventListener(k.slice(2), v);
    else if (v !== false && v != null) el.setAttribute(k, v === true ? "" : v);
  }
  for (const k of kids.flat()) if (k != null) el.append(k.nodeType ? k : document.createTextNode(k));
  return el;
};

async function api(method, path, body) {
  const res = await fetch(path, {
    method,
    headers: { "X-OSMaker-Token": token, "Content-Type": "application/json" },
    body: body === undefined ? undefined : typeof body === "string" ? body : JSON.stringify(body),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || res.statusText);
  return data;
}

// ---------------------------------------------------------------------------
// State

let info;          // catalog, keyboards, build methods… from the server
let D;             // the design
let selected = null;
let history = [], future = [];
let scale = 1;

const WIDGETS = {
  search:   { name: "Search bar",     icon: "🔍", hint: "Find apps or search the web", w: 500, h: 48,  color: "#ffffff", textColor: "#111827", opacity: 0.95, radius: 24, fontSize: 18, text: "Search apps and the web…" },
  clock:    { name: "Clock",          icon: "🕒", hint: "Time and date",               w: 240, h: 110, color: "#000000", textColor: "#ffffff", opacity: 0.35, radius: 16, fontSize: 40, format: "24h" },
  calendar: { name: "Calendar",       icon: "📅", hint: "This month",                  w: 260, h: 230, color: "#000000", textColor: "#ffffff", opacity: 0.35, radius: 16, fontSize: 13 },
  sysmon:   { name: "System monitor", icon: "📈", hint: "CPU and memory use",          w: 280, h: 100, color: "#000000", textColor: "#ffffff", opacity: 0.35, radius: 16, fontSize: 14 },
  note:     { name: "Sticky note",    icon: "🗒️", hint: "Notes you can type in",       w: 220, h: 200, color: "#fff59d", textColor: "#333333", opacity: 1,    radius: 6,  fontSize: 15, text: "Things to do:\n- " },
  text:     { name: "Text",           icon: "🔤", hint: "A title or message",          w: 420, h: 70,  color: "#000000", textColor: "#ffffff", opacity: 0,    radius: 0,  fontSize: 34, text: "Welcome!" },
  install:  { name: "Install button", icon: "💿", hint: "Puts your OS on a computer's disk", w: 90, h: 90, color: "#000000", textColor: "#ffffff", opacity: 0, radius: 12, fontSize: 13, text: "Install" },
  app:      { name: "App icon",       icon: "🧩", hint: "Opens an app",                w: 90,  h: 90,  color: "#000000", textColor: "#ffffff", opacity: 0,    radius: 12, fontSize: 13, app: "files" },
};

const PRESETS = [
  ["gradient", "#1e3c72", "#7b2ff7", 135], ["gradient", "#ff512f", "#dd2476", 135],
  ["gradient", "#11998e", "#38ef7d", 160], ["gradient", "#0f2027", "#2c5364", 180],
  ["gradient", "#f7971e", "#ffd200", 120], ["gradient", "#654ea3", "#eaafc8", 200],
  ["gradient", "#000428", "#004e92", 180], ["gradient", "#ff9a9e", "#fad0c4", 90],
  ["color", "#1d1f27", "#1d1f27", 0],      ["color", "#2d6a4f", "#2d6a4f", 0],
  ["color", "#3a86ff", "#3a86ff", 0],      ["color", "#e9ecef", "#e9ecef", 0],
];

const appById = id => info.catalog.find(a => a.id === id);
const isInstalled = id => { const a = appById(id); return a && (a.base || D.apps.includes(id)); };

// ---------------------------------------------------------------------------
// History and saving

function snapshot() { return JSON.stringify(D); }
let lastSnap = "";
function commit() {
  const s = snapshot();
  if (s === lastSnap) return;
  history.push(lastSnap);
  if (history.length > 150) history.shift();
  future = [];
  lastSnap = s;
  scheduleAutosave();
  render();
}
function undo() {
  if (!history.length) return;
  future.push(lastSnap);
  lastSnap = history.pop();
  D = JSON.parse(lastSnap);
  if (selected && !D.items.some(i => i.id === selected)) selected = null;
  scheduleAutosave(); renderAll();
}
function redo() {
  if (!future.length) return;
  history.push(lastSnap);
  lastSnap = future.pop();
  D = JSON.parse(lastSnap);
  scheduleAutosave(); renderAll();
}

let autosaveTimer;
function scheduleAutosave() {
  clearTimeout(autosaveTimer);
  autosaveTimer = setTimeout(() => api("PUT", "/api/autosave", D).catch(e => status(e.message, true)), 800);
}

function status(msg, err = false) {
  const el = $("#status");
  el.textContent = msg;
  el.classList.toggle("err", err);
}

// ---------------------------------------------------------------------------
// Rendering the designed screen

function bgCSS(bg) {
  if (bg.kind === "image" && bg.image) return `center / cover no-repeat url("${bg.image}")`;
  if (bg.kind === "gradient") return `linear-gradient(${bg.angle}deg, ${bg.color1}, ${bg.color2})`;
  return bg.color1;
}

function rgba(hex, a) {
  const n = parseInt(hex.slice(1), 16);
  return `rgba(${n >> 16},${(n >> 8) & 255},${n & 255},${a})`;
}

function tile(app, size) {
  return h("span", { class: "tile", style: {
    background: app.color, width: size + "px", height: size + "px", fontSize: Math.round(size * 0.58) + "px",
  } }, app.emoji);
}

function clockText(fmt) {
  const now = new Date();
  const t = now.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", hour12: fmt === "12h" }).replace(/^0/, "");
  const d = now.toLocaleDateString([], { weekday: "long", day: "numeric", month: "long" });
  return [t, d];
}

const isIcon = it => it.type === "app" || it.type === "install";

function renderItem(it) {
  const cls = { calendar: "cal", sysmon: "sys", install: "app" }[it.type] || it.type;
  const el = h("div", { class: `item w-${cls}`, "data-id": it.id });
  Object.assign(el.style, {
    left: it.x + "px", top: it.y + "px", width: it.w + "px", height: it.h + "px",
    borderRadius: it.radius + "px", color: it.textColor, fontSize: it.fontSize + "px",
    background: isIcon(it) ? "transparent" : rgba(it.color, it.opacity),
  });
  switch (it.type) {
    case "app":
    case "install": {
      const a = it.type === "install"
        ? { name: it.text || "Install", emoji: "💿", color: D.desktop.accent, description: "Installs your OS onto a computer's disk. Only shows when running from the USB stick." }
        : appById(it.app) || { name: "?", emoji: "❓", color: "#555" };
      const size = Math.max(16, Math.round(Math.min(it.w, it.h - it.fontSize * 1.6) * 0.6));
      el.append(tile(a, size), h("span", { class: "lbl" }, a.name));
      el.title = a.description || a.name;
      break;
    }
    case "search":
      el.append(h("span", {}, "🔍"), h("span", { class: "ph" }, it.text || "Search…"));
      break;
    case "clock": {
      const [t, d] = clockText(it.format);
      el.append(h("div", { class: "t" }, t), h("div", { class: "d", style: { fontSize: Math.max(8, Math.round(it.fontSize * 0.38)) + "px" } }, d));
      break;
    }
    case "calendar": {
      const now = new Date();
      const first = new Date(now.getFullYear(), now.getMonth(), 1);
      const days = new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
      const grid = h("div", { class: "days" }, ..."MTWTFSS".split("").map(c => h("span", { class: "dn" }, c)));
      for (let i = 0; i < (first.getDay() + 6) % 7; i++) grid.append(h("span"));
      for (let d = 1; d <= days; d++) grid.append(h("span", { class: d === now.getDate() ? "today" : "" }, d));
      el.append(h("div", { class: "head" }, now.toLocaleDateString([], { month: "long", year: "numeric" })), grid);
      break;
    }
    case "sysmon":
      for (const [n, v] of [["CPU", 23], ["Memory", 41]])
        el.append(h("div", { class: "r" }, h("span", {}, n), h("span", { class: "bar" }, h("i", { style: { width: v + "%" } })), h("span", {}, v + "%")));
      break;
    case "note":
    case "text":
      el.textContent = it.text || "";
      break;
  }
  el.append(h("span", { class: "handle" }));
  if (it.id === selected) el.classList.add("sel");
  return el;
}

function renderTaskbar() {
  const t = D.taskbar;
  const light = isLightColor(t.color);
  const bar = h("div", { class: "taskbar" + (light ? " light" : ""), style: {
    height: t.size + "px", background: rgba(t.color, t.opacity), [t.position]: "0",
    fontSize: Math.max(11, Math.round(t.size * 0.33)) + "px",
  } });
  bar.append(h("div", { class: "start" }, t.menuLabel || ""));
  for (const id of t.pinned) {
    const a = appById(id);
    if (a) bar.append(h("div", { class: "pin", title: a.name }, tile(a, Math.round(t.size * 0.62))));
  }
  bar.append(h("div", { class: "fill" }));
  const tray = h("div", { class: "tray" }, "📶", "🔊");
  if (t.showClock) tray.append(h("span", { class: "tbclock" }, clockText("24h")[0]));
  tray.append("⏻");
  bar.append(tray);
  return bar;
}

function isLightColor(hex) {
  const n = parseInt(hex.slice(1), 16);
  return ((n >> 16) * 299 + ((n >> 8) & 255) * 587 + (n & 255) * 114) / 1000 > 150;
}

function render() {
  renderScreen();
  renderProps();
  $("#btn-undo").disabled = !history.length;
  $("#btn-redo").disabled = !future.length;
}

function renderScreen() {
  const scr = $("#screen");
  scr.replaceChildren();
  scr.style.background = bgCSS(D.desktop.background);
  scr.style.setProperty("--os-accent", D.desktop.accent);
  for (const it of D.items) scr.append(renderItem(it));
  scr.append(renderTaskbar());
}

function renderAll() {
  render();
  renderApps();
  renderLook();
  renderTaskbarPanel();
  renderSystem();
}

function fit() {
  const stage = $("#stage");
  const pad = 28;
  const W = stage.clientWidth - pad * 2, H = stage.clientHeight - pad * 2;
  scale = Math.max(0.1, Math.min(W / info.canvas.w, H / info.canvas.h));
  const scr = $("#screen");
  scr.style.transform = `scale(${scale})`;
  scr.style.left = Math.round((stage.clientWidth - info.canvas.w * scale) / 2) + "px";
  scr.style.top = Math.round((stage.clientHeight - info.canvas.h * scale) / 2) + "px";
}

// ---------------------------------------------------------------------------
// Placing, moving and resizing

const newId = type => type + Math.random().toString(36).slice(2, 8);
const clamp = (v, lo, hi) => Math.max(lo, Math.min(hi, v));
const snapOn = () => $("#snap").checked;
const snapV = v => snapOn() ? Math.round(v / 10) * 10 : Math.round(v);

function addItem(type, x, y, extra = {}) {
  const w = WIDGETS[type];
  const it = {
    id: newId(type), type, w: w.w, h: w.h,
    color: w.color, textColor: w.textColor, opacity: w.opacity, radius: w.radius, fontSize: w.fontSize,
    ...(w.text ? { text: w.text } : {}), ...(w.format ? { format: w.format } : {}), ...(w.app ? { app: w.app } : {}),
    ...extra,
  };
  if (type === "app" && !isInstalled(it.app)) D.apps.push(it.app);
  it.x = clamp(snapV((x ?? info.canvas.w / 2) - it.w / 2), 0, info.canvas.w - it.w);
  it.y = clamp(snapV((y ?? info.canvas.h / 2) - it.h / 2), 0, info.canvas.h - it.h);
  D.items.push(it);
  selected = it.id;
  commit();
  renderApps();
  return it;
}

function screenPoint(e) {
  const r = $("#screen").getBoundingClientRect();
  return [(e.clientX - r.left) / scale, (e.clientY - r.top) / scale];
}

function setupScreen() {
  const scr = $("#screen");
  let drag = null;

  scr.addEventListener("pointerdown", e => {
    if (document.body.classList.contains("preview")) return;
    const el = e.target.closest(".item");
    if (!el) { selected = null; render(); return; }
    const it = D.items.find(i => i.id === el.dataset.id);
    selected = it.id;
    const [px, py] = screenPoint(e);
    drag = { it, mode: e.target.classList.contains("handle") ? "resize" : "move", px, py, x: it.x, y: it.y, w: it.w, h: it.h, moved: false };
    scr.setPointerCapture(e.pointerId);
    render();
  });
  scr.addEventListener("pointermove", e => {
    if (!drag) return;
    const [px, py] = screenPoint(e);
    const dx = px - drag.px, dy = py - drag.py;
    const it = drag.it, C = info.canvas;
    if (drag.mode === "move") {
      it.x = clamp(snapV(drag.x + dx), 0, C.w - it.w);
      it.y = clamp(snapV(drag.y + dy), 0, C.h - it.h);
    } else {
      it.w = clamp(snapV(drag.w + dx), 16, C.w - it.x);
      it.h = clamp(snapV(drag.h + dy), 16, C.h - it.y);
    }
    drag.moved = true;
    const el = $(`.item[data-id="${it.id}"]`);
    if (drag.mode === "move") Object.assign(el.style, { left: it.x + "px", top: it.y + "px" });
    else renderScreen();
  });
  const end = () => { if (drag && drag.moved) commit(); else if (drag) renderProps(); drag = null; };
  scr.addEventListener("pointerup", end);
  scr.addEventListener("pointercancel", end);

  // Dropping from the palette or the app list.
  const stage = $("#stage");
  stage.addEventListener("dragover", e => { e.preventDefault(); stage.classList.add("drop"); });
  stage.addEventListener("dragleave", () => stage.classList.remove("drop"));
  stage.addEventListener("drop", e => {
    e.preventDefault();
    stage.classList.remove("drop");
    const data = e.dataTransfer.getData("text/osmaker");
    if (!data) return;
    const [x, y] = screenPoint(e);
    const [type, app] = data.split(":");
    addItem(type, x, y, app ? { app } : {});
  });
}

function setupPalette() {
  const pal = $("#palette");
  for (const [type, w] of Object.entries(WIDGETS)) {
    if (type === "app") continue;
    pal.append(h("div", {
      class: "pal", draggable: "true", title: "Click to add, or drag onto the screen",
      ondragstart: e => e.dataTransfer.setData("text/osmaker", type),
      onclick: () => addItem(type),
    }, h("span", { class: "ic" }, w.icon), h("b", {}, w.name), h("small", {}, w.hint)));
  }
  pal.append(h("div", {
    class: "pal", title: "Pick an app in the Apps tab",
    onclick: () => switchTab("apps"),
  }, h("span", { class: "ic" }, "🧩"), h("b", {}, "App icons"), h("small", {}, "In the Apps tab")));
}

// ---------------------------------------------------------------------------
// Properties of the selected item

function field(label, input) { return h("label", { class: "col" }, label, input); }

function numInput(it, key, min, max, step = 1) {
  return h("input", { type: "number", min, max, step, value: it[key], onchange: e => {
    let v = Number(e.target.value);
    if (Number.isNaN(v)) return render();
    it[key] = clamp(v, min, max);
    fixBounds(it);
    commit();
  } });
}

function fixBounds(it) {
  const C = info.canvas;
  it.w = clamp(Math.round(it.w), 16, C.w); it.h = clamp(Math.round(it.h), 16, C.h);
  it.x = clamp(Math.round(it.x), 0, C.w - it.w); it.y = clamp(Math.round(it.y), 0, C.h - it.h);
}

function renderProps() {
  const box = $("#props");
  const it = D.items.find(i => i.id === selected);
  if (!it) {
    box.replaceChildren(h("div", { class: "empty" },
      h("h2", {}, "Nothing selected"),
      h("p", {}, "Click something on the screen to change it."),
      h("p", {}, h("kbd", {}, "Delete"), " removes it, ", h("kbd", {}, "arrows"), " nudge it, ", h("kbd", {}, "Ctrl+D"), " copies it."),
      h("p", {}, `${D.items.length} things on the desktop, ${D.apps.length} extra apps.`)));
    return;
  }
  const w = WIDGETS[it.type];
  const kids = [h("h2", {}, `${w.icon} ${w.name}`)];

  if (it.type === "app") {
    const sel = h("select", { onchange: e => { it.app = e.target.value; if (!isInstalled(it.app)) D.apps.push(it.app); commit(); renderApps(); } });
    for (const c of info.categories) {
      const g = h("optgroup", { label: c });
      for (const a of info.catalog.filter(a => a.category === c)) g.append(h("option", { value: a.id, selected: a.id === it.app }, a.name));
      sel.append(g);
    }
    kids.push(field("App", sel));
  }
  if (["search", "note", "text", "install"].includes(it.type)) {
    const label = { search: "Hint text", note: "Starting text", text: "Text", install: "Label" }[it.type];
    const ta = it.type === "search" || it.type === "install" ? h("input", { type: "text", value: it.text || "" }) : h("textarea", {}, it.text || "");
    ta.addEventListener("input", e => { it.text = e.target.value; renderScreen(); });
    ta.addEventListener("change", commit);
    kids.push(field(label, ta));
  }
  if (it.type === "clock") {
    const s = h("select", { onchange: e => { it.format = e.target.value; commit(); } },
      h("option", { value: "24h", selected: it.format !== "12h" }, "24 hour (14:30)"),
      h("option", { value: "12h", selected: it.format === "12h" }, "12 hour (2:30 PM)"));
    kids.push(field("Clock style", s));
  }

  kids.push(h("div", { class: "grid2" },
    field("X", numInput(it, "x", 0, info.canvas.w)), field("Y", numInput(it, "y", 0, info.canvas.h)),
    field("Width", numInput(it, "w", 16, info.canvas.w)), field("Height", numInput(it, "h", 16, info.canvas.h))));

  const color = (key) => h("input", { type: "color", value: it[key], oninput: e => { it[key] = e.target.value; renderScreen(); }, onchange: commit });
  const range = (key, min, max, step) => h("input", { type: "range", min, max, step, value: it[key], oninput: e => { it[key] = Number(e.target.value); renderScreen(); }, onchange: commit });
  if (it.type === "install") {
    kids.push(h("p", { class: "hint small" }, "This only appears when your OS is running from its USB stick. Clicking it installs the OS onto the computer's disk."));
  }
  if (!isIcon(it)) {
    kids.push(h("label", { class: "row" }, "Background", color("color")));
    kids.push(h("label", { class: "row" }, "Solid", range("opacity", 0, 1, 0.05)));
  }
  kids.push(h("label", { class: "row" }, "Text colour", color("textColor")));
  kids.push(h("label", { class: "row" }, "Text size", range("fontSize", 8, 120, 1)));
  if (!isIcon(it)) kids.push(h("label", { class: "row" }, "Round corners", range("radius", 0, 100, 1)));

  kids.push(h("div", { class: "actions" },
    h("button", { onclick: duplicate }, "Copy"),
    h("button", { onclick: () => { D.items = D.items.filter(i => i !== it); D.items.push(it); commit(); } }, "To front"),
    h("button", { onclick: () => { D.items = D.items.filter(i => i !== it); D.items.unshift(it); commit(); } }, "To back"),
    h("button", { class: "danger", onclick: removeSelected }, "Delete")));
  box.replaceChildren(...kids);
}

function duplicate() {
  const it = D.items.find(i => i.id === selected);
  if (!it) return;
  const copy = { ...it, id: newId(it.type), x: it.x + 20, y: it.y + 20 };
  fixBounds(copy);
  D.items.push(copy);
  selected = copy.id;
  commit();
}

function removeSelected() {
  D.items = D.items.filter(i => i.id !== selected);
  selected = null;
  commit();
}

// ---------------------------------------------------------------------------
// Apps tab

function renderApps() {
  const list = $("#app-list");
  const q = $("#app-filter").value.trim().toLowerCase();
  list.replaceChildren();
  for (const c of info.categories) {
    const apps = info.catalog.filter(a => a.category === c && (!q || (a.name + " " + a.description).toLowerCase().includes(q)));
    if (!apps.length) continue;
    list.append(h("div", { class: "cat" }, c));
    for (const a of apps) {
      const pinned = D.taskbar.pinned.includes(a.id);
      const check = a.base
        ? h("span", { class: "badge", title: "Every OS has this" }, "built in")
        : h("input", { type: "checkbox", checked: D.apps.includes(a.id), title: "Install this app", onchange: e => {
            if (e.target.checked) D.apps.push(a.id);
            else {
              D.apps = D.apps.filter(x => x !== a.id);
              D.items = D.items.filter(i => !(i.type === "app" && i.app === a.id));
              D.taskbar.pinned = D.taskbar.pinned.filter(x => x !== a.id);
            }
            commit(); renderApps(); renderTaskbarPanel();
          } });
      list.append(h("div", { class: "app-row", draggable: "true", ondragstart: e => e.dataTransfer.setData("text/osmaker", "app:" + a.id) },
        tile(a, 34),
        h("div", { class: "meta" }, h("b", {}, a.name), h("span", { title: a.description }, a.description + (a.multilib ? " · 32-bit" : ""))),
        h("button", { title: "Put an icon on the desktop", onclick: () => placeApp(a.id) }, "＋"),
        h("button", { class: pinned ? "on" : "", title: pinned ? "Unpin from the taskbar" : "Pin to the taskbar", onclick: () => togglePin(a.id) }, "📌"),
        check));
    }
  }
  const pkgs = D.apps.length;
  $("#app-count").textContent = `${pkgs} extra app${pkgs === 1 ? "" : "s"} will be installed.`;
}

function placeApp(id) {
  // Put new icons in the next free slot down the left side, like a desktop does.
  const W = WIDGETS.app;
  for (let col = 0; col < 12; col++) {
    for (let row = 0; row < 6; row++) {
      const x = 30 + col * 110, y = 30 + row * 110;
      if (!D.items.some(i => x < i.x + i.w && x + W.w > i.x && y < i.y + i.h && y + W.h > i.y) && y + W.h <= info.canvas.h - D.taskbar.size)
        return addItem("app", x + W.w / 2, y + W.h / 2, { app: id });
    }
  }
  addItem("app", undefined, undefined, { app: id });
}

function togglePin(id) {
  const t = D.taskbar;
  if (t.pinned.includes(id)) t.pinned = t.pinned.filter(x => x !== id);
  else {
    if (t.pinned.length >= 12) return status("The taskbar holds 12 pinned apps at most.", true);
    t.pinned.push(id);
    if (!isInstalled(id)) D.apps.push(id);
  }
  commit(); renderApps(); renderTaskbarPanel();
}

// ---------------------------------------------------------------------------
// Look, Taskbar and System tabs

function seg(id, value, onpick) {
  for (const b of $$(`#${id} button`)) {
    b.classList.toggle("on", b.dataset.v === value);
    b.onclick = () => onpick(b.dataset.v);
  }
}

function renderLook() {
  const dt = D.desktop, bg = dt.background;
  seg("theme-seg", dt.theme, v => { dt.theme = v; commit(); renderLook(); });
  seg("bg-kind", bg.kind, v => {
    if (v === "image" && !bg.image) { $("#bg-file").click(); return; }
    bg.kind = v; commit(); renderLook();
  });
  $("#accent").value = dt.accent;
  $("#bg-c1").value = bg.color1;
  $("#bg-c2").value = bg.color2;
  $("#bg-angle").value = bg.angle;
  $("#bg-c2-row").hidden = bg.kind !== "gradient";
  $("#bg-angle-row").hidden = bg.kind !== "gradient";
  $(".bg-colors").hidden = bg.kind === "image";
  $("#bg-image-row").hidden = bg.kind !== "image";
}

function setupLook() {
  const dt = () => D.desktop, bg = () => D.desktop.background;
  const live = (sel, fn) => { $(sel).addEventListener("input", e => { fn(e.target.value); render(); }); $(sel).addEventListener("change", commit); };
  live("#accent", v => dt().accent = v);
  live("#bg-c1", v => { bg().color1 = v; if (bg().kind === "color") bg().color2 = v; });
  live("#bg-c2", v => bg().color2 = v);
  live("#bg-angle", v => bg().angle = Number(v));

  const pre = $("#bg-presets");
  for (const [kind, c1, c2, angle] of PRESETS) {
    pre.append(h("button", { style: { background: kind === "gradient" ? `linear-gradient(${angle}deg, ${c1}, ${c2})` : c1 }, title: "Use this wallpaper",
      onclick: () => { Object.assign(D.desktop.background, { kind, color1: c1, color2: c2, angle }); commit(); renderLook(); } }));
  }

  $("#bg-pick").onclick = () => $("#bg-file").click();
  $("#bg-file").onchange = async e => {
    const file = e.target.files[0];
    e.target.value = "";
    if (!file) return;
    try {
      bg().image = await shrinkImage(file);
      bg().kind = "image";
      commit(); renderLook();
      status(`Wallpaper set to ${file.name}.`);
    } catch (err) { status("Could not use that picture: " + err.message, true); }
  };
}

// Pictures straight off a phone can be 10+ MB; a desktop never needs more
// than a 1920x1080 JPEG, and that keeps the design file small.
function shrinkImage(file) {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file);
    const img = new Image();
    img.onload = () => {
      URL.revokeObjectURL(url);
      const k = Math.min(1, 1920 / img.width, 1080 / img.height);
      const c = document.createElement("canvas");
      c.width = Math.round(img.width * k); c.height = Math.round(img.height * k);
      c.getContext("2d").drawImage(img, 0, 0, c.width, c.height);
      resolve(c.toDataURL("image/jpeg", 0.9));
    };
    img.onerror = () => { URL.revokeObjectURL(url); reject(new Error("not a picture this browser can read")); };
    img.src = url;
  });
}

function renderTaskbarPanel() {
  const t = D.taskbar;
  seg("tb-pos", t.position, v => { t.position = v; commit(); renderTaskbarPanel(); });
  $("#tb-size").value = t.size;
  $("#tb-color").value = t.color;
  $("#tb-opacity").value = t.opacity;
  $("#tb-label").value = t.menuLabel;
  $("#tb-clock").checked = t.showClock;
  const ol = $("#pinned");
  ol.replaceChildren();
  t.pinned.forEach((id, i) => {
    const a = appById(id);
    if (!a) return;
    const li = h("li", { draggable: "true",
      ondragstart: e => e.dataTransfer.setData("text/pin", String(i)),
      ondragover: e => e.preventDefault(),
      ondrop: e => {
        e.preventDefault(); e.stopPropagation();
        const from = Number(e.dataTransfer.getData("text/pin"));
        if (Number.isNaN(from)) return;
        const [moved] = t.pinned.splice(from, 1);
        t.pinned.splice(i, 0, moved);
        commit(); renderTaskbarPanel();
      } },
      tile(a, 26), h("b", {}, a.name), h("button", { title: "Unpin", onclick: () => togglePin(id) }, "✕"));
    ol.append(li);
  });
  if (!t.pinned.length) ol.append(h("li", { class: "hint" }, "Nothing pinned yet."));
}

function setupTaskbar() {
  const t = () => D.taskbar;
  const live = (sel, fn, ev = "input") => { $(sel).addEventListener(ev, e => { fn(e.target); render(); }); $(sel).addEventListener("change", commit); };
  live("#tb-size", el => t().size = Number(el.value));
  live("#tb-color", el => t().color = el.value);
  live("#tb-opacity", el => t().opacity = Number(el.value));
  live("#tb-label", el => t().menuLabel = el.value);
  live("#tb-clock", el => t().showClock = el.checked, "change");
}

let hostnameTouched = false;
const slug = s => s.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "").slice(0, 63) || "my-os";

function renderSystem() {
  const s = D.system;
  $("#sys-osname").value = s.osName;
  $("#sys-hostname").value = s.hostname;
  $("#sys-fullname").value = s.fullName;
  $("#sys-username").value = s.username;
  $("#sys-autologin").checked = s.autologin;
  $("#sys-timezone").value = s.timezone;
  $("#sys-keyboard").value = s.keyboard;
  $("#pw-state").textContent = s.passwordHash
    ? "✓ A password is set. Type a new one to change it."
    : (s.autologin ? "No password: the OS logs in by itself. Set one if you want sudo to ask for it." : "⚠ Set a password, or turn on automatic login.");
  hostnameTouched = s.hostname !== slug(s.osName);
}

function setupSystem() {
  const s = () => D.system;
  const text = (sel, fn) => { $(sel).addEventListener("input", e => fn(e.target.value)); $(sel).addEventListener("change", () => { commit(); renderSystem(); }); };
  text("#sys-osname", v => { s().osName = v; if (!hostnameTouched) { s().hostname = slug(v); $("#sys-hostname").value = s().hostname; } });
  text("#sys-hostname", v => { s().hostname = v; hostnameTouched = true; });
  text("#sys-fullname", v => s().fullName = v);
  text("#sys-username", v => s().username = v.toLowerCase());
  text("#sys-timezone", v => s().timezone = v);
  $("#sys-autologin").onchange = e => { s().autologin = e.target.checked; commit(); renderSystem(); };
  $("#sys-keyboard").onchange = e => { s().keyboard = e.target.value; commit(); };
  const setPw = async () => {
    const pw = $("#sys-password").value;
    if (!pw) return;
    try {
      s().passwordHash = (await api("POST", "/api/password", { password: pw })).hash;
      $("#sys-password").value = "";
      commit(); renderSystem();
      status("Password set. Only a scrambled copy is saved in your design.");
    } catch (e) { status(e.message, true); }
  };
  $("#pw-set").onclick = setPw;
  $("#sys-password").addEventListener("keydown", e => { if (e.key === "Enter") setPw(); });

  for (const k of info.keyboards) $("#sys-keyboard").append(h("option", { value: k.id }, k.name));
  try {
    for (const tz of Intl.supportedValuesOf("timeZone")) $("#tz-list").append(h("option", { value: tz }));
  } catch { /* older browsers: free text still works */ }
}

// ---------------------------------------------------------------------------
// Files: new, open, save

function loadDesign(d) {
  const def = info.default;
  D = {
    ...structuredClone(def), ...d,
    system: { ...def.system, ...d.system },
    desktop: { ...def.desktop, ...d.desktop, background: { ...def.desktop.background, ...(d.desktop || {}).background } },
    taskbar: { ...def.taskbar, ...d.taskbar },
    apps: d.apps || [], items: d.items || [],
  };
  D.taskbar.pinned = D.taskbar.pinned || [];
  selected = null; history = []; future = [];
  lastSnap = snapshot();
  renderAll();
}

function saveFile() {
  const blob = new Blob([JSON.stringify(D, null, 2)], { type: "application/json" });
  const a = h("a", { href: URL.createObjectURL(blob), download: (D.system.hostname || "my-os") + ".osmaker" });
  document.body.append(a); a.click(); a.remove();
  setTimeout(() => URL.revokeObjectURL(a.href), 1000);
  status(`Saved ${a.download} to your downloads.`);
}

function setupFiles() {
  $("#btn-new").onclick = () => {
    if (!confirm("Start a new OS? Save first if you want to keep this one.")) return;
    loadDesign(structuredClone(info.default));
    scheduleAutosave();
    status("Started a new OS.");
  };
  $("#btn-open").onclick = () => $("#file-open").click();
  $("#file-open").onchange = async e => {
    const file = e.target.files[0];
    e.target.value = "";
    if (!file) return;
    try {
      const d = JSON.parse(await file.text());
      if (!d.system || !d.desktop) throw new Error("that is not an OS Maker design");
      loadDesign(d);
      scheduleAutosave();
      status(`Opened ${file.name}.`);
    } catch (err) { status("Could not open it: " + err.message, true); }
  };
  $("#btn-save").onclick = saveFile;
  $("#btn-undo").onclick = undo;
  $("#btn-redo").onclick = redo;
  $("#btn-exit-preview").onclick = () => { document.body.classList.remove("preview"); fit(); };
  $("#btn-preview").onclick = () => { selected = null; document.body.classList.add("preview"); render(); fit(); status(""); };
}

// ---------------------------------------------------------------------------
// Building the ISO

let pollTimer;

async function openBuild() {
  const body = $("#build-body");
  $("#modal").hidden = false;
  body.replaceChildren(h("p", {}, h("span", { class: "spinner" }), "Checking your design…"));
  try {
    await api("POST", "/api/validate", D);
  } catch (e) {
    body.replaceChildren(
      h("p", {}, "A few things need fixing first:"),
      h("ul", { class: "errors" }, e.message.split("\n").map(m => h("li", {}, m))),
      h("div", { class: "btns" }, h("button", { onclick: closeBuild }, "OK")));
    return;
  }
  const status = await api("GET", "/api/build");
  if (status.running) return showProgress();

  // One button: OS Maker picks the best way this computer can build.
  const best = info.methods[0];
  const out = h("input", { type: "text", value: info.outDir });
  const n = D.apps.length;
  const how = !best ? null
    : best.kind === "bootstrap" ? "OS Maker downloads Arch Linux (about 150 MB, only the first time), changes it to match your design, and turns it into an ISO."
    : best.kind === "archiso" ? "This computer runs Arch Linux, so OS Maker builds your ISO right here."
    : `OS Maker builds your ISO inside an Arch Linux container in ${best.kind}.`;
  body.replaceChildren(
    h("p", {}, `This makes `, h("b", {}, D.system.osName), ` as a bootable Arch Linux ISO with your desktop and ${n} extra app${n === 1 ? "" : "s"}.`),
    best
      ? h("div", { class: "note" }, how, best.password ? " You'll be asked for your password once." : " No password needed.")
      : h("div", { class: "note" }, "This computer can't build ISOs: OS Maker needs a 64-bit (x86-64) Linux computer. ",
          "You can still make the build folder and run its ", h("code", {}, "build.sh"), " on one."),
    h("div", { class: "note", style: { marginTop: "12px" } },
      "Building downloads every app (often 1–3 GB) and takes 10–40 minutes. You need about 10 GB of free space. ",
      "You can keep designing while it runs; the build uses the design as it is right now."),
    h("details", { style: { marginTop: "12px" } }, h("summary", {}, "Where to save it"), field("Folder", out)),
    h("div", { class: "btns" },
      h("button", { onclick: closeBuild }, "Cancel"),
      best ? null : h("button", { onclick: () => startBuild(out.value, "", true) }, "Make the build folder"),
      best ? h("button", { class: "primary", onclick: () => startBuild(out.value, best.kind, false) }, "💿 Build ISO") : null));
}

async function startBuild(outDir, method, folderOnly) {
  try {
    await api("POST", "/api/build", { design: D, outDir, method, folderOnly });
  } catch (e) {
    $("#build-body").prepend(h("ul", { class: "errors" }, e.message.split("\n").map(m => h("li", {}, m))));
    return;
  }
  showProgress();
}

function showProgress() {
  const body = $("#build-body");
  const head = h("p", {});
  const log = h("pre", {});
  const btns = h("div", { class: "btns" });
  body.replaceChildren(head, log, btns);
  const tick = async () => {
    let s;
    try { s = await api("GET", "/api/build"); } catch { return; }
    const atBottom = log.scrollTop + log.clientHeight >= log.scrollHeight - 20;
    log.textContent = s.log || "";
    if (atBottom) log.scrollTop = log.scrollHeight;
    btns.replaceChildren();
    if (s.running) {
      head.replaceChildren(h("span", { class: "spinner" }), "Building… this takes a while.");
      btns.append(h("button", { onclick: closeBuild }, "Hide"),
        h("button", { class: "danger", onclick: () => api("POST", "/api/build/cancel") }, "Stop the build"));
      return;
    }
    clearInterval(pollTimer);
    if (s.ok && s.iso) {
      head.replaceChildren(h("span", { class: "done" }, "✓ Your OS is ready: "), h("code", {}, s.iso));
      body.insertBefore(h("div", { class: "note" },
        "Try it in a virtual machine (VirtualBox, GNOME Boxes or QEMU), or write it to a USB stick with ",
        "balenaEtcher or Fedora Media Writer and boot from it."), btns);
    } else if (s.ok) {
      head.replaceChildren(h("span", { class: "done" }, "✓ Build folder ready: "), h("code", {}, s.dir));
      body.insertBefore(h("div", { class: "note" }, "On an Arch Linux computer, run ", h("code", {}, "sudo bash build.sh"),
        " in that folder. The README.txt inside explains the other ways."), btns);
    } else {
      head.replaceChildren(h("span", { class: "failed" }, "✗ " + (s.error || "The build stopped.")));
    }
    btns.append(h("button", { onclick: () => api("POST", "/api/open-folder", { path: s.dir }).catch(e => alert(e.message)) }, "Open folder"),
      h("button", { class: "primary", onclick: closeBuild }, "Done"));
  };
  clearInterval(pollTimer);
  tick();
  pollTimer = setInterval(tick, 1500);
}

function closeBuild() {
  clearInterval(pollTimer);
  $("#modal").hidden = true;
}

// ---------------------------------------------------------------------------
// Start-up

function switchTab(name) {
  for (const b of $$("#tabs button")) b.classList.toggle("on", b.dataset.tab === name);
  for (const p of $$(".panel")) p.classList.toggle("on", p.dataset.panel === name);
}

function setupKeys() {
  document.addEventListener("keydown", e => {
    const typing = /^(INPUT|TEXTAREA|SELECT)$/.test(document.activeElement.tagName);
    const mod = e.ctrlKey || e.metaKey;
    if (e.key === "Escape") {
      if (!$("#modal").hidden) return closeBuild();
      document.body.classList.remove("preview"); fit(); return;
    }
    if (mod && e.key.toLowerCase() === "s") { e.preventDefault(); return saveFile(); }
    if (typing) return;
    if (mod && e.key.toLowerCase() === "z") { e.preventDefault(); return e.shiftKey ? redo() : undo(); }
    if (mod && e.key.toLowerCase() === "y") { e.preventDefault(); return redo(); }
    const it = D.items.find(i => i.id === selected);
    if (!it) return;
    if (mod && e.key.toLowerCase() === "d") { e.preventDefault(); return duplicate(); }
    if (e.key === "Delete" || e.key === "Backspace") { e.preventDefault(); return removeSelected(); }
    const step = e.shiftKey ? 10 : 1;
    const moves = { ArrowLeft: [-step, 0], ArrowRight: [step, 0], ArrowUp: [0, -step], ArrowDown: [0, step] };
    if (moves[e.key]) {
      e.preventDefault();
      it.x += moves[e.key][0]; it.y += moves[e.key][1];
      fixBounds(it); commit();
    }
  });
}

async function main() {
  try {
    info = await api("GET", "/api/info");
    loadDesign(await api("GET", "/api/design"));
  } catch (e) {
    document.body.replaceChildren(h("p", { style: { padding: "40px" } },
      "OS Maker could not start: " + e.message + ". Close this tab and open OS Maker again."));
    return;
  }
  for (const b of $$("#tabs button")) b.onclick = () => switchTab(b.dataset.tab);
  setupPalette(); setupScreen(); setupLook(); setupTaskbar(); setupSystem(); setupFiles(); setupKeys();
  $("#app-filter").addEventListener("input", renderApps);
  $("#btn-build").onclick = openBuild;
  renderAll();
  fit();
  new ResizeObserver(fit).observe($("#stage"));
  // Keep clocks ticking and tell the server this page is still open.
  setInterval(renderScreen, 30000);
  setInterval(() => api("POST", "/api/heartbeat").catch(() => status("Lost contact with OS Maker. Is it still running?", true)), 10000);
  status("Drag things around the screen. Your work saves itself as you go; use Save to keep a copy.");
}

main();
