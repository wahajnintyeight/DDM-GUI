import './downgrade.css';
import { createGameSelector } from './game-selector';
import { InspectDowngradeManifest, PickManifestFile, PreflightDowngrade, SelectDownloader, SelectGameFolder, SelectKeys, StartDowngrade } from '../wailsjs/go/main/App';

const folderIcon = '<svg aria-hidden="true" viewBox="0 0 24 24"><path d="M4 5.5h6l2 2h8v11H4zM4 9h16"/></svg>';
const checkIcon = '<svg aria-hidden="true" viewBox="0 0 24 24"><path d="m5 12.5 4.5 4.5L19 7.5"/></svg>';
const baseName = (path) => path.split(/[\\/]/).pop();

const fileRow = (id, label, help, placeholder = 'No file chosen') => `<div class="file-row" data-row="${id}"><label for="${id}">${label}</label><div class="file-row__control"><input id="${id}" type="text" readonly placeholder="${placeholder}"><button class="button button--secondary" type="button" data-pick="${id}">${folderIcon}<span class="button__text">Browse</span></button></div>${help ? `<p>${help}</p>` : ''}</div>`;

export function createDowngrade({ el, state, beginRun, abortRun, errorText, formatBytes }) {
  const workspace = document.querySelector('.workspace');
  const view = document.createElement('section');
  view.className = 'setup';
  view.dataset.view = 'downgrade';
  view.setAttribute('aria-labelledby', 'downgrade-title');
  view.innerHTML = `<div class="setup__inner">
    <div class="setup__intro"><h1 id="downgrade-title">Go back to an older version</h1><p>Restore a game to an older version using two manifest files. Files that already match are kept, so only what changed is downloaded.</p></div>
    <section class="step" data-step="game"><div class="step__badge" aria-hidden="true"><span>1</span>${checkIcon}</div><div class="step__body">
      <h2>Which game?</h2><p class="step__hint">Search by name. If you already know the Steam App ID, switch to “Enter App ID”.</p>
      <div id="dg-game-selector"></div>
      <div class="field app-id-field"><label for="dg-app">Steam App ID</label><input id="dg-app" type="text" inputmode="numeric" placeholder="e.g. 3468650"></div>
      <div class="field"><label for="dg-depot">Depot ID</label><input id="dg-depot" type="text" inputmode="numeric" placeholder="Filled in when you choose a manifest file"><p id="dg-depot-help">Filled in for you from the manifest file. Change it only if you know you need to.</p></div>
    </div></section>
    <section class="step" data-step="versions"><div class="step__badge" aria-hidden="true"><span>2</span>${checkIcon}</div><div class="step__body">
      <h2>Pick the two manifest files</h2><p class="step__hint">DepotDrop reads the depot and version numbers from inside each file.</p>
      ${fileRow('dg-current', 'Current manifest file', 'The manifest of the version installed on your computer right now.')}
      ${fileRow('dg-target', 'Previous version manifest', 'The manifest of the older version you want back.')}
      <div id="dg-extras"></div>
      <button class="button button--quiet dg-add" id="dg-add-depot" type="button">Add another depot</button>
      <p class="dg-note">Some games use more than one depot. Add a pair for each so every depot goes back together.</p>
    </div></section>
    <section class="step" data-step="tools"><div class="step__badge" aria-hidden="true"><span>3</span>${checkIcon}</div><div class="step__body">
      <h2>Add your tools</h2>
      ${fileRow('dg-downloader', 'Downloader program', 'A DepotDownloaderMod.exe build that supports downgrades.')}
      ${fileRow('dg-keys', 'Keys file', 'keys.txt with a key for each depot above.')}
    </div></section>
    <section class="step" data-step="folder"><div class="step__badge" aria-hidden="true"><span>4</span>${checkIcon}</div><div class="step__body">
      <h2>Pick the game folder</h2><p class="step__hint">The folder the game is installed in. Files from the old version that the older one doesn’t use are removed only after everything succeeds.</p>
      ${fileRow('dg-folder', 'Game folder', '', 'No folder chosen')}
      <details class="advanced"><summary>Advanced: this game has other depots</summary>
        <div class="validate-option"><label for="dg-partial"><input id="dg-partial" type="checkbox">Update only the depots above</label><p>Use this if the folder holds depots you aren’t adding. Nothing will be removed, because those depots might use the same files.</p></div>
      </details>
    </div></section>
  </div>`;
  workspace.insertBefore(view, workspace.querySelector('.monitor'));

  const $ = (selector) => view.querySelector(selector);
  const gameSelector = createGameSelector($('#dg-game-selector'), $('#dg-app'), 'dg');
  const panel = document.querySelector('[data-checklist=downgrade]');
  const summaryBox = document.querySelector('#dg-summary');
  const button = document.querySelector('#start-downgrade');
  const message = document.querySelector('#form-message');
  const dg = { appId: '', depotId: '', current: '', target: '', downloader: '', keys: '', folder: '', partial: false, extras: [], reviewed: false, busy: false, depotAuto: false, currentDepot: '', targetDepot: '' };

  const set = (id, path, shown = baseName(path)) => { const input = $(`#${id}`); input.value = shown; input.title = path; input.closest('.file-row').dataset.state = 'success'; input.closest('.file-row').querySelector('.button__text').textContent = 'Change'; };

  async function pick(id, choose, assign, after) {
    const row = $(`#${id}`).closest('.file-row');
    row.dataset.state = 'loading';
    try {
      const path = await choose();
      if (path) { assign(path); set(id, path, id === 'dg-folder' ? path : baseName(path)); invalidate(); if (after) await after(path, row); }
      else row.dataset.state = $(`#${id}`).value ? 'success' : 'default';
    } catch (error) {
      row.dataset.state = 'error';
      message.textContent = errorText(error);
    }
  }

  // Read the depot and manifest IDs from inside a manifest file and use them to fill in the depot.
  async function learn(path, row, side, extra) {
    const help = row.querySelector('p');
    try {
      const info = await InspectDowngradeManifest(path);
      if (help) help.textContent = `Depot ${info.depotId}, manifest ${info.manifestId}.`;
      if (extra) {
        if (!extra.depotId) { extra.depotId = info.depotId; extra.input.value = info.depotId; }
      } else {
        dg[`${side}Depot`] = info.depotId;
        if (!dg.depotId || dg.depotAuto) { dg.depotId = info.depotId; dg.depotAuto = true; $('#dg-depot').value = info.depotId; }
        const other = side === 'current' ? dg.targetDepot : dg.currentDepot;
        if (other && other !== info.depotId) {
          row.dataset.state = 'error';
          message.textContent = 'The two manifest files belong to different depots. Choose two manifests from the same depot.';
          message.dataset.keep = 'true';
        }
      }
    } catch (error) {
      row.dataset.state = 'error';
      if (help) help.textContent = errorText(error);
    }
    invalidate();
  }

  const complete = () => {
    const extrasDone = dg.extras.every((extra) => /^\d+$/.test(extra.depotId) && extra.current && extra.target);
    return {
      game: /^\d+$/.test(dg.appId) && /^\d+$/.test(dg.depotId),
      versions: Boolean(dg.current && dg.target) && extrasDone,
      downloader: Boolean(dg.downloader),
      keys: Boolean(dg.keys),
      folder: Boolean(dg.folder),
    };
  };

  function refresh() {
    const done = complete();
    view.querySelector('[data-step=game]').dataset.done = String(done.game);
    view.querySelector('[data-step=versions]').dataset.done = String(done.versions);
    view.querySelector('[data-step=tools]').dataset.done = String(done.downloader && done.keys);
    view.querySelector('[data-step=folder]').dataset.done = String(done.folder);
    panel.querySelectorAll('li').forEach((item) => { item.dataset.done = String(done[item.dataset.item]); });
    const remaining = Object.values(done).filter((value) => !value).length;
    document.querySelector('#dg-ready-title').textContent = dg.reviewed ? 'Ready to start' : remaining === 0 ? 'Review before you start' : remaining === 1 ? 'Just 1 thing left' : `${remaining} things left`;
    summaryBox.hidden = !dg.reviewed;
    if (!state.running) {
      button.disabled = remaining !== 0 || dg.busy;
      button.querySelector('.button__label').textContent = dg.reviewed ? 'Start downgrade' : 'Review downgrade';
      if (remaining !== 0) message.textContent = 'Complete the steps to unlock this button.';
      else if (!dg.reviewed && !dg.busy && !message.dataset.keep) message.textContent = 'Nothing changes until you press Start downgrade.';
    }
  }

  // Any change to the inputs means the earlier review no longer applies.
  function invalidate() {
    dg.reviewed = false;
    delete message.dataset.keep;
    refresh();
  }

  const request = () => ({
    appId: dg.appId,
    gameFolder: dg.folder,
    downloaderPath: dg.downloader,
    keysPath: dg.keys,
    acknowledgePartial: dg.partial,
    depots: [{ depotId: dg.depotId, currentManifestPath: dg.current, targetManifestPath: dg.target }, ...dg.extras.map((extra) => ({ depotId: extra.depotId, currentManifestPath: extra.current, targetManifestPath: extra.target }))],
  });

  function renderSummary(summary) {
    const rows = summary.depots.map((depot) => `<li><strong>Depot ${depot.depotId}</strong><span>Version ${depot.currentManifestId} → ${depot.targetManifestId}</span></li>`).join('');
    const removal = summary.partial ? 'Nothing will be removed (partial downgrade).' : summary.removeCount ? `${summary.removeCount} old ${summary.removeCount === 1 ? 'file' : 'files'} (${formatBytes(summary.removeBytes)}) will be removed after the download succeeds.` : 'No old files need to be removed.';
    const notes = (summary.notes || []).map((note) => `<li>${note.replace(/[&<>]/g, (char) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' })[char])}</li>`).join('');
    summaryBox.innerHTML = `<p class="monitor__label">What will happen</p><ul class="dg-depots">${rows}</ul><p class="dg-removal">${removal}</p>${notes ? `<ul class="dg-notes">${notes}</ul>` : ''}<p class="dg-where">In: <code></code></p>`;
    summaryBox.querySelector('code').textContent = summary.gameFolder;
  }

  async function onPrimary() {
    if (state.running || button.disabled) return;
    if (!dg.reviewed) {
      dg.busy = true; button.dataset.state = 'loading'; button.disabled = true; message.textContent = 'Checking your files…';
      try {
        renderSummary(await PreflightDowngrade(request()));
        dg.reviewed = true;
        message.textContent = '';
      } catch (error) {
        message.textContent = errorText(error);
        message.dataset.keep = 'true';
      }
      dg.busy = false; button.dataset.state = 'default';
      refresh();
      return;
    }
    beginRun();
    button.dataset.state = 'loading';
    button.querySelector('.button__label').textContent = 'Downgrading';
    try { await StartDowngrade(request()); }
    catch (error) { abortRun(); el.message.textContent = errorText(error); message.dataset.keep = 'true'; dg.reviewed = false; button.dataset.state = 'error'; refresh(); }
  }

  function addExtra() {
    const extra = { depotId: '', current: '', target: '' };
    dg.extras.push(extra);
    const block = document.createElement('div');
    block.className = 'dg-extra';
    const n = dg.extras.length;
    block.innerHTML = `<div class="dg-extra__head"><strong>Extra depot ${n}</strong><button class="button button--quiet" type="button" data-remove>Remove</button></div><div class="field"><label for="dg-x${n}-depot">Depot ID</label><input id="dg-x${n}-depot" type="text" inputmode="numeric"></div>${fileRow(`dg-x${n}-current`, 'Current manifest file', '')}${fileRow(`dg-x${n}-target`, 'Previous version manifest', '')}`;
    $('#dg-extras').append(block);
    extra.input = block.querySelector('input[type=text]:not([readonly])');
    extra.input.addEventListener('input', (event) => { extra.depotId = event.target.value.trim(); invalidate(); });
    block.querySelector(`[data-pick=dg-x${n}-current]`).addEventListener('click', () => pick(`dg-x${n}-current`, () => PickManifestFile('Choose the current manifest file (the version installed now)'), (path) => { extra.current = path; }, (path, row) => learn(path, row, 'current', extra)));
    block.querySelector(`[data-pick=dg-x${n}-target]`).addEventListener('click', () => pick(`dg-x${n}-target`, () => PickManifestFile('Choose the previous version manifest (the older version you want)'), (path) => { extra.target = path; }, (path, row) => learn(path, row, 'target', extra)));
    block.querySelector('[data-remove]').addEventListener('click', () => { dg.extras.splice(dg.extras.indexOf(extra), 1); block.remove(); invalidate(); });
    invalidate();
  }

  $('#dg-app').addEventListener('input', (event) => { dg.appId = event.target.value.trim(); invalidate(); });
  $('#dg-depot').addEventListener('input', (event) => { dg.depotId = event.target.value.trim(); dg.depotAuto = false; invalidate(); });
  $('#dg-partial').addEventListener('change', (event) => { dg.partial = event.target.checked; invalidate(); });
  $('[data-pick=dg-current]').addEventListener('click', () => pick('dg-current', () => PickManifestFile('Choose the current manifest file (the version installed now)'), (path) => { dg.current = path; }, (path, row) => learn(path, row, 'current')));
  $('[data-pick=dg-target]').addEventListener('click', () => pick('dg-target', () => PickManifestFile('Choose the previous version manifest (the older version you want)'), (path) => { dg.target = path; }, (path, row) => learn(path, row, 'target')));
  $('[data-pick=dg-downloader]').addEventListener('click', () => pick('dg-downloader', SelectDownloader, (path) => { dg.downloader = path; }));
  $('[data-pick=dg-keys]').addEventListener('click', () => pick('dg-keys', SelectKeys, (path) => { dg.keys = path; }));
  $('[data-pick=dg-folder]').addEventListener('click', () => pick('dg-folder', SelectGameFolder, (path) => { dg.folder = path; }));
  $('#dg-add-depot').addEventListener('click', addExtra);
  button.addEventListener('click', onPrimary);

  // The shared status panel reports running state; mirror it on this view's button and locks.
  document.addEventListener('depotdrop:running', (event) => {
    const running = event.detail;
    view.querySelectorAll('input, button').forEach((control) => { control.disabled = running; });
    gameSelector.setDisabled(running);
    if (!running) { button.dataset.state = button.dataset.state === 'loading' ? 'default' : button.dataset.state; dg.reviewed = false; }
    refresh();
  });
  document.addEventListener('depotdrop:update', (event) => {
    if (workspace.dataset.mode !== 'downgrade') return;
    const update = event.detail;
    if (update.state === 'complete') { button.dataset.state = 'success'; button.querySelector('.button__label').textContent = 'Downgrade again'; }
    else if (update.state === 'error') button.dataset.state = 'error';
    else if (update.state === 'paused') { button.dataset.state = 'default'; button.querySelector('.button__label').textContent = 'Paused'; }
  });
  document.addEventListener('depotdrop:refresh', refresh);

  return {
    // Carry over choices the user already made on the Download tab.
    adopt(shared) {
      const take = (key, id, assign, shown) => { if (!dg[key] && shared[key]) { assign(shared[key]); set(id, shared[key], shown); } };
      take('downloader', 'dg-downloader', (path) => { dg.downloader = path; });
      take('keys', 'dg-keys', (path) => { dg.keys = path; });
      take('folder', 'dg-folder', (path) => { dg.folder = path; }, shared.folder);
      if (!dg.appId && shared.appId) { gameSelector.showManual(); $('#dg-app').value = shared.appId; $('#dg-app').dispatchEvent(new Event('input', { bubbles: true })); }
      if (!dg.depotId && shared.depotId) { dg.depotId = shared.depotId; $('#dg-depot').value = shared.depotId; }
      invalidate();
    },
    refresh,
  };
}
