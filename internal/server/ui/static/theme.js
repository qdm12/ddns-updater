/**
 * Theme Toggle - Dark/Light Mode
 * Manages theme switching with localStorage persistence
 */
(function () {
  'use strict';

  const STORAGE_KEY = 'ddns-theme';
  const THEME_ATTR = 'data-theme';

  /**
   * Get initial theme from localStorage or system preference
   */
  function getInitialTheme() {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      return stored;
    }

    // Check system preference
    return window.matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : 'light';
  }

  /**
   * Apply theme to document and update UI
   */
  function applyTheme(theme) {
    document.documentElement.setAttribute(THEME_ATTR, theme);
    localStorage.setItem(STORAGE_KEY, theme);

    // Update toggle button icon and aria-label
    const toggle = document.getElementById('theme-toggle');
    if (toggle) {
      const nextTheme = theme === 'dark' ? 'light' : 'dark';
      toggle.setAttribute('aria-label', `Switch to ${nextTheme} mode`);
      toggle.setAttribute('aria-pressed', String(theme === 'dark'));
      toggle.innerHTML = theme === 'dark' ? getSunIcon() : getMoonIcon();
    }
  }

  /**
   * Toggle between dark and light theme
   */
  function toggleTheme() {
    const current = document.documentElement.getAttribute(THEME_ATTR);
    const next = current === 'dark' ? 'light' : 'dark';
    applyTheme(next);
  }

  /**
   * Moon icon for light mode (click to enable dark mode)
   */
  function getMoonIcon() {
    return `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
    </svg>`;
  }

  /**
   * Sun icon for dark mode (click to enable light mode)
   */
  function getSunIcon() {
    return `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <circle cx="12" cy="12" r="5"></circle>
      <line x1="12" y1="1" x2="12" y2="3"></line>
      <line x1="12" y1="21" x2="12" y2="23"></line>
      <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
      <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
      <line x1="1" y1="12" x2="3" y2="12"></line>
      <line x1="21" y1="12" x2="23" y2="12"></line>
      <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
      <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
    </svg>`;
  }

  // Initialize theme immediately (before DOM load to prevent flash)
  const initialTheme = getInitialTheme();
  applyTheme(initialTheme);

  // Expose toggle function to global scope for button onclick
  window.toggleTheme = toggleTheme;

  // Update button icon when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function () {
      applyTheme(initialTheme);
    });
  }
})();
