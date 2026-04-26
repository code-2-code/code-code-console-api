package chats

import (
	"context"
	"fmt"
	"io"
	"strings"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	runeventv1 "code-code.internal/go-contract/platform/run_event/v1"
)

type runOutputStreamService interface {
	Stream(context.Context, string, uint64, func(runOutputEvent) error) error
}

type runOutputEvent struct {
	Delta  *runeventv1.RunDeltaEvent
	Result *runeventv1.RunResultEvent
}

type GRPCRunOutputClient struct {
	client managementv1.AgentSessionManagementServiceClient
}

func NewGRPCRunOutputClient(client managementv1.AgentSessionManagementServiceClient) *GRPCRunOutputClient {
	if client == nil {
		return nil
	}
	return &GRPCRunOutputClient{client: client}
}

func (c *GRPCRunOutputClient) Stream(ctx context.Context, runID string, afterSequence uint64, yield func(runOutputEvent) error) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("consoleapi/chats: run output client is not configured")
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return fmt.Errorf("consoleapi/chats: run id is required")
	}
	if yield == nil {
		return fmt.Errorf("consoleapi/chats: yield is nil")
	}
	stream, err := c.client.StreamAgentRunOutput(ctx, &managementv1.StreamAgentRunOutputRequest{
		RunId:         runID,
		AfterSequence: afterSequence,
	})
	if err != nil {
		return err
	}
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch payload := event.GetEvent().(type) {
		case *managementv1.StreamAgentRunOutputResponse_Delta:
			if err := yield(runOutputEvent{Delta: payload.Delta}); err != nil {
				return err
			}
		case *managementv1.StreamAgentRunOutputResponse_Result:
			if err := yield(runOutputEvent{Result: payload.Result}); err != nil {
				return err
			}
		default:
		}
	}
}
