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

  /**
   * History Modal Logic
   */
  let currentHistoryData = [];
  let currentPage = 1;
  let totalPages = 1;
  const ITEMS_PER_PAGE = 20;

  /**
   * Open history modal for a domain
   */
  function openHistoryModal(button) {
    const row = button.closest('tr');
    const historyJSON = row.dataset.history;

    // Get domain name from the domain cell's text content
    const domainCell = row.querySelector('.domain-cell');
    const domain = domainCell ? domainCell.textContent.trim() : 'Unknown';

    try {
      currentHistoryData = JSON.parse(historyJSON);
    } catch (e) {
      currentHistoryData = [];
    }

    // Reverse to show newest first (data is oldest first)
    currentHistoryData = currentHistoryData.reverse();

    // Calculate total pages
    totalPages = Math.max(1, Math.ceil(currentHistoryData.length / ITEMS_PER_PAGE));
    currentPage = 1;

    // Update modal title
    document.getElementById('modal-title').textContent = `IP History: ${domain}`;

    // Render first page
    renderHistoryPage();

    // Show modal
    const modal = document.getElementById('history-modal');
    modal.classList.add('active');
  }

  /**
   * Close history modal
   */
  function closeHistoryModal() {
    const modal = document.getElementById('history-modal');
    modal.classList.remove('active');
  }

  /**
   * Go to previous page
   */
  function previousPage() {
    if (currentPage > 1) {
      currentPage--;
      renderHistoryPage();
    }
  }

  /**
   * Go to next page
   */
  function nextPage() {
    if (currentPage < totalPages) {
      currentPage++;
      renderHistoryPage();
    }
  }

  /**
   * Render current page of history
   */
  function renderHistoryPage() {
    const container = document.getElementById('history-table-container');

    // Empty state
    if (currentHistoryData.length === 0) {
      container.innerHTML = `
        <div class="history-empty">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"></circle>
            <polyline points="12 6 12 12 16 14"></polyline>
          </svg>
          <p>No history available</p>
        </div>
      `;
      updatePaginationControls();
      return;
    }

    // Calculate page range
    const startIndex = (currentPage - 1) * ITEMS_PER_PAGE;
    const endIndex = Math.min(startIndex + ITEMS_PER_PAGE, currentHistoryData.length);
    const pageData = currentHistoryData.slice(startIndex, endIndex);

    // Build table HTML
    let tableHTML = `
      <table class="history-table">
        <thead>
          <tr>
            <th class="col-num">#</th>
            <th class="col-ip">IP Address</th>
            <th class="col-time">Changed At</th>
            <th class="col-duration">Duration</th>
          </tr>
        </thead>
        <tbody>
    `;

    const now = new Date();

    pageData.forEach((event, index) => {
      const absoluteIndex = startIndex + index + 1;
      const eventTime = new Date(event.time);
      const formattedTime = formatDateTime(eventTime);

      // Calculate duration (how long this IP was active)
      // Since we reversed the array, index 0 is newest (current)
      let duration = '—';
      if (index === 0 && startIndex === 0) {
        // This is the most recent IP (currently active)
        duration = '<span class="history-current-badge">Current</span>';
      } else {
        // Calculate duration from this event to the previous one (which is earlier in the reversed array)
        const prevEventIndex = startIndex + index - 1;
        if (prevEventIndex >= 0 && prevEventIndex < currentHistoryData.length) {
          const prevEvent = currentHistoryData[prevEventIndex];
          const prevTime = new Date(prevEvent.time);
          duration = formatDuration(prevTime - eventTime);
        }
      }

      tableHTML += `
        <tr>
          <td class="col-num">${absoluteIndex}</td>
          <td class="col-ip">${event.ip}</td>
          <td class="col-time">${formattedTime}</td>
          <td class="col-duration">${duration}</td>
        </tr>
      `;
    });

    tableHTML += `
        </tbody>
      </table>
    `;

    container.innerHTML = tableHTML;
    updatePaginationControls();
  }

  /**
   * Update pagination button states
   */
  function updatePaginationControls() {
    const prevBtn = document.getElementById('prev-page');
    const nextBtn = document.getElementById('next-page');
    const pageInfo = document.getElementById('page-info');

    prevBtn.disabled = currentPage <= 1;
    nextBtn.disabled = currentPage >= totalPages;
    pageInfo.textContent = `Page ${currentPage} of ${totalPages}`;
  }

  /**
   * Format date/time for display
   */
  function formatDateTime(date) {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    const seconds = String(date.getSeconds()).padStart(2, '0');
    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
  }

  /**
   * Format duration between two times in human-readable format
   */
  function formatDuration(milliseconds) {
    const seconds = Math.floor(milliseconds / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);
    const weeks = Math.floor(days / 7);
    const months = Math.floor(days / 30);
    const years = Math.floor(days / 365);

    if (years > 0) {
      const remainingMonths = Math.floor((days % 365) / 30);
      return remainingMonths > 0 ? `${years} year${years > 1 ? 's' : ''} ${remainingMonths} month${remainingMonths > 1 ? 's' : ''}` : `${years} year${years > 1 ? 's' : ''}`;
    }
    if (months > 0) {
      const remainingDays = days % 30;
      return remainingDays > 0 ? `${months} month${months > 1 ? 's' : ''} ${remainingDays} day${remainingDays > 1 ? 's' : ''}` : `${months} month${months > 1 ? 's' : ''}`;
    }
    if (weeks > 0) {
      const remainingDays = days % 7;
      return remainingDays > 0 ? `${weeks} week${weeks > 1 ? 's' : ''} ${remainingDays} day${remainingDays > 1 ? 's' : ''}` : `${weeks} week${weeks > 1 ? 's' : ''}`;
    }
    if (days > 0) {
      const remainingHours = hours % 24;
      return remainingHours > 0 ? `${days} day${days > 1 ? 's' : ''} ${remainingHours} hour${remainingHours > 1 ? 's' : ''}` : `${days} day${days > 1 ? 's' : ''}`;
    }
    if (hours > 0) {
      const remainingMinutes = minutes % 60;
      return remainingMinutes > 0 ? `${hours} hour${hours > 1 ? 's' : ''} ${remainingMinutes} minute${remainingMinutes > 1 ? 's' : ''}` : `${hours} hour${hours > 1 ? 's' : ''}`;
    }
    if (minutes > 0) {
      return `${minutes} minute${minutes > 1 ? 's' : ''}`;
    }
    return `${seconds} second${seconds !== 1 ? 's' : ''}`;
  }

  // Expose functions to global scope
  window.openHistoryModal = openHistoryModal;
  window.closeHistoryModal = closeHistoryModal;
  window.previousPage = previousPage;
  window.nextPage = nextPage;

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

    // Close modal when clicking outside
    const modal = document.getElementById('history-modal');
    if (modal) {
      modal.addEventListener('click', function(e) {
        if (e.target === modal) {
          closeHistoryModal();
        }
      });
    }

    // Close modal with ESC key
    document.addEventListener('keydown', function(e) {
      if (e.key === 'Escape') {
        const modal = document.getElementById('history-modal');
        if (modal && modal.classList.contains('active')) {
          closeHistoryModal();
        }
      }
    });
  });
})();
