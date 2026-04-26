package platformclient

import (
	"context"

	cliruntimev1 "code-code.internal/go-contract/platform/cli_runtime/v1"
)

func (c *CLIRuntimes) LatestAvailableImages(ctx context.Context) ([]*cliruntimev1.CLIRuntimeImage, error) {
	client, err := c.client.requireCLIRuntime()
	if err != nil {
		return nil, err
	}
	response, err := client.GetLatestAvailableCLIRuntimeImages(ctx, &cliruntimev1.GetLatestAvailableCLIRuntimeImagesRequest{})
	if err != nil {
		return nil, err
	}
	return response.GetItems(), nil
}
