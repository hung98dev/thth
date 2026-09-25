# External Integrations & Service Providers
status: LOCKED

## Scope

Đặc tả kỹ thuật khóa danh sách nhà cung cấp dịch vụ ngoài (Identity, IAP Payment, Notifications) và mô hình bảo mật cho ngày phát hành game Thỉnh Thần.

Mục tiêu: Đảm bảo AI agent triển khai các task Auth (`IMP-006`), IAP (`IMP-053`), và An ninh (`IMP-045`) không phải tự lựa chọn nhà cung cấp, thông số mật mã hay cấu hình môi trường.

## 1. Identity Providers (IdP)

### 1.1 Danh sách Provider được duyệt
Chỉ hỗ trợ 3 nhà cung cấp xác thực liên kết (Federated Identity Providers) chính thức:
1. **Apple Sign In (`apple`):** Dành cho iOS, macOS và PC client. Xác thực JWT Identity Token qua Apple Public Keys (JWKS: `https://appleid.apple.com/auth/keys`).
2. **Google Play Games / Google OAuth (`google`):**
   - **Production Path:** Bắt buộc xác thực chữ ký OIDC JWT cục bộ bằng bộ khóa công khai Google JWKS (`https://www.googleapis.com/oauth2/v3/certs`). Kiểm tra thuật toán RS256, `iss` (`https://accounts.google.com`), `aud` khớp với Client ID được cấu hình, và thời gian hết hạn `exp > now()`. JWKS được cache trong bộ nhớ kèm cơ chế refresh tự động.
   - **Debug / Development Path Only:** Endpoint `https://oauth2.googleapis.com/tokeninfo?id_token={token}` chỉ được phép dùng cho kiểm thử thủ công/debug cục bộ; cấm dùng trên production để tránh phụ thuộc network roundtrip và rate limit của Google.
3. **Steam OpenID (`steam`):** Dành cho phiên bản phân phối PC qua Steamworks SDK. Xác thực qua Steam User Authentication Ticket (`ISteamUserAuth/AuthenticateUserTicket`).

### 1.2 First-Party Password Authentication: ENABLED (ADR-0051)
- Provider `password`: đăng ký bằng username + mật khẩu + email (không xác thực email), đăng nhập bằng username + mật khẩu.
- Quy tắc, băm Argon2id và mã lỗi: `auth.md` § Password Provider. Không có reset mật khẩu lúc ra mắt.
- Email không gửi đi đâu; server không tích hợp dịch vụ gửi mail.

## 2. In-App Purchase (IAP) Verification

### 2.1 Endpoint xác thực
- **Apple App Store:**
  - Production: `https://api.storekit.itunes.apple.com/inApps/v1/transactions/{transactionId}`
  - Sandbox: `https://api.storekit-sandbox.itunes.apple.com/inApps/v1/transactions/{transactionId}`
  - Xác thực qua App Store Server API với JWT sinh bằng private key ES256 được Apple cấp.
- **Google Play:**
  - Google Play Developer API (Android Publisher v3): `https://androidpublisher.googleapis.com/androidpublisher/v3/applications/{packageName}/purchases/products/{productId}/tokens/{token}`
  - Xác thực qua OAuth2 Service Account với quyền `androidpublisher`.
  - Sau khi `GRANTED`: gọi `.../purchases/products/{productId}/tokens/{token}:acknowledge` (retry có backoff; Google tự hoàn tiền nếu chưa acknowledge sau 3 ngày và báo qua RTDN).
- **Steam (PC, ADR-0060):** nhà cung cấp thanh toán duy nhất trên PC là Steam Microtransactions (Steamworks Web API, publisher key):
  - Khởi tạo: `POST https://partner.steam-api.com/ISteamMicroTxn/InitTxn/v3/` (server gọi khi client gọi `/api/v1/iap/steam/init`; `orderid` do server sinh, 64-bit, lưu làm `platform_receipt = orderid`).
  - Hoàn tất: `POST https://partner.steam-api.com/ISteamMicroTxn/FinalizeTxn/v2/` sau callback `MicroTxnAuthorizationResponse_t` phía client; `GRANTED` chỉ khi kết quả `OK` và `QueryTxn/v3` trả `Succeeded`/`Approved` đã finalize.
  - Hoàn tiền/chargeback: không có webhook; worker gọi `GetReport/v5` (`type=GAMESALES`) mỗi 10 phút với con trỏ `time` bền vững, xử lý các dòng `Refunded`/`Chargedback` giống notification refund (dedup theo `orderid + status` trong `iap_notification_dedup`).
  - Sandbox: `ISteamMicroTxnSandbox` khi `IAP_SANDBOX=true`.
- **Client submission:** mọi nền tảng gửi kết quả mua qua `POST /api/v1/iap/verify` (`validation.md` § IAP Receipt Verification); "regional payment gateway" trong `../03_systems/monetization.md` = Steam trên PC.

### 2.2 Server Notifications & Webhooks
- **Apple Server Notifications v2:** Nhận qua webhook endpoint `/api/v1/iap/apple/webhook`. Payload là signed JWS. Server verify chữ ký bằng chứng chỉ Apple root CA.
- **Google Cloud Pub/Sub RTDN:** Nhận qua webhook endpoint `/api/v1/iap/google/webhook`. Bắt buộc xác thực OIDC JWT Bearer Token trong header `Authorization` do Google gửi kèm theo [Google Pub/Sub Push Authentication](https://docs.cloud.google.com/pubsub/docs/authenticate-push-subscriptions), kiểm tra audience khớp với URL webhook và email khớp với Google service account được ủy quyền.
- **Idempotency:** Mỗi notification chứa `notificationUUID` (Apple) hoặc `messageId` (Google). Server ghi nhận vào bảng `iap_notification_dedup` trước khi xử lý, loại trừ trùng lặp.

## 3. Shared Auth Rate-Limiting (Không dùng Redis)

Quy chuẩn điều phối giới hạn tốc độ yêu cầu (Rate Limiting) trong tiến trình world duy nhất (ADR-0052) mà không vi phạm quy tắc cấm Redis:

1. **Local In-Process Tier (Level 1):**
   - Edge subsystem duy trì in-memory Token Bucket cài bằng Go standard library (deterministic refill theo `time.Now`, không dependency ngoài).
   - Áp dụng cho kết nối TCP thô, chống tấn công SYN flood và spam frame WebSocket.
2. **Global Database Tier (Level 2):**
   - Áp dụng cho các thao tác nhạy cảm: Login handshake, IAP claim.
   - Lưu trữ trong bảng PostgreSQL `rate_limit_counters`:
     ```text
     key_hash         BYTEA PRIMARY KEY (SHA-256 của IP + Action)
     window_start     TIMESTAMPTZ NOT NULL
     request_count    INTEGER NOT NULL DEFAULT 1
     ```
   - Câu lệnh atomic upsert (resets counter when the window rolls over):
     ```sql
     INSERT INTO rate_limit_counters (key_hash, window_start, request_count)
     VALUES ($1, date_trunc('minute', NOW()), 1)
     ON CONFLICT (key_hash) DO UPDATE
     SET window_start  = CASE
                           WHEN rate_limit_counters.window_start = date_trunc('minute', NOW())
                           THEN rate_limit_counters.window_start
                           ELSE date_trunc('minute', NOW())
                         END,
         request_count = CASE
                           WHEN rate_limit_counters.window_start = date_trunc('minute', NOW())
                           THEN rate_limit_counters.request_count + 1
                           ELSE 1
                         END
     RETURNING request_count;
     ```
   - Nếu `request_count > LIMIT`, trả về `RATE_LIMITED` (canonical error code; see `../05_network/errors.md`). Note: rate limit key should include account_id where available, per `rate_limits.md`.

## 4. Environment Config & Secrets Schema

Mọi cấu hình môi trường được nạp qua biến môi trường tiêu chuẩn (không commit file `.env` vào repository):

| Biến môi trường | Bắt buộc | Mô tả |
|---|---|---|
| `DATABASE_URL` | Có | Chuỗi kết nối PostgreSQL (pgx pool format) |
| `SERVER_PORT` | Có | Cổng TCP mở listener WSS (mặc định 8080) |
| `IAP_SANDBOX` | Không | `true` \| `false` (mặc định `false`) |
| `APPLE_KEY_ID` | Khi bật IAP | Key ID cấp bởi Apple Developer Portal |
| `APPLE_ISSUER_ID` | Khi bật IAP | Issuer ID của App Store Connect |
| `APPLE_PRIVATE_KEY_PEM` | Khi bật IAP | Nội dung private key ES256 |
| `OPERATOR_TOTP_KEY` | Luôn luôn | 32-byte key (base64) mã hóa AES-256-GCM cho `operators.totp_secret_encrypted` (`auth.md` § Operator) |
| `GOOGLE_SERVICE_ACCOUNT_JSON` | Khi bật IAP | Nội dung JSON của Google Service Account |
| `STEAM_APP_ID` | Khi bật IAP | Steam App ID của game |
| `STEAM_PUBLISHER_KEY` | Khi bật IAP | Steamworks Web API publisher key (MicroTxn, GetReport) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Không | Địa chỉ OpenTelemetry collector |

## Invariants

```text
first-party password = enabled (provider `password`, ADR-0051); federated: Apple, Google, Steam; no password reset at launch
IAP verification = server-to-server qua Apple/Google/Steam official APIs; client chỉ gửi receipt qua POST /api/v1/iap/verify
rate limiting = in-memory L1 + PostgreSQL L2; cấm dùng Redis
secrets = environment variables only; không hardcode trong repo
```
