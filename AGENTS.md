# Agent Rules

- This repository owns console BFF, chat HTTP service, platform client
  adapters, and console API telemetry setup.
- Do not edit protobuf source or generated contract bindings here.
- If UI or BFF work needs a new public contract, change `code-code-contracts`
  first and then update this repository to the released contract version.
- If session persistence behavior must change, make that change in
  `code-code-platform-session` first.
- Keep platform runtime internals out of console code. Cross the boundary
  through public services and contract types.
- Do not move React UI, showcase API, deployment, or platform runtime behavior
  into this repository.
