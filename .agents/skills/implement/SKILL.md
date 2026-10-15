---
name: implement
description: "Implement a piece of work based on a spec or set of tickets."
disable-model-invocation: true
---

Implement the work described by the user in the spec or tickets, inside the ticket worktree created by the orchestrator, on that worktree's `ticket/...` branch.

Use /tdd where possible, at pre-agreed seams.

Run typechecking regularly, single test files regularly, and the full test suite once at the end.

Once done, use /code-review to review the work.

Git policy for this repo:

- Verify with `git branch --show-current` that you are on your own `ticket/...` branch before making changes.
- After the relevant checks pass, commit locally to that ticket branch.
- Push only with explicit user permission approval; never force push.
- Never merge to `main`, delete branches, or clean worktrees — those belong to the human and the orchestrator after a confirmed merge.
