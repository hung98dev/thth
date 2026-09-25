# Personal Data Register & Column Mapping
status: LOCKED

## Scope

Bảng đăng ký và ánh xạ chi tiết các trường dữ liệu cá nhân (Personal Data) tới từng bảng và cột trong cơ sở dữ liệu PostgreSQL của Thỉnh Thần, tuân thủ Luật Bảo vệ dữ liệu cá nhân số 91/2025/QH15 và Nghị định số 13/2023/NĐ-CP.

Tài liệu này là căn cứ thực thi kỹ thuật trực tiếp cho task `IMP-056` (Data Protection / Retention & Data-Subject Requests).

Category letters and retention intent are canonical in `data_protection.md`; this file maps them to columns.

## 1. Table.Column Mapping Matrix

| Category | Mô tả dữ liệu | Bảng & Cột cụ thể | Cơ sở pháp lý (Luật 91/2025 & NĐ 356/2025) | Thời hạn lưu trữ | Hành động khi Xóa / Yêu cầu Hủy (Erasure) |
|---|---|---|---|---|---|
| **A. Định danh tài khoản** | ID tài khoản, ngày tạo, trạng thái rủi ro | `accounts.account_id`<br>`accounts.created_at`<br>`accounts.status`<br>`account_refund_consumed_events` (điểm hoàn tiền IAP suy ra, ADR-0060)<br>`account_password_credentials.username_key`<br>`account_password_credentials.email`<br>`account_password_credentials.password_hash` | Điều 11 (Thực hiện hợp đồng dịch vụ) | 1 năm sau khi đóng tài khoản | Chuyển `status = 'TOMBSTONE_ERASED'`. Cắt toàn bộ liên kết nhân vật. Sau 1 năm, xóa bản ghi nếu không còn nghĩa vụ pháp lý. |
| **B. Danh tính liên kết (Federated ID)** | ID người dùng từ bên thứ ba (Apple, Google, Steam) | `account_identities.provider_id`<br>`account_identities.provider_subject`<br>`account_identities.linked_at` | Điều 11 (Xác thực đăng nhập theo yêu cầu người dùng) | 30 ngày sau khi đóng tài khoản | Hard delete ngay lập tức các dòng trong `account_identities` khi xử lý yêu cầu xóa hợp lệ. |
| **C. Phiên xác thực & Thiết bị** | Session family, refresh token hash, ngày hết hạn | `auth_session_families.*`<br>`auth_refresh_credentials.*` | Điều 11 (Bảo mật phiên đăng nhập) | Tối đa 30 ngày hoặc khi token hết hạn | Hard delete toàn bộ bản ghi session và refresh credentials của tài khoản. |
| **D. IP & Tín hiệu an ninh** | Danh sách token bị thu hồi, IP rate limit | `auth_revocations.*`<br>`rate_limit_counters.*` | Điều 17 (Bảo vệ an ninh mạng và chống gian lận) | 90 ngày (rolling window) | Dữ liệu tự động hết hạn và xóa sau 90 ngày theo scheduler; không thể yêu cầu xóa sớm do phục vụ phòng chống tấn công. |
| **E. Chứng từ thanh toán IAP** | Token giao dịch, biên lai Store, lịch sử hoàn tiền | `account_iap_entitlements.platform_receipt`<br>`account_refund_consumed_events.*` | Luật Kế toán 88/2015/QH13 (Nghĩa vụ lưu trữ chứng từ tài chính) | **10 năm** bắt buộc | **Không xóa chứng từ tài chính**. Thực hiện cắt liên kết định danh: chuyển `account_id` trỏ về `TOMBSTONE_ACCOUNT_ID` để lưu trữ chứng từ vô danh. |
| **F. Nội dung giao tiếp (Chat)** | Lịch sử chat thế giới, bang hội, tin nhắn riêng | `chat_messages.sender_account_id`<br>`chat_messages.sender_character_id`<br>`chat_messages.content`<br>`chat_messages.created_at` | Điều 11 (Cung cấp tính năng mạng xã hội in-game) | 90 ngày (rolling window) | Xóa toàn bộ tin nhắn do tài khoản gửi trong vòng 15 ngày lịch kể từ yêu cầu xóa hợp lệ (data_protection.md). |
| **G. Dữ liệu Gameplay & Tiến trình** | Cấp độ, trang bị, kho đồ, Atlas, nhiệm vụ | `characters.*`<br>`character_atlas.*`<br>`character_progression.*`<br>`character_inventories.*`<br>`operations.*` | Không thuộc dữ liệu cá nhân nhạy cảm; thuộc sở hữu hệ thống trò chơi | Nhân vật: vĩnh viễn ở dạng ẩn danh; event log gameplay: 180 ngày (rolling) | **Cắt đứt liên kết sở hữu (Anonymization):**<br>- `characters.account_id = TOMBSTONE_ACCOUNT_ID`<br>- Đổi tên nhân vật: `Anonymized_<short_uuid>`<br>- Giải phóng `name_key` ban đầu<br>- Hủy bỏ các niêm yết trên Chợ Đấu Giá. |
| **H. Nhật ký kiểm toán (Audit Logs)** | Log can thiệp tài khoản, thay đổi quyền, thao tác GM | `audit_events.actor_id`<br>`audit_events.action`<br>`audit_events.payload`<br>`audit_events.created_at` | Điều 17 (An toàn thông tin & truy vết gian lận) | 3 năm | Lưu giữ nguyên vẹn trong 3 năm phục vụ điều tra gian lận; sau 3 năm tự động purge. |

## 2. Quy trình Thực thi Quyền Dữ liệu (Data-Subject Requests)

### 2.1 Yêu cầu Xóa Dữ liệu (Right to Erasure / RTBF)
1. Thời hạn xử lý: Phản hồi thủ tục trong vòng **2 ngày làm việc**; hoàn tất thực thi hợp lệ trong vòng **15 ngày** kể từ khi nhận được yêu cầu hợp lệ qua kênh hỗ trợ chính thức per Decree 356/2025/NĐ-CP.
2. Thứ tự thực thi kỹ thuật trong một PostgreSQL Transaction:
   ```sql
   -- 1. Chuyển trạng thái tài khoản
   UPDATE accounts SET status = 'TOMBSTONE_ERASED' WHERE account_id = $1;
   -- 2. Xóa liên kết danh tính bên thứ ba
   DELETE FROM account_identities WHERE account_id = $1;
   DELETE FROM account_password_credentials WHERE account_id = $1;
   -- 3. Hủy phiên làm việc
   DELETE FROM auth_session_families WHERE account_id = $1;
   DELETE FROM auth_refresh_credentials WHERE account_id = $1;
   -- 4. Chuyển giao nhân vật sang Tombstone và ẩn danh tên
   UPDATE characters
   SET account_id = '00000000-0000-0000-0000-000000000001',
       name = 'Anonymized_' || SUBSTRING(character_id::text, 1, 8),
       name_key = 'anonymized_' || SUBSTRING(character_id::text, 1, 8)
   WHERE account_id = $1;
   -- 5. Cắt liên kết chứng từ IAP (lưu trữ 10 năm theo Luật Kế toán)
   UPDATE account_iap_entitlements
   SET account_id = '00000000-0000-0000-0000-000000000001'
   WHERE account_id = $1;
   ```
3. Lưu biên bản thực thi vào hồ sơ DPO kèm mã yêu cầu để giải trình cơ quan quản lý (A05 Bộ Công an) khi có thanh tra.

### 2.2 Yêu cầu Trích xuất Dữ liệu (Data Portability)
- Cung cấp file JSON chứa thông tin nhóm A, B, D và E (lịch sử giao dịch IAP).
- Tiến trình chơi game (cấp độ, trang bị, vật phẩm) được xuất kèm dưới dạng bản tóm tắt thân thiện (game summary export).

## Invariants

```text
Category E (IAP receipts) = lưu trữ 10 năm theo Luật Kế toán 88/2015/QH13; cắt liên kết ID
nhân vật sau khi xóa tài khoản = chuyển sang TOMBSTONE_ACCOUNT_ID và đổi tên Anonymized_*
thời hạn thực thi yêu cầu quyền dữ liệu = tối đa 15 ngày (phản hồi thủ tục 2 ngày làm việc)
dữ liệu Category B, C = xóa hoàn toàn (hard delete) khi tài khoản bị xóa
```
