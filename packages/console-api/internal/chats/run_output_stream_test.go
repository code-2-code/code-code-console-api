package chats

import (
	"net/http/httptest"
	"strings"
	"testing"

	outputv1 "code-code.internal/go-contract/agent/output/v1"
	resultv1 "code-code.internal/go-contract/agent/result/v1"
	runeventv1 "code-code.internal/go-contract/platform/run_event/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestAGUIRunOutputStateForwardsAGUIMessageEvents(t *testing.T) {
	response := httptest.NewRecorder()
	recorder, err := newAGUIStreamWriter(response)
	if err != nil {
		t.Fatalf("newAGUIStreamWriter() error = %v", err)
	}
	state := newAGUIRunOutputState("run-1")

	err = state.apply(recorder, runOutputEvent{
		Delta: &runeventv1.RunDeltaEvent{
			RunId:  "run-1",
			Output: testRunOutput(1, map[string]any{"type": "TEXT_MESSAGE_START", "messageId": "assistant-message", "role": "assistant"}),
		},
	})
	if err != nil {
		t.Fatalf("apply() error = %v", err)
	}
	for _, output := range []*outputv1.RunOutput{
		testRunOutput(2, map[string]any{"type": "TEXT_MESSAGE_CONTENT", "messageId": "assistant-message", "delta": "hel"}),
		testRunOutput(3, map[string]any{"type": "TEXT_MESSAGE_END", "messageId": "assistant-message"}),
	} {
		if err := state.apply(recorder, runOutputEvent{Delta: &runeventv1.RunDeltaEvent{
			RunId:  "run-1",
			Output: output,
		}}); err != nil {
			t.Fatalf("apply output error = %v", err)
		}
	}
	if err := state.apply(recorder, runOutputEvent{
		Result: &runeventv1.RunResultEvent{
			RunId: "run-1",
			Payload: &runeventv1.RunResultEvent_TerminalResult{
				TerminalResult: &resultv1.RunResult{Status: resultv1.RunStatus_RUN_STATUS_COMPLETED},
			},
		},
	}); err != nil {
		t.Fatalf("apply terminal error = %v", err)
	}
	joined := response.Body.String()
	for _, fragment := range []string{
		`"type":"TEXT_MESSAGE_START"`,
		`"type":"TEXT_MESSAGE_CONTENT"`,
		`"delta":"hel"`,
		`"type":"TEXT_MESSAGE_END"`,
	} {
		if !strings.Contains(joined, fragment) {
			t.Fatalf("events missing %q: %s", fragment, joined)
		}
	}
}

func TestAGUIRunOutputStateIgnoresDuplicateSequence(t *testing.T) {
	response := httptest.NewRecorder()
	recorder, err := newAGUIStreamWriter(response)
	if err != nil {
		t.Fatalf("newAGUIStreamWriter() error = %v", err)
	}
	state := newAGUIRunOutputState("run-1")

	if err := state.apply(recorder, runOutputEvent{
		Result: &runeventv1.RunResultEvent{
			RunId: "run-1",
			Payload: &runeventv1.RunResultEvent_Output{
				Output: testRunOutput(1, map[string]any{"type": "TEXT_MESSAGE_CONTENT", "messageId": "assistant-message", "delta": "hello"}),
			},
		},
	}); err != nil {
		t.Fatalf("apply output error = %v", err)
	}
	if err := state.apply(recorder, runOutputEvent{
		Delta: &runeventv1.RunDeltaEvent{
			RunId:  "run-1",
			Output: testRunOutput(1, map[string]any{"type": "TEXT_MESSAGE_CONTENT", "messageId": "assistant-message", "delta": "duplicate"}),
		},
	}); err != nil {
		t.Fatalf("apply duplicate delta error = %v", err)
	}
	joined := response.Body.String()
	if strings.Count(joined, `"type":"TEXT_MESSAGE_CONTENT"`) != 1 {
		t.Fatalf("unexpected content count: %s", joined)
	}
}

func TestAGUIRunOutputStateEmitsToolCallEvents(t *testing.T) {
	response := httptest.NewRecorder()
	recorder, err := newAGUIStreamWriter(response)
	if err != nil {
		t.Fatalf("newAGUIStreamWriter() error = %v", err)
	}
	state := newAGUIRunOutputState("run-1")

	events := []*outputv1.RunOutput{
		testRunOutput(1, map[string]any{"type": "TOOL_CALL_START", "toolCallId": "tool-1", "toolCallName": "shell"}),
		testRunOutput(2, map[string]any{"type": "TOOL_CALL_ARGS", "toolCallId": "tool-1", "delta": `{"summary":"ls -la"}`}),
		testRunOutput(3, map[string]any{"type": "TOOL_CALL_END", "toolCallId": "tool-1"}),
		testRunOutput(4, map[string]any{"type": "TOOL_CALL_RESULT", "messageId": "tool-message-tool-1", "toolCallId": "tool-1", "role": "tool", "content": `{"summary":"ls -la"}`}),
	}
	for _, output := range events {
		if err := state.apply(recorder, runOutputEvent{Delta: &runeventv1.RunDeltaEvent{
			RunId:  "run-1",
			Output: output,
		}}); err != nil {
			t.Fatalf("apply tool call error = %v", err)
		}
	}
	joined := response.Body.String()
	for _, fragment := range []string{
		`"type":"TOOL_CALL_START"`,
		`"toolCallId":"tool-1"`,
		`"toolCallName":"shell"`,
		`"type":"TOOL_CALL_ARGS"`,
		`"delta":"{\"summary\":\"ls -la\"}"`,
		`"type":"TOOL_CALL_END"`,
		`"type":"TOOL_CALL_RESULT"`,
	} {
		if !strings.Contains(joined, fragment) {
			t.Fatalf("events missing %q: %s", fragment, joined)
		}
	}
}

func testRunOutput(sequence uint64, event map[string]any) *outputv1.RunOutput {
	payload, err := structpb.NewStruct(event)
	if err != nil {
		panic(err)
	}
	return &outputv1.RunOutput{Sequence: sequence, Event: payload}
}
