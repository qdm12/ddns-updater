# IP Change History Modal - Design Document

**Date:** 2026-01-06
**Status:** Approved

## Overview

Add an IP change history viewer that displays all historical IP changes for a domain in a modal popup. Users can view comprehensive information about when IPs changed, how long each IP was active, and navigate through paginated history.

## User Requirements

- View complete IP change history for each domain
- Display in a modal popup (small page)
- Show all available information from history data
- Easy to access from the main table

## Architecture & Data Flow

### Data Embedding Strategy

When the Go template renders `index.html`, each table row includes complete history data as a JSON data attribute:

```html
<tr data-history='[{"ip":"1.2.3.4","time":"2026-01-06T10:30:00Z"},...]'>
```

### User Interaction Flow

1. User clicks the history icon button in a domain row
2. JavaScript reads the `data-history` attribute from that row
3. Parses the JSON to get all HistoryEvent entries (IP + Time)
4. Calculates duration between consecutive changes
5. Creates paginated table HTML (20 entries per page)
6. Shows modal overlay with the history table
7. User can navigate pages with prev/next buttons
8. User closes modal by clicking close button or outside the modal

### No Network Calls Required

All data is already embedded in the page from the initial Go template render. This keeps the feature fast and simple, following the existing architecture pattern where `/` returns fully rendered HTML with embedded data.

## Modal Structure

### Visual Layout

- Semi-transparent dark backdrop (blocks interaction with main page)
- Centered white card with rounded corners
- Header: Domain name + close button
- Body: Paginated history table
- Footer: Page navigation (« Previous | Page X of Y | Next »)

### HTML Structure

```html
<div id="history-modal" class="modal-overlay">
  <div class="modal-content">
    <div class="modal-header">
      <h2>IP History: example.com</h2>
      <button class="modal-close" aria-label="Close">×</button>
    </div>
    <div class="modal-body">
      <table class="history-table">
        <!-- History entries here -->
      </table>
    </div>
    <div class="modal-footer">
      <button class="page-nav-btn" id="prev-page">« Previous</button>
      <span class="page-info">Page 1 of 5</span>
      <button class="page-nav-btn" id="next-page">Next »</button>
    </div>
  </div>
</div>
```

## History Table Design

### Table Columns

| Column | Description | Example |
|--------|-------------|---------|
| # | Sequential number (newest = 1) | 1, 2, 3... |
| IP Address | The IP from HistoryEvent.IP | 192.168.1.1 |
| Changed At | Formatted timestamp | 2026-01-06 15:30:45 |
| Time Ago | Human-readable duration since change | 2h ago, 3d ago |
| Duration | How long this IP was active | 2h 15m, 3d 4h, — (current) |

### Data Calculations

**Time Ago:**
- Use similar logic to `GetDurationSinceSuccess()`
- Calculate from event time to now
- Format: "2h ago", "3d ago", "45s ago"

**Duration:**
- Calculate difference between consecutive HistoryEvent timestamps
- For current IP (newest): show "—" or "Current"
- For previous IPs: `nextEvent.Time - currentEvent.Time`
- Format: "2h 15m", "3d", "45s"

### Display Order

History array is already antichronological (newest first):
- Page 1: entries 1-20 (most recent changes)
- Page 2: entries 21-40
- etc.

### Table Styling

- Use existing CSS classes from `styles.css` (same table styling as main page)
- Zebra striping for readability
- Monospace font for IP addresses
- Responsive: stack columns on mobile if needed

## Modal UI Design

### Visual Design

- **Overlay**: Semi-transparent black background (`rgba(0,0,0,0.5)`)
- **Modal card**: Max-width 800px, white background (adapts to dark theme)
- **Shadow**: Elevated shadow to appear above main content
- **Animation**: Fade-in overlay + scale-up card (0.2s ease-out)
- **Responsive**: Full-screen on mobile with padding

### Interaction Behavior

- Click history icon → Modal appears with fade-in
- Click close button (×) → Modal disappears
- Click outside modal (on overlay) → Modal closes
- ESC key → Modal closes
- Pagination buttons disabled when on first/last page

### History Icon Button

Add a small clock/history icon to each row in the main table, positioned as a new dedicated column or integrated into an existing column.

## Implementation Plan

### Files to Modify

1. **`/internal/server/ui/index.html`** (Go template)
   - Add history icon button to each table row
   - Embed history data as JSON in `data-history` attribute on each row
   - Add modal HTML structure (hidden by default)

2. **`/internal/server/ui/static/app.js`** (JavaScript)
   - Add `openHistoryModal(rowElement)` function
   - Add `renderHistoryTable(historyData, page)` function
   - Add `formatDuration(seconds)` helper
   - Add `formatTimeAgo(timestamp)` helper
   - Add pagination logic
   - Add modal close handlers (close button, overlay click, ESC key)

3. **`/internal/server/ui/static/styles.css`** (Styling)
   - Add `.modal-overlay` styles
   - Add `.modal-content` styles
   - Add `.modal-header`, `.modal-body`, `.modal-footer` styles
   - Add `.history-table` styles
   - Add animation keyframes for fade-in
   - Add dark theme support for modal

### No Backend Changes Required

The history data is already available in the records (from `internal/models/history.go`), so no new endpoints or database queries are needed.

### Implementation Steps

1. Add CSS for modal styling
2. Add modal HTML structure to template
3. Add history icon to table rows with embedded data
4. Add JavaScript for modal logic and pagination
5. Test with domains that have varying amounts of history (0, 1, 5, 50+ changes)

## Testing Scenarios

- Domain with no history (0 changes)
- Domain with single IP (1 entry)
- Domain with few changes (2-5 entries)
- Domain with many changes (50+ entries requiring pagination)
- Mobile responsiveness
- Dark theme compatibility
- Keyboard navigation (ESC to close)
- Accessibility (ARIA labels, focus management)

## Design Decisions

### Why embed data instead of API endpoint?
- Follows existing architecture pattern (server-rendered templates)
- No additional server roundtrips
- Simpler implementation
- Data is already in memory when rendering the page

### Why 20 entries per page?
- Balance between showing enough data and keeping modal scrollable
- Matches common pagination patterns
- Can be adjusted if needed

### Why modal instead of inline expansion?
- Better for showing comprehensive information
- Doesn't disrupt the main table layout
- Easier to make responsive on mobile
- User explicitly requested "small page" / "pop up"

## Future Enhancements (Out of Scope)

- Export history to CSV/JSON
- Search/filter by date range
- Chart visualization of IP changes over time
- Comparison between multiple domains
