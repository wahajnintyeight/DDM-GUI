// The live numbers shown while a job runs: download speed (with a small history graph), time elapsed and time left.
//
// Speed comes from the computer's network adapters, polled once a second and smoothed over a few seconds.
// It therefore also reflects other programs using the network, and reads near zero while files are only
// being checked on disk, which is useful to see in itself.

import { createJobTimer } from './eta.js';

const HISTORY = 60; // seconds of speed history in the graph
const SPEED_SMOOTHING_SECONDS = 4;

const tile = (label, id, hint = '') => `
  <div class="job-stat" ${hint ? `title="${hint}"` : ''}>
    <span class="job-stat__label">${label}</span>
    <strong class="job-stat__value" id="${id}">—</strong>
  </div>`;

export function createJobStats(root, { readNetworkBytes, formatBytes }) {
  root.className = 'job-stats';
  root.hidden = true;
  root.innerHTML = `
    <div class="job-stats__grid">
      ${tile('Speed', 'job-speed', 'Network speed of this computer. It also counts other programs using the connection.')}
      ${tile('Elapsed', 'job-elapsed')}
      ${tile('Time left', 'job-left', 'An estimate that gets steadier as the job goes on.')}
    </div>
    <svg class="job-stats__graph" viewBox="0 0 ${HISTORY} 24" preserveAspectRatio="none" role="img" aria-label="Download speed over the last minute">
      <polyline class="job-stats__area" points=""></polyline>
      <polyline class="job-stats__line" points=""></polyline>
    </svg>
    <p class="job-stats__note" id="job-received"></p>`;

  const speedEl = root.querySelector('#job-speed');
  const elapsedEl = root.querySelector('#job-elapsed');
  const leftEl = root.querySelector('#job-left');
  const receivedEl = root.querySelector('#job-received');
  const line = root.querySelector('.job-stats__line');
  const area = root.querySelector('.job-stats__area');

  let pollId = 0;
  let paused = false;
  let lastBytes = -1;
  let startBytes = -1;
  let lastAt = 0;
  let speed = 0;
  let history = [];
  let supported = true;

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

  async function poll() {
    if (paused || !supported) return;
    let bytes;
    try { bytes = await readNetworkBytes(); } catch { bytes = -1; }
    if (bytes < 0) { supported = false; speedEl.textContent = 'n/a'; receivedEl.textContent = ''; return; }
    const now = performance.now();
    if (startBytes < 0) startBytes = bytes;
    if (lastBytes >= 0) {
      const dt = (now - lastAt) / 1000;
      if (dt > 0) {
        const instant = Math.max(0, (bytes - lastBytes) / dt);
        speed += (1 - Math.exp(-dt / SPEED_SMOOTHING_SECONDS)) * (instant - speed);
        history.push(speed);
        if (history.length > HISTORY) history.shift();
      }
    }
    lastBytes = bytes;
    lastAt = now;
    speedEl.textContent = `${formatBytes(Math.round(speed))}/s`;
    receivedEl.textContent = `${formatBytes(bytes - startBytes)} received since this job started (whole computer)`;
    drawGraph();
  }

  return {
    start() {
      paused = false; supported = true;
      lastBytes = startBytes = -1; speed = 0; history = [];
      speedEl.textContent = '—'; receivedEl.textContent = '';
      drawGraph();
      timer.start();
      window.clearInterval(pollId);
      pollId = window.setInterval(poll, 1000);
      poll();
    },
    stop() {
      window.clearInterval(pollId);
      pollId = 0;
      timer.stop();
    },
    pause() {
      paused = true;
      speed = 0;
      speedEl.textContent = '0 B/s';
      timer.pause();
    },
    resume() {
      paused = false;
      lastBytes = -1; // do not count the time spent paused
      timer.resume();
    },
    report: (percent) => timer.report(percent),
  };
}
