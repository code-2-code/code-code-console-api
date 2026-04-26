package chats

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"code-code.internal/console-api/internal/httpjson"
	agentcorev1 "code-code.internal/go-contract/agent/core/v1"
	inputv1 "code-code.internal/go-contract/agent/input/v1"
	"code-code.internal/go-contract/agui"
	agentrunv1 "code-code.internal/go-contract/platform/agent_run/v1"
	agentsessionv1 "code-code.internal/go-contract/platform/agent_session/v1"
	agentsessionactionv1 "code-code.internal/go-contract/platform/agent_session_action/v1"
	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	aguitypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	aguisse "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/encoding/sse"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

const (
	aguiPollInterval      = 1 * time.Second
	aguiHeartbeatInterval = 15 * time.Second
	aguiTerminalWait      = 1500 * time.Millisecond
)

type aguiStreamWriter struct {
	writer  http.ResponseWriter
	flusher http.Flusher
	sse     *aguisse.SSEWriter
}

type aguiProjectionState struct {
	Session aguiProjectionSession `json:"session"`
	Usage   *aguiProjectionUsage  `json:"usage,omitempty"`
}

type aguiProjectionSession struct {
	ID                    string `json:"id"`
	ProviderID            string `json:"providerId,omitempty"`
	ProfileID             string `json:"profileId,omitempty"`
	Phase                 string `json:"phase,omitempty"`
	Message               string `json:"message,omitempty"`
	ActiveRunID           string `json:"activeRunId,omitempty"`
	RealizedRuleRevision  string `json:"realizedRuleRevision,omitempty"`
	RealizedSkillRevision string `json:"realizedSkillRevision,omitempty"`
	RealizedMCPRevision   string `json:"realizedMcpRevision,omitempty"`
}

func handleAGUIRun(w http.ResponseWriter, r *http.Request, chats chatService, sessions sessionControlService, turns turnService, runs runService, runOutputs runOutputStreamService, chatID string) {
	if r.Method != http.MethodPost {
		httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		httpjson.WriteError(w, http.StatusNotFound, "not_found", "chat not found")
		return
	}

	request, err := decodeAGUIRunRequest(r)
	if err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	threadID := strings.TrimSpace(request.ThreadID)
	if threadID == "" {
		threadID = chatID
	}
	runID := strings.TrimSpace(request.RunID)
	if runID == "" {
		generated, err := newRandomID("run")
		if err != nil {
			httpjson.WriteError(w, http.StatusInternalServerError, "generate_run_id_failed", "failed to generate run id")
			return
		}
		runID = generated
	}
	request.ThreadID = threadID
	request.RunID = runID
	if err := normalizeAGUIRunInput(request); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_ag_ui_input", err.Error())
		return
	}
	prompt, err := agui.LatestUserText(request.Messages)
	if err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_prompt", err.Error())
		return
	}

	sessionID, session, err := loadAGUIChatSession(r.Context(), chats, sessions, chatID)
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "get_chat_session_failed", err)
		return
	}
	if threadID == chatID {
		threadID = sessionID
	}
	request.ThreadID = threadID

	turnID, err := newRandomID("turn")
	if err != nil {
		httpjson.WriteError(w, http.StatusInternalServerError, "generate_turn_id_failed", "failed to generate turn id")
		return
	}
	action, err := sessions.CreateTurn(r.Context(), sessionID, turnID, turnID, &agentcorev1.RunRequest{
		RunId: runID,
		Input: &inputv1.RunInput{Text: prompt},
	})
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "create_turn_failed", err)
		return
	}
	actionState := action.GetAction()
	actionID := strings.TrimSpace(actionState.GetSpec().GetActionId())
	if actionID == "" {
		actionID = turnID
	}

	history, err := loadAGUIMessages(r.Context(), chats, chatID)
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "list_chat_messages_failed", err)
		return
	}
	stream, err := newAGUIStreamWriter(w)
	if err != nil {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	if err := stream.event(aguievents.NewMessagesSnapshotEvent(history)); err != nil {
		return
	}
	if err := stream.raw(agui.RunStartedPayload(threadID, runID, stringPointerValue(request.ParentRunID), request)); err != nil {
		return
	}
	if err := streamRunLifecycle(r.Context(), stream, sessions, turns, runs, runOutputs, sessionID, actionID, threadID, runID, session, actionState); err != nil {
		_ = stream.event(aguievents.NewRunErrorEvent(err.Error()))
		return
	}
}

func loadAGUIChatSession(ctx context.Context, chats chatService, sessions sessionControlService, chatID string) (string, *agentsessionv1.AgentSessionState, error) {
	chat, err := chats.GetChat(ctx, chatID)
	if err != nil {
		return "", nil, err
	}
	sessionID := currentSessionID(chatID, chat)
	state, err := sessions.Get(ctx, sessionID)
	if err != nil {
		return "", nil, err
	}
	return sessionID, state, nil
}

func streamRunLifecycle(
	ctx context.Context,
	stream *aguiStreamWriter,
	sessions sessionControlService,
	turns turnService,
	runs runService,
	runOutputs runOutputStreamService,
	sessionID string,
	actionID string,
	threadID string,
	requestedRunID string,
	sessionState *agentsessionv1.AgentSessionState,
	actionState *agentsessionactionv1.AgentSessionActionState,
) error {
	pollTicker := time.NewTicker(aguiPollInterval)
	heartbeatTicker := time.NewTicker(aguiHeartbeatInterval)
	defer pollTicker.Stop()
	defer heartbeatTicker.Stop()

	currentRunID := strings.TrimSpace(requestedRunID)
	runState, currentRunID, err := loadCurrentRun(ctx, runs, actionState, currentRunID)
	if err != nil {
		return err
	}
	outputUpdates := make(chan aguiRunOutputEvent, 32)
	currentOutputRunID := ""
	resumeOutputRunID := ""
	var outputAfterSequence uint64
	var cancelOutput context.CancelFunc
	var outputState *aguiRunOutputState
	outputUsage := &aguiUsageState{}
	outputTerminalObserved := false
	var pendingTerminal aguievents.Event
	var pendingTerminalAt time.Time

	lastProjection := ""
	maybeEmitProjection := func() error {
		projection := buildAGUIProjection(sessionID, sessionState, outputUsage.snapshot())
		encoded, err := json.Marshal(projection)
		if err != nil {
			return err
		}
		if string(encoded) == lastProjection {
			return nil
		}
		lastProjection = string(encoded)
		if err := stream.event(aguievents.NewStateSnapshotEvent(projection)); err != nil {
			return err
		}
		return nil
	}
	lastActivity := ""
	maybeEmitActivity := func() error {
		event, key, ok, err := newAGUITurnActivitySnapshot(sessionID, actionState, runState, currentRunID)
		if err != nil || !ok {
			return err
		}
		if key == lastActivity {
			return nil
		}
		lastActivity = key
		return stream.event(event)
	}
	switchOutputRun := func(nextRunID string, nextRunState *agentrunv1.AgentRunState) {
		nextRunID = strings.TrimSpace(nextRunID)
		if currentOutputRunID == nextRunID && cancelOutput != nil {
			return
		}
		if cancelOutput != nil {
			cancelOutput()
			cancelOutput = nil
		}
		if nextRunID != resumeOutputRunID {
			outputAfterSequence = 0
		}
		resumeOutputRunID = ""
		currentOutputRunID = ""
		outputState = nil
		outputUsage = &aguiUsageState{}
		outputTerminalObserved = false
		pendingTerminal = nil
		pendingTerminalAt = time.Time{}
		if runOutputs == nil || nextRunState == nil || nextRunID == "" {
			return
		}
		outputState = newAGUIRunOutputState(nextRunID)
		cancelOutput = startAGUIRunOutputStream(ctx, runOutputs, nextRunID, outputAfterSequence, outputUpdates)
		currentOutputRunID = nextRunID
	}
	defer func() {
		if cancelOutput != nil {
			cancelOutput()
		}
	}()

	if err := maybeEmitProjection(); err != nil {
		return err
	}
	if err := maybeEmitActivity(); err != nil {
		return err
	}
	switchOutputRun(currentRunID, runState)
	if terminal, ok := terminalRunEvent(threadID, currentRunID, actionState, runState); ok {
		if currentOutputRunID != "" && !outputTerminalObserved {
			pendingTerminal = terminal
			pendingTerminalAt = time.Now()
		} else {
			return stream.event(terminal)
		}
	}

	for {
		select {
		case update := <-outputUpdates:
			if strings.TrimSpace(update.runID) != currentOutputRunID {
				continue
			}
			if update.err != nil {
				if cancelOutput != nil {
					cancelOutput()
					cancelOutput = nil
				}
				resumeOutputRunID = currentOutputRunID
				if outputState != nil {
					outputAfterSequence = outputState.afterSequence()
				}
				currentOutputRunID = ""
				outputState = nil
				outputTerminalObserved = false
				if pendingTerminal != nil {
					return stream.event(pendingTerminal)
				}
				continue
			}
			if outputState == nil {
				continue
			}
			if err := outputState.apply(stream, update.event); err != nil {
				bestEffortStopTurn(actionID, turns)
				return err
			}
			if outputUsage.apply(update.event) {
				if err := maybeEmitProjection(); err != nil {
					bestEffortStopTurn(actionID, turns)
					return err
				}
			}
			if update.event.Result != nil && update.event.Result.GetTerminalResult() != nil {
				outputTerminalObserved = true
				if pendingTerminal != nil && pendingTerminalAt.IsZero() {
					pendingTerminalAt = time.Now()
				}
			}
		case <-ctx.Done():
			bestEffortStopTurn(actionID, turns)
			return context.Canceled
		case <-heartbeatTicker.C:
			if err := stream.heartbeat(); err != nil {
				bestEffortStopTurn(actionID, turns)
				return err
			}
		case <-pollTicker.C:
			nextSession, err := sessions.Get(ctx, sessionID)
			if err != nil {
				return err
			}
			sessionState = nextSession

			nextAction, err := turns.Get(ctx, actionID)
			if err != nil {
				return err
			}
			actionState = nextAction.GetAction()

			previousRunID := currentRunID
			runState, currentRunID, err = loadCurrentRun(ctx, runs, actionState, currentRunID)
			if err != nil {
				return err
			}
			if previousRunID != currentRunID || (currentOutputRunID == "" && runState != nil && currentRunID != "") {
				switchOutputRun(currentRunID, runState)
			}

			if err := maybeEmitProjection(); err != nil {
				bestEffortStopTurn(actionID, turns)
				return err
			}
			if err := maybeEmitActivity(); err != nil {
				bestEffortStopTurn(actionID, turns)
				return err
			}

			terminal, ok := terminalRunEvent(threadID, currentRunID, actionState, runState)
			if !ok {
				pendingTerminal = nil
				pendingTerminalAt = time.Time{}
				continue
			}
			if currentOutputRunID != "" {
				if pendingTerminal == nil {
					pendingTerminalAt = time.Now()
				}
				pendingTerminal = terminal
				if !outputTerminalObserved || time.Since(pendingTerminalAt) < aguiTerminalWait {
					continue
				}
			}
			pendingTerminal = nil
			pendingTerminalAt = time.Time{}
			return stream.event(terminal)
		}
	}
}

func decodeAGUIRunRequest(r *http.Request) (*aguitypes.RunAgentInput, error) {
	limited := &io.LimitedReader{R: r.Body, N: 1<<20 + 1}
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	if int64(len(body)) > 1<<20 {
		return nil, fmt.Errorf("request body exceeds %d bytes", 1<<20)
	}
	var request aguitypes.RunAgentInput
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}
	return &request, nil
}

func normalizeAGUIRunInput(input *aguitypes.RunAgentInput) error {
	if input == nil {
		return nil
	}
	input.ThreadID = strings.TrimSpace(input.ThreadID)
	input.RunID = strings.TrimSpace(input.RunID)
	if input.ParentRunID != nil {
		parentRunID := strings.TrimSpace(*input.ParentRunID)
		if parentRunID == "" {
			input.ParentRunID = nil
		} else {
			input.ParentRunID = &parentRunID
		}
	}
	if input.Messages == nil {
		input.Messages = []aguitypes.Message{}
	}
	for index, message := range input.Messages {
		normalized, err := agui.NormalizeMessage(message)
		if err != nil {
			return fmt.Errorf("messages[%d]: %w", index, err)
		}
		input.Messages[index] = normalized
	}
	if input.Tools == nil {
		input.Tools = []aguitypes.Tool{}
	}
	if input.Context == nil {
		input.Context = []aguitypes.Context{}
	}
	return nil
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func newAGUIStreamWriter(w http.ResponseWriter) (*aguiStreamWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming unsupported")
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	return &aguiStreamWriter{writer: w, flusher: flusher, sse: aguisse.NewSSEWriter()}, nil
}

func (s *aguiStreamWriter) event(event aguievents.Event) error {
	if s == nil {
		return nil
	}
	return s.sse.WriteEvent(context.Background(), s.writer, event)
}

func (s *aguiStreamWriter) raw(payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.rawJSON(data)
}

func (s *aguiStreamWriter) rawJSON(data []byte) error {
	if s == nil {
		return nil
	}
	return s.sse.WriteBytes(context.Background(), s.writer, data)
}

func (s *aguiStreamWriter) heartbeat() error {
	if _, err := fmt.Fprint(s.writer, ": keepalive\n\n"); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}

func bestEffortStopTurn(actionID string, turns turnService) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _ = turns.Stop(ctx, actionID)
}

func isTerminalRunPhase(phase agentrunv1.AgentRunPhase) bool {
	switch phase {
	case agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED,
		agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED,
		agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELED:
		return true
	default:
		return false
	}
}

func isTerminalActionPhase(phase agentsessionactionv1.AgentSessionActionPhase) bool {
	switch phase {
	case agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_SUCCEEDED,
		agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_FAILED,
		agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_CANCELED:
		return true
	default:
		return false
	}
}

func sessionPhaseLabel(phase agentsessionv1.AgentSessionPhase) string {
	switch phase {
	case agentsessionv1.AgentSessionPhase_AGENT_SESSION_PHASE_PENDING:
		return "pending"
	case agentsessionv1.AgentSessionPhase_AGENT_SESSION_PHASE_READY:
		return "ready"
	case agentsessionv1.AgentSessionPhase_AGENT_SESSION_PHASE_RUNNING:
		return "running"
	case agentsessionv1.AgentSessionPhase_AGENT_SESSION_PHASE_FAILED:
		return "failed"
	default:
		return ""
	}
}

func actionPhaseLabel(phase agentsessionactionv1.AgentSessionActionPhase) string {
	switch phase {
	case agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_PENDING:
		return "pending"
	case agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_RUNNING:
		return "running"
	case agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_SUCCEEDED:
		return "succeeded"
	case agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_FAILED:
		return "failed"
	case agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_CANCELED:
		return "canceled"
	default:
		return ""
	}
}

func actionDisplayPhaseLabel(phase agentsessionactionv1.AgentSessionActionDisplayPhase) string {
	switch phase {
	case agentsessionactionv1.AgentSessionActionDisplayPhase_AGENT_SESSION_ACTION_DISPLAY_PHASE_QUEUED:
		return "queued"
	case agentsessionactionv1.AgentSessionActionDisplayPhase_AGENT_SESSION_ACTION_DISPLAY_PHASE_RETRYING:
		return "retrying"
	case agentsessionactionv1.AgentSessionActionDisplayPhase_AGENT_SESSION_ACTION_DISPLAY_PHASE_FALLBACKING:
		return "fallbacking"
	case agentsessionactionv1.AgentSessionActionDisplayPhase_AGENT_SESSION_ACTION_DISPLAY_PHASE_RUNNING:
		return "running"
	case agentsessionactionv1.AgentSessionActionDisplayPhase_AGENT_SESSION_ACTION_DISPLAY_PHASE_STOPPING:
		return "stopping"
	case agentsessionactionv1.AgentSessionActionDisplayPhase_AGENT_SESSION_ACTION_DISPLAY_PHASE_STOPPED:
		return "stopped"
	case agentsessionactionv1.AgentSessionActionDisplayPhase_AGENT_SESSION_ACTION_DISPLAY_PHASE_SUCCEEDED:
		return "succeeded"
	case agentsessionactionv1.AgentSessionActionDisplayPhase_AGENT_SESSION_ACTION_DISPLAY_PHASE_FAILED:
		return "failed"
	default:
		return ""
	}
}

func actionFailureClassLabel(class agentsessionactionv1.AgentSessionActionFailureClass) string {
	switch class {
	case agentsessionactionv1.AgentSessionActionFailureClass_AGENT_SESSION_ACTION_FAILURE_CLASS_BLOCKED:
		return "blocked"
	case agentsessionactionv1.AgentSessionActionFailureClass_AGENT_SESSION_ACTION_FAILURE_CLASS_TRANSIENT:
		return "transient"
	case agentsessionactionv1.AgentSessionActionFailureClass_AGENT_SESSION_ACTION_FAILURE_CLASS_PERMANENT:
		return "permanent"
	case agentsessionactionv1.AgentSessionActionFailureClass_AGENT_SESSION_ACTION_FAILURE_CLASS_MANUAL_RETRY:
		return "manual_retry"
	default:
		return ""
	}
}

func aguiRunMessage(status *agentrunv1.AgentRunStatus) string {
	if status == nil {
		return ""
	}
	if message := strings.TrimSpace(status.GetMessage()); message != "" {
		return message
	}
	if result := status.GetResult(); result != nil && result.GetError() != nil {
		if message := strings.TrimSpace(result.GetError().GetMessage()); message != "" {
			return message
		}
	}
	switch status.GetPhase() {
	case agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING:
		return "Run is pending."
	case agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_SCHEDULED:
		return "Run is scheduled."
	case agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING:
		return "Run is running."
	case agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED:
		return "Run completed successfully."
	case agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED:
		return "Run failed."
	case agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELED:
		return "Run canceled."
	default:
		return ""
	}
}

func runErrorCode(status *agentrunv1.AgentRunStatus) string {
	if status == nil || status.GetResult() == nil || status.GetResult().GetError() == nil {
		return ""
	}
	return strings.TrimSpace(status.GetResult().GetError().GetCode())
}

func actionMessage(action *agentsessionactionv1.AgentSessionActionState, run *agentrunv1.AgentRunState) string {
	if action != nil && action.GetStatus() != nil {
		if message := strings.TrimSpace(action.GetStatus().GetMessage()); message != "" {
			return message
		}
	}
	if run != nil && run.GetStatus() != nil {
		if message := strings.TrimSpace(aguiRunMessage(run.GetStatus())); message != "" {
			return message
		}
	}
	if action != nil && action.GetStatus() != nil {
		switch action.GetStatus().GetPhase() {
		case agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_PENDING:
			return "Turn is pending."
		case agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_RUNNING:
			return "Turn is running."
		case agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_SUCCEEDED:
			return "Turn completed successfully."
		case agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_FAILED:
			return "Turn failed."
		case agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_CANCELED:
			return "Turn canceled."
		}
	}
	return ""
}

func loadCurrentRun(
	ctx context.Context,
	runs runService,
	action *agentsessionactionv1.AgentSessionActionState,
	fallbackRunID string,
) (*agentrunv1.AgentRunState, string, error) {
	runID := strings.TrimSpace(fallbackRunID)
	if action != nil && action.GetStatus() != nil {
		if actionRunID := strings.TrimSpace(action.GetStatus().GetRun().GetRunId()); actionRunID != "" {
			runID = actionRunID
		}
	}
	if runID == "" {
		return nil, "", nil
	}
	response, err := runs.Get(ctx, runID)
	if err != nil {
		if grpcstatus.Code(err) == codes.NotFound {
			return nil, runID, nil
		}
		return nil, runID, err
	}
	return response.GetRun(), runID, nil
}

func buildAGUIProjection(
	sessionID string,
	session *agentsessionv1.AgentSessionState,
	usage *aguiProjectionUsage,
) aguiProjectionState {
	projection := aguiProjectionState{
		Session: aguiProjectionSession{ID: sessionID},
	}
	if session != nil {
		spec := session.GetSpec()
		status := session.GetStatus()
		if specSessionID := strings.TrimSpace(spec.GetSessionId()); specSessionID != "" {
			projection.Session.ID = specSessionID
		}
		projection.Session.ProviderID = strings.TrimSpace(spec.GetProviderId())
		projection.Session.ProfileID = strings.TrimSpace(spec.GetProfileId())
		projection.Session.Phase = sessionPhaseLabel(status.GetPhase())
		projection.Session.Message = strings.TrimSpace(status.GetMessage())
		projection.Session.ActiveRunID = strings.TrimSpace(status.GetActiveRun().GetRunId())
		projection.Session.RealizedRuleRevision = strings.TrimSpace(status.GetRealizedRuleRevision())
		projection.Session.RealizedSkillRevision = strings.TrimSpace(status.GetRealizedSkillRevision())
		projection.Session.RealizedMCPRevision = strings.TrimSpace(status.GetRealizedMcpRevision())
	}
	if usage != nil {
		copy := *usage
		projection.Usage = &copy
	}
	return projection
}

func terminalRunEvent(
	threadID string,
	runID string,
	action *agentsessionactionv1.AgentSessionActionState,
	run *agentrunv1.AgentRunState,
) (aguievents.Event, bool) {
	if action != nil && action.GetStatus() != nil {
		status := action.GetStatus()
		if isTerminalActionPhase(status.GetPhase()) {
			if status.GetPhase() == agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_FAILED {
				code := actionFailureClassLabel(status.GetFailureClass())
				if code == "" && run != nil && run.GetStatus() != nil {
					code = runErrorCode(run.GetStatus())
				}
				return newRunErrorEvent(actionMessage(action, run), code), true
			}
			return aguievents.NewRunFinishedEventWithOptions(
				threadID,
				terminalRunID(runID, action, run),
				aguievents.WithResult(map[string]any{
					"phase":          status.GetPhase().String(),
					"displayPhase":   status.GetView().GetDisplayPhase().String(),
					"message":        actionMessage(action, run),
					"retryCount":     status.GetRetryCount(),
					"attemptCount":   status.GetAttemptCount(),
					"candidateIndex": status.GetCandidateIndex(),
				}),
			), true
		}
		if status.GetPhase() != agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_UNSPECIFIED {
			return nil, false
		}
	}
	if run == nil || run.GetStatus() == nil || !isTerminalRunPhase(run.GetStatus().GetPhase()) {
		return nil, false
	}
	status := run.GetStatus()
	if status.GetPhase() == agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED {
		return newRunErrorEvent(aguiRunMessage(status), runErrorCode(status)), true
	}
	return aguievents.NewRunFinishedEventWithOptions(
		threadID,
		terminalRunID(runID, action, run),
		aguievents.WithResult(map[string]any{
			"phase":   status.GetPhase().String(),
			"message": aguiRunMessage(status),
		}),
	), true
}

func newRunErrorEvent(message, code string) aguievents.Event {
	if normalized := strings.TrimSpace(code); normalized != "" {
		return aguievents.NewRunErrorEvent(message, aguievents.WithErrorCode(normalized))
	}
	return aguievents.NewRunErrorEvent(message)
}

func terminalRunID(runID string, action *agentsessionactionv1.AgentSessionActionState, run *agentrunv1.AgentRunState) string {
	if action != nil && action.GetStatus() != nil {
		if actionRunID := strings.TrimSpace(action.GetStatus().GetRun().GetRunId()); actionRunID != "" {
			return actionRunID
		}
	}
	if run != nil {
		if specRunID := strings.TrimSpace(run.GetSpec().GetRunId()); specRunID != "" {
			return specRunID
		}
		if statusRunID := strings.TrimSpace(run.GetStatus().GetRunId()); statusRunID != "" {
			return statusRunID
		}
	}
	return runID
}
