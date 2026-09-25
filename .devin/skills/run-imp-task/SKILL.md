---
name: run-imp-task
description: Execute one claimed IMP-* task end to end — readiness, implement, verify, review, CI evidence, auto-merge. Use when the coordinator hands you a task.
---

# Run an IMP Task

Canonical: `docs/10_implementation/agent_execution_protocol.md` (§4 implementation, §5 evidence, §5a merge sequence, §5b retries), `definition_of_done.md`, `audit_gates.md`, ADR-0057. This skill is a checklist; if it differs, the protocol wins.

## Workflow

1. **Sync.** Clean tree; `git fetch && git merge origin/main`. Never rebase.
2. **Readiness.** The task is `IN_PROGRESS` with you as `claimed_by` (claims come only from the coordinator). Re-check DoR (§2); if a box fails, stop and report.
3. **Branch + draft PR.** `imp/IMP-XXX-<slug>`; open a draft PR at once; title `<type>(IMP-XXX): <summary>`; body = Change Packet (§4a).
4. **Map.** `/repo-architecture`; every file you touch is inside `owned_paths`.
5. **Implement + test.** `/implement-backend-feature`, `/implement-unity-feature`, `/client-server-feature`, `/database-change` or `/produce-art-asset`. Every `## Tests` entry exists, runs, and asserts the numbers of any requirement IDs named in `## Acceptance`.
6. **Spec gap?** Append a `BLK-xxx` to `known_blockers.md`, set `status: BLOCKED` + `blocked_by`, push, stop. Never edit protected docs.
7. **Verify locally.** `bash .devin/scripts/verify_delta.sh --full` until PASS.
8. **Review.** Request the `reviewer` (separate session). Fix findings; any later push needs a new review.
9. **CI.** Wait for `Q0-Q6 verify (Linux)` and `Q0-Q6 verify (Windows)` green on the head.
10. **Evidence + DONE.** `gh run download <run_id> -n evidence -D docs/10_implementation/evidence/IMP-XXX/`; set `status: DONE` + summary row; push; wait for CI green and the reviewer's re-posted `policy-review`. Two-phase gate tasks (IMP-000/061/003/004/005/065/068) skip this step and set DONE in a follow-up status PR after the post-merge `main` run.
11. **Merge.** `gh pr ready <N>` then `gh pr merge <N> --auto --squash --delete-branch`. Never merge manually or push to `main`.

## Failures
- 3 red CI runs with the same root cause → `BLOCKED` with `BLK-xxx` (contract) or `OPS-xxx` (environment); stop.
- Infrastructure failure → `gh run rerun --failed` once citing the log; again → `OPS-xxx` + issue labelled `ops-blocked`; stop.
- `AUTO_MERGE_FROZEN=true` → stop.
- Your squash commit was reverted → task is `IN_PROGRESS` again; restart from step 1 on a new branch.

## Acceptance
- Task `DONE` on `main` per `definition_of_done.md`; reviewer `APPROVE`; no file outside `owned_paths`.
