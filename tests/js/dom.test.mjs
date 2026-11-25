// dom.js helpers that run without a browser.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { debounce, isTextInput, isMobile } from '../../web/js/dom.js';

test('debounce runs once, with the last arguments', async () => {
  const calls = [];
  const f = debounce((x) => calls.push(x), 10);
  f(1); f(2); f(3);
  await new Promise((r) => setTimeout(r, 30));
  assert.deepEqual(calls, [3]);
});

test('isTextInput recognises fields', () => {
  assert.equal(isTextInput({ tagName: 'INPUT' }), true);
  assert.equal(isTextInput({ tagName: 'DIV', isContentEditable: true }), true);
  assert.equal(isTextInput({ tagName: 'BUTTON' }), false);
  assert.equal(isTextInput(null), false);
});

test('isMobile is false outside a browser', () => {
  assert.equal(isMobile(), false);
});
