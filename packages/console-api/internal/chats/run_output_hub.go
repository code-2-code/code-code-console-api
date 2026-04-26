package chats

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	runOutputHubBufferSize       = 1024
	runOutputSubscriberBuffer    = 256
	runOutputTerminalReplayDelay = time.Minute
)

type runOutputHub struct {
	source  runOutputStreamService
	mu      sync.Mutex
	entries map[string]*runOutputHubEntry
}

type runOutputHubEntry struct {
	hub         *runOutputHub
	runID       string
	cancel      context.CancelFunc
	mu          sync.Mutex
	buffer      []runOutputEvent
	subscribers map[*runOutputSubscriber]struct{}
	terminal    bool
	err         error
}

type runOutputSubscriber struct {
	updates chan runOutputDelivery
}

type runOutputDelivery struct {
	event runOutputEvent
	err   error
	done  bool
}

func newRunOutputHub(source runOutputStreamService) runOutputStreamService {
	if source == nil {
		return nil
	}
	return &runOutputHub{source: source, entries: map[string]*runOutputHubEntry{}}
}

func (h *runOutputHub) Stream(ctx context.Context, runID string, afterSequence uint64, yield func(runOutputEvent) error) error {
	if h == nil || h.source == nil {
		return fmt.Errorf("consoleapi/chats: run output source is not configured")
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return fmt.Errorf("consoleapi/chats: run id is required")
	}
	if yield == nil {
		return fmt.Errorf("consoleapi/chats: yield is nil")
	}
	entry := h.entry(runID)
	subscriber, replay, terminal, err := entry.subscribe(afterSequence)
	for _, event := range replay {
		if err := yield(event); err != nil {
			if subscriber != nil {
				entry.unsubscribe(subscriber)
			}
			return err
		}
	}
	if terminal || err != nil {
		return err
	}
	defer entry.unsubscribe(subscriber)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update, ok := <-subscriber.updates:
			if !ok || update.done {
				return update.err
			}
			if update.err != nil {
				return update.err
			}
			if err := yield(update.event); err != nil {
				return err
			}
		}
	}
}

func (h *runOutputHub) entry(runID string) *runOutputHubEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	if entry := h.entries[runID]; entry != nil {
		return entry
	}
	ctx, cancel := context.WithCancel(context.Background())
	entry := &runOutputHubEntry{
		hub:         h,
		runID:       runID,
		cancel:      cancel,
		subscribers: map[*runOutputSubscriber]struct{}{},
	}
	h.entries[runID] = entry
	go entry.run(ctx)
	return entry
}

func (h *runOutputHub) remove(runID string, entry *runOutputHubEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.entries[runID] == entry {
		delete(h.entries, runID)
		entry.cancel()
	}
}

func (e *runOutputHubEntry) run(ctx context.Context) {
	err := e.hub.source.Stream(ctx, e.runID, 0, func(event runOutputEvent) error {
		e.publish(event)
		return nil
	})
	e.finish(err)
}

func (e *runOutputHubEntry) subscribe(afterSequence uint64) (*runOutputSubscriber, []runOutputEvent, bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	replay := make([]runOutputEvent, 0, len(e.buffer))
	for _, event := range e.buffer {
		if sequence := runOutputSequence(event); sequence != 0 && sequence <= afterSequence {
			continue
		}
		replay = append(replay, event)
	}
	if e.terminal || e.err != nil {
		return nil, replay, e.terminal, e.err
	}
	subscriber := &runOutputSubscriber{updates: make(chan runOutputDelivery, runOutputSubscriberBuffer)}
	e.subscribers[subscriber] = struct{}{}
	return subscriber, replay, false, nil
}

func (e *runOutputHubEntry) unsubscribe(subscriber *runOutputSubscriber) {
	if subscriber == nil {
		return
	}
	e.mu.Lock()
	delete(e.subscribers, subscriber)
	e.mu.Unlock()
}

func (e *runOutputHubEntry) publish(event runOutputEvent) {
	e.mu.Lock()
	e.buffer = append(e.buffer, event)
	if len(e.buffer) > runOutputHubBufferSize {
		e.buffer = append([]runOutputEvent(nil), e.buffer[len(e.buffer)-runOutputHubBufferSize:]...)
	}
	if runOutputTerminal(event) {
		e.terminal = true
	}
	subscribers := e.subscriberList()
	e.mu.Unlock()

	for _, subscriber := range subscribers {
		subscriber.updates <- runOutputDelivery{event: event}
	}
}

func (e *runOutputHubEntry) finish(err error) {
	e.mu.Lock()
	if err != nil && !e.terminal {
		e.err = err
	}
	terminal := e.terminal
	subscribers := e.subscriberList()
	e.subscribers = map[*runOutputSubscriber]struct{}{}
	e.mu.Unlock()

	for _, subscriber := range subscribers {
		subscriber.updates <- runOutputDelivery{err: err, done: true}
		close(subscriber.updates)
	}
	if terminal {
		time.AfterFunc(runOutputTerminalReplayDelay, func() {
			e.hub.remove(e.runID, e)
		})
		return
	}
	e.hub.remove(e.runID, e)
}

func (e *runOutputHubEntry) subscriberList() []*runOutputSubscriber {
	out := make([]*runOutputSubscriber, 0, len(e.subscribers))
	for subscriber := range e.subscribers {
		out = append(out, subscriber)
	}
	return out
}

func runOutputSequence(event runOutputEvent) uint64 {
	if output := runOutput(event); output != nil {
		return output.GetSequence()
	}
	return 0
}

func runOutputTerminal(event runOutputEvent) bool {
	return event.Result != nil && event.Result.GetTerminalResult() != nil
}
