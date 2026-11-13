import { createJobTimer } from './eta.js';

const HISTORY = 60;
const tile = (label, id, hint = '') => `
  <div class="job-stat" ${hint ? `title="${hint}"` : ''}>
    <span class="job-stat__label">${label}</span>
    <strong class="job-stat__value" id="${id}">—</strong>
  </div>`;

export function createJobStats(root, { formatBytes }) {
  root.className = 'job-stats';
  root.hidden = true;
  root.innerHTML = `
    <div class="job-stats__grid">
      ${tile('Download speed', 'job-speed', 'Compressed bytes from completed chunks per second. Transfers in progress and retries are not counted yet.')}
      ${tile('Validation read', 'job-read', 'Bytes read by the validator per second, including reads served from cache.')}
      ${tile('Elapsed', 'job-elapsed')}
      ${tile('Time left', 'job-left', 'Rough estimate based on completed-file progress. Validation and downloads can run at different rates.')}
    </div>
    <svg class="job-stats__graph" viewBox="0 0 ${HISTORY} 24" preserveAspectRatio="none" role="img" aria-label="Completed chunk download speed over the last minute">
      <polyline class="job-stats__area" points=""></polyline>
      <polyline class="job-stats__line" points=""></polyline>
    </svg>
    <p class="job-stats__note" id="job-received"></p>`;

  const speedEl = root.querySelector('#job-speed');
  const readEl = root.querySelector('#job-read');
  const elapsedEl = root.querySelector('#job-elapsed');
  const leftEl = root.querySelector('#job-left');
  const receivedEl = root.querySelector('#job-received');
  const line = root.querySelector('.job-stats__line');
  const area = root.querySelector('.job-stats__area');
  let lastBytes = -1;
  let receivedBytes = 0;
  let history = [];
  let telemetryTimeout = 0;

  const timer = createJobTimer((view) => {
    if (!view.active) { root.hidden = true; return; }
    root.hidden = false;
    elapsedEl.textContent = view.elapsed;
    leftEl.textContent = view.left;
  });

  function drawGraph() {
    const peak = Math.max(...history, 1);
    const points = history.map((value, index) => `${index + HISTORY - history.length},${(24 - (value / peak) * 22).toFixed(1)}`);
    line.setAttribute('points', points.join(' '));
    area.setAttribute('points', points.length ? `${HISTORY - history.length},24 ${points.join(' ')} ${HISTORY - 1},24` : '');
  }

  function showMetrics(metrics) {
    if (!metrics) return;
    window.clearTimeout(telemetryTimeout);
    const bytes = Math.max(0, Number(metrics.downloadedBytes) || 0);
    receivedBytes += lastBytes < 0 || bytes < lastBytes ? bytes : bytes - lastBytes;
    lastBytes = bytes;
    const downloadRate = Math.max(0, Number(metrics.downloadRate) || 0);
    const readRate = Math.max(0, Number(metrics.validationRate) || 0);
    speedEl.textContent = `${formatBytes(Math.round(downloadRate))}/s`;
    readEl.textContent = `${formatBytes(Math.round(readRate))}/s`;
    receivedEl.textContent = `${formatBytes(receivedBytes)} in completed chunks`;
    history.push(downloadRate);
    if (history.length > HISTORY) history.shift();
    drawGraph();
  }

  return {
    start() {
      lastBytes = -1; receivedBytes = 0; history = [];
      speedEl.textContent = '—'; readEl.textContent = '—';
      receivedEl.textContent = 'Waiting for downloader telemetry';
      window.clearTimeout(telemetryTimeout);
      telemetryTimeout = window.setTimeout(() => { receivedEl.textContent = 'Job statistics unavailable with this downloader'; }, 5000);
      drawGraph();
      timer.start();
    },
    stop() { window.clearTimeout(telemetryTimeout); timer.stop(); },
    pause() { window.clearTimeout(telemetryTimeout); speedEl.textContent = '0 B/s'; readEl.textContent = '0 B/s'; timer.pause(); },
    resume() { lastBytes = -1; timer.resume(); },
    report(percent, metrics) { timer.report(percent); showMetrics(metrics); },
  };
}
