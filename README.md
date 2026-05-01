# code-code-console-api

Console BFF and chat HTTP services for Code Code.

This repository owns:

- `packages/console-api`: console HTTP/BFF handlers, chat service behavior,
  platform service clients, provider observability projections, and console API
  telemetry setup.
- `code-code-contracts`: generated shared contracts as a Git submodule.
- `code-code-platform-session`: session persistence/domain helpers as a Git
  submodule.

Useful checks:

```bash
cd packages/console-api && go test ./...
```
