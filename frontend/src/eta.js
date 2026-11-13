// Elapsed time and a smoothed "time left" estimate for a running job.
//
// The downloader only reports a percentage, so the estimate is built from how fast that percentage moves.
// Speed is sampled about once a second and blended with an exponentially weighted average whose weight
// depends on the real time between samples, so slow stretches and bursts do not make the estimate jump.
// Nothing is shown until there is enough data, and the estimate pauses when progress stalls or the job is paused.

const SAMPLE_MS = 1000;
const SMOOTHING_SECONDS = 20; // larger = steadier but slower to react
const MIN_SECONDS = 5;
const MIN_PERCENT = 0.3;
const STALL_SECONDS = 20;

export function formatClock(seconds) {
  const total = Math.max(0, Math.round(seconds));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  if (h) return `${h}h ${String(m).padStart(2, '0')}m`;
  if (m) return `${m}m ${String(s).padStart(2, '0')}s`;
  return `${s}s`;
}

function formatLeft(seconds) {
  if (seconds < 10) return 'a few seconds';
  if (seconds < 90) return `~${Math.round(seconds / 5) * 5}s`;
  if (seconds < 3600) return `~${Math.ceil(seconds / 60)} min`;
  const minutes = Math.ceil(seconds / 60);
  return `~${Math.floor(minutes / 60)}h ${minutes % 60}m`;
}

export function createJobTimer(onRender) {
  let timerId = 0;
  let active = false;
  let activeMs = 0;
  let lastTick = 0;
  let progress = 0;
  let sampleTime = 0;
  let sampleProgress = 0;
  let rate = 0; // percent per second
  let lastMoveTime = 0;

  const now = () => performance.now();

  function render(paused = false) {
    if (!active) { onRender({ active: false }); return; }
    const elapsed = activeMs / 1000;
    let left = 'estimating…';
    if (paused) left = 'paused';
    else if (progress >= 100) left = 'finishing up';
    else if (elapsed >= MIN_SECONDS && progress >= MIN_PERCENT && rate > 0) {
      left = (now() - lastMoveTime) / 1000 > STALL_SECONDS ? 'catching up…' : formatLeft((100 - progress) / rate);
    }
    onRender({ active: true, elapsed: formatClock(elapsed), left, paused });
  }

  function tick() {
    const t = now();
    if (lastTick) activeMs += t - lastTick;
    lastTick = t;
    render();
  }

  function start() {
    stop();
    active = true;
    activeMs = 0; progress = 0; rate = 0;
    sampleTime = lastTick = lastMoveTime = now();
    sampleProgress = 0;
    timerId = window.setInterval(tick, 1000);
    render();
  }

  function stop() {
    window.clearInterval(timerId);
    timerId = 0; active = false; lastTick = 0;
    render();
  }

  function pause() {
    if (!active) return;
    tick();
    window.clearInterval(timerId);
    timerId = 0; lastTick = 0;
    render(true);
  }

  function resume() {
    if (!active || timerId) return;
    sampleTime = lastTick = lastMoveTime = now();
    sampleProgress = progress;
    rate = 0;
    timerId = window.setInterval(tick, 1000);
    render();
  }

  // Call with each running update's percentage.
  function report(percent) {
    if (!active || !timerId) return;
    const value = Math.min(100, Math.max(0, Number(percent) || 0));
    const t = now();
    if (value < progress) {
      sampleTime = t;
      sampleProgress = value;
      rate = 0;
    }
    if (value > progress) lastMoveTime = t;
    progress = value;
    const dt = (t - sampleTime) / 1000;
    if (progress > sampleProgress && dt * 1000 >= SAMPLE_MS) {
      const instant = (progress - sampleProgress) / dt;
      if (instant > 0) {
        const weight = 1 - Math.exp(-dt / SMOOTHING_SECONDS);
        rate = rate > 0 ? rate + weight * (instant - rate) : instant;
      }
      sampleTime = t;
      sampleProgress = progress;
    }
    render();
  }

  return { start, stop, pause, resume, report };
}
