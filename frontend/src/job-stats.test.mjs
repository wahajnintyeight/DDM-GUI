import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createJobStats } from './job-stats.js';

test('shows job telemetry and accumulates completed chunks across a resume', () => {
  const elements = new Map();
  const root = { querySelector(selector) {
    if (!elements.has(selector)) elements.set(selector, { textContent: '', setAttribute() {} });
    return elements.get(selector);
  } };
  const previousWindow = globalThis.window;
  globalThis.window = { setInterval: () => 1, clearInterval() {}, setTimeout: () => 2, clearTimeout() {} };
  try {
    const stats = createJobStats(root, { formatBytes: (n) => `${n} B` });
    stats.start();
    stats.report(1, { downloadedBytes: 1000, downloadRate: 500, validationRate: 250 });
    assert.equal(elements.get('#job-speed').textContent, '500 B/s');
    assert.equal(elements.get('#job-read').textContent, '250 B/s');
    stats.pause();
    stats.resume();
    stats.report(1, { downloadedBytes: 0, downloadRate: 0, validationRate: 0 });
    stats.report(2, { downloadedBytes: 500, downloadRate: 500, validationRate: 0 });
    assert.equal(elements.get('#job-received').textContent, '1500 B in completed chunks');
    stats.stop();
  } finally { globalThis.window = previousWindow; }
});
