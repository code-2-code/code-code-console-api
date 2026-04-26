package chats

import (
	"encoding/json"
	"strings"

	agentrunv1 "code-code.internal/go-contract/platform/agent_run/v1"
	agentsessionactionv1 "code-code.internal/go-contract/platform/agent_session_action/v1"
	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
)

const aguiTurnActivityType = "TURN"

type aguiProjectionTurn struct {
	ID             string `json:"id"`
	Phase          string `json:"phase,omitempty"`
	DisplayPhase   string `json:"displayPhase,omitempty"`
	Message        string `json:"message,omitempty"`
	CanStop        bool   `json:"canStop,omitempty"`
	CanRetry       bool   `json:"canRetry,omitempty"`
	RetryCount     int32  `json:"retryCount,omitempty"`
	AttemptCount   int32  `json:"attemptCount,omitempty"`
	CandidateIndex int32  `json:"candidateIndex,omitempty"`
	FailureClass   string `json:"failureClass,omitempty"`
}

type aguiTurnActivityContent struct {
	ID             string                 `json:"id"`
	SessionID      string                 `json:"sessionId,omitempty"`
	RunID          string                 `json:"runId,omitempty"`
	Phase          string                 `json:"phase,omitempty"`
	DisplayPhase   string                 `json:"displayPhase,omitempty"`
	Message        string                 `json:"message,omitempty"`
	CanStop        bool                   `json:"canStop,omitempty"`
	CanRetry       bool                   `json:"canRetry,omitempty"`
	RetryCount     int32                  `json:"retryCount,omitempty"`
	AttemptCount   int32                  `json:"attemptCount,omitempty"`
	CandidateIndex int32                  `json:"candidateIndex,omitempty"`
	FailureClass   string                 `json:"failureClass,omitempty"`
	Steps          []aguiTurnActivityStep `json:"steps,omitempty"`
}

type aguiTurnActivityStep struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Phase   string `json:"phase,omitempty"`
	Message string `json:"message,omitempty"`
}

func newAGUITurnActivitySnapshot(
	sessionID string,
	action *agentsessionactionv1.AgentSessionActionState,
	run *agentrunv1.AgentRunState,
	runID string,
) (*aguievents.ActivitySnapshotEvent, string, bool, error) {
	content, ok := buildAGUITurnActivityContent(sessionID, action, run, runID)
	if !ok {
		return nil, "", false, nil
	}
	key, err := json.Marshal(content)
	if err != nil {
		return nil, "", false, err
	}
	event := aguievents.NewActivitySnapshotEvent("turn-activity-"+content.ID, aguiTurnActivityType, content)
	return event, string(key), true, nil
}

func buildAGUITurnActivityContent(
	sessionID string,
	action *agentsessionactionv1.AgentSessionActionState,
	run *agentrunv1.AgentRunState,
	runID string,
) (aguiTurnActivityContent, bool) {
	if action == nil {
		return aguiTurnActivityContent{}, false
	}
	turn := buildAGUITurnProjection(action, run)
	id := strings.TrimSpace(turn.ID)
	if id == "" {
		return aguiTurnActivityContent{}, false
	}
	return aguiTurnActivityContent{
		ID:             id,
		SessionID:      strings.TrimSpace(sessionID),
		RunID:          strings.TrimSpace(runID),
		Phase:          turn.Phase,
		DisplayPhase:   turn.DisplayPhase,
		Message:        turn.Message,
		CanStop:        turn.CanStop,
		CanRetry:       turn.CanRetry,
		RetryCount:     turn.RetryCount,
		AttemptCount:   turn.AttemptCount,
		CandidateIndex: turn.CandidateIndex,
		FailureClass:   turn.FailureClass,
		Steps:          buildAGUITurnActivitySteps(run),
	}, true
}

func buildAGUITurnProjection(
	action *agentsessionactionv1.AgentSessionActionState,
	run *agentrunv1.AgentRunState,
) aguiProjectionTurn {
	spec := action.GetSpec()
	status := action.GetStatus()
	turn := aguiProjectionTurn{
		ID:             strings.TrimSpace(spec.GetTurnId()),
		Phase:          actionPhaseLabel(status.GetPhase()),
		DisplayPhase:   actionDisplayPhaseLabel(status.GetView().GetDisplayPhase()),
		Message:        strings.TrimSpace(status.GetMessage()),
		CanStop:        status.GetView().GetCanStop(),
		CanRetry:       status.GetView().GetCanRetry(),
		RetryCount:     status.GetRetryCount(),
		AttemptCount:   status.GetAttemptCount(),
		CandidateIndex: status.GetCandidateIndex(),
		FailureClass:   actionFailureClassLabel(status.GetFailureClass()),
	}
	if turn.ID == "" {
		turn.ID = strings.TrimSpace(spec.GetActionId())
	}
	if turn.Message == "" {
		turn.Message = actionMessage(action, run)
	}
	return turn
}
