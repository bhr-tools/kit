import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createSelection } from '../../web/js/select.js';

const order = ['a', 'b', 'c', 'd', 'e'];
const click = (mods = {}) => ({ shiftKey: false, ctrlKey: false, metaKey: false, ...mods });

test('a plain click with nothing selected is left to the app', () => {
  const s = createSelection();
  assert.equal(s.click(click(), 'b', order), false);
  assert.equal(s.size, 0);
});

test('Ctrl+click toggles, then plain clicks keep selecting', () => {
  const s = createSelection();
  assert.equal(s.click(click({ ctrlKey: true }), 'b', order), true);
  assert.equal(s.click(click(), 'd', order), true);
  assert.deepEqual(s.keys, ['b', 'd']);
  s.click(click(), 'b', order);
  assert.deepEqual(s.keys, ['d']);
});

test('Shift+click selects the range from the anchor, both ways', () => {
  const s = createSelection();
  s.toggle('d');
  s.click(click({ shiftKey: true }), 'b', order);
  assert.deepEqual(s.keys.sort(), ['b', 'c', 'd']);
});

test('retain drops rows gone from the list; clear reports whether anything was selected', () => {
  const changes = [];
  const s = createSelection({ onChange: (k) => changes.push(k) });
  s.toggle('a');
  s.toggle('z');
  s.retain(order);
  assert.deepEqual(s.keys, ['a']);
  assert.equal(s.clear(), true);
  assert.equal(s.clear(), false);
  assert.deepEqual(changes.at(-1), []);
});
