/**
 * DDNS Updater - Main Application Logic
 * Handles refresh and update button functionality with toast notifications
 */
(function () {
  'use strict';

  /**
   * Handle refresh button click - reload the page
   */
  async function handleRefresh() {
    const btn = document.getElementById('refresh-btn');
    if (!btn || btn.disabled) return;

    // Show loading state
    btn.disabled = true;
    btn.classList.add('loading');
    const originalHTML = btn.innerHTML;
    btn.innerHTML = `
      <svg class="animate-spin" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M21 12a9 9 0 11-6.219-8.56"></path>
      </svg>
    `;

    // Small delay to show the loading state, then reload
    setTimeout(() => {
      window.location.reload();
    }, 300);
  }

  /**
   * Handle manual update button click - trigger DNS update
   */
  async function handleManualUpdate() {
    const btn = document.getElementById('update-btn');
    if (!btn || btn.disabled) return;

    // Show loading state
    btn.disabled = true;
    btn.classList.add('loading');
    const originalHTML = btn.innerHTML;
    btn.innerHTML = `
      <svg class="animate-spin" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M21 12a9 9 0 11-6.219-8.56"></path>
      </svg>
      <span class="btn-text">Updating...</span>
    `;

    try {
      const response = await fetch('/update', {
        method: 'GET',
      });

      const responseText = await response.text();

      if (response.ok) {
        // Success - show toast and refresh after delay
        showToast('DNS update initiated successfully. Refreshing...', 'success');

        // Refresh page after 2 seconds to show updated status
        setTimeout(() => {
          window.location.reload();
        }, 2000);
      } else {
        // Error - show error message
        showToast(`Update failed: ${responseText}`, 'error');

        // Reset button state
        btn.disabled = false;
        btn.classList.remove('loading');
        btn.innerHTML = originalHTML;
      }
    } catch (error) {
      // Network or other error
      showToast(`Update failed: ${error.message}`, 'error');

      // Reset button state
      btn.disabled = false;
      btn.classList.remove('loading');
      btn.innerHTML = originalHTML;
    }
  }

  /**
   * Show toast notification
   * @param {string} message - Message to display
   * @param {string} type - Toast type: 'success', 'error', or 'info'
   */
  function showToast(message, type = 'info') {
    // Create toast element
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;

    // Add to DOM
    document.body.appendChild(toast);

    // Trigger show animation after brief delay
    setTimeout(() => {
      toast.classList.add('show');
    }, 10);

    // Remove toast after 5 seconds
    setTimeout(() => {
      toast.classList.remove('show');
      // Remove from DOM after animation completes
      setTimeout(() => {
        toast.remove();
      }, 300);
    }, 5000);
  }

  /**
   * Get refresh icon SVG
   */
  function getRefreshIcon() {
    return `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <polyline points="23 4 23 10 17 10"></polyline>
      <polyline points="1 20 1 14 7 14"></polyline>
      <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path>
    </svg>`;
  }

  /**
   * Get play/update icon SVG
   */
  function getUpdateIcon() {
    return `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <polyline points="23 4 23 10 17 10"></polyline>
      <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
    </svg>`;
  }

  /**
   * Auto-refresh functionality with Page Visibility API
   */
  let autoRefreshTimer = null;
  let autoRefreshInterval = 0; // 0 = off, values in seconds
  let isPageVisible = !document.hidden;
  let countdownTimer = null;
  let secondsRemaining = 0;

  /**
   * Start auto-refresh timer
   */
  function startAutoRefresh() {
    stopAutoRefresh();
    if (autoRefreshInterval === 0 || !isPageVisible) return;

    secondsRemaining = autoRefreshInterval;

    autoRefreshTimer = setInterval(() => {
      if (isPageVisible) {
        window.location.reload();
      }
    }, autoRefreshInterval * 1000);

    // Start countdown
    countdownTimer = setInterval(() => {
      if (isPageVisible && secondsRemaining > 0) {
        secondsRemaining--;
        updateAutoRefreshIndicator();

        if (secondsRemaining === 0) {
          secondsRemaining = autoRefreshInterval;
        }
      }
    }, 1000);

    updateAutoRefreshIndicator();
  }

  /**
   * Stop auto-refresh timer
   */
  function stopAutoRefresh() {
    if (autoRefreshTimer) {
      clearInterval(autoRefreshTimer);
      autoRefreshTimer = null;
    }
    if (countdownTimer) {
      clearInterval(countdownTimer);
      countdownTimer = null;
    }
  }

  /**
   * Update auto-refresh visual indicator
   */
  function updateAutoRefreshIndicator() {
    const indicator = document.getElementById('auto-refresh-indicator');
    if (!indicator) return;

    if (autoRefreshInterval === 0) {
      indicator.style.display = 'none';
      return;
    }

    indicator.style.display = 'flex';

    // Format countdown time
    let countdownText;
    if (secondsRemaining >= 60) {
      const minutes = Math.floor(secondsRemaining / 60);
      const seconds = secondsRemaining % 60;
      countdownText = seconds > 0 ? `${minutes}:${seconds.toString().padStart(2, '0')}` : `${minutes}:00`;
    } else {
      countdownText = `${secondsRemaining}s`;
    }

    const statusText = isPageVisible ? countdownText : 'Paused';
    indicator.innerHTML = `
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="${isPageVisible ? 'animate-spin-slow' : ''}">
        <path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.2"></path>
      </svg>
      <span>${statusText}</span>
    `;
  }

  /**
   * Toggle auto-refresh menu
   */
  function toggleAutoRefreshMenu() {
    const menu = document.getElementById('auto-refresh-menu');
    if (!menu) return;

    const isVisible = menu.style.display === 'block';
    menu.style.display = isVisible ? 'none' : 'block';
  }

  /**
   * Set auto-refresh interval
   */
  function setAutoRefreshInterval(seconds) {
    autoRefreshInterval = seconds;
    localStorage.setItem('ddns-auto-refresh', seconds.toString());

    const menu = document.getElementById('auto-refresh-menu');
    if (menu) menu.style.display = 'none';

    if (seconds > 0) {
      startAutoRefresh();
    } else {
      stopAutoRefresh();
      updateAutoRefreshIndicator();
    }

    // Update button states
    updateAutoRefreshButtons();
  }

  /**
   * Update auto-refresh button states
   */
  function updateAutoRefreshButtons() {
    const buttons = document.querySelectorAll('[data-interval]');
    buttons.forEach(btn => {
      const interval = parseInt(btn.dataset.interval);
      if (interval === autoRefreshInterval) {
        btn.classList.add('active');
      } else {
        btn.classList.remove('active');
      }
    });
  }

  /**
   * Handle page visibility change
   */
  function handleVisibilityChange() {
    isPageVisible = !document.hidden;

    if (isPageVisible) {
      // Page became visible - restart auto-refresh if enabled
      if (autoRefreshInterval > 0) {
        // Reset countdown when page becomes visible again
        secondsRemaining = autoRefreshInterval;
        startAutoRefresh();
      }
    } else {
      // Page became hidden - stop auto-refresh to save resources
      stopAutoRefresh();
      updateAutoRefreshIndicator();
    }
  }

  // Expose functions to global scope for button onclick handlers
  window.handleRefresh = handleRefresh;
  window.handleManualUpdate = handleManualUpdate;
  window.toggleAutoRefreshMenu = toggleAutoRefreshMenu;
  window.setAutoRefreshInterval = setAutoRefreshInterval;

  /**
   * Handle status badge tooltips on mobile (touch devices)
   */
  function initStatusTooltips() {
    // Get all status badges with tooltips
    const badges = document.querySelectorAll('.badge.has-status-tooltip');

    badges.forEach(badge => {
      // Handle touch/click events for mobile
      badge.addEventListener('click', function(e) {
        e.preventDefault();
        e.stopPropagation();

        // Close any other open tooltips
        document.querySelectorAll('.badge.tooltip-active').forEach(b => {
          if (b !== badge) {
            b.classList.remove('tooltip-active');
          }
        });

        // Toggle this tooltip
        badge.classList.toggle('tooltip-active');
      });
    });

    // Close tooltip when clicking outside
    document.addEventListener('click', function(e) {
      if (!e.target.closest('.badge.has-status-tooltip')) {
        document.querySelectorAll('.badge.tooltip-active').forEach(badge => {
          badge.classList.remove('tooltip-active');
        });
      }
    });
  }

  // Initialize button icons and tooltips when DOM is ready
  document.addEventListener('DOMContentLoaded', function () {
    // Set refresh button icon if it exists
    const refreshBtn = document.getElementById('refresh-btn');
    if (refreshBtn && !refreshBtn.innerHTML.includes('svg')) {
      refreshBtn.innerHTML = getRefreshIcon();
    }

    // Set update button icon if it exists
    const updateBtn = document.getElementById('update-btn');
    if (updateBtn && !updateBtn.innerHTML.includes('svg')) {
      updateBtn.innerHTML = getUpdateIcon() + '<span class="btn-text">Update</span>';
    }

    // Initialize status tooltips for mobile support
    initStatusTooltips();

    // Initialize auto-refresh from localStorage
    const savedInterval = localStorage.getItem('ddns-auto-refresh');
    if (savedInterval) {
      autoRefreshInterval = parseInt(savedInterval);
      if (autoRefreshInterval > 0) {
        startAutoRefresh();
      }
      updateAutoRefreshButtons();
    }

    // Close auto-refresh menu when clicking outside
    document.addEventListener('click', function(e) {
      const menu = document.getElementById('auto-refresh-menu');
      const btn = document.getElementById('auto-refresh-btn');
      if (menu && btn && !menu.contains(e.target) && !btn.contains(e.target)) {
        menu.style.display = 'none';
      }
    });

    // Listen for page visibility changes
    document.addEventListener('visibilitychange', handleVisibilityChange);
  });
})();
