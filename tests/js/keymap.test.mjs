import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createMatcher, displayKey, helpGroups, keyName } from '../../web/js/keymap.js';

const table = [
  { keys: ['j', 'ArrowDown'], label: 'Next', group: 'Move', run() {} },
  { keys: ['gg'], label: 'First', group: 'Move', run() {} },
  { keys: ['G'], label: 'Last', group: 'Move', run() {} },
  { keys: ['C-d'], label: 'Half page down', group: 'Move', run() {} },
  { digit: true, label: 'Account', group: 'Accounts', run() {} },
  { keys: ['x'], label: 'Hidden', hidden: true, run() {} },
];
const labelOf = (hit) => hit?.binding?.label ?? null;

test('single keys and aliases', () => {
  const m = createMatcher(() => table);
  assert.equal(labelOf(m.feed('j')), 'Next');
  assert.equal(labelOf(m.feed('ArrowDown')), 'Next');
  assert.equal(m.feed('z'), null);
});

test('a sequence waits for its second key', () => {
  const m = createMatcher(() => table);
  assert.deepEqual(m.feed('g'), { waiting: true });
  assert.equal(labelOf(m.feed('g')), 'First');
});

test('a wrong second key is tried on its own', () => {
  const m = createMatcher(() => table);
  m.feed('g');
  assert.equal(labelOf(m.feed('j')), 'Next');
  assert.equal(m.pending, '');
});

test('a key alone that also starts a sequence fires on timeout', async () => {
  let fired = null;
  const t = [{ keys: ['g'], label: 'Go to date', run() {} }, { keys: ['gg'], label: 'First', run() {} }];
  const m = createMatcher(() => t, { onTimeout: (hit) => { fired = hit.binding.label; } });
  assert.deepEqual(m.feed('g'), { waiting: true });
  await new Promise((r) => setTimeout(r, 1100));
  assert.equal(fired, 'Go to date');
});

test('digits on QWERTY and AZERTY', () => {
  const m = createMatcher(() => table);
  assert.deepEqual([m.feed('2'), m.feed('é'), m.feed('"')].map((h) => h.digit), [2, 2, 3]);
});

test('Ctrl bindings and key names', () => {
  assert.equal(keyName({ key: 'd', ctrlKey: true }), 'C-d');
  assert.equal(keyName({ key: 'd', ctrlKey: false }), 'd');
  assert.equal(labelOf(createMatcher(() => table).feed('C-d')), 'Half page down');
  assert.equal(displayKey('C-d'), 'Ctrl+d');
  assert.equal(displayKey(' '), 'Space');
});

test('help groups keep table order and skip hidden bindings', () => {
  const groups = helpGroups(table);
  assert.deepEqual(groups.map((g) => g.group), ['Move', 'Accounts']);
  assert.deepEqual(groups[0].rows[0], { label: 'Next', keys: ['j', '↓'] });
  assert.deepEqual(groups[1].rows[0].keys, ['1–9']);
  assert.ok(!groups.flatMap((g) => g.rows).some((r) => r.label === 'Hidden'));
});

test('bindings without run are only documented', () => {
  const t = [{ keys: ['C-Enter'], label: 'Save', group: 'Details' }, { keys: ['j'], label: 'Next', run() {} }];
  assert.equal(createMatcher(() => t).feed('C-Enter'), null);
  assert.deepEqual(helpGroups(t)[0], { group: 'Details', rows: [{ label: 'Save', keys: ['Ctrl+Enter'] }] });
});
