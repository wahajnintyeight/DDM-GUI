import '@awesome.me/webawesome/dist/styles/themes/default.css';
import '@awesome.me/webawesome/dist/components/details/details.js';
import './style.css';
import { createGameSelector } from './game-selector';
import { CancelDownload, PauseDownload, ResumeDownload, SelectDownloader, SelectGameFolder, SelectKeys, SelectManifest, StartDownload } from '../wailsjs/go/main/App';
import { EventsOn, Quit, WindowIsMaximised, WindowMinimise, WindowToggleMaximise } from '../wailsjs/runtime/runtime';

const themeKey = 'depotdrop-theme';
const savedTheme = localStorage.getItem(themeKey);
const initialTheme = savedTheme === 'dark' || savedTheme === 'light' ? savedTheme : (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
document.documentElement.dataset.theme = initialTheme;
document.documentElement.classList.add('wa-theme-default', 'wa-palette-default', 'wa-brand-blue');
document.documentElement.classList.toggle('wa-dark', initialTheme === 'dark');
document.querySelector('#app').innerHTML = `
<header class="app-header"><div class="brand"><span class="brand__mark">D</span><span class="brand__name">DepotDrop</span></div><div class="header-actions"><button class="theme-toggle" id="theme-toggle" type="button" aria-pressed="${initialTheme === 'dark'}"><svg class="theme-toggle__sun" aria-hidden="true" viewBox="0 0 24 24"><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"/></svg><svg class="theme-toggle__moon" aria-hidden="true" viewBox="0 0 24 24"><path d="M20.3 15.2A8.5 8.5 0 0 1 8.8 3.7 8.5 8.5 0 1 0 20.3 15.2Z"/></svg><span id="theme-label">${initialTheme === 'dark' ? 'Light' : 'Dark'}</span></button><div class="header-state" aria-live="polite"><span class="header-state__light"></span><span id="header-state-label">Ready</span></div><div class="window-controls" role="group" aria-label="Window controls"><button class="window-control" id="window-minimize" type="button" aria-label="Minimize DepotDrop" title="Minimize"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="M5 12h14"/></svg></button><button class="window-control" id="window-maximize" type="button" aria-label="Maximize DepotDrop" title="Maximize"><svg class="maximize-icon" aria-hidden="true" viewBox="0 0 24 24"><path d="M6.5 6.5h11v11h-11z"/></svg><svg class="restore-icon" aria-hidden="true" viewBox="0 0 24 24"><path d="M8 8h10v10H8zM6 16V6h10"/></svg></button><button class="window-control window-control--close" id="window-close" type="button" aria-label="Close DepotDrop" title="Close"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="m7 7 10 10M17 7 7 17"/></svg></button></div></div></header>
<main class="workspace">
  <section class="setup" aria-labelledby="setup-title"><div class="setup__inner">
    <div class="setup__intro"><h1 id="setup-title">Set up your download</h1><p>Three quick steps. DepotDrop handles the technical part for you.</p></div>

    <section class="step" id="step-game" data-done="false" aria-labelledby="step-game-title">
      <div class="step__badge" aria-hidden="true"><span>1</span><svg viewBox="0 0 24 24"><path d="m5 12.5 4.5 4.5L19 7.5"/></svg></div>
      <div class="step__body">
        <h2 id="step-game-title">Choose your game</h2>
        <p class="step__hint">Search by name. If you already know the Steam App ID, switch to “Enter App ID”.</p>
        <div id="game-selector"></div>
        <div class="field app-id-field"><label for="app-id">Steam App ID</label><input id="app-id" type="text" inputmode="numeric" aria-required="true" aria-describedby="app-id-help" placeholder="e.g. 3468650"><p id="app-id-help">The number at the end of the game’s Steam store link.</p></div>
      </div>
    </section>

    <section class="step" id="step-files" data-done="false" aria-labelledby="step-files-title">
      <div class="step__badge" aria-hidden="true"><span>2</span><svg viewBox="0 0 24 24"><path d="m5 12.5 4.5 4.5L19 7.5"/></svg></div>
      <div class="step__body">
        <h2 id="step-files-title">Add your files</h2>
        <div class="file-row" id="manifest-field" data-state="default">
          <label for="manifest-path">Manifest file</label>
          <div class="file-row__control"><input id="manifest-path" type="text" readonly aria-required="true" aria-describedby="manifest-help" placeholder="No file chosen"><button class="button button--secondary" id="choose-manifest" type="button"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="M4 5.5h6l2 2h8v11H4zM4 9h16"/></svg><span class="button__text">Browse</span></button></div>
          <p id="manifest-help">A .manifest file, named like 3468651_8958793990673293420.manifest</p>
        </div>
        <div class="file-row" id="keys-field" data-state="default">
          <label for="keys-path">Keys file</label>
          <div class="file-row__control"><input id="keys-path" type="text" readonly aria-required="true" aria-describedby="keys-help" placeholder="No file chosen"><button class="button button--secondary" id="choose-keys" type="button"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="M4 5.5h6l2 2h8v11H4zM4 9h16"/></svg><span class="button__text">Browse</span></button></div>
          <p id="keys-help">keys.txt: a text file with one depotID;hexKey entry per line.</p>
        </div>
        <div class="file-row" id="downloader-field" data-state="default">
          <label for="downloader-path">Downloader program</label>
          <div class="file-row__control"><input id="downloader-path" type="text" readonly aria-required="true" aria-describedby="downloader-help" placeholder="No file chosen"><button class="button button--secondary" id="choose-downloader" type="button"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="M4 5.5h6l2 2h8v11H4zM4 9h16"/></svg><span class="button__text">Browse</span></button></div>
          <p id="downloader-help">The DepotDownloaderMod.exe program that does the downloading.</p>
        </div>
        <details class="advanced"><summary>Advanced: change the Depot ID</summary>
          <div class="field"><label for="depot-id">Depot ID</label><input id="depot-id" type="text" inputmode="numeric" aria-describedby="depot-id-help" placeholder="Filled in from the manifest"><p id="depot-id-help">Leave this alone unless you know you need a different depot.</p></div>
        </details>
      </div>
    </section>

    <section class="step" id="step-folder" data-done="false" aria-labelledby="step-folder-title">
      <div class="step__badge" aria-hidden="true"><span>3</span><svg viewBox="0 0 24 24"><path d="m5 12.5 4.5 4.5L19 7.5"/></svg></div>
      <div class="step__body">
        <h2 id="step-folder-title">Pick where to save</h2>
        <p class="step__hint">Existing files are reused; only missing ones are downloaded.</p>
        <div class="file-row" id="folder-field" data-state="default">
          <label for="game-folder">Game folder</label>
          <div class="file-row__control"><input id="game-folder" type="text" readonly aria-required="true" placeholder="No folder chosen"><button class="button button--secondary" id="choose-folder" type="button"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="M3.5 6h6l2 2h9v10.5h-17z"/></svg><span class="button__text">Browse</span></button></div>
        </div>
        <div class="validate-option"><label for="validate-download"><input id="validate-download" type="checkbox" checked aria-describedby="validate-help">Check existing files (recommended)</label><p id="validate-help">Finds damaged or incomplete files already in the folder. Slower, but safest.</p></div>
      </div>
    </section>
  </div></section>

  <aside class="monitor" id="monitor" data-phase="idle" aria-label="Download status">
    <div class="ready-view" aria-live="polite">
      <p class="monitor__label">Your download</p>
      <h2 id="ready-title">Let’s get started</h2>
      <p id="ready-sub">Finish the steps on the left to unlock the download button.</p>
      <ul class="checklist">
        <li data-item="game" data-done="false"><span class="check"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="m5 12.5 4.5 4.5L19 7.5"/></svg></span>Game chosen</li>
        <li data-item="manifest" data-done="false"><span class="check"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="m5 12.5 4.5 4.5L19 7.5"/></svg></span>Manifest file added</li>
        <li data-item="keys" data-done="false"><span class="check"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="m5 12.5 4.5 4.5L19 7.5"/></svg></span>Keys file added</li>
        <li data-item="downloader" data-done="false"><span class="check"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="m5 12.5 4.5 4.5L19 7.5"/></svg></span>Downloader program added</li>
        <li data-item="folder" data-done="false"><span class="check"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="m5 12.5 4.5 4.5L19 7.5"/></svg></span>Save folder chosen</li>
      </ul>
    </div>
    <div class="progress-view">
      <p class="monitor__label" id="progress-title">Download progress</p>
      <p class="monitor__status" id="progress-status">Getting started</p>
      <output class="monitor__percent" id="progress-percent" aria-live="polite">0.00%</output>
      <div class="progress-track" role="progressbar" aria-label="Download progress" aria-valuemin="0" aria-valuemax="100" aria-valuenow="0"><span id="progress-fill"></span></div>
      <p class="current-line" id="current-line">Waiting for the downloader to report in…</p>
      <div class="download-report" id="download-report" hidden aria-live="polite"><div class="report-stat"><span>Downloaded</span><strong id="report-downloaded">0 B</strong></div><div class="report-stat"><span>Uncompressed</span><strong id="report-uncompressed">0 B</strong></div><div class="report-stat"><span>Time taken</span><strong id="report-elapsed">0s</strong></div></div>
    </div>
    <div class="actions">
      <button class="button button--primary button--block" id="start-download" type="button" disabled><span class="button__spinner"></span><span class="button__label">Start download</span></button>
      <div class="actions__secondary"><button class="button button--secondary" id="pause-download" type="button" hidden>Pause</button><button class="button button--quiet" id="cancel-download" type="button" hidden>Cancel</button></div>
      <p id="form-message" class="form-message" aria-live="polite">Complete the steps to unlock this button.</p>
    </div>
    <div class="monitor__footer"><wa-details class="console-disclosure" id="console-panel" appearance="plain"><span class="console-summary" slot="summary"><span class="console-toggle__label">Show detailed log</span><span id="line-count">0 lines</span></span><pre id="output-log" tabindex="0">Waiting to start.</pre></wa-details></div>
  </aside>
</main>`;

const $ = (selector) => document.querySelector(selector);
const el = { manifestField: $('#manifest-field'), manifestPath: $('#manifest-path'), manifestHelp: $('#manifest-help'), downloaderField: $('#downloader-field'), downloaderPath: $('#downloader-path'), downloaderHelp: $('#downloader-help'), keysField: $('#keys-field'), keysPath: $('#keys-path'), keysHelp: $('#keys-help'), folderField: $('#folder-field'), gameFolder: $('#game-folder'), appID: $('#app-id'), depotID: $('#depot-id'), validate: $('#validate-download'), monitor: $('#monitor'), readyTitle: $('#ready-title'), readySub: $('#ready-sub'), stepGame: $('#step-game'), stepFiles: $('#step-files'), stepFolder: $('#step-folder'), checklist: [...document.querySelectorAll('.checklist li')], chooseManifest: $('#choose-manifest'), chooseDownloader: $('#choose-downloader'), chooseKeys: $('#choose-keys'), chooseFolder: $('#choose-folder'), start: $('#start-download'), cancel: $('#cancel-download'), pause: $('#pause-download'), message: $('#form-message'), header: $('.header-state'), headerLabel: $('#header-state-label'), status: $('#progress-status'), percent: $('#progress-percent'), track: $('.progress-track'), fill: $('#progress-fill'), currentLine: $('#current-line'), report: $('#download-report'), reportDownloaded: $('#report-downloaded'), reportUncompressed: $('#report-uncompressed'), reportElapsed: $('#report-elapsed'), consolePanel: $('#console-panel'), consoleLabel: $('.console-toggle__label'), lineCount: $('#line-count'), log: $('#output-log'), themeToggle: $('#theme-toggle'), themeLabel: $('#theme-label'), windowControls: $('.window-controls'), minimize: $('#window-minimize'), maximize: $('#window-maximize'), close: $('#window-close') };
const state = { manifest: null, gameFolder: '', downloaderPath: '', keysPath: '', appID: '', depotID: '', running: false, paused: false, lines: [] };
const errorText = (error) => typeof error === 'string' ? error : error?.message || String(error);

const gameSelector = createGameSelector($('#game-selector'), el.appID);

function updateReadyState() {
  const done = {
    game: /^\d+$/.test(state.appID),
    manifest: Boolean(state.manifest),
    keys: Boolean(state.keysPath),
    downloader: Boolean(state.downloaderPath),
    folder: Boolean(state.gameFolder),
  };
  const remaining = Object.values(done).filter((value) => !value).length;
  const complete = remaining === 0;
  const ready = complete && !state.running;
  el.stepGame.dataset.done = String(done.game);
  el.stepFiles.dataset.done = String(done.manifest && done.keys && done.downloader);
  el.stepFolder.dataset.done = String(done.folder);
  el.checklist.forEach((item) => { item.dataset.done = String(done[item.dataset.item]); });
  el.readyTitle.textContent = complete ? 'You’re ready to go' : remaining === 1 ? 'Just 1 thing left' : `${remaining} things left`;
  el.readySub.textContent = complete ? 'Everything is set. Press Start download whenever you like.' : 'Finish the steps on the left to unlock the download button.';
  el.start.disabled = !ready;
  el.start.setAttribute('aria-disabled', String(!ready));
  if (!state.running) el.message.textContent = complete ? '' : 'Complete the steps to unlock this button.';
}

const baseName = (path) => path.split(/[\\/]/).pop();
function showChosen(input, button, value, title = value) {
  input.value = value;
  input.title = title;
  button.querySelector('.button__text').textContent = 'Change';
}

function setRunning(running) {
  state.running = running;
  gameSelector.setDisabled(running);
  if (!running) state.paused = false;
  el.pause.hidden = !running;
  el.pause.disabled = false;
  el.pause.textContent = state.paused ? 'Resume' : 'Pause';
  el.chooseManifest.disabled = running;
  el.chooseDownloader.disabled = running;
  el.chooseKeys.disabled = running;
  el.chooseFolder.disabled = running;
  el.validate.disabled = running;
  el.appID.disabled = running;
  el.depotID.disabled = running;
  el.cancel.hidden = !running;
  el.start.dataset.state = running ? 'loading' : 'default';
  $('.button__label').textContent = running ? 'Downloading' : 'Start download';
  el.header.dataset.state = running ? 'running' : 'ready';
  el.headerLabel.textContent = running ? 'Running' : 'Ready';
  updateReadyState();
}

function updateProgress(value) {
  const progress = Math.min(100, Math.max(0, Number(value) || 0));
  el.percent.value = progress;
  el.percent.textContent = `${progress.toFixed(2)}%`;
  el.fill.style.setProperty('--progress', progress / 100);
  el.track.setAttribute('aria-valuenow', progress.toFixed(2));
}

const formatBytes = (value) => {
  const bytes = Number(value);
  if (!Number.isFinite(bytes) || bytes < 0) return '—';
  if (bytes < 1024) return `${bytes} B`;
  const units = ['KB', 'MB', 'GB', 'TB'];
  let size = bytes;
  let unit = 'B';
  for (const next of units) { size /= 1024; unit = next; if (size < 1024 || next === units.at(-1)) break; }
  return `${size >= 100 ? size.toFixed(0) : size >= 10 ? size.toFixed(1) : size.toFixed(2)} ${unit}`;
};
const formatDuration = (value) => {
  const total = Math.max(0, Number(value) || 0);
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const seconds = Math.floor(total % 60);
  return hours ? `${hours}h ${minutes}m ${seconds}s` : minutes ? `${minutes}m ${seconds}s` : `${total < 10 ? total.toFixed(1) : seconds}s`;
};
function updateReport(update, visible = false) {
  el.reportDownloaded.textContent = formatBytes(update.downloadedBytes);
  el.reportUncompressed.textContent = formatBytes(update.uncompressedBytes);
  el.reportElapsed.textContent = formatDuration(update.elapsedSeconds);
  if (visible) el.report.hidden = false;
}

function appendLine(line) {
  if (!line) return;
  state.lines.push(line);
  state.lines = state.lines.slice(-160);
  el.log.textContent = state.lines.join('\n');
  el.log.scrollTop = el.log.scrollHeight;
  el.lineCount.textContent = `${state.lines.length} ${state.lines.length === 1 ? 'line' : 'lines'}`;
  el.currentLine.textContent = line;
}

el.chooseManifest.addEventListener('click', async () => {
  el.manifestField.dataset.state = 'loading';
  try {
    const manifest = await SelectManifest();
    if (!manifest?.path) { el.manifestField.dataset.state = state.manifest ? 'success' : 'default'; return; }
    state.manifest = manifest;
    showChosen(el.manifestPath, el.chooseManifest, baseName(manifest.path), manifest.path);
    state.depotID = manifest.depotId;
    el.depotID.value = manifest.depotId;
    el.manifestHelp.textContent = `Found manifest ${manifest.manifestId} for depot ${manifest.depotId}.`;
    el.manifestPath.removeAttribute('aria-invalid');
    el.manifestField.dataset.state = 'success';
  } catch (error) {
    el.manifestHelp.textContent = `${errorText(error)} Choose a file named DEPOTID_MANIFESTID.manifest.`;
    el.manifestPath.setAttribute('aria-invalid', 'true');
    el.manifestField.dataset.state = 'error';
  }
  updateReadyState();
});

el.chooseDownloader.addEventListener('click', async () => {
  el.downloaderField.dataset.state = 'loading';
  try {
    const path = await SelectDownloader();
    if (!path) { el.downloaderField.dataset.state = state.downloaderPath ? 'success' : 'default'; return; }
    state.downloaderPath = path;
    showChosen(el.downloaderPath, el.chooseDownloader, baseName(path), path);
    el.downloaderHelp.textContent = 'The DepotDownloaderMod.exe program that does the downloading.';
    el.downloaderPath.removeAttribute('aria-invalid');
    el.downloaderField.dataset.state = 'success';
  } catch (error) {
    el.downloaderHelp.textContent = `${errorText(error)} Choose DepotDownloaderMod.exe.`;
    el.downloaderPath.setAttribute('aria-invalid', 'true');
    el.downloaderField.dataset.state = 'error';
  }
  updateReadyState();
});

el.chooseKeys.addEventListener('click', async () => {
  el.keysField.dataset.state = 'loading';
  try {
    const path = await SelectKeys();
    if (!path) { el.keysField.dataset.state = state.keysPath ? 'success' : 'default'; return; }
    state.keysPath = path;
    showChosen(el.keysPath, el.chooseKeys, baseName(path), path);
    el.keysHelp.textContent = 'keys.txt: a text file with one depotID;hexKey entry per line.';
    el.keysPath.removeAttribute('aria-invalid');
    el.keysField.dataset.state = 'success';
  } catch (error) {
    el.keysHelp.textContent = `${errorText(error)} Choose the keys.txt file you received for this app/depot.`;
    el.keysPath.setAttribute('aria-invalid', 'true');
    el.keysField.dataset.state = 'error';
  }
  updateReadyState();
});

el.appID.addEventListener('input', () => { state.appID = el.appID.value.trim(); updateReadyState(); });
el.depotID.addEventListener('input', () => { state.depotID = el.depotID.value.trim(); updateReadyState(); });

el.chooseFolder.addEventListener('click', async () => {
  el.folderField.dataset.state = 'loading';
  try {
    const folder = await SelectGameFolder();
    if (!folder) { el.folderField.dataset.state = state.gameFolder ? 'success' : 'default'; return; }
    state.gameFolder = folder;
    showChosen(el.gameFolder, el.chooseFolder, folder);
    el.gameFolder.removeAttribute('aria-invalid');
    el.folderField.dataset.state = 'success';
  } catch (error) {
    el.gameFolder.setAttribute('aria-invalid', 'true');
    el.message.textContent = `${errorText(error)} Choose an existing destination folder.`;
    el.folderField.dataset.state = 'error';
  }
  updateReadyState();
});

el.start.addEventListener('click', async () => {
  el.monitor.dataset.phase = 'active';
  setRunning(true); state.lines = []; el.log.textContent = ''; el.lineCount.textContent = '0 lines'; updateProgress(0); el.report.hidden = true;
  try { await StartDownload({ manifestPath: state.manifest.path, gameFolder: state.gameFolder, downloaderPath: state.downloaderPath, keysPath: state.keysPath, appId: state.appID, depotId: state.depotID, validate: el.validate.checked }); }
  catch (error) { el.monitor.dataset.phase = 'idle'; setRunning(false); el.header.dataset.state = 'error'; el.headerLabel.textContent = 'Needs attention'; el.status.textContent = 'Download could not start'; el.message.textContent = errorText(error); el.start.dataset.state = 'error'; }
});

function setTheme(theme) {
  document.documentElement.dataset.theme = theme;
  document.documentElement.classList.toggle('wa-dark', theme === 'dark');
  localStorage.setItem(themeKey, theme);
  el.themeToggle.setAttribute('aria-pressed', String(theme === 'dark'));
  el.themeLabel.textContent = theme === 'dark' ? 'Light' : 'Dark';
}

el.themeToggle.addEventListener('click', () => setTheme(document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark'));
el.minimize.addEventListener('click', () => WindowMinimise());

let maximized = false;
function setMaximized(value) {
  maximized = value;
  el.maximize.dataset.state = value ? 'maximized' : 'default';
  el.maximize.setAttribute('aria-label', `${value ? 'Restore' : 'Maximize'} DepotDrop`);
  el.maximize.title = value ? 'Restore' : 'Maximize';
}

el.maximize.addEventListener('click', () => {
  WindowToggleMaximise();
  setMaximized(!maximized);
});

try { WindowIsMaximised().then(setMaximized).catch(() => {}); } catch {}

const updateConsoleLabel = () => { el.consoleLabel.textContent = el.consolePanel.open ? 'Hide detailed log' : 'Show detailed log'; };
el.consolePanel.addEventListener('wa-show', updateConsoleLabel);
el.consolePanel.addEventListener('wa-hide', updateConsoleLabel);
updateConsoleLabel();

el.close.addEventListener('click', () => Quit());

el.pause.addEventListener('click', async () => {
  el.pause.disabled = true;
  el.status.textContent = state.paused ? 'Resuming download...' : 'Pausing downloader...';
  try { await (state.paused ? ResumeDownload() : PauseDownload()); }
  catch (error) { el.pause.disabled = false; el.message.textContent = errorText(error); }
});

el.cancel.addEventListener('click', () => {
  el.cancel.disabled = true; el.status.textContent = 'Stopping downloader…';
  CancelDownload().finally(() => { el.cancel.disabled = false; });
});

try {
  EventsOn('download:update', (update) => {
    updateProgress(update.progress); updateReport(update, update.state !== 'running'); el.status.textContent = update.message; appendLine(update.line);
    if (update.state === 'paused') {
      state.paused = true;
      el.pause.disabled = false;
      el.pause.textContent = 'Resume';
      el.start.dataset.state = 'default';
      $('.button__label').textContent = 'Paused';
      el.header.dataset.state = 'paused';
      el.headerLabel.textContent = 'Paused';
      el.message.textContent = update.message;
      return;
    }
    if (update.state === 'running') {
      if (state.paused) { state.paused = false; setRunning(true); el.report.hidden = true; }
      return;
    }
    setRunning(false);
    el.header.dataset.state = update.state;
    el.headerLabel.textContent = { complete: 'Complete', cancelled: 'Cancelled', error: 'Needs attention' }[update.state] || 'Ready';
    el.start.dataset.state = update.state === 'complete' ? 'success' : update.state === 'error' ? 'error' : 'default';
    if (update.state === 'complete') $('.button__label').textContent = 'Download again';
    el.message.textContent = update.message;
  });
} catch {}

updateReadyState();
