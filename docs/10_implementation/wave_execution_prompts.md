# Implementation Wave Prompts
status: LOCKED

Operational prompts for executing every `IMP-*` task. Status, dependencies, scope, paths, tests and evidence are canonical in `task_queue.md`; the workflow, claim and merge sequence are canonical in `agent_execution_protocol.md` (§5a). This file never overrides them.

Waves are the longest-path levels of the `depends_on` DAG: wave 0 has no dependency and wave N tasks depend on a task in wave N-1. A task may start as soon as all its own dependencies are `DONE` on `main`; the coordinator recomputes readiness from `main` before every claim instead of waiting for a whole wave.

## How the Owner Runs the Project
1. Once, after Owner Setup: start the **Reviewer** session (§3 below) and the **Spec-owner** session (§4 below); they keep running and react to PRs / blockers by themselves.
2. For each wave in order, open one AI session in the repo and just say **"Làm wave N"** (e.g. "Làm wave 0 đi"; "làm wave tiếp" = next unfinished wave). `AGENTS.md` § Owner commands and the `/run-wave` skill make the agent execute the Wave Prompt below with that N. Wait for its final report, then send the next wave. Nothing else is needed from the owner except resolving `OPS-xxx` issues labelled `ops-blocked`.

### Wave Prompt (copy-paste, replace `<N>`)
```text
Thực hiện Wave <N> của repo thinhthan. Bạn là coordinator cho wave này.
Đọc AGENTS.md, docs/10_implementation/wave_execution_prompts.md, agent_execution_protocol.md (§3–§5b) và audit_gates.md (Bootstrap Mode).
1. Pull main. Kiểm tra mọi task của các wave trước đã DONE trên main (hoặc BLOCKED đã được spec-owner xử lý); nếu chưa, báo lại danh sách task còn thiếu và dừng.
   Nếu AUTO_MERGE_FROZEN=true hoặc có OPS-xxx mở: báo lại và dừng.
2. Lấy danh sách task của Wave <N> trong bảng Waves. Với từng task: claim theo §3 (IMP-000 tự claim trong PR của nó),
   rồi giao cho một agent con riêng (worktree, branch imp/IMP-XXX-<slug>, DB port và Unity cache riêng) chạy Implementer Prompt bên dưới.
   Chạy song song tối đa bằng số runner slot; task còn lại chờ slot trống.
3. Theo dõi tới khi mọi task của wave là DONE trên main (two-phase task: cả PR status theo §5a) hoặc BLOCKED.
   Task BLOCKED vì BLK: để spec-owner xử lý, khi task trở lại NOT_STARTED thì claim và chạy lại. BLOCKED vì OPS: dừng task đó.
4. Báo cáo cuối: từng task -> DONE/BLOCKED (mã BLK/OPS), PR đã merge, số lần CI chạy, việc chủ repo cần làm (nếu có).
Không tự sửa spec, không hỏi con người trong lúc chạy, không push thẳng main, không rebase/force-push.
```

## Session Prompts (copy-paste)

Start one long-running session per role. If the harness has no `/run-imp-task` skill, the agent follows `.devin/skills/run-imp-task/SKILL.md` as a checklist. Role rules: `agent_execution_protocol.md` §1.

### 1. Coordinator (one session, runs the whole pipeline)
```text
Bạn là coordinator của repo thinhthan. Đọc AGENTS.md, docs/10_implementation/agent_execution_protocol.md (§3, §5a, §5b), audit_gates.md và file này.
Lặp liên tục cho tới khi IMP-048 DONE:
1. Pull main. Nếu biến repo AUTO_MERGE_FROZEN=true hoặc có OPS-xxx mở thì dừng và chờ.
2. Tính các task sẵn sàng (mọi depends_on DONE trên main, không BLK mở ảnh hưởng, tuân Bootstrap Mode trước IMP-068).
3. Claim theo §3 (IMP-000: claim trong chính PR của nó), tối đa số runner slot task IN_PROGRESS cùng lúc, ưu tiên thứ tự topo nhỏ nhất.
4. Giao mỗi task cho đúng một implementer bằng Implementer Prompt (task, branch, worktree, base SHA).
5. Task BLOCKED vì BLK -> giao cho spec-owner; claim quá 24h không hoạt động -> trả về NOT_STARTED.
Không tự viết code, không sửa spec, không hỏi con người.
```

### 2. Implementer (one session per task)
```text
Thực hiện <IMP-XXX> trên branch <branch>, worktree <path>, base <sha>.
Chạy /run-imp-task (hoặc làm đúng checklist .devin/skills/run-imp-task/SKILL.md) theo docs/10_implementation/agent_execution_protocol.md §4–§5b.
Chỉ sửa owned_paths và path test/evidence của packet. Không hỏi con người. Không push thẳng main, không workflow_dispatch, không rebase/force-push.
Thiếu hoặc mâu thuẫn spec: ghi BLK-xxx vào known_blockers.md, đặt task BLOCKED, push và dừng.
Xong khi PR đã ready và auto-merge đã bật (two-phase task: theo §5a).
```

### 3. Reviewer (one session, separate OS account holding the App key)
```text
Bạn là reviewer độc lập (.devin/agents/reviewer.md). Với mỗi PR mở hoặc có push mới:
review `git diff origin/main...HEAD` theo checklist của profile (spec-change PR dùng spec checklist), chạy verify_delta --full,
đăng verdict + danh sách spec đã đối chiếu thành PR review comment, rồi post status `policy-review` bằng App cho đúng head SHA.
Không sửa code, không tự duyệt PR do chính session này tạo.
```

### 4. Spec-owner (one session, env THINHTHAN_AGENT_ROLE=spec-owner)
```text
Bạn là spec-owner (.devin/agents/spec-owner.md). Xử lý từng BLK-xxx mở trong docs/10_implementation/known_blockers.md:
quyết định phương án tốt nhất, sửa spec/ADR và mọi consumer trong một spec-change PR (branch spec/BLK-xxx-<slug>), gắn requirement ID cho yêu cầu đo được,
thêm regression test vào ## Tests của task liên quan, đóng BLK và trả task về NOT_STARTED. Không viết code triển khai, không hỏi con người.
```

## Waves

| Wave | Tasks | Notes |
|---|---|---|
| 0 | IMP-000 | two-phase: IMP-000; bootstrap-eligible: IMP-000 |
| 1 | IMP-001, IMP-061, IMP-063, IMP-064, IMP-101 | two-phase: IMP-061; bootstrap-eligible: IMP-001, IMP-061, IMP-063, IMP-064, IMP-101 |
| 2 | IMP-002, IMP-005, IMP-070, IMP-083 | two-phase: IMP-005; bootstrap-eligible: IMP-002, IMP-005, IMP-070, IMP-083 |
| 3 | IMP-003, IMP-071, IMP-073, IMP-074, IMP-075, IMP-104 | two-phase: IMP-003; bootstrap-eligible: IMP-003, IMP-071, IMP-073, IMP-074, IMP-075, IMP-104 |
| 4 | IMP-004 | two-phase: IMP-004; bootstrap-eligible: IMP-004 |
| 5 | IMP-050, IMP-068 | two-phase: IMP-068; bootstrap-eligible: IMP-050 |
| 6 | IMP-098 |  |
| 7 | IMP-078, IMP-079, IMP-081, IMP-082, IMP-097 |  |
| 8 | IMP-006, IMP-007, IMP-008, IMP-062, IMP-080 |  |
| 9 | IMP-072, IMP-100, IMP-105 |  |
| 10 | IMP-065, IMP-076 | two-phase: IMP-065 |
| 11 | IMP-013 |  |
| 12 | IMP-066 |  |
| 13 | IMP-009, IMP-011, IMP-018, IMP-095 |  |
| 14 | IMP-010, IMP-012, IMP-014, IMP-020, IMP-029, IMP-030, IMP-034, IMP-035, IMP-059, IMP-099 |  |
| 15 | IMP-015, IMP-026, IMP-036, IMP-054, IMP-058, IMP-060, IMP-094 |  |
| 16 | IMP-016, IMP-027, IMP-037, IMP-038, IMP-049 |  |
| 17 | IMP-017, IMP-019, IMP-028, IMP-031, IMP-032, IMP-033, IMP-053, IMP-084, IMP-088 |  |
| 18 | IMP-021, IMP-022, IMP-051, IMP-055, IMP-057, IMP-102 |  |
| 19 | IMP-023, IMP-025, IMP-039, IMP-089, IMP-092 |  |
| 20 | IMP-024, IMP-040, IMP-042, IMP-086, IMP-087, IMP-090, IMP-091 |  |
| 21 | IMP-041, IMP-052, IMP-085 |  |
| 22 | IMP-043, IMP-093 |  |
| 23 | IMP-047, IMP-056, IMP-077 |  |
| 24 | IMP-103 |  |
| 25 | IMP-067, IMP-069 | serialized integration: IMP-067, IMP-069 |
| 26 | IMP-044, IMP-045, IMP-046, IMP-096 |  |
| 27 | IMP-048 | serialized integration: IMP-048 |

Before `IMP-068 = DONE`, only tasks without `IMP-068` in their transitive `depends_on` run (Bootstrap Mode, `audit_gates.md`). Two-phase tasks merge `IN_PROGRESS` and get `DONE` from a follow-up status PR. Serialized integration tasks run alone on their owned paths (`server/cmd/server/`, `server/internal/app/`, `client/Assets/Scripts/App/`, release artifacts).

## Parallel Execution
The coordinator claims a batch of ready tasks in one claim PR, then gives each implementer only the Implementer Prompt with its own `IMP-*`, branch, worktree path and base SHA. Concurrent tasks never share a worktree, database, port or Unity project/cache directory. Runner slots from Owner Setup bound concurrency; each PR follows §5a independently and merges on its own green checks.
