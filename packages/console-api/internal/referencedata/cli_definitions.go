package referencedata

import (
	"context"
	"net/http"

	"code-code.internal/console-api/internal/httpjson"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
)

type cliSupportService interface {
	ListCLIs(context.Context) ([]*supportv1.CLI, error)
}

// RegisterCLIDefinitionHandlers registers read-only CLI definition HTTP routes.
func RegisterCLIDefinitionHandlers(mux *http.ServeMux, service cliSupportService) {
	mux.HandleFunc("/api/cli-definitions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		clis, err := service.ListCLIs(r.Context())
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_cli_definitions_failed", err)
			return
		}
		items := make([]*managementv1.CLIDefinitionView, 0, len(clis))
		for _, cli := range clis {
			items = append(items, cliToManagementView(cli))
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, &managementv1.ListCLIDefinitionsResponse{Items: items})
	})
}

func cliToManagementView(cli *supportv1.CLI) *managementv1.CLIDefinitionView {
	if cli == nil {
		return nil
	}
	out := &managementv1.CLIDefinitionView{
		CliId:       cli.GetCliId(),
		DisplayName: cli.GetDisplayName(),
		IconUrl:     cli.GetIconUrl(),
		WebsiteUrl:  cli.GetWebsiteUrl(),
		Description: cli.GetDescription(),
	}
	out.ContainerImages = make([]*managementv1.CLIContainerImageView, 0, len(cli.GetContainerImages()))
	for _, image := range cli.GetContainerImages() {
		if image == nil {
			continue
		}
		out.ContainerImages = append(out.ContainerImages, &managementv1.CLIContainerImageView{
			ExecutionClass: image.GetExecutionClass(),
			Image:          image.GetImage(),
			CpuRequest:     image.GetCpuRequest(),
			MemoryRequest:  image.GetMemoryRequest(),
		})
	}
	if caps := cli.GetCapability(); caps != nil {
		out.Capabilities = &managementv1.CLIDefinitionCapabilityView{
			SupportsStreaming:    caps.GetSupportsStreaming(),
			SupportsApprovalMode: caps.GetSupportsApprovalMode(),
		}
		for _, proto := range caps.GetSupportedProtocols() {
			out.Capabilities.SupportedProtocols = append(out.Capabilities.SupportedProtocols, proto.String())
		}
	}
	return out
}
