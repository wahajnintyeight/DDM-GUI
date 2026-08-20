import './style.css';
import { CancelDownload, SelectGameFolder, SelectManifest, StartDownload } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

document.querySelector('#app').innerHTML = `
<header class="app-header"><div class="brand"><span class="brand__mark">D</span><span class="brand__name">DepotDrop</span></div><div class="header-state" aria-live="polite"><span class="header-state__light"></span><span id="header-state-label">Ready</span></div></header>
<main class="workspace">
  <section class="setup" aria-labelledby="setup-title">
    <div class="setup__intro"><h1 id="setup-title">Pick two paths. Start the download.</h1><p>Choose the depot manifest and the folder where Cricket 26 should be installed. DepotDrop handles the command.</p></div>
    <div class="path-field" id="manifest-field" data-state="default">
      <div class="path-field__copy"><label for="manifest-path">Manifest file</label><p id="manifest-help">Expected format: 3468651_795493147105120679.manifest</p></div>
      <div class="path-field__control"><input id="manifest-path" type="text" readonly aria-required="true" aria-describedby="manifest-help" placeholder="No manifest selected"><button class="button button--secondary" id="choose-manifest" type="button"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="M4 5.5h6l2 2h8v11H4zM4 9h16"/></svg>Choose manifest</button></div>
    </div>
    <div class="path-field" id="folder-field" data-state="default">
      <div class="path-field__copy"><label for="game-folder">Game folder</label><p id="folder-help">Existing files are validated; missing files are downloaded.</p></div>
      <div class="path-field__control"><input id="game-folder" type="text" readonly aria-required="true" aria-describedby="folder-help" placeholder="No destination selected"><button class="button button--secondary" id="choose-folder" type="button"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="M3.5 6h6l2 2h9v10.5h-17z"/></svg>Choose folder</button></div>
    </div>
    <div class="action-row"><button class="button button--primary" id="start-download" type="button" disabled><span class="button__spinner"></span><span class="button__label">Start download</span></button><button class="button button--quiet" id="cancel-download" type="button" hidden>Cancel</button><p id="form-message" class="form-message" aria-live="polite">Select both paths to continue.</p></div>
  </section>
  <section class="monitor" aria-labelledby="progress-title">
    <div class="monitor__summary"><div><p class="monitor__label" id="progress-title">Download progress</p><p class="monitor__status" id="progress-status">Waiting for paths</p></div><output class="monitor__percent" id="progress-percent" aria-live="polite">0.00%</output></div>
    <div class="progress-track" role="progressbar" aria-label="Download progress" aria-valuemin="0" aria-valuemax="100" aria-valuenow="0"><span id="progress-fill"></span></div>
    <p class="current-line" id="current-line">Downloader output will appear here.</p>
    <details class="output-panel"><summary>View output log <span id="line-count">0 lines</span></summary><pre id="output-log" tabindex="0">Waiting to start.</pre></details>
  </section>
</main>
<footer class="app-footer"><span>APP 3468650</span><span>DEPOT FROM MANIFEST</span><span>VALIDATE ON</span></footer>`;

const $ = (selector) => document.querySelector(selector);
const el = { manifestField: $('#manifest-field'), manifestPath: $('#manifest-path'), manifestHelp: $('#manifest-help'), folderField: $('#folder-field'), gameFolder: $('#game-folder'), chooseManifest: $('#choose-manifest'), chooseFolder: $('#choose-folder'), start: $('#start-download'), cancel: $('#cancel-download'), message: $('#form-message'), header: $('.header-state'), headerLabel: $('#header-state-label'), status: $('#progress-status'), percent: $('#progress-percent'), track: $('.progress-track'), fill: $('#progress-fill'), currentLine: $('#current-line'), lineCount: $('#line-count'), log: $('#output-log') };
const state = { manifest: null, gameFolder: '', running: false, lines: [] };
const errorText = (error) => typeof error === 'string' ? error : error?.message || String(error);

function updateReadyState() {
  const ready = Boolean(state.manifest && state.gameFolder && !state.running);
  el.start.disabled = !ready;
  el.start.setAttribute('aria-disabled', String(!ready));
  if (!state.running) el.message.textContent = ready ? 'Ready to download and validate.' : 'Select both paths to continue.';
}

function setRunning(running) {
  state.running = running;
  el.chooseManifest.disabled = running;
  el.chooseFolder.disabled = running;
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
    el.manifestHelp.textContent = `Manifest ${manifest.manifestId} · Depot ${manifest.depotId}`;
    el.manifestPath.removeAttribute('aria-invalid');
    el.manifestField.dataset.state = 'success';
  } catch (error) {
    el.manifestHelp.textContent = `${errorText(error)} Choose a manifest named DEPOT_MANIFEST.manifest.`;
    el.manifestPath.setAttribute('aria-invalid', 'true');
    el.manifestField.dataset.state = 'error';
  }
  updateReadyState();
});

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
  setRunning(true); state.lines = []; el.log.textContent = ''; el.lineCount.textContent = '0 lines'; updateProgress(0);
  try { await StartDownload({ manifestPath: state.manifest.path, gameFolder: state.gameFolder }); }
  catch (error) { setRunning(false); el.header.dataset.state = 'error'; el.headerLabel.textContent = 'Needs attention'; el.status.textContent = 'Download could not start'; el.message.textContent = errorText(error); el.start.dataset.state = 'error'; }
});

el.cancel.addEventListener('click', () => {
  el.cancel.disabled = true; el.status.textContent = 'Stopping downloader…';
  CancelDownload().finally(() => { el.cancel.disabled = false; });
});

EventsOn('download:update', (update) => {
  updateProgress(update.progress); el.status.textContent = update.message; appendLine(update.line);
  if (update.state === 'running') return;
  setRunning(false);
  el.header.dataset.state = update.state;
  el.headerLabel.textContent = { complete: 'Complete', cancelled: 'Cancelled', error: 'Needs attention' }[update.state] || 'Ready';
  el.start.dataset.state = update.state === 'complete' ? 'success' : update.state === 'error' ? 'error' : 'default';
  el.message.textContent = update.message;
});

updateReadyState();
