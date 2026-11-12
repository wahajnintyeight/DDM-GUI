import { CheckForUpdate, DownloadUpdate, RestartForUpdate } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import './updater.css';

export function setupUpdater() {
  const indicator = document.querySelector('.update-indicator');
  const button = document.querySelector('#check-update');
  const announcement = document.querySelector('#update-announcement');
  const dialog = document.createElement('dialog');
  dialog.className = 'update-dialog';
  dialog.setAttribute('aria-labelledby', 'update-title');
  dialog.innerHTML = `<div class="update-dialog__icon" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M12 3v11m-4-4 4 4 4-4M4.5 16.5v2.25A1.75 1.75 0 0 0 6.25 20.5h11.5a1.75 1.75 0 0 0 1.75-1.75V16.5"/></svg></div><h2 id="update-title">A new version is ready</h2><p id="update-status" role="status" aria-live="polite"></p><div class="update-actions"><button type="button" class="button button--secondary" id="update-close">Not now</button><button type="button" class="button button--primary" id="update-action"></button></div>`;
  document.body.append(dialog);
  const status = dialog.querySelector('#update-status');
  const action = dialog.querySelector('#update-action');
  const close = dialog.querySelector('#update-close');
  const title = dialog.querySelector('#update-title');
  let busy = false;
  let checking = false;
  let phase = 'check';
  const setBusy = (value) => { busy = value; action.disabled = value; close.disabled = value; };
  const showAction = (label, next) => { action.textContent = label; phase = next; };
  const message = (error) => String(error?.message || error);

  async function check() {
    if (checking || busy || phase === 'restart' || phase === 'download') return;
    checking = true;
    try {
      const result = await CheckForUpdate();
      if (!result.available) return;
      status.textContent = `Version ${result.available} is available. You’re using version ${result.current}.`;
      showAction('Download update', 'download');
      indicator.hidden = false;
      indicator.dataset.notify = 'true';
      announcement.textContent = `DepotDrop version ${result.available} is available. Select the update icon to learn more.`;
      setTimeout(() => { delete indicator.dataset.notify; }, 8000);
    } catch {
      // Retry quietly on the next check when offline or rate-limited.
    } finally { checking = false; }
  }

  button.addEventListener('click', () => { delete indicator.dataset.notify; dialog.showModal(); });
  close.addEventListener('click', () => dialog.close());
  dialog.addEventListener('cancel', (event) => { if (busy) event.preventDefault(); });
  action.addEventListener('click', async () => {
    setBusy(true);
    try {
      if (phase === 'download') {
        title.textContent = 'Downloading update';
        status.textContent = 'Getting the new version ready…';
        await DownloadUpdate();
        title.textContent = 'Ready to restart';
        status.textContent = 'Restart DepotDrop to finish installing the update.';
        showAction('Restart and update', 'restart');
        button.setAttribute('aria-label', 'Update ready to install');
        button.title = 'Update ready to install';
      } else {
        status.textContent = 'Restarting…';
        await RestartForUpdate();
      }
    } catch (error) {
      title.textContent = phase === 'restart' ? 'Couldn’t restart' : 'Couldn’t download update';
      status.textContent = message(error);
      showAction(phase === 'restart' ? 'Try restart again' : 'Try again', phase);
    } finally { setBusy(false); }
  });
  EventsOn('wails:updater:download-progress', (progress) => {
    if (busy && progress.total > 0) status.textContent = `Downloading… ${Math.min(100, Math.round(progress.written / progress.total * 100))}%`;
  });
  check();
  setInterval(check, 60 * 60 * 1000);
}
