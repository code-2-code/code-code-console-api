package chats

import (
	"testing"

	agentrunv1 "code-code.internal/go-contract/platform/agent_run/v1"
	agentsessionactionv1 "code-code.internal/go-contract/platform/agent_session_action/v1"
)

func TestBuildAGUITurnActivityContentIncludesWorkflowSteps(t *testing.T) {
	action := &agentsessionactionv1.AgentSessionActionState{
		Spec: &agentsessionactionv1.AgentSessionActionSpec{
			ActionId:  "turn-1",
			SessionId: "session-1",
			TurnId:    "turn-1",
		},
		Status: &agentsessionactionv1.AgentSessionActionStatus{
			Phase: agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_RUNNING,
			View: &agentsessionactionv1.AgentSessionActionView{
				DisplayPhase: agentsessionactionv1.AgentSessionActionDisplayPhase_AGENT_SESSION_ACTION_DISPLAY_PHASE_RUNNING,
			},
		},
	}
	run := &agentrunv1.AgentRunState{
		Spec: &agentrunv1.AgentRunSpec{
			RunId: "run-1",
			PrepareJobs: []*agentrunv1.AgentRunPrepareJob{
				{JobId: "auth", JobType: "auth"},
				{JobId: "rules", JobType: "rules"},
			},
		},
		Status: &agentrunv1.AgentRunStatus{
			Phase: agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING,
			PrepareJobs: []*agentrunv1.AgentRunPrepareJobStatus{
				{JobId: "auth", Phase: agentrunv1.AgentRunPrepareJobPhase_AGENT_RUN_PREPARE_JOB_PHASE_SUCCEEDED},
				{JobId: "rules", Phase: agentrunv1.AgentRunPrepareJobPhase_AGENT_RUN_PREPARE_JOB_PHASE_RUNNING},
			},
		},
	}

	content, ok := buildAGUITurnActivityContent("session-1", action, run, "run-1")
	if !ok {
		t.Fatal("buildAGUITurnActivityContent() ok = false")
	}
	if got, want := len(content.Steps), 3; got != want {
		t.Fatalf("steps = %d, want %d", got, want)
	}
	if content.Steps[0].Label != "Prepare auth" || content.Steps[0].Phase != "succeeded" {
		t.Fatalf("step 0 = %+v", content.Steps[0])
	}
	if content.Steps[1].Label != "Prepare rules" || content.Steps[1].Phase != "running" {
		t.Fatalf("step 1 = %+v", content.Steps[1])
	}
	if content.Steps[2].Label != "Execute prompt" || content.Steps[2].Phase != "pending" {
		t.Fatalf("step 2 = %+v", content.Steps[2])
	}
}
