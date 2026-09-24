# Client Experience & Presentation Contract
status: LOCKED

## Scope

Hợp đồng quy chuẩn trải nghiệm người dùng (UX), máy trạng thái giao diện (UI State Machine), bố cục màn hình (HUD Layout), sơ đồ gán phím (Input Action Mapping), vùng an toàn thiết bị di động (Safe Area) và các trạng thái lỗi/chuyển vùng cho Unity client (Unity 6000.6.1f1).

Tài liệu này là đặc tả cho `IMP-065` (Client Bootstrap / Session), `IMP-066` (Input / UI State Machine) và các task giao diện người dùng.

## 1. UI State Machine Tổng thể

Toàn bộ client vận hành dựa trên một Finite State Machine cấp cao duy nhất:

```text
[BOOT]
  -> Kiểm tra phiên bản & nạp Addressables Catalog
[PATCHING_UPDATE]
  -> Tải cập nhật tài nguyên nếu có; hiển thị % và thanh tiến trình
[AUTH_TITLE]
  -> Màn hình đăng nhập liên kết (Apple / Google / Steam); nút Chạm để vào
[CHARACTER_SELECT]
  -> Chọn hoặc tạo nhân vật (tối đa 3 nhân vật; cấm xóa nhân vật)
[MAP_LOADING_TRANSFER]
  -> Màn hình chờ tải cảnh chuyển vùng (giữ kết nối WSS, tải Addressables nhóm bản đồ mới)
[IN_WORLD]
  -> Vào thế giới thực tế, hiển thị HUD chiến đấu đầy đủ
[DISCONNECTED_RETRY]
  -> Mất kết nối, hiển thị đếm ngược thử lại tự động (1..5s)
```

## 2. Bố cục Giao diện In-World (HUD Layout)

Độ phân giải/viewport tham chiếu chuẩn: `1280 × 720` (`16:9`, ADR-0046). Canvas Scaler: `Scale With Screen Size`, Reference Resolution = `1280 × 720`, Match = `0.5`, Reference Pixels Per Unit = `100`; UI sprites are 2x textures imported at `200 PPU` (ADR-0055) so they render at reference size.

Đây chỉ là vùng nhìn camera/UI, không phải kích thước map. Một normal-world map rộng `2..5` viewport và camera cuộn theo nhân vật trong authored camera regions; kích thước cụ thể và topology từng map nằm trong `world_route_catalog.md`.

```text
+-----------------------------------------------------------------------------------+
| [Avatar] Lv.60 TenNhanVat                      [MiniMap: Lang Da - Khu 1] [Ping]  |
| HP: [=======================] 45000/45000      [Quy 3/3]                 [Menu =] |
| MP: [=======================]  8200/ 8200                                         |
| [Buff1][Buff2][Debuff1]                        [Nhiệm vụ:                         |
|                                                  - Diet 10 Chuot Dong (8/10)]     |
|                                                                                   |
|                                                                                   |
| [Chat Dock - 3 dong]                                                              |
| [Tab: The gioi / Bang]                                                            |
|                                                       (Active 4)  (Active 5)      |
|     ( ^ )                                        (Active 2)  (Active 3)           |
| ( <       > ) Virtual Joystick               (Active 1)                           |
|     ( v )                                         [== BASIC ATTACK ==]   (Jump ^) |
+-----------------------------------------------------------------------------------+
```

### Chi tiết các Cụm HUD:
1. **Top-Left (Thông tin Nhân vật):** Ảnh đại diện theo phái, cấp độ, thanh máu (Đỏ), thanh mana (Xanh lam), hàng icon trạng thái (Buff hình khiên/tròn viền xanh, Debuff hình tam giác nhọn viền đỏ kèm số giây đếm ngược).
2. **Top-Right (Bản đồ & Kênh):** Tên bản đồ hiện tại, số hiệu kênh (`Khu 1` .. `Khu 30`), biểu tượng cường độ mạng (Ping ms), nút mở Menu chính.
3. **Right-Center (Bảng Theo dõi Nhiệm vụ):** Tối đa 3 nhiệm vụ hiển thị (1 chính tuyến màu vàng cam, 2 phụ tuyến màu xanh ngọc).
4. **Bottom-Left (Di chuyển & Chat):** Cần điều khiển ảo (Virtual Joystick) trên màn hình cảm ứng; dock chat thu nhỏ hiển thị 3 dòng mới nhất, chạm vào để mở toàn màn hình chat.
5. **Bottom-Right (Hệ thống Kỹ năng):** Nút Đánh Thường (Basic Attack) kích thước lớn nhất, 5 nút Kỹ Năng Kích Hoạt (Active 1..5) xếp hình cánh cung quanh nút đánh thường, nút Nhảy (Jump).

## 3. Sơ đồ Gán Phím (Input Action Mapping)

Sử dụng Unity Input System (`com.unity.inputsystem 1.20.0`):

| Thao tác | Bàn phím & Chuột (PC) | Tay cầm (Gamepad / Xbox) | Màn hình cảm ứng (Mobile) | Gói tin mạng gửi đi |
|---|---|---|---|---|
| **Chạy Trái / Phải** | `A` / `D` hoặc `←` / `→` | Cần gạt trái (Left Stick) | Kéo Virtual Joystick | `C2S_MOVEMENT_EDGE` (108, PRESS \| FLIP) |
| **Dừng di chuyển** | Nhả phím di chuyển | Nhả cần gạt trái | Nhả Virtual Joystick | `C2S_MOVEMENT_EDGE` (108, RELEASE) |
| **Nhảy (Jump)** | `Space` hoặc `W` | Nút `A` | Chạm nút Nhảy | `C2S_JUMP` (101) |
| **Rơi sàn một chiều** | `S + Space` | `Down + A` | Kéo Joystick xuống + Nhảy | `C2S_DROP_THROUGH` (102) |
| **Đánh Thường** | `J` hoặc Chuột Trái | Nút `X` | Chạm nút Đánh Thường | `C2S_BASIC_ATTACK` (201) |
| **Kỹ năng Active 1..5** | `K`, `L`, `U`, `I`, `O` | `Y`, `B`, `RB`, `RT`, `LB` | Chạm nút Active 1..5 | `C2S_SKILL_USE` (200) |
| **Tương tác NPC / Nhặt** | `F` | Nút `X` (khi gần vật phẩm) | Nút Tương tác ngữ cảnh | `C2S_INTERACT` (103) |
| **Đổi Mục tiêu** | `Tab` | `R3` (Nhấn cần phải) | Chạm trực tiếp vào quái | Xử lý client targeting |
| **Mở Chat** | `Enter` | Nút `Back` / `View` | Chạm vào Chat Dock | N/A (Mở UI nội bộ) |

## 4. Vùng An toàn & Ma trận Thiết bị (Safe Area & Device Matrix)

### 4.1 Resolution và aspect ratio

- Desktop mặc định mở cửa sổ `1280x720`; người chơi được phép chọn fullscreen/native hoặc độ phân giải lớn hơn.
- Gameplay camera tại 16:9 luôn nhìn `25.6m x 14.4m`, orthographic size `7.2m`, theo `physics_geometry_contract.md`.
- Màn hình rộng hơn 16:9 giữ chiều cao world view và mở rộng chiều ngang tối đa 21:9; phần vượt 21:9 dùng pillarbox.
- Màn hình hẹp hơn 16:9 dùng letterbox để không cắt gameplay/HUD hoặc làm thay đổi vùng nhìn cạnh tranh.
- UI scale theo Canvas Scaler và safe area. Không đặt vị trí HUD bằng pixel tuyệt đối ngoài mặt phẳng tham chiếu `1280x720`.
- Thay đổi độ phân giải không thay `ART_PIXELS_PER_METER`, collider, movement speed, map bounds hoặc khoảng cách kỹ năng.

1. **Khắc phục Tai thỏ / Dynamic Island / Nốt ruồi camera:**
   - Sử dụng component `SafeAreaFitter` trên Canvas gốc.
   - Truy vấn `Screen.safeArea` khi khởi động và khi xoay màn hình để tự động chèn padding lề trái/phải tối thiểu `48px` trên các thiết bị có viền khuyết tật.
2. **Ngăn chặn xung đột cảm ứng (Touch Gesture Conflicts):**
   - Vùng Virtual Joystick chiếm nửa dưới bên trái màn hình (`X: 0..0.4`, `Y: 0..0.5`).
   - Chạm trong vùng Joystick không kích hoạt mở Chat Dock hay bấm vào thông tin nhân vật.

## 5. Trạng thái Lỗi, Chuyển Vùng & Mất Kết Nối

1. **Chuyển vùng bản đồ (`TRANSFERRING_MAP`):**
   - Khóa toàn bộ input di chuyển và chiến đấu của người chơi.
   - Hiển thị màn hình mờ với tranh dân gian đặc trưng của vùng đất sắp đến.
   - Thanh tiến trình hiển thị tiến độ tải Addressables.
   - Nếu quá ngân sách `30s` (hoặc `120s` với phó bản), tự động hủy và hiển thị thông báo `Chuyển vùng thất bại, đang quay lại điểm an toàn`.
2. **Mất kết nối mạng (`DISCONNECTED`):**
   - Hiển thị popup modal giữa màn hình: `Mất kết nối tới máy chủ. Đang thử kết nối lại... (Lần 1/5)`.
   - Nút `Thử lại ngay` và nút `Thoát ra màn hình chính`.
3. **Đăng nhập đè phiên (`SESSION_REPLACED`):**
   - Nhận message ID 11 từ server: Lập tức đóng kết nối WSS.
   - Hiển thị thông báo không thể đóng: `Tài khoản của bạn đã được đăng nhập từ một thiết bị khác.` kèm nút `Đồng ý` để quay về màn hình Title.

## 6. Khả năng Tiếp cận (Accessibility)

1. **Tương phản số liệu:** Toàn bộ chữ số sát thương bay (floating combat text) và thanh máu phải có viền đen dày `2px` (outline) để đọc rõ trên mọi nền địa hình sáng/tối.
2. **Phân biệt bằng hình dạng:** Không bao giờ dùng màu sắc đơn độc để biểu thị trạng thái.
   - Buff có khung viền tròn và icon mũi tên hướng lên.
   - Debuff có khung viền tam giác và icon đầu lâu/vết rách hướng xuống.
3. **Kích thước nút bấm tối thiểu:** Nút cảm ứng trên di động có kích thước tối thiểu `44 × 44` points để người chơi không bị bấm trượt.

## Invariants

```text
độ phân giải tham chiếu = 1280x720; Canvas Scaler match = 0.5
1280x720 là viewport, không phải map size; normal-world map rộng 2..5 viewport
desktop default window = 1280x720; fullscreen/native vẫn được hỗ trợ
SafeAreaFitter bắt buộc để tránh tai thỏ camera
input di chuyển gửi qua C2S_MOVEMENT_EDGE (108)
SESSION_REPLACED = modal ngắt kết nối không thể đóng
icon trạng thái phân biệt bằng hình dạng (tròn/tam giác), không chỉ bằng màu
```
