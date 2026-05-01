# Agent Rules

- This repository owns console API, console web UI, and showcase-facing UI/API code.
- Do not edit protobuf source or generated contract bindings here.
- If UI or BFF work needs a new public contract, change `code-code-contracts` first and then update this repository to the released contract version.
- Keep platform runtime internals out of console code. Cross the boundary through public services and contract types.
- Keep UI changes scoped to the relevant package under `packages/console-web`.
