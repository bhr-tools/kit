import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createUndo } from '../../web/js/undo.js';

const recorder = () => {
  const toasts = [];
  return { toasts, toast: (msg, opts) => toasts.push({ msg, ...opts }) };
};

test('undoes the latest action first', async () => {
  const done = [];
  const undo = createUndo({ quiet: true });
  undo.push({ label: 'A', undo: () => done.push('A') }, { quiet: true });
  undo.push({ label: 'B', undo: () => done.push('B') }, { quiet: true });
  await undo.run();
  await undo.run();
  assert.deepEqual(done, ['B', 'A']);
  assert.equal(await undo.run(), false);
});

test('a cancelled entry is not undone', async () => {
  const done = [];
  const undo = createUndo();
  undo.push({ label: 'A', undo: () => done.push('A') }, { quiet: true });
  undo.push({ label: 'B', undo: () => done.push('B') }, { quiet: true }).cancel();
  await undo.run();
  assert.deepEqual(done, ['A']);
});

test('a failed undo stays on the stack and shows the error', async () => {
  const r = recorder();
  const undo = createUndo({ toast: r.toast });
  undo.push({ label: 'Moved', undo: async () => { throw new Error('Offline'); } });
  assert.equal(r.toasts[0].action.label, 'Undo');
  assert.equal(await undo.run(), false);
  assert.equal(undo.size, 1);
  assert.deepEqual(r.toasts.at(-1), { msg: 'Offline', type: 'error' });
});

test('the stack is bounded', () => {
  const undo = createUndo({ limit: 3 });
  for (let i = 0; i < 5; i++) undo.push({ label: `${i}`, undo() {} }, { quiet: true });
  assert.equal(undo.size, 3);
});

test('an undo that reports false stays, a custom message is shown', async () => {
  const r = recorder();
  const undo = createUndo({ toast: r.toast });
  let ok = false;
  undo.push({ label: 'Done', undone: 'Restored "Milk"', undo: () => ok }, { quiet: true });
  assert.equal(await undo.run(), false);
  assert.equal(undo.size, 1);
  assert.equal(r.toasts.length, 0);
  ok = true;
  assert.equal(await undo.run(), true);
  assert.equal(r.toasts.at(-1).msg, 'Restored "Milk"');
});
