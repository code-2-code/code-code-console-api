package chats

import (
	"strings"

	agentrunv1 "code-code.internal/go-contract/platform/agent_run/v1"
)

func buildAGUITurnActivitySteps(run *agentrunv1.AgentRunState) []aguiTurnActivityStep {
	if run == nil || run.GetSpec() == nil {
		return nil
	}
	steps := make([]aguiTurnActivityStep, 0, len(run.GetSpec().GetPrepareJobs())+1)
	statusByID := prepareStatusByID(run.GetStatus().GetPrepareJobs())
	for _, job := range run.GetSpec().GetPrepareJobs() {
		if job == nil {
			continue
		}
		status := statusByID[strings.TrimSpace(job.GetJobId())]
		phase := "pending"
		message := ""
		if status != nil {
			phase = prepareJobPhaseLabel(status.GetPhase())
			message = strings.TrimSpace(status.GetMessage())
		}
		steps = append(steps, aguiTurnActivityStep{
			ID:      "prepare:" + strings.TrimSpace(job.GetJobId()),
			Label:   prepareJobLabel(job),
			Phase:   phase,
			Message: message,
		})
	}
	if step, ok := executeActivityStep(run, len(steps) > 0); ok {
		steps = append(steps, step)
	}
	if len(steps) == 0 {
		return nil
	}
	return steps
}

func prepareStatusByID(items []*agentrunv1.AgentRunPrepareJobStatus) map[string]*agentrunv1.AgentRunPrepareJobStatus {
	out := map[string]*agentrunv1.AgentRunPrepareJobStatus{}
	for _, item := range items {
		id := strings.TrimSpace(item.GetJobId())
		if id != "" {
			out[id] = item
		}
	}
	return out
}

func prepareJobLabel(job *agentrunv1.AgentRunPrepareJob) string {
	name := firstNonEmpty(job.GetJobType(), job.GetJobId(), "session")
	return "Prepare " + humanizeActivityToken(name)
}

func executeActivityStep(run *agentrunv1.AgentRunState, hasPrepare bool) (aguiTurnActivityStep, bool) {
	phase := executeActivityPhase(run, hasPrepare)
	if phase == "" {
		return aguiTurnActivityStep{}, false
	}
	return aguiTurnActivityStep{
		ID:      "execute",
		Label:   "Execute prompt",
		Phase:   phase,
		Message: strings.TrimSpace(aguiRunMessage(run.GetStatus())),
	}, true
}

func executeActivityPhase(run *agentrunv1.AgentRunState, hasPrepare bool) string {
	status := run.GetStatus()
	if status == nil {
		return ""
	}
	if hasPrepare {
		switch {
		case prepareJobsFailed(status.GetPrepareJobs()):
			return "skipped"
		case !prepareJobsSucceeded(status.GetPrepareJobs()):
			return "pending"
		}
	}
	return runPhaseLabel(status.GetPhase())
}

func prepareJobsSucceeded(items []*agentrunv1.AgentRunPrepareJobStatus) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if item.GetPhase() != agentrunv1.AgentRunPrepareJobPhase_AGENT_RUN_PREPARE_JOB_PHASE_SUCCEEDED &&
			item.GetPhase() != agentrunv1.AgentRunPrepareJobPhase_AGENT_RUN_PREPARE_JOB_PHASE_SKIPPED {
			return false
		}
	}
	return true
}

func prepareJobsFailed(items []*agentrunv1.AgentRunPrepareJobStatus) bool {
	for _, item := range items {
		switch item.GetPhase() {
		case agentrunv1.AgentRunPrepareJobPhase_AGENT_RUN_PREPARE_JOB_PHASE_FAILED,
			agentrunv1.AgentRunPrepareJobPhase_AGENT_RUN_PREPARE_JOB_PHASE_CANCELED:
			return true
		}
	}
	return false
}

func prepareJobPhaseLabel(phase agentrunv1.AgentRunPrepareJobPhase) string {
	switch phase {
	case agentrunv1.AgentRunPrepareJobPhase_AGENT_RUN_PREPARE_JOB_PHASE_PENDING:
		return "pending"
	case agentrunv1.AgentRunPrepareJobPhase_AGENT_RUN_PREPARE_JOB_PHASE_SKIPPED:
		return "skipped"
	case agentrunv1.AgentRunPrepareJobPhase_AGENT_RUN_PREPARE_JOB_PHASE_RUNNING:
		return "running"
	case agentrunv1.AgentRunPrepareJobPhase_AGENT_RUN_PREPARE_JOB_PHASE_SUCCEEDED:
		return "succeeded"
	case agentrunv1.AgentRunPrepareJobPhase_AGENT_RUN_PREPARE_JOB_PHASE_FAILED:
		return "failed"
	case agentrunv1.AgentRunPrepareJobPhase_AGENT_RUN_PREPARE_JOB_PHASE_CANCELED:
		return "canceled"
	default:
		return ""
	}
}

func runPhaseLabel(phase agentrunv1.AgentRunPhase) string {
	switch phase {
	case agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING,
		agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_SCHEDULED:
		return "pending"
	case agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING:
		return "running"
	case agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED:
		return "succeeded"
	case agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED:
		return "failed"
	case agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELED:
		return "canceled"
	default:
		return ""
	}
}

func humanizeActivityToken(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "_", " "))
	value = strings.ReplaceAll(value, "-", " ")
	return value
}
