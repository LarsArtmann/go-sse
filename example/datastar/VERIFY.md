# Browser Verification Checklist

Manual verification of the DataStar activity feed example. Complete in under 2 minutes.

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

### 2. Fan-out (open multiple tabs)
- [ ] Open a second tab on `http://localhost:8765`
- [ ] Both tabs show the same events arriving at the same time
- [ ] The "clients connected" counter shows 2 (or more)

### 3. Filtering
- [ ] Click "Alerts only" in one tab
- [ ] Only ALERT events appear in that tab
- [ ] Other tabs (All events) continue showing all categories
- [ ] Click "All events" to return to the unfiltered view

### 4. Reconnection replay
- [ ] Note the last event ID you see
- [ ] Close a tab, wait 6+ seconds (3+ missed events)
- [ ] Reopen the tab
- [ ] "Replayed N missed events" banner appears briefly
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
