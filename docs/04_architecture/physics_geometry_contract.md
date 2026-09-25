# Deterministic Physics & Geometry Contract
status: LOCKED

## Scope

Hợp đồng vật lý và hình học không gian 2D cho game Thỉnh Thần, đảm bảo tính tất định (determinism) và sự đồng nhất tuyệt đối giữa mô phỏng dự đoán trên Unity client và thẩm quyền quyết định trên Go server.

Tài liệu này là đặc tả cho `IMP-013` (Movement / Collision), `IMP-015` (Skill Geometry), và `IMP-062` (Geometry Exporter).

## 1. Hệ Tọa độ & Đơn vị Đo lường

1. **Không gian 2D Side-Scrolling:**
   - Trục hoành $X$: Hướng ngang. $+X$ hướng sang Phải (Đông), $-X$ hướng sang Trái (Tây).
   - Trục tung $Y$: Hướng đứng. $+Y$ hướng lên Trời, $-Y$ hướng xuống Đất (trọng lực kéo về $-Y$).
   - Trục $Z$: Bằng $0$ trong toàn bộ tính toán va chạm và di chuyển (chỉ dùng cho thứ tự vẽ layer hiển thị trong Unity).
2. **Quy đổi Đơn vị:**
   - `1 unit = 1.0 meter (m)`.
   - Vận tốc tính bằng `m/s`, gia tốc tính bằng `m/s²`.
   - Tọa độ gốc $(0, 0)$: Góc dưới cùng bên trái của vùng bao bản đồ (Map Bounds Min).

### 1.1 Tỷ lệ trình bày và camera tham chiếu

Các hằng số dưới đây là canonical theo ADR-0046:

```text
REFERENCE_VIEWPORT_WIDTH_PX  = 1280
REFERENCE_VIEWPORT_HEIGHT_PX = 720
REFERENCE_ASPECT             = 16:9
ART_PIXELS_PER_METER         = 50
REFERENCE_VIEW_WIDTH_M       = 25.6
REFERENCE_VIEW_HEIGHT_M      = 14.4
REFERENCE_ORTHO_SIZE_M       = 7.2
```

`1280x720` là vùng nhìn của camera và mặt phẳng thiết kế UI, **không phải kích thước map**. Map bình thường rộng `2.0..5.0` lần vùng nhìn (`51.2..128.0m`, tương đương `2560..6400px` ở tỷ lệ tham chiếu); camera cuộn trong bounds của map.

Quy đổi:

```text
world_m = art_px / ART_PIXELS_PER_METER
art_px  = world_m * ART_PIXELS_PER_METER
```

Pixel chỉ dùng cho trình bày/import asset. Mọi mô phỏng, collision, spawn, hitbox và wire coordinate tiếp tục dùng mét/milimet. Cấm lấy `Screen.width`, độ phân giải texture, `Sprite.bounds`, hoặc Transform scale làm dữ liệu vật lý.

## 2. Mô hình Số học & Lượng tử hóa (Quantization)

Để triệt tiêu sai số dấu phẩy động (floating-point divergence) giữa Go (x86_64) và Unity C# (IL2CPP / ARM / x86):
1. **Lượng tử hóa Milimet:**
   - Tọa độ và vận tốc nội bộ được chuẩn hóa với độ phân giải milimet: $1\text{ mm} = 0.001\text{ m}$.
   - Trong replication snapshot, tọa độ được lượng tử hóa thành số nguyên milimet (`x_mm`, `y_mm`). `C2S_MOVEMENT_EDGE` (108) chỉ mang `edge_type`, `direction`, `client_seq`, và `client_mono_ms` — không mang tọa độ (xem `../05_network/messages.md`).
2. **Ngưỡng Epsilon tiếp xúc:**
   - $\epsilon = 0.001\text{ m}$ ($1\text{ mm}$).
   - Hai bề mặt cách nhau $\le \epsilon$ được coi là đang tiếp xúc (Grounded / Wall Contact).

## 3. Kích thước Thực thể (Entity Size Profiles)

Mọi va chạm di chuyển trong game sử dụng trục AABB (Axis-Aligned Bounding Box):

| `size_profile` | Silhouette tham chiếu tối đa | Cell nguồn | Collider W x H | Collider ở 50 px/m | Điểm neo |
|---|---:|---:|---:|---:|---|
| `CHARACTER` | `64x96px` | `96x128px` | `0.8m x 1.8m` | `40x90px` | chân giữa `(0,0)` |
| `MONSTER_SMALL` | `50x50px` | `64x64px` | `0.6m x 0.6m` | `30x30px` | chân giữa `(0,0)` |
| `MONSTER_MEDIUM` | `75x100px` | `96x128px` | `1.0m x 1.4m` | `50x70px` | chân giữa `(0,0)` |
| `MONSTER_ELITE` | `125x150px` | `160x192px` | `1.6m x 2.4m` | `80x120px` | chân giữa `(0,0)` |
| `BOSS_LARGE` | `200x220px` | `256x256px` | `2.4m x 3.2m` | `120x160px` | chân giữa `(0,0)` |
| `WORLD_BOSS` | `250x280px` | `320x320px` | `3.0m x 4.0m` | `150x200px` | chân giữa `(0,0)` |

`Silhouette` là hộp bao pixel không trong suốt của cơ thể ở frame idle tham chiếu, không tính vũ khí rời, bóng đổ, projectile và VFX. Cell nguồn có transparent padding để animation không bị cắt. Pivot của mọi frame là chân giữa; flip hướng chỉ dùng `SpriteRenderer.flipX`, không dùng scale âm hoặc thay pivot.

Nhân vật ở idle/run/jump phải giữ silhouette cao `88..96px`; `96px` là trần tham chiếu. Trang phục, tóc và vũ khí có thể vượt silhouette tối đa `8px` mỗi phía trong cell nguồn nhưng không thay collider. Attack/VFX cần vùng lớn hơn phải tách thành sprite/VFX con.

Kích thước va chạm được tải tĩnh từ catalog; cấm suy ra từ mesh đồ họa hay khung xương hoạt hình (animation rig).

### 3.1 Quy tắc resolve profile

- Character luôn là `CHARACTER`.
- Mỗi row trong `monster_catalog.md` resolve theo bảng canonical `size_profile` của catalog; không có runtime default theo tên.
- Mỗi row trong `boss_catalog.md` khai báo trực tiếp `size_profile`.
- Hurtbox di chuyển dùng collider ở bảng trên. Hitbox kỹ năng/projectile vẫn theo typed geometry trong `skills.md`; không dùng silhouette làm hitbox.
- Kiểm tra reach/hit của skill dùng giao hình với hurtbox đã lượng tử hóa: khoảng cách bề mặt `<=0.001m` là tiếp xúc, `>=0.002m` là tách rời. `SKILL_ORIGIN_Y` và các reach budget thuộc `skills.md`/ADR-0047.
- Mọi giá trị px trong bảng trên là **reference px** (50 px/m). Texture thực được vẽ ở 2x và import `100 PPU` (ADR-0055, `../07_content/presentation_asset_manifest.md` §3), nên kích thước thế giới không đổi.
- Asset/prefab phải có Transform scale `(1,1,1)`. Sai cell, PPU, pivot, profile hoặc scale làm asset validation fail.

## 4. Tham số Động lực học & Thứ tự Tích phân (Integration Order)

Chu kỳ mô phỏng cố định: $dt = 50\text{ ms}$ ($20\text{ Hz}$, `server_tick`).

### 4.1 Hằng số Chuyển động Cơ bản
```text
BASE_RUN_SPEED          = 6.0 m/s (canonical target in movement.md)
FIRST_JUMP_IMPULSE      = +11.0 m/s (canonical target in movement.md)
SECOND_JUMP_IMPULSE     = +10.0 m/s (canonical target in movement.md)
GRAVITY                 = -28.0 m/s²
MAX_FALL_SPEED          = -20.0 m/s (canonical target in movement.md)
AIR_CONTROL             = 0.85
MAX_WALKABLE_SLOPE      = 45.0 độ (độ dốc 1:1)
MAX_STEP_HEIGHT         = 0.30 m (bậc thang tự bước 30cm)
ONE_WAY_DROP_IGNORE_MS  = 300 ms (cửa sổ bỏ qua va chạm khi drop through)
```

### 4.2 Thứ tự Tích phân trong mỗi Tick (50ms)
1. **Tiếp nhận Intent:**
   - Horizontal intent: `C2S_MOVEMENT_EDGE` (108, ADR-0038: PRESS / RELEASE / FLIP cho hướng LEFT / RIGHT).
   - Vertical / jump intent: `C2S_JUMP` (101, discrete command).
   - Drop intent: `C2S_DROP_THROUGH` (102, discrete command).
2. **Cập nhật Vận tốc ngang ($v_x$):**
   - $v_x = \text{intent\_dir} \times \text{final\_move\_speed}$ (nhân với AIR_CONTROL nếu đang trên không).
3. **Cập nhật Vận tốc dọc ($v_y$):**
   - Nếu không chạm đất: $v_y = \max(v_y + \text{GRAVITY} \times dt, \text{MAX\_FALL\_SPEED})$.
   - Nếu nhận `C2S_JUMP` và đủ điều kiện nhảy (grounded hoặc jump count < 2): $v_y = \text{FIRST_JUMP_IMPULSE}$ (hoặc SECOND_JUMP_IMPULSE).
4. **Quét & Xử lý Va chạm Ngang ($X$-Sweep):**
   - Di chuyển $\Delta x = v_x \times dt$.
   - Nếu gặp bậc thang $\le \text{MAX\_STEP\_HEIGHT}$: Tự động nâng nhân vật lên bậc.
   - Nếu gặp tường hoặc dốc $> 45^\circ$: Dừng chuyển động ngang, triệt tiêu $v_x$.
5. **Quét & Xử lý Va chạm Dọc ($Y$-Sweep):**
   - Di chuyển $\Delta y = v_y \times dt$.
   - Nếu va chạm sàn (rơi xuống): Đặt $v_y = 0$, bật cờ `is_grounded = true`.
   - Nếu va chạm trần (nhảy lên): Đặt $v_y = 0$, bắt đầu rơi.
6. **Xử lý Sàn một chiều (One-Way Platforms):**
   - Chỉ có va chạm khi nhân vật rơi từ trên xuống ($v_y \le 0$) và chân nhân vật ở tick trước nằm trên mặt sàn.
   - Khi nhận `C2S_DROP_THROUGH` (102): Bỏ qua va chạm với sàn một chiều đó trong `300ms`.

## 5. Thẩm quyền Server & Dung sai Sửa sai (Server Authority & Reconciliation)

1. **Client Prediction:** Client Unity chạy mô phỏng dự đoán cục bộ theo đúng công thức trên để đảm bảo phản hồi tức thì cho người chơi. Tọa độ client gửi lên chỉ mang tính chất dự đoán/gợi ý (prediction hint); server **tuyệt đối không bao giờ lấy tọa độ client làm chân lý**.
2. **Server Authority:** Server tính toán vị trí thực tế hợp lệ tại mỗi tick 20 Hz. Mọi hitbox chiến đấu, tương tác portal và nhặt đồ chỉ sử dụng tọa độ thẩm quyền của server.
3. **Dung sai Sửa sai (Reconciliation Threshold):**
   - Khoảng cách sai lệch: $\Delta r = \sqrt{(x_{\text{client}} - x_{\text{server}})^2 + (y_{\text{client}} - y_{\text{server}})^2}$.
   - Nếu $\Delta r \le 0.50\text{ m}$: Server tiếp tục cập nhật trạng thái bình thường mà không ép dịch chuyển (tránh giật hình do jitter/latency nhỏ). Tọa độ server vẫn là chân lý duy nhất.
   - Nếu $\Delta r > 0.50\text{ m}$: Server gửi gói tin `S2C_MOVEMENT_CORRECTION` (107) ép client snap về vị trí chuẩn của server và re-simulate các input chưa được xác nhận.
## 6. Hợp đồng Kích thước và Hình dạng Map

### 6.1 Bounds và screen spans

Mỗi playable space khai báo:

```text
space_id
space_kind = WORLD | DUNGEON | FINALE | PVP | GUILD_WAR
bounds_m = { min_x=0, min_y=0, max_x, max_y }
reference_span = { width_screens, height_screens }
layout_profile
```

Với viewport tham chiếu `25.6m x 14.4m`:

```text
max_x = width_screens  * 25.6m
max_y = height_screens * 14.4m
reference_extent_px = (max_x * 50, max_y * 50)
```

Normal-world width phải nằm trong `2.0..5.0` screens. `bounds_m` chỉ là envelope ngoài cho simulation/camera; toàn bộ hình chữ nhật không mặc nhiên đi được. `layout_profile` và geometry đã export quyết định sàn, tường, tầng cao, nhánh, vòng nối và vùng cấm thực tế.

Mỗi FIELD/dungeon phải có một main route liên tục từ entry đến exit/final stage, ít nhất một optional branch, và mọi nhánh cụt dài hơn `0.5` screen phải kết thúc bằng content anchor (chest, objective, elite, fishing spot hoặc lore interaction). Mỗi map dùng đúng topology trong catalog; cấm thay tất cả bằng một hành lang ngang.

### 6.2 Camera

- Camera gameplay có orthographic size `7.2m` tại vùng nhìn 16:9.
- Thiết bị rộng hơn 16:9 giữ chiều cao `14.4m` và cho thấy thêm chiều ngang, tối đa tỷ lệ 21:9; phần vượt 21:9 dùng pillarbox.
- Thiết bị hẹp hơn 16:9 dùng letterbox để giữ vùng gameplay 16:9; HUD đặt trong safe area.
- Camera clamp theo authored camera regions nằm trong `bounds_m`, không clamp theo kích thước sprite nền.
- Parallax/background có thể vượt bounds nhưng không tạo collision hoặc spawn hợp lệ.

## 7. Hợp đồng Xuất Hình học (Unity Geometry Exporter)

1. **Công cụ Xuất:** Script Unity Editor `ThinhThan.Core.Geometry.Editor.GeometryExporter` (assembly `ThinhThan.Core.Geometry.Editor`, IMP-062) quét các Collider trong Scene (gắn tag `ServerGeometry`) of the collision-only authoring scene `client/Assets/Scenes/Collision/<space_id>.unity` (IMP-062); visual scenes of IMP-072/IMP-105 contain no `ServerGeometry` colliders (ADR-0068).
2. **Định dạng Xuất:** File JSON lưu tại `server/internal/sim/spatial/maps/<space_id>.geom.json`.
3. **Cấu trúc dữ liệu:**
   ```json
   {
     "space_id": "map.lang_da.bo_ruong",
     "space_kind": "WORLD",
     "layout_profile": "IRRIGATION_BRAID",
     "content_revision": "sha256_hash",
     "bounds": { "min_x": 0.0, "min_y": 0.0, "max_x": 76.8, "max_y": 18.0 },
     "segments": [
       { "id": 1, "kind": "SOLID_GROUND", "x1": 0.0, "y1": 2.0, "x2": 24.0, "y2": 2.0 },
       { "id": 2, "kind": "ONE_WAY_PLATFORM", "x1": 27.0, "y1": 5.0, "x2": 35.0, "y2": 5.0 },
       { "id": 3, "kind": "SLOPE", "x1": 35.0, "y1": 5.0, "x2": 43.0, "y2": 9.0 }
     ]
   }
   ```
4. **Bất biến:** Server Go chỉ đọc file `.geom.json` này; tuyệt đối không import Unity runtime DLLs hay phụ thuộc vào file binary của Unity.
5. Export fail nếu `space_id`, `space_kind`, `layout_profile`, bounds, camera region hoặc logical anchor không khớp catalog; segment/anchor nằm ngoài bounds; hay một FIELD/dungeon mất main route.

## Invariants

```text
1 unit = 1.0m; lượng tử hóa mm (0.001m); epsilon = 0.001m
reference viewport = 1280x720; ART_PIXELS_PER_METER = 50
reference camera = 25.6m x 14.4m; orthographic size = 7.2m
normal-world map width = 2.0..5.0 reference screens, never one screen by default
fixed tick = 50ms (20 Hz); trọng lực g = -28.0 m/s²
character silhouette <= 64x96px; collider = AABB 0.8m x 1.8m; điểm neo ở chân giữa
dung sai sửa sai vị trí = 0.50m
server geometry là file json tĩnh theo space_id; cấm suy ra từ sprite hay animation
map bounds là envelope; layout_profile + exported geometry mới quyết định vùng đi được
```
