package opcbrowser

import (
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
)

func TestBuildWatchTargets(t *testing.T) {
	t.Parallel()

	matches := []Match{
		{Path: "/Area/Counter", NodeID: "ns=2;s=Counter", NodeClass: ua.NodeClassVariable.String()},
		{Path: "/Area/Folder", NodeID: "ns=2;s=Folder", NodeClass: ua.NodeClassObject.String()},
	}

	targets, requests, err := buildWatchTargets(matches)
	if err != nil {
		t.Fatalf("buildWatchTargets() error = %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("buildWatchTargets() targets = %d, want 1", len(targets))
	}
	if len(requests) != 1 {
		t.Fatalf("buildWatchTargets() requests = %d, want 1", len(requests))
	}
	if got := requests[0].RequestedParameters.ClientHandle; got != 1 {
		t.Fatalf("client handle = %d, want 1", got)
	}
}

func TestActiveWatchTargets(t *testing.T) {
	t.Parallel()

	targets := []watchTarget{
		{Handle: 1, Match: Match{Path: "/Area/A"}},
		{Handle: 2, Match: Match{Path: "/Area/B"}},
	}

	active, failures := activeWatchTargets(targets, []*ua.MonitoredItemCreateResult{
		{StatusCode: ua.StatusOK},
		{StatusCode: ua.StatusBadNodeIDUnknown},
	})

	if len(active) != 1 {
		t.Fatalf("activeWatchTargets() active = %d, want 1", len(active))
	}
	if len(failures) != 1 {
		t.Fatalf("activeWatchTargets() failures = %d, want 1", len(failures))
	}
}

func TestExtractWatchEvents(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 20, 14, 30, 0, 0, time.UTC)
	notification := &ua.DataChangeNotification{
		MonitoredItems: []*ua.MonitoredItemNotification{
			{
				ClientHandle: 7,
				Value: &ua.DataValue{
					Value:           ua.MustVariant(int32(42)),
					Status:          ua.StatusOK,
					SourceTimestamp: now,
				},
			},
		},
	}

	events := extractWatchEvents(notification, map[uint32]watchTarget{
		7: {Handle: 7, Match: Match{Path: "/Area/Counter", NodeID: "ns=2;s=Counter"}},
	})

	if len(events) != 1 {
		t.Fatalf("extractWatchEvents() events = %d, want 1", len(events))
	}
	if events[0].Value != "42" {
		t.Fatalf("extractWatchEvents() value = %q, want %q", events[0].Value, "42")
	}
	if events[0].Path != "/Area/Counter" {
		t.Fatalf("extractWatchEvents() path = %q, want %q", events[0].Path, "/Area/Counter")
	}
	if !events[0].SourceTimestamp.Equal(now) {
		t.Fatalf("extractWatchEvents() source timestamp = %v, want %v", events[0].SourceTimestamp, now)
	}
}
