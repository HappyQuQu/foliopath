/* FolioPath — Theme Toggle Script */
(function () {
  'use strict';

  const STORAGE_KEY = 'foliopath-theme';

  function getPreference() {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === 'light' || stored === 'dark' || stored === 'system') return stored;
    return 'system';
  }

  function resolvePreference(preference) {
    if (preference === 'light' || preference === 'dark') return preference;
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }

  function applyResolved(theme) {
    document.documentElement.setAttribute('data-theme', theme);
  }

  // Apply immediately to prevent flash
  applyResolved(resolvePreference(getPreference()));

  window.setThemePreference = function (preference) {
    const valid = preference === 'light' || preference === 'dark' || preference === 'system';
    const next = valid ? preference : 'system';
    localStorage.setItem(STORAGE_KEY, next);
    applyResolved(resolvePreference(next));
  };

  window.previewThemePreference = function (preference) {
    applyResolved(resolvePreference(preference));
  };

  // Expose toggle function globally
  window.toggleTheme = function () {
    const current = document.documentElement.getAttribute('data-theme');
    window.setThemePreference(current === 'dark' ? 'light' : 'dark');
  };

  function closeAccountMenu(options) {
    const menu = document.querySelector('[data-account-menu]');
    const trigger = document.querySelector('[data-account-trigger]');
    if (!menu || !trigger) return;

    menu.hidden = true;
    trigger.setAttribute('aria-expanded', 'false');
    if (options && options.restoreFocus) trigger.focus();
  }

  window.toggleAccountMenu = function () {
    const menu = document.querySelector('[data-account-menu]');
    const trigger = document.querySelector('[data-account-trigger]');
    if (!menu || !trigger) return;

    const shouldOpen = menu.hidden;
    menu.hidden = !shouldOpen;
    trigger.setAttribute('aria-expanded', String(shouldOpen));
  };

  const GLOBAL_HEADER_HTML = `
    <a href="03-browse.html" class="brand" aria-label="FolioPath 首页">
      <img class="brand-mark" src="assets/foliopath-mark-tree.svg" alt="" aria-hidden="true">
      <span class="brand-wordmark">FolioPath</span>
    </a>
    <form class="global-search" action="04-search.html" role="search" data-global-search>
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
      <input type="search" name="q" placeholder="搜索全部照片和视频" aria-label="全局搜索" data-global-search-input>
      <button class="global-search-submit" type="submit">搜索</button>
    </form>
    <div class="global-header-right">
      <button class="admin-badge" type="button" aria-label="打开管理员菜单" aria-haspopup="menu" aria-expanded="false" onclick="toggleAccountMenu()" data-account-trigger>
        <span class="admin-avatar" aria-hidden="true">管</span>
        <span class="admin-label">管理员</span>
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><polyline points="6 9 12 15 18 9"/></svg>
      </button>
      <div class="account-menu" role="menu" hidden data-account-menu>
        <div class="account-menu-header"><strong>管理员</strong><span>admin</span></div>
        <a href="06-settings-libraries.html" role="menuitem">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-2 2 2 2 0 01-2-2v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 01-2-2 2 2 0 012-2h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 010-2.83 2 2 0 012.83 0l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 012-2 2 2 0 012 2v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 0 2 2 0 010 2.83l-.06.06a1.65 1.65 0 00-.33 1.82V9a1.65 1.65 0 001.51 1H21a2 2 0 012 2 2 2 0 01-2 2h-.09a1.65 1.65 0 00-1.51 1z"/></svg>
          管理中心
        </a>
        <a href="01-auth.html" class="danger" role="menuitem">退出登录</a>
      </div>
    </div>`;

  document.addEventListener('click', function (event) {
    const menu = document.querySelector('[data-account-menu]');
    const trigger = document.querySelector('[data-account-trigger]');
    if (!menu || menu.hidden || !trigger) return;
    if (menu.contains(event.target) || trigger.contains(event.target)) return;
    closeAccountMenu();
  });

  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape') closeAccountMenu({ restoreFocus: true });
  });

  document.addEventListener('DOMContentLoaded', function () {
    document.querySelectorAll('[data-global-header]').forEach(function (header) {
      header.innerHTML = GLOBAL_HEADER_HTML;
    });

    const params = new URLSearchParams(window.location.search);
    const query = params.get('q');
    const search = document.querySelector('[data-global-search-input]');
    if (query && search) search.value = query;

    function updateSettingsNavigation() {
      const items = document.querySelectorAll('.settings-nav-item');
      if (!items.length || !document.querySelector('[data-settings-default]')) return;

      const hash = window.location.hash;
      items.forEach(function (item) {
        const target = item.getAttribute('href');
        const isDefault = item.hasAttribute('data-settings-default');
        const active = (hash === '#storage' && target === '#storage') ||
          (hash === '#account' && target === '#account') ||
          ((!hash || hash === '#appearance') && isDefault);
        item.classList.toggle('active', active);
        if (active) item.setAttribute('aria-current', 'page');
        else item.removeAttribute('aria-current');
      });
    }

    updateSettingsNavigation();
    window.addEventListener('hashchange', updateSettingsNavigation);
  });

  // Listen for system changes
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function (e) {
    if (getPreference() === 'system') {
      applyResolved(e.matches ? 'dark' : 'light');
    }
  });
})();
