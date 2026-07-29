/* FolioPath — Theme Toggle Script */
(function () {
  'use strict';

  const STORAGE_KEY = 'foliopath-theme';

  function getPreferred() {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === 'light' || stored === 'dark') return stored;
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }

  function apply(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem(STORAGE_KEY, theme);
  }

  // Apply immediately to prevent flash
  apply(getPreferred());

  // Expose toggle function globally
  window.toggleTheme = function () {
    const current = document.documentElement.getAttribute('data-theme');
    apply(current === 'dark' ? 'light' : 'dark');
  };

  // Listen for system changes
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function (e) {
    if (!localStorage.getItem(STORAGE_KEY)) {
      apply(e.matches ? 'dark' : 'light');
    }
  });
})();
