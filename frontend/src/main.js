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
  <section class="setup" aria-labelledby="setup-title">
    <div class="setup__intro"><p class="section-kicker">Setup</p><h1 id="setup-title">Set up your download.</h1><p>Choose the manifest, downloader, keys, and game folder. DepotDrop takes care of the command while you watch the progress.</p></div>
    <div class="path-field" id="manifest-field" data-state="default">
      <div class="path-field__copy"><label for="manifest-path">Manifest file</label><p id="manifest-help">Depot + manifest IDs are read from its filename.</p></div>
      <div class="path-field__control"><input id="manifest-path" type="text" readonly aria-required="true" aria-describedby="manifest-help" placeholder="No manifest selected"><button class="button button--secondary" id="choose-manifest" type="button"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="M4 5.5h6l2 2h8v11H4zM4 9h16"/></svg>Choose manifest</button></div>
    </div>
    <div class="identity-grid">
      <div class="identity-field"><label for="app-id">Game app ID <span class="required-mark">Required</span></label><div id="game-selector"></div><input id="app-id" type="text" inputmode="numeric" aria-required="true" aria-describedby="app-id-help" placeholder="e.g. 3468650"><p id="app-id-help">Select a Steam result or enter an app ID manually.</p></div>
      <div class="identity-field"><label for="depot-id">Depot ID <span class="optional-mark">Optional</span></label><input id="depot-id" type="text" inputmode="numeric" aria-describedby="depot-id-help" placeholder="Auto from manifest"><p id="depot-id-help">Blank uses the manifest depot.</p></div>
    </div>
    <div class="path-field" id="downloader-field" data-state="default">
      <div class="path-field__copy"><label for="downloader-path">Downloader executable</label><p id="downloader-help">Required. Choose DepotDownloaderMod.exe.</p></div>
      <div class="path-field__control"><input id="downloader-path" type="text" readonly aria-required="true" aria-describedby="downloader-help" placeholder="No downloader selected"><button class="button button--secondary" id="choose-downloader" type="button"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="M4 5.5h6l2 2h8v11H4zM4 9h16"/></svg>Choose .exe</button></div>
    </div>
    <div class="path-field" id="keys-field" data-state="default">
      <div class="path-field__copy"><label for="keys-path">keys.txt</label><p id="keys-help">Required. One depotID;hexKey entry per line.</p></div>
      <div class="path-field__control"><input id="keys-path" type="text" readonly aria-required="true" aria-describedby="keys-help" placeholder="No keys file selected"><button class="button button--secondary" id="choose-keys" type="button"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="M4 5.5h6l2 2h8v11H4zM4 9h16"/></svg>Choose keys</button></div>
    </div>
    <div class="path-field" id="folder-field" data-state="default">
      <div class="path-field__copy"><label for="game-folder">Game folder</label><p id="folder-help">Existing files can be reused; missing files are downloaded.</p></div>
      <div class="path-field__control"><input id="game-folder" type="text" readonly aria-required="true" aria-describedby="folder-help" placeholder="No destination selected"><button class="button button--secondary" id="choose-folder" type="button"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="M3.5 6h6l2 2h9v10.5h-17z"/></svg>Choose folder</button></div>
    </div>
    <div class="validate-option"><label for="validate-download"><input id="validate-download" type="checkbox" checked aria-describedby="validate-help">Validate</label><p id="validate-help">Verify all existing files. When off, the downloader may still check files needed for recovery.</p></div>
    <div class="action-row"><button class="button button--primary" id="start-download" type="button" disabled><span class="button__spinner"></span><span class="button__label">Start download</span></button><button class="button button--secondary" id="pause-download" type="button" hidden>Pause</button><button class="button button--quiet" id="cancel-download" type="button" hidden>Cancel</button><p id="form-message" class="form-message" aria-live="polite">Complete the required setup fields to continue.</p></div>
  </section>
  <section class="monitor" aria-labelledby="progress-title">
    <div class="monitor__summary"><div><p class="monitor__label" id="progress-title">Download progress</p><p class="monitor__status" id="progress-status">Waiting for paths</p></div><output class="monitor__percent" id="progress-percent" aria-live="polite">0.00%</output></div>
    <div class="progress-track" role="progressbar" aria-label="Download progress" aria-valuemin="0" aria-valuemax="100" aria-valuenow="0"><span id="progress-fill"></span></div>
    <p class="current-line" id="current-line">Downloader output will appear here.</p>
    <div class="download-report" id="download-report" hidden aria-live="polite"><div class="report-stat"><span>Downloaded</span><strong id="report-downloaded">0 B</strong></div><div class="report-stat"><span>Uncompressed</span><strong id="report-uncompressed">0 B</strong></div><div class="report-stat"><span>Elapsed</span><strong id="report-elapsed">0s</strong></div></div>
    <div class="monitor__footer"><p class="monitor__hint">The detailed console is optional.</p><wa-details class="console-disclosure" id="console-panel" appearance="plain"><span class="console-summary" slot="summary"><span class="console-toggle__label">Show console</span><span id="line-count">0 lines</span></span><pre id="output-log" tabindex="0">Waiting to start.</pre></wa-details></div>
  </section>
</main>
<footer class="app-footer"><span id="footer-app-id">APP ID NOT SET</span><span>DEPOT FROM MANIFEST</span><span id="footer-validate">VALIDATE ON</span></footer>`;

const $ = (selector) => document.querySelector(selector);
const el = { manifestField: $('#manifest-field'), manifestPath: $('#manifest-path'), manifestHelp: $('#manifest-help'), downloaderField: $('#downloader-field'), downloaderPath: $('#downloader-path'), downloaderHelp: $('#downloader-help'), keysField: $('#keys-field'), keysPath: $('#keys-path'), keysHelp: $('#keys-help'), folderField: $('#folder-field'), gameFolder: $('#game-folder'), appID: $('#app-id'), depotID: $('#depot-id'), footerAppID: $('#footer-app-id'), validate: $('#validate-download'), footerValidate: $('#footer-validate'), chooseManifest: $('#choose-manifest'), chooseDownloader: $('#choose-downloader'), chooseKeys: $('#choose-keys'), chooseFolder: $('#choose-folder'), start: $('#start-download'), cancel: $('#cancel-download'), pause: $('#pause-download'), message: $('#form-message'), header: $('.header-state'), headerLabel: $('#header-state-label'), status: $('#progress-status'), percent: $('#progress-percent'), track: $('.progress-track'), fill: $('#progress-fill'), currentLine: $('#current-line'), report: $('#download-report'), reportDownloaded: $('#report-downloaded'), reportUncompressed: $('#report-uncompressed'), reportElapsed: $('#report-elapsed'), consolePanel: $('#console-panel'), consoleLabel: $('.console-toggle__label'), lineCount: $('#line-count'), log: $('#output-log'), themeToggle: $('#theme-toggle'), themeLabel: $('#theme-label'), windowControls: $('.window-controls'), minimize: $('#window-minimize'), maximize: $('#window-maximize'), close: $('#window-close') };
const state = { manifest: null, gameFolder: '', downloaderPath: '', keysPath: '', appID: '', depotID: '', running: false, paused: false, lines: [] };
const errorText = (error) => typeof error === 'string' ? error : error?.message || String(error);

const gameSelector = createGameSelector($('#game-selector'), el.appID);

function updateReadyState() {
  const missing = [];
  if (!state.manifest) missing.push('a manifest');
  if (!state.gameFolder) missing.push('a game folder');
  if (!state.downloaderPath) missing.push('DepotDownloaderMod.exe');
  if (!state.keysPath) missing.push('keys.txt');
  if (!/^\d+$/.test(state.appID)) missing.push(state.appID ? 'a numeric game app ID' : 'the game app ID');
  const ready = missing.length === 0 && !state.running;
  el.start.disabled = !ready;
  el.start.setAttribute('aria-disabled', String(!ready));
  if (!state.running) el.message.textContent = ready ? 'Ready to download.' : missing.length === 1 ? `Add ${missing[0]} to continue.` : 'Complete the required setup fields to continue.';
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
    el.manifestPath.value = manifest.path;
    state.depotID = manifest.depotId;
    el.depotID.value = manifest.depotId;
    el.manifestHelp.textContent = `Manifest ${manifest.manifestId} · Depot ${manifest.depotId}`;
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
    el.downloaderPath.value = path;
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
    el.keysPath.value = path;
    el.keysPath.removeAttribute('aria-invalid');
    el.keysField.dataset.state = 'success';
  } catch (error) {
    el.keysHelp.textContent = `${errorText(error)} Choose the keys.txt file you received for this app/depot.`;
    el.keysPath.setAttribute('aria-invalid', 'true');
    el.keysField.dataset.state = 'error';
  }
  updateReadyState();
});

el.validate.addEventListener('change', () => { el.footerValidate.textContent = el.validate.checked ? 'VALIDATE ON' : 'VALIDATE OFF'; });

el.appID.addEventListener('input', () => { state.appID = el.appID.value.trim(); el.footerAppID.textContent = state.appID ? `APP ${state.appID}` : 'APP ID NOT SET'; updateReadyState(); });
el.depotID.addEventListener('input', () => { state.depotID = el.depotID.value.trim(); updateReadyState(); });

el.chooseFolder.addEventListener('click', async () => {
  el.folderField.dataset.state = 'loading';
  try {
    const folder = await SelectGameFolder();
    if (!folder) { el.folderField.dataset.state = state.gameFolder ? 'success' : 'default'; return; }
    state.gameFolder = folder;
    el.gameFolder.value = folder;
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
  setRunning(true); state.lines = []; el.log.textContent = ''; el.lineCount.textContent = '0 lines'; updateProgress(0); el.report.hidden = true;
  try { await StartDownload({ manifestPath: state.manifest.path, gameFolder: state.gameFolder, downloaderPath: state.downloaderPath, keysPath: state.keysPath, appId: state.appID, depotId: state.depotID, validate: el.validate.checked }); }
  catch (error) { setRunning(false); el.header.dataset.state = 'error'; el.headerLabel.textContent = 'Needs attention'; el.status.textContent = 'Download could not start'; el.message.textContent = errorText(error); el.start.dataset.state = 'error'; }
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

const updateConsoleLabel = () => { el.consoleLabel.textContent = el.consolePanel.open ? 'Hide console' : 'Show console'; };
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
    el.message.textContent = update.message;
  });
} catch {}

updateReadyState();
