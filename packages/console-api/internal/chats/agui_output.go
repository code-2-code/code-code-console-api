package chats

import (
	"context"
	"strings"

	outputv1 "code-code.internal/go-contract/agent/output/v1"
	"code-code.internal/go-contract/agui"
	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
)

type aguiRunOutputEvent struct {
	runID string
	event runOutputEvent
	err   error
}

type aguiRunOutputState struct {
	lastSequence uint64
}

func newAGUIRunOutputState(string) *aguiRunOutputState {
	return &aguiRunOutputState{}
}

func (s *aguiRunOutputState) apply(stream *aguiStreamWriter, event runOutputEvent) error {
	if s == nil || stream == nil {
		return nil
	}
	if event.Result != nil && event.Result.GetTerminalResult() != nil {
		return nil
	}
	output := runOutput(event)
	if output == nil || output.GetEvent() == nil {
		return nil
	}
	if output.GetSequence() != 0 {
		if output.GetSequence() <= s.lastSequence {
			return nil
		}
		s.lastSequence = output.GetSequence()
	}
	if agui.EventType(output) == aguievents.EventTypeMessagesSnapshot {
		return nil
	}
	if _, err := agui.EventFromOutput(output); err != nil {
		return err
	}
	payload, err := agui.StructJSON(output.GetEvent())
	if err != nil {
		return err
	}
	return stream.rawJSON(payload)
}

func (s *aguiRunOutputState) afterSequence() uint64 {
	if s == nil {
		return 0
	}
	return s.lastSequence
}

func runOutput(event runOutputEvent) *outputv1.RunOutput {
	switch {
	case event.Delta != nil:
		return event.Delta.GetOutput()
	case event.Result != nil:
		return event.Result.GetOutput()
	default:
		return nil
	}
}

func aguiOutputEventType(output *outputv1.RunOutput) aguievents.EventType {
	return agui.EventType(output)
}

func startAGUIRunOutputStream(
	parent context.Context,
	service runOutputStreamService,
	runID string,
	afterSequence uint64,
	updates chan<- aguiRunOutputEvent,
) context.CancelFunc {
	if service == nil || strings.TrimSpace(runID) == "" || updates == nil {
		return nil
	}
	ctx, cancel := context.WithCancel(parent)
	go func() {
		err := service.Stream(ctx, runID, afterSequence, func(event runOutputEvent) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case updates <- aguiRunOutputEvent{runID: runID, event: event}:
				return nil
			}
		})
		if err == nil || ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
		case updates <- aguiRunOutputEvent{runID: runID, err: err}:
		}
	}()
	return cancel
}
