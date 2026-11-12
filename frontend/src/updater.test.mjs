import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

class Element {
  constructor() { this.hidden = false; this.dataset = {}; this.listeners = {}; this.attributes = {}; this.textContent = ''; }
  setAttribute(name, value) { this.attributes[name] = value; }
  addEventListener(name, callback) { this.listeners[name] = callback; }
  querySelector(selector) { return this.children[selector]; }
  showModal() { this.open = true; }
  close() { this.open = false; }
}

async function setup(result) {
  const indicator = new Element(); indicator.hidden = true;
  const button = new Element(); const announcement = new Element();
  const dialog = new Element();
  dialog.children = Object.fromEntries(['#update-status', '#update-action', '#update-close', '#update-title'].map((key) => [key, new Element()]));
  globalThis.document = { querySelector: (key) => ({ '.update-indicator': indicator, '#check-update': button, '#update-announcement': announcement })[key], createElement: () => dialog, body: { append() {} } };
  globalThis.CheckForUpdate = async () => result;
  globalThis.DownloadUpdate = async () => {};
  globalThis.RestartForUpdate = async () => {};
  globalThis.EventsOn = () => {};
  const originalInterval = globalThis.setInterval;
  const originalTimeout = globalThis.setTimeout;
  globalThis.setInterval = () => 0;
  globalThis.setTimeout = () => 0;
  try {
    const source = (await readFile(new URL('./updater.js', import.meta.url), 'utf8')).replace(/^import .*;\s*$/gm, '');
    const module = await import(`data:text/javascript,${encodeURIComponent(source)}#${Math.random()}`);
    module.setupUpdater();
    await Promise.resolve();
    return { indicator, button, announcement, dialog };
  } finally { globalThis.setInterval = originalInterval; globalThis.setTimeout = originalTimeout; }
}

test('shows the update indicator only when a newer version is available', async () => {
  let ui = await setup({ current: '0.2.5', available: '' });
  assert.equal(ui.indicator.hidden, true);
  ui = await setup({ current: '0.2.5', available: '0.2.6' });
  assert.equal(ui.indicator.hidden, false);
  assert.match(ui.announcement.textContent, /0\.2\.6/);
  ui.button.listeners.click();
  assert.equal(ui.dialog.open, true);
  assert.match(ui.dialog.children['#update-status'].textContent, /0\.2\.6/);
  await ui.dialog.children['#update-action'].listeners.click();
  assert.equal(ui.dialog.children['#update-action'].textContent, 'Restart and update');
});
