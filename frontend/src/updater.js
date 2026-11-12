import { CheckForUpdate, DownloadUpdate, RestartForUpdate } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import './updater.css';

export function setupUpdater() {
  const button = document.querySelector('#check-update');
  const dialog = document.createElement('dialog');
  dialog.className = 'update-dialog';
  dialog.setAttribute('aria-labelledby', 'update-title');
  dialog.innerHTML = `<h2 id="update-title">DepotDrop updates</h2><p id="update-status" role="status" aria-live="polite"></p><div class="update-actions"><button type="button" class="button button--secondary" id="update-close">Close</button><button type="button" class="button button--primary" id="update-action" hidden></button></div>`;
  document.body.append(dialog);
  const status = dialog.querySelector('#update-status');
  const action = dialog.querySelector('#update-action');
  const close = dialog.querySelector('#update-close');
  let busy = false;
  let phase = 'check';
  const setBusy = (value) => { busy = value; action.disabled = value; close.disabled = value; };
  const showAction = (label, next) => { action.hidden = false; action.textContent = label; phase = next; };
  const message = (error) => String(error?.message || error);
  async function check() {
    setBusy(true); action.hidden = true; status.textContent = 'Checking for updates…';
    try {
      const result = await CheckForUpdate();
      if (result.available) { status.textContent = `Version ${result.available} is available. You have version ${result.current}.`; showAction('Download update', 'download'); }
      else status.textContent = `You’re up to date (version ${result.current}).`;
    } catch (error) { status.textContent = message(error); showAction('Try again', 'check'); }
    finally { setBusy(false); }
  }
  button.addEventListener('click', () => { dialog.showModal(); if (phase !== 'restart') check(); });
  close.addEventListener('click', () => dialog.close());
  dialog.addEventListener('cancel', (event) => { if (busy) event.preventDefault(); });
  action.addEventListener('click', async () => {
    if (phase === 'check') return check();
    setBusy(true);
    try {
      if (phase === 'download') {
        status.textContent = 'Downloading update…';
        await DownloadUpdate();
        status.textContent = 'Update ready. Restart DepotDrop to apply it.';
        showAction('Restart and update', 'restart');
      } else { status.textContent = 'Restarting…'; await RestartForUpdate(); }
    } catch (error) { status.textContent = message(error); showAction(phase === 'restart' ? 'Try restart again' : 'Try again', phase); }
    finally { setBusy(false); }
  });
  EventsOn('wails:updater:download-progress', (progress) => {
    if (busy && progress.total > 0) status.textContent = `Downloading update… ${Math.min(100, Math.round(progress.written / progress.total * 100))}%`;
  });
}
