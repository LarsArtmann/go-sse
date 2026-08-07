package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-sse"
)

const (
	categoryAlert   = "alert"
	categorySuccess = "success"
	categoryInfo    = "info"
)

const (
	badgeAlert   = "ALERT"
	badgeSuccess = "OK"
	badgeInfo    = "INFO"
)

// activityItem is a single entry in the activity feed.
type activityItem struct {
	category string // "info", "alert", "success"
	badge    string // "INFO", "ALERT", "OK"
	message  string
	time     string
}

// randomN wraps math/rand/v2.IntN for demo data generation. Not crypto-safe;
// this is a demo server generating fake activity messages.
//
//nolint:gosec // G404: non-crypto random for demo data
func randomN(n int) int {
	return rand.IntN(n)
}

// msgTemplate pairs a display category with a random message generator.
type msgTemplate struct {
	category string
	badge    string
	gen      func() string
}

//nolint:gochecknoglobals,mnd // read-only template pool, intentional for example
var msgTemplates = []msgTemplate{
	{
		categoryAlert,
		badgeAlert,
		func() string { return fmt.Sprintf("CPU usage above 90%% on node-%d", randomN(nodeCount)+1) },
	},
	{
		categoryAlert,
		badgeAlert,
		func() string { return fmt.Sprintf("Disk space below 10%% on node-%d", randomN(nodeCount)+1) },
	},
	{
		categoryAlert,
		badgeAlert,
		func() string { return fmt.Sprintf("Memory leak detected in service-%d", randomN(serviceCount)+1) },
	},
	{
		categoryAlert,
		badgeAlert,
		func() string {
			return fmt.Sprintf(
				"Response time exceeding SLA on endpoint-%d",
				randomN(endpointCount)+1,
			)
		},
	},
	{
		categorySuccess,
		badgeSuccess,
		func() string { return fmt.Sprintf("Deploy #%d passed all checks", randomN(deployCount)+1) },
	},
	{
		categorySuccess,
		badgeSuccess,
		func() string {
			return fmt.Sprintf(
				"Migration v1.%d.%d applied successfully",
				randomN(versionMinor),
				randomN(versionPatch),
			)
		},
	},
	{
		categorySuccess,
		badgeSuccess,
		func() string {
			return fmt.Sprintf(
				"Build #%d completed in %ds",
				randomN(deployCount)+1,
				randomN(maxBuildSeconds)+1,
			)
		},
	},
	{
		categorySuccess,
		badgeSuccess,
		func() string { return fmt.Sprintf("Health check passed for service-%d", randomN(serviceCount)+1) },
	},
	{
		categoryInfo, badgeInfo,
		func() string { return fmt.Sprintf("User session-%d started", randomN(sessionCount)+1) },
	},
	{
		categoryInfo,
		badgeInfo,
		func() string { return fmt.Sprintf("Cache invalidated for region-%d", randomN(endpointCount)+1) },
	},
	{
		categoryInfo, badgeInfo,
		func() string { return fmt.Sprintf("Scheduled task-%d completed", randomN(taskCount)+1) },
	},
	{
		categoryInfo,
		badgeInfo,
		func() string { return fmt.Sprintf("Configuration reloaded for service-%d", randomN(serviceCount)+1) },
	},
}

func generateItem() activityItem {
	t := msgTemplates[randomN(len(msgTemplates))]

	return activityItem{
		category: t.category,
		badge:    t.badge,
		message:  t.gen(),
		time:     time.Now().Format("15:04:05"),
	}
}

// feedItemEvent builds a DataStar patch-elements SSE event that prepends
// a single feed item to the #feed div. The event carries a sequential ID
// so it can be replayed on reconnection.
//
// The "category" data line embeds the event category for server-side
// predicate filtering. DataStar ignores unknown keys in patch-elements
// payloads, so this line has no client-side effect.
func feedItemEvent(id int64, item activityItem) sse.Event {
	data := strings.Join([]string{
		"selector #feed",
		"mode prepend",
		"category " + item.category,
		sse.KeyedLines("elements", feedItemHTML(item)),
	}, "\n")

	return sse.Event{
		Event: "datastar-patch-elements",
		Data:  data,
		ID:    sse.NewEventID(strconv.FormatInt(id, 10)),
	}
}

// countEvent builds a DataStar patch-signals event that updates the
// subscriberCount signal. No event ID — this is an ephemeral meta event
// that should not be replayed.
func countEvent(count int) sse.Event {
	return sse.Event{
		Event: "datastar-patch-signals",
		Data:  sse.KeyedLines("signals", fmt.Sprintf(`{"subscriberCount":%d}`, count)),
	}
}

// replayEvent builds a DataStar patch-signals event that sets the replayed
// signal, triggering the "Replayed N missed events" banner.
func replayEvent(n int) sse.Event {
	return sse.Event{
		Event: "datastar-patch-signals",
		Data:  sse.KeyedLines("signals", fmt.Sprintf(`{"replayed":%d}`, n)),
	}
}

// totalEventSignal builds a DataStar patch-signals event that updates the
// totalEvents counter. Used for the empty-state check and the "N events sent"
// stat in the UI.
func totalEventSignal(n int64) sse.Event {
	return sse.Event{
		Event: "datastar-patch-signals",
		Data:  sse.KeyedLines("signals", fmt.Sprintf(`{"totalEvents":%d}`, n)),
	}
}

func feedItemHTML(item activityItem) string {
	return fmt.Sprintf(
		`<div class="feed-item feed-item--%s"><span class="feed-item__badge">%s</span><span class="feed-item__message">%s</span><span class="feed-item__time">%s</span></div>`,
		item.category,
		item.badge,
		item.message,
		item.time,
	)
}

// startProducer runs a background goroutine that emits a random activity
// event every emitInterval. Each event gets a monotonically increasing ID
// so reconnecting clients can replay missed events.
func (s *activityServer) startProducer(ctx context.Context) {
	var id int64

	emit := func() {
		id++

		item := generateItem()

		evt := feedItemEvent(id, item)

		s.store.Append(evt)
		s.broadcaster.BroadcastMany(evt, totalEventSignal(id))
	}

	emit() // emit one immediately so the user sees something fast

	ticker := time.NewTicker(emitInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			emit()
		}
	}
}
