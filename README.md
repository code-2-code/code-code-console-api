# code-code-console

Console BFF, console web UI, and showcase-facing application surfaces for Code Code.

This repository owns:

- `packages/console-api`: console HTTP/BFF services and the current chat service implementation.
- `packages/console-web`: React console and showcase web workspaces.
- `packages/showcase-api`: showcase HTTP API.

This split preserves source history from the original monorepo. Contract
dependency migration is the next step: console code should consume
`code-code-contracts` through versioned Go/TypeScript packages instead of local
workspace paths.

Useful checks:

```bash
cd packages/console-api && go test ./...
cd packages/showcase-api && go test ./...
cd packages/console-web && pnpm install && pnpm typecheck
```
