package chats

import (
	"strings"

	"code-code.internal/go-contract/agui"
	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"google.golang.org/protobuf/types/known/structpb"
)

type aguiProjectionUsage struct {
	InputTokens           int64  `json:"inputTokens,omitempty"`
	OutputTokens          int64  `json:"outputTokens,omitempty"`
	CachedInputTokens     int64  `json:"cachedInputTokens,omitempty"`
	ReasoningOutputTokens int64  `json:"reasoningOutputTokens,omitempty"`
	RequestCount          int64  `json:"requestCount,omitempty"`
	ToolCallCount         int64  `json:"toolCallCount,omitempty"`
	ModelID               string `json:"modelId,omitempty"`
	ContextWindowTokens   int64  `json:"contextWindowTokens,omitempty"`
}

type aguiUsageState struct {
	lastLLMSequence  uint64
	lastTurnSequence uint64
	projection       *aguiProjectionUsage
}

func (s *aguiUsageState) apply(event runOutputEvent) bool {
	if s == nil {
		return false
	}
	output := runOutput(event)
	if output == nil || aguiOutputEventType(output) != aguievents.EventTypeCustom {
		return false
	}
	fields := output.GetEvent().GetFields()
	name := strings.TrimSpace(fields["name"].GetStringValue())
	value := structMap(fields["value"])
	switch name {
	case agui.CustomRunLLMUsage:
		if output.GetSequence() <= s.lastLLMSequence {
			return false
		}
		s.lastLLMSequence = output.GetSequence()
		next := s.ensureProjection()
		next.ModelID = firstString(value, "modelId", "resolvedModelId", "providerModelId", "surfaceId", "surfaceId")
		next.ContextWindowTokens = intValue(value["contextWindowTokens"])
		applyTokenUsage(next, structMapFromAny(value["usage"]))
		return true
	case agui.CustomRunTurnUsage:
		if output.GetSequence() <= s.lastTurnSequence {
			return false
		}
		s.lastTurnSequence = output.GetSequence()
		next := s.ensureProjection()
		applyTokenUsage(next, structMapFromAny(value["usage"]))
		counters := structMapFromAny(value["counters"])
		next.RequestCount = intValue(counters["requestCount"])
		next.ToolCallCount = intValue(counters["toolCallCount"])
		return true
	default:
		return false
	}
}

func (s *aguiUsageState) snapshot() *aguiProjectionUsage {
	if s == nil || s.projection == nil {
		return nil
	}
	copy := *s.projection
	return &copy
}

func (s *aguiUsageState) ensureProjection() *aguiProjectionUsage {
	if s.projection == nil {
		s.projection = &aguiProjectionUsage{}
	}
	return s.projection
}

func applyTokenUsage(target *aguiProjectionUsage, usage map[string]any) {
	if target == nil {
		return
	}
	target.InputTokens = intValue(usage["inputTokens"])
	target.OutputTokens = intValue(usage["outputTokens"])
	target.CachedInputTokens = intValue(usage["cachedInputTokens"])
	target.ReasoningOutputTokens = intValue(usage["reasoningOutputTokens"])
}

func structMap(value *structpb.Value) map[string]any {
	if value == nil {
		return nil
	}
	return structMapFromAny(value.AsInterface())
}

func structMapFromAny(value any) map[string]any {
	item, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return item
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok {
			if normalized := strings.TrimSpace(value); normalized != "" {
				return normalized
			}
		}
	}
	return ""
}

func intValue(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case float32:
		return int64(typed)
	default:
		return 0
	}
}
