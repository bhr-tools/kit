// Every kit primitive on one page, for the browser tests.
import { h, clear } from '/kit/js/dom.js';
import { openModal, openMenu, openContextMenu, promptDialog, confirmDialog, toast } from '/kit/js/overlay.js';
import { initKeys } from '/kit/js/keys.js';
import { createUndo } from '/kit/js/undo.js';
import { sortable } from '/kit/js/drag.js';
import { createSelection } from '/kit/js/select.js';
import { tabBar } from '/kit/js/tabbar.js';
import { createLock } from '/kit/js/lock.js';

const $ = (id) => document.getElementById(id);
const state = {
  folder: 'Inbox',
  cursor: 0,
  items: ['Alpha', 'Bravo', 'Charlie', 'Delta'].map((title, i) => ({ key: `k${i}`, title, folder: 'Inbox' })),
};
const folders = ['Inbox', 'Archive'];
const undo = createUndo({ toast });
const selection = createSelection({ onChange: () => render() });

const visible = () => state.items.filter((i) => i.folder === state.folder);
const current = () => visible()[state.cursor];
const targets = () => (selection.size ? selection.keys : current() ? [current().key] : []);

function render() {
  state.cursor = Math.max(0, Math.min(state.cursor, visible().length - 1));
  clear($('folders'), folders.map((f) => h('div', {
    class: ['folder', f === state.folder && 'active'], dataset: { folder: f },
    onclick: () => go(f),
  }, f, h('span', { class: 'count' }, String(state.items.filter((i) => i.folder === f).length)))));
  clear($('list'), visible().map((item, i) => h('div', {
    class: ['row', i === state.cursor && 'cursor', selection.has(item.key) && 'selected'],
    dataset: { key: item.key },
    onclick: (e) => { if (!selection.click(e, item.key, visible().map((x) => x.key))) { state.cursor = i; render(); } },
    oncontextmenu: (e) => openContextMenu(e, [{ label: 'Delete', danger: true, run: () => remove([item.key]) }]),
  }, item.title)));
  $('title').textContent = state.folder;
  bar.setActive(state.folder);
  bar.setBadge('Archive', state.items.filter((i) => i.folder === 'Archive').length);
}

function go(folder) {
  state.folder = folder;
  state.cursor = 0;
  selection.clear();
  render();
}

function move(keys, folder) {
  const before = keys.map((k) => [k, state.items.find((i) => i.key === k).folder]);
  for (const k of keys) state.items.find((i) => i.key === k).folder = folder;
  selection.clear();
  render();
  undo.push({ label: `Moved to ${folder}`, undo: () => { for (const [k, f] of before) state.items.find((i) => i.key === k).folder = f; render(); } });
}

function remove(keys) {
  const before = [...state.items];
  state.items = state.items.filter((i) => !keys.includes(i.key));
  selection.clear();
  render();
  undo.push({ label: keys.length > 1 ? `${keys.length} items deleted` : 'Item deleted', undo: () => { state.items = before; render(); } });
}

function reorder(key, index) {
  const list = visible().filter((i) => i.key !== key);
  const item = state.items.find((i) => i.key === key);
  list.splice(index, 0, item);
  state.items = [...list, ...state.items.filter((i) => i.folder !== state.folder)];
  state.cursor = index;
  render();
}

async function edit() {
  const item = current();
  if (!item) return;
  const title = await promptDialog('Rename', { value: item.title });
  if (title) { item.title = title; render(); }
}

async function add() {
  const title = await promptDialog('New item', { placeholder: 'Title' });
  if (!title) return;
  state.items.push({ key: `k${Date.now()}`, title, folder: state.folder });
  render();
}

function details() {
  const item = current();
  if (!item) return;
  // A page that only shows something: "Back" on phones.
  openModal({ title: item.title, body: h('p', { class: 'dialog-message' }, `In ${item.folder}.`) });
}

const table = () => [
  { keys: ['j', 'ArrowDown'], label: 'Next', group: 'Move', run: () => { state.cursor++; render(); } },
  { keys: ['k', 'ArrowUp'], label: 'Previous', group: 'Move', run: () => { state.cursor = Math.max(0, state.cursor - 1); render(); } },
  { keys: ['gg'], label: 'First', group: 'Move', run: () => { state.cursor = 0; render(); } },
  { keys: ['G'], label: 'Last', group: 'Move', run: () => { state.cursor = visible().length - 1; render(); } },
  { keys: ['J'], label: 'Move down', group: 'Move', run: () => current() && reorder(current().key, Math.min(state.cursor + 1, visible().length - 1)) },
  { keys: ['K'], label: 'Move up', group: 'Move', run: () => current() && reorder(current().key, Math.max(state.cursor - 1, 0)) },
  { digit: true, label: 'Folder', group: 'Move', run: (n) => folders[n - 1] && go(folders[n - 1]) },
  { keys: [' '], label: 'Select', group: 'Edit', run: () => current() && selection.toggle(current().key) },
  { keys: ['d'], label: 'Delete', group: 'Edit', run: () => targets().length && remove(targets()) },
  { keys: ['u'], label: 'Undo', group: 'Edit', run: () => undo.run() },
  { keys: ['e'], label: 'Rename', group: 'Edit', run: edit },
  { keys: ['Enter'], label: 'Details', group: 'Edit', run: details },
  { keys: ['a'], label: 'Add', group: 'Edit', run: add },
  { keys: ['m'], label: 'Move to folder', group: 'Edit', run: () => targets().length && openMenu($('list').querySelector('.cursor') || $('list'), folders.map((f) => ({ label: f, run: () => move(targets(), f) })), { title: 'Move to' }) },
  { keys: ['x'], label: 'Clear all', group: 'Edit', run: async () => { if (await confirmDialog('Clear all', 'Delete every item?', { okText: 'Delete' })) remove(state.items.map((i) => i.key)); } },
  { keys: ['Escape'], label: 'Clear selection', group: 'Edit', run: () => selection.clear() },
];

const bar = tabBar($('tabbar'), folders.map((f) => ({ id: f, label: f, icon: 'check' })), { onSelect: go });
initKeys({ table, helpTitle: 'Kit demo' });
sortable($('list'), {
  label: (row) => row.textContent,
  targets: () => [...document.querySelectorAll('.folder')].filter((el) => el.dataset.folder !== state.folder).map((el) => ({ el, key: el.dataset.folder })),
  onMove: ({ key, index }) => reorder(key, index),
  onDropOn: ({ key, target }) => move(selection.has(key) ? selection.keys : [key], target.key),
});

const lock = createLock({ appName: 'Kit demo', onUnlocked: async () => render() });
lock.start().then(() => { if (!lock.status.no_auth) $('lock').append(lock.button()); });
render();
window.kitDemo = { state };
