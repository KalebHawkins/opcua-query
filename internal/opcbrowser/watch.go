package opcbrowser

import (
	"context"
	"fmt"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

type WatchSession struct {
	Result          Result
	SubscriptionID  uint32
	RevisedInterval time.Duration
	Targets         []Match
	MonitorFailures []string
}

type WatchEvent struct {
	Path            string
	NodeID          string
	Value           string
	Status          string
	SourceTimestamp time.Time
	ServerTimestamp time.Time
}

type WatchCallbacks struct {
	Ready func(WatchSession)
	Event func(WatchEvent)
}

type watchTarget struct {
	Handle uint32
	Match  Match
}

func Watch(ctx context.Context, req Request, callbacks WatchCallbacks) error {
	if callbacks.Ready == nil {
		callbacks.Ready = func(WatchSession) {}
	}
	if callbacks.Event == nil {
		callbacks.Event = func(WatchEvent) {}
	}

	endpoint := normalizeEndpoint(req.Endpoint)
	authType := ua.UserTokenTypeAnonymous
	if req.Username != "" {
		authType = ua.UserTokenTypeUserName
	}

	client, err := newClient(ctx, endpoint, req, authType)
	if err != nil {
		return err
	}
	defer closeClient(client)

	browseReq := req
	browseReq.ReadValues = false
	result, err := browseWithClient(ctx, client, browseReq)
	if err != nil {
		return err
	}

	targets, requests, err := buildWatchTargets(result.MatchedNodes)
	if err != nil {
		return err
	}

	notifyCh := make(chan *opcua.PublishNotificationData, len(targets)+1)
	sub, err := client.Subscribe(ctx, &opcua.SubscriptionParameters{Interval: subscriptionInterval(req.Interval)}, notifyCh)
	if err != nil {
		return fmt.Errorf("create subscription: %w", err)
	}
	defer cancelSubscription(sub)

	monitorResult, err := sub.Monitor(ctx, ua.TimestampsToReturnBoth, requests...)
	if err != nil {
		return fmt.Errorf("monitor nodes: %w", err)
	}

	activeTargets, failures := activeWatchTargets(targets, monitorResult.Results)
	if len(activeTargets) == 0 {
		return fmt.Errorf("monitor nodes: no variable nodes could be subscribed")
	}

	callbacks.Ready(WatchSession{
		Result:          result,
		SubscriptionID:  sub.SubscriptionID,
		RevisedInterval: sub.RevisedPublishingInterval,
		Targets:         matchesFromTargets(activeTargets),
		MonitorFailures: failures,
	})

	targetByHandle := make(map[uint32]watchTarget, len(activeTargets))
	for _, target := range activeTargets {
		targetByHandle[target.Handle] = target
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case notification := <-notifyCh:
			if notification == nil {
				continue
			}
			if notification.Error != nil {
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("subscription notification: %w", notification.Error)
			}

			switch value := notification.Value.(type) {
			case *ua.DataChangeNotification:
				for _, event := range extractWatchEvents(value, targetByHandle) {
					callbacks.Event(event)
				}
			case *ua.StatusChangeNotification:
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("subscription status changed: %s", value.Status)
			}
		}
	}
}

func buildWatchTargets(matches []Match) ([]watchTarget, []*ua.MonitoredItemCreateRequest, error) {
	variableMatches := make([]Match, 0, len(matches))
	for _, match := range matches {
		if match.NodeClass != ua.NodeClassVariable.String() {
			continue
		}
		variableMatches = append(variableMatches, match)
	}

	if len(variableMatches) == 0 {
		return nil, nil, fmt.Errorf("no variable nodes matched the supplied filter")
	}

	targets := make([]watchTarget, 0, len(variableMatches))
	requests := make([]*ua.MonitoredItemCreateRequest, 0, len(variableMatches))
	for index, match := range variableMatches {
		nodeID, err := ua.ParseNodeID(match.NodeID)
		if err != nil {
			return nil, nil, fmt.Errorf("parse node id %q: %w", match.NodeID, err)
		}

		handle := uint32(index + 1)
		targets = append(targets, watchTarget{Handle: handle, Match: match})
		requests = append(requests, opcua.NewMonitoredItemCreateRequestWithDefaults(nodeID, ua.AttributeIDValue, handle))
	}

	return targets, requests, nil
}

func activeWatchTargets(targets []watchTarget, results []*ua.MonitoredItemCreateResult) ([]watchTarget, []string) {
	active := make([]watchTarget, 0, len(targets))
	failures := make([]string, 0)

	for index, target := range targets {
		if index >= len(results) || results[index] == nil {
			failures = append(failures, fmt.Sprintf("%s: missing monitor result", target.Match.Path))
			continue
		}

		if results[index].StatusCode != ua.StatusOK {
			failures = append(failures, fmt.Sprintf("%s: %s", target.Match.Path, results[index].StatusCode))
			continue
		}

		active = append(active, target)
	}

	return active, failures
}

func extractWatchEvents(notification *ua.DataChangeNotification, targets map[uint32]watchTarget) []WatchEvent {
	if notification == nil {
		return nil
	}

	events := make([]WatchEvent, 0, len(notification.MonitoredItems))
	for _, item := range notification.MonitoredItems {
		if item == nil || item.Value == nil {
			continue
		}

		target, ok := targets[item.ClientHandle]
		if !ok {
			continue
		}

		event := WatchEvent{
			Path:            target.Match.Path,
			NodeID:          target.Match.NodeID,
			SourceTimestamp: item.Value.SourceTimestamp,
			ServerTimestamp: item.Value.ServerTimestamp,
		}
		if item.Value.Value != nil {
			event.Value = renderVariant(item.Value.Value)
		}
		if item.Value.Status != ua.StatusOK {
			event.Status = item.Value.Status.Error()
		}

		events = append(events, event)
	}

	return events
}

func matchesFromTargets(targets []watchTarget) []Match {
	matches := make([]Match, 0, len(targets))
	for _, target := range targets {
		matches = append(matches, target.Match)
	}
	return matches
}

func subscriptionInterval(interval time.Duration) time.Duration {
	if interval <= 0 {
		return opcua.DefaultSubscriptionInterval
	}
	return interval
}

func cancelSubscription(sub *opcua.Subscription) {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = sub.Cancel(cleanupCtx)
}
