# Engineering Conventions & Code Standards
status: LOCKED

## Scope

Quy chuẩn kỹ thuật bắt buộc cho toàn bộ mã nguồn Go, C# (Unity), Protobuf và SQL trong repository Thỉnh Thần.

Mục tiêu: Đảm bảo mọi AI agent khi sinh code đều tuân thủ cùng một phong cách, kiến trúc, xử lý lỗi, logging, thời gian, RNG và quản lý bộ nhớ, không tạo ra các pattern phân kỳ.

## 1. Go Backend Conventions

### 1.1 Toolchain & Formatting
- **Go Toolchain:** `1.27.1` (pin tại `docs/00_context/technology_versions.md`).
- **Formatting:** `gofmt` tiêu chuẩn với tabs cho thụt lề; cấm dùng linter tự ý reformat khác chuẩn `gofmt`.
- **Package Layout:** Nằm trong `server/internal/<domain>/`. Tên package ngắn gọn, chữ thường, không gạch dưới, không camelCase.

### 1.2 Structured Logging
- **Thư viện chuẩn:** `log/slog` từ standard library. Tuyệt đối cấm thêm `zap`, `logrus`, hay `zerolog`.
- **Định dạng:** JSON format cho production (`slog.NewJSONHandler`), Text format cho local development.
- **Thuộc tính bắt buộc (Attributes):**
  - Mọi log ghi nhận nghiệp vụ/lỗi phải có: `op` (operation name), `revision` (content revision), `actor_id` / `session_id` (nếu trong context session).
  - Không log dữ liệu cá nhân nhạy cảm (plaintext passwords, raw payment tokens).

### 1.3 Context, Time & RNG Injection
- **Context:** `ctx context.Context` luôn là tham số đầu tiên của mọi hàm I/O, database query hoặc network call. Tôn trọng deadline và `ctx.Done()`.
- **Time Authority:**
  - Server là nguồn chân lý thời gian tuyệt đối. Luôn dùng `time.Now().UTC()`.
  - Không tin cậy và không dùng timestamp do client gửi lên cho bất kỳ logic tính toán hay lưu trữ nào.
- **RNG Injection:**
  - Gameplay RNG (combat rolls, drop tables, crits, status procs): Phải dùng `math/rand/v2` PCG-64 được inject qua interface từ `server/internal/core/rng/`. Tuyệt đối cấm import `math/rand` (v1).
  - Security / Cryptography / UUIDs: Sử dụng `crypto/rand`.

### 1.4 Concurrency & Simulation Single-Writer
- Mỗi map channel / instance simulation loop là một single-writer duy nhất chạy trên một goroutine chuyên trách.
- Không chia sẻ bộ nhớ mutable giữa các goroutines mà không thông qua channel hoặc atomic state snapshot.
- Ephemeral Global (party, chat fanout, matchmaking): Single-writer in-process; không dùng Redis hay message broker ngoài.

### 1.5 Error Handling & Wire Error Mapping
- Lỗi nghiệp vụ nội bộ dùng domain error types hoặc sentinel errors (`errors.Is`, `errors.As`).
- Khi phản hồi cho client qua network, lỗi phải được ánh xạ 1-1 với mã lỗi trong `docs/05_network/errors.md`.
- Cấm panic trong code xử lý request hoặc simulation tick. `panic` chỉ được phép ở giai đoạn khởi động (startup assertion) khi thiếu config cốt lõi không thể phục hồi.

### 1.6 Database & SQL (pgx/v5)
- **Truy cập:** Sử dụng `github.com/jackc/pgx/v5` với native pgxpool. Cấm dùng ORM (GORM, ent) hoặc abstraction layer không được duyệt.
- **Parameterized SQL:** 100% câu lệnh SQL phải dùng tham số (`$1`, `$2`), tuyệt đối cấm nối chuỗi câu lệnh SQL.
- **Giao dịch (Transactions):** Các thao tác cập nhật số dư, inventory, reward claims phải nằm trong transaction với lock ordering nhất quán (e.g. sort theo ID trước khi SELECT FOR UPDATE) để ngăn deadlock.
- **Migrations:**
  - Sử dụng `golang-migrate/migrate/v4 v4.20.1`.
  - Planned repository path is `server/migrations/` with `000001_baseline_schema.up.sql` / `000001_baseline_schema.down.sql`; `IMP-005` owns materializing it.
  - Không sửa file migration cũ đã merge; mọi sửa đổi schema phải là migration mới tăng dần.

## 2. C# / Unity Client Conventions

### 2.1 Language & Style
- **Language Level:** C# 9.0 (.NET Standard 2.1) tương thích Unity 6000.6.1f1.
- **Style:** Allman brace style (dấu `{` ở dòng riêng), 4 spaces cho thụt lề, PascalCase cho Types và Methods, camelCase cho biến cục bộ và parameters, `_camelCase` cho private fields.

### 2.2 Assembly Definitions (.asmdef)
Mọi code C# phải nằm dưới các assembly definition được cô lập rõ ràng:
- `ThinhThan.Protocol.asmdef`: Chỉ chứa code sinh tự động từ protobuf; không tham chiếu tới bất kỳ Unity assembly nào khác.
- `ThinhThan.Core.asmdef`: Chứa math, UUID utilities, text normalization, pure domain models.
- `ThinhThan.Net.asmdef`: WSS client, session state machine, serialization handling.
- `ThinhThan.Systems.asmdef`: Gameplay presentation, movement interpolation, combat controllers.
- `ThinhThan.UI.asmdef`: HUD, menu screens, input overlays.
- `ThinhThan.Tests.EditMode.asmdef` và `ThinhThan.Tests.PlayMode.asmdef`: Thư mục test riêng.
Cấm circular dependencies giữa các asmdef.

### 2.3 Memory & Allocation Hygiene
- Realtime loop (`Update`, `FixedUpdate`): Cấm cấp phát bộ nhớ rác (`GC.Alloc = 0`) trên hot path.
- Dùng object pooling cho projectile, floating text, visual effects.
- Tránh LINQ trong các hàm chạy theo frame.

### 2.4 Input & UI State Machine
- Sử dụng Unity Input System (`com.unity.inputsystem 1.20.0`).
- Gửi cạnh di chuyển qua `C2S_MOVEMENT_EDGE` (ID 108, `DISCRETE_INTENT`) với wire enum `PRESS | RELEASE | FLIP`; không phát minh `STOP`. `client_mono_ms` chỉ advisory và server clamp bù trễ tối đa 80 ms theo `../05_network/messages.md`.
- UI điều khiển qua finite state machine, không gọi trực tiếp network socket từ view UI.

## 3. Protocol Buffers Wire Conventions

- **Syntax:** `proto3`. Thụt lề 2 spaces.
- **Package:** `package thinhthan.v1;`.
- **Options:**
  - `option go_package = "thinhthan/internal/protocol/v1;protocolv1";`
  - `option csharp_namespace = "ThinhThan.Protocol.V1";`
- **Tên field:** snake_case cho field names, PascalCase cho message và enum names, SCREAMING_SNAKE_CASE cho enum values (với tiền tố enum name).
- **Generated Code:** Cấm sửa tay code sinh ra trong `server/internal/protocol/v1/` và `client/Assets/Scripts/Protocol/`. Mọi thay đổi phải sinh qua `scripts/codegen.ps1`; Go parity tests nằm ngoài generated-only tree tại `server/internal/testing/protocol/`.

## 4. Testing & Verification Conventions

- **Go Tests:** File đặt cạnh code nguồn `*_test.go`. Tên test: `Test<Feature>_<Scenario>`. Bắt buộc test cả happy path lẫn error/boundary conditions.
- **Unity Tests:** EditMode tests cho pure logic / data validation; PlayMode tests cho WSS session flow và component lifecycle.
- **Fixtures:** Fixture dữ liệu phải đặt trong thư mục `testdata/`, nội dung bất biến, có thể tái lập 100%.
- **Zero Flakiness:** Mọi test ngẫu nhiên phải dùng fixed seed được kiểm soát; không dùng thời gian thực để so sánh nếu không có tolerance hợp lý.

## 4a. Branch, Commit and PR Conventions

```text
branches   imp/IMP-XXX-<slug>   spec/BLK-XXX-<slug>   claim/<yyyymmdd>-<n>   revert/<sha>
titles     <type>(IMP-XXX): <summary>      type = feat | fix | test | chore | docs | ci | perf
PR body    Change Packet (.github/pull_request_template.md, agent_execution_protocol.md §4a)
updates    git merge origin/main only; never rebase, amend, force-push or push to main
merge      squash via auto-merge after the §5a sequence
```

## 5. Clean Cutover & Anti-Drift Rule

- Khi thay đổi một API hoặc contract:
  - Migrate toàn bộ caller trong cùng một commit.
  - Xóa bỏ hoàn toàn code cũ, không để lại alias, shim, deprecated stub hay commented code.
  - Đảm bảo git status hoàn toàn sạch sau khi chạy codegen và test verify.

## Invariants

```text
Go toolchain = 1.27.1; Unity = 6000.6.1f1 (C# 9.0 Allman)
slog cho Go logging; pgx/v5 raw SQL cho database
math/rand/v2 cho gameplay RNG; crypto/rand cho security/UUID
proto/ là wire SoT; không sửa tay generated code
zero commented-out code, zero fake stubs, zero unapproved packages
```
