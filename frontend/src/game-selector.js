import { SearchSteamGames } from '../wailsjs/go/main/App';
import './game-selector.css';

export function createGameSelector(root, appInput) {
  root.innerHTML = `<div class="game-mode" role="group" aria-label="Choose how to enter the game"><button type="button" data-mode="search" aria-pressed="true">Search game</button><button type="button" data-mode="manual" aria-pressed="false">Enter app ID</button></div>
    <div class="game-search"><label for="game-query" class="game-search__label">Steam game</label><input id="game-query" type="text" role="combobox" aria-autocomplete="list" aria-expanded="false" aria-controls="game-results" aria-describedby="game-search-status" autocomplete="off" maxlength="200" placeholder="Type a game name"><div class="game-popup" hidden><div id="game-results" role="listbox" aria-label="Steam search results"></div></div><p id="game-search-status" role="status">Type at least 1 character. Up to 10 results.</p></div>`;
  const input = root.querySelector('#game-query');
  const search = root.querySelector('.game-search');
  const popup = root.querySelector('.game-popup');
  const list = root.querySelector('#game-results');
  const status = root.querySelector('#game-search-status');
  const modes = [...root.querySelectorAll('[data-mode]')];
  let games = [], active = -1, version = 0, timer, disabled = false;

  function close() {
    popup.hidden = true;
    input.setAttribute('aria-expanded', 'false');
    input.removeAttribute('aria-activedescendant');
    active = -1;
  }
  function open() {
    if (disabled || !games.length || document.activeElement !== input) return;
    popup.hidden = false;
    input.setAttribute('aria-expanded', 'true');
  }
  function highlight(index) {
    active = index;
    [...list.children].forEach((option, i) => option.setAttribute('aria-selected', String(i === active)));
    if (active >= 0) {
      input.setAttribute('aria-activedescendant', list.children[active].id);
      list.children[active].scrollIntoView({ block: 'nearest' });
    }
  }
  function select(index) {
    if (disabled || !games[index]) return;
    const game = games[index];
    version++;
    clearTimeout(timer);
    input.value = game.name;
    appInput.value = String(game.id);
    appInput.dispatchEvent(new Event('input', { bubbles: true }));
    status.textContent = `Selected ${game.name}. App ID ${game.id}.`;
    close();
  }
  async function runSearch(term, requestVersion) {
    try {
      const results = await SearchSteamGames(term);
      if (requestVersion !== version || disabled) return;
      games = results.slice(0, 10);
      list.replaceChildren();
      games.forEach((game, index) => {
        const option = document.createElement('div');
        option.id = `game-option-${index}`;
        option.className = 'game-option';
        option.setAttribute('role', 'option');
        option.setAttribute('aria-selected', 'false');
        const title = document.createElement('span');
        title.textContent = game.name;
        const id = document.createElement('small');
        id.textContent = `App ${game.id}`;
        option.append(title, id);
        option.addEventListener('pointerdown', event => event.preventDefault());
        option.addEventListener('click', () => select(index));
        list.append(option);
      });
      status.textContent = games.length ? `${games.length} results. Select a title to fill its app ID.` : 'No results. Try another name or enter the app ID.';
      open();
    } catch (error) {
      if (requestVersion !== version || disabled) return;
      status.textContent = typeof error === 'string' ? error : error?.message || 'Search unavailable. Enter the app ID manually.';
      close();
    }
  }
  input.addEventListener('input', () => {
    const requestVersion = ++version;
    clearTimeout(timer);
    games = [];
    list.replaceChildren();
    close();
    // A changed query must not keep the previously selected game's ID.
    appInput.value = '';
    appInput.dispatchEvent(new Event('input', { bubbles: true }));
    const term = input.value.trim();
    status.textContent = term ? 'Searching Steam...' : 'Type at least 1 character. Up to 10 results.';
    if (term) timer = setTimeout(() => runSearch(term, requestVersion), 300);
  });
  input.addEventListener('focus', open);
  input.addEventListener('blur', close);
  input.addEventListener('keydown', event => {
    if (event.key === 'Escape') { close(); return; }
    if (['ArrowDown', 'ArrowUp'].includes(event.key) && games.length) {
      event.preventDefault();
      open();
      highlight(event.key === 'ArrowDown' ? (active + 1) % games.length : (active <= 0 ? games.length : active) - 1);
    } else if (event.key === 'Enter' && !popup.hidden && active >= 0) {
      event.preventDefault();
      select(active);
    }
  });
  modes.forEach(button => button.addEventListener('click', () => {
    version++;
    clearTimeout(timer);
    close();
    const manual = button.dataset.mode === 'manual';
    modes.forEach(mode => mode.setAttribute('aria-pressed', String(mode === button)));
    search.hidden = manual;
    appInput.readOnly = !manual;
    if (manual) { appInput.focus(); status.textContent = 'Type at least 1 character. Up to 10 results.'; }
    else { input.focus(); if (!games.length && input.value.trim()) input.dispatchEvent(new Event('input')); }
  }));
  appInput.readOnly = true;
  appInput.addEventListener('input', () => {
    if (appInput.readOnly) return;
    input.value = '';
    games = [];
    list.replaceChildren();
    status.textContent = 'Type at least 1 character. Up to 10 results.';
  });
  return {
    setDisabled(value) {
      disabled = value;
      input.disabled = value;
      modes.forEach(button => { button.disabled = value; });
      if (value) { version++; clearTimeout(timer); close(); }
    }
  };
}
