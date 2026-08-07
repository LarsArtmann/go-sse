# Browser Verification Checklist

Manual verification of the DataStar activity feed example. Complete in under 3 minutes.

## Prerequisites

```bash
go run ./example/datastar/
# open http://localhost:8765
```

## Checklist

### 1. Basic event delivery
- [ ] Feed items appear immediately (first event within 1 second)
- [ ] New items appear every ~2 seconds
- [ ] Items prepend to the top of the feed (newest first)
- [ ] Each item shows a badge (ALERT/OK/INFO), message, and timestamp
- [ ] Total event count increments with each new item

### 2. Fan-out (open multiple tabs)
- [ ] Open a second tab on `http://localhost:8765`
- [ ] Both tabs show the same events arriving at the same time
- [ ] The "clients connected" counter shows 2 (or more)

### 3. Filtering
- [ ] Click "Alerts only" in one tab (or press `a`)
- [ ] Only ALERT events appear in that tab
- [ ] Other tabs (All events) continue showing all categories
- [ ] Click "All events" (or press `e`) to return to the unfiltered view

### 4. Reconnection replay
- [ ] Note the last event ID you see
- [ ] Close a tab, wait 6+ seconds (3+ missed events)
- [ ] Reopen the tab
- [ ] "Replayed N missed events" banner appears and fades out
- [ ] The missed events appear at the top of the feed

### 5. Heartbeat / idle survival
- [ ] Leave a tab open and idle for 60+ seconds
- [ ] Connection stays alive (no disconnect, events keep flowing)

### 6. Subscriber count updates
- [ ] Open a new tab — counter increments within ~1 second
- [ ] Close a tab — counter decrements within ~1 second

### 7. Graceful shutdown
- [ ] With tabs open, press `Ctrl+C` in the terminal
- [ ] "Shutting down..." log appears
- [ ] Server exits cleanly within ~5 seconds

### 8. Theme toggle
- [ ] Click the Dark/Light button in the header
- [ ] Theme switches between dark and light
- [ ] Refresh the page — theme persists (localStorage)

### 9. Pause/resume
- [ ] Click "Pause" — feed hides, "Paused" banner appears
- [ ] Events continue arriving server-side (subscriber count unchanged)
- [ ] Click "Resume" — feed reappears with buffered events

### 10. Scroll-to-top
- [ ] Wait until the feed overflows (10+ items, scroll bar appears)
- [ ] New items auto-scroll to show the newest at the top
- [ ] Scroll down manually — auto-scroll stops (respects user position)
- [ ] Scroll back to top — auto-scroll resumes
