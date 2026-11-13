import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createJobTimer } from './eta.js';

test('estimates progress using the full time between completed files', () => {
  let clock = 1000;
  let view;
  let tick;
  const originalPerformance = globalThis.performance;
  const originalWindow = globalThis.window;
  Object.defineProperty(globalThis, 'performance', { configurable: true, value: { now: () => clock } });
  globalThis.window = { setInterval: (callback) => { tick = callback; return 1; }, clearInterval() {} };
  try {
    const timer = createJobTimer((value) => { view = value; });
    timer.start();
    for (let i = 0; i < 10; i++) { clock += 1000; tick(); timer.report(0); }
    timer.report(0.4);
    for (let i = 0; i < 10; i++) { clock += 1000; tick(); timer.report(0.4); }
    timer.report(0.8);
    assert.equal(view.left, '~42 min');
  } finally {
    Object.defineProperty(globalThis, 'performance', { configurable: true, value: originalPerformance });
    globalThis.window = originalWindow;
  }
});
