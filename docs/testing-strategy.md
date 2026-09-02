# FolioPath 测试策略

## 状态

本文同时定义目标验证体系和当前已落地的 Go 验证面。仓库已有路径边界、媒体库、
scanner、SQLite 单元测试，贯通 files → scanner → SQLite 的临时目录集成测试，OpenAPI
契约测试，真实 HTTP 边界、合成媒体 fixture 和显式容量测试。正式 `cmd/foliopath`、
composition root、SQLite 生命周期、health、认证及媒体库/扫描/浏览/缩略图/搜索/原内容
HTTP handler 已可启动和测试；React 认证、媒体库/扫描、浏览/预览及搜索/完整查看器切片
均已有真实浏览器 E2E，发布镜像和最终浏览器/真机矩阵仍待 Stage 5。仓库已有固定 Node/npm、
确定性 OpenAPI TypeScript 生成、strict typecheck、依赖 audit 和唯一 client 边界。
仓库不再为 PR 或 `main` push 运行自动 CI；验证由开发者在本地执行，过去的双架构、
FS-05 runtime/recovery 和 SBOM/license 结果只作为历史证据保留。
只有实际执行成功的目标才能声称可用。

当前证据分别见 [FS-01 路径边界](spikes/fs-01-path-boundary.md)、
[FS-02 SQLite/generation](spikes/fs-02-sqlite-generation.md)、
[FS-03 媒体矩阵](spikes/fs-03-media-matrix.md) 和
[FS-04 容量基线](spikes/fs-04-capacity-baseline.md)与
[FS-05 运行恢复](spikes/fs-05-runtime-recovery.md)。FS-01 路径和 FS-02 当前正确性 scope
已通过；FS-04 的 Stage 0 扫描/索引范围通过，FS-03 与 FS-04 完整产品/发布范围保持
Conditional，不能替代完整媒体、容量、浏览器
和发布验证。FS-01 的 Stage 0 范围包括原生 Linux amd64/arm64 `openat2` mount 拒绝和 HTTP
harness；生产 handler/auth 与发布 volume/unmount 分别由后续 Backend/Release Gate 强制。

## 质量目标

测试优先保护以下不可逆或高风险边界：

1. FolioPath 永不修改、移动、改名或删除原始媒体。
2. 任意 API 输入都不能让读取逃出 `/library` 或目标媒体库。
3. 离线、权限失败和中断扫描不会误清已有索引。
4. 大型目录的扫描、列表和缩略图工作受内存、并发和事务边界限制。
5. 游标分页、虚拟列表和媒体 Range 在数据变化与取消请求时仍保持可用。
6. 发布镜像可备份、恢复和升级，不依赖开发机环境。

## 测试层次

### Go 单元测试

与源码同包，覆盖快速、确定且无外部进程的行为：

- 路径清理、组件级包含关系、重叠媒体库判定、唯一名称、根路径不可变和错误码。
- 扫描 generation 状态机、协作取消、失败是否具备清理资格、24 小时调度、跳过统计、任务幂等与重试退避。
- 游标编码/校验、稳定排序、查询过滤和 MIME/格式判定。
- 缓存键、源指纹、变换版本、10 GiB 默认配额、LRU 清理水位和临时文件提交规则。
- 单管理员原子初始化、密码验证、会话过期/撤销、退出、CSRF 和登录限流。
- HTTP handler 的请求校验、状态映射和错误脱敏（使用 fake service）。
- 原媒体 full/HEAD/单 Range/条件请求、空文件、读取 admission、取消和源故障状态。

路径解析、游标解码和媒体头解析加入 Go fuzz tests。任何 fuzz 失败输入都保存为回归样例。

当前已实现的 Go 单元覆盖包括：

- 共享相对路径策略、重复编码、无效 UTF-8/NUL、symlink、根移除/替换、A → B → A 身份回归、特殊节点、遍历取消和错误脱敏；
- Linux `openat2` 的失败关闭映射、`RESOLVE_BENEATH | RESOLVE_NO_SYMLINKS |
  RESOLVE_NO_XDEV` 策略与旧子目录 FD 移出边界后的拒绝；
- 媒体库根规范化、组件边界重叠判断、NFC 显示名、NFKC + Unicode full case-fold 唯一键、
  无变化/并发改名自身表示与根不可变；
- 固定 MVP 扩展名/MIME 正向矩阵、SVG/HEIC/HEIF/AVIF/RAW 负向矩阵与系统目录跳过清单；
- 真实文件 SQLite、Goose migration、WAL/外键/busy timeout、SQLite 安全版本门槛、integrity/foreign-key/checkpoint；
- 认证 version 1 → 2 migration、并发单管理员约束、密码 verifier 字段、会话/CSRF 摘要、
  正的绝对期限、禁止明文令牌列和 session 级联；
- Unicode NFKC/full case folding 管理员身份规范化、8～128 Unicode 字符 setup 密码边界、
  前后端字符计数一致、Argon2id 严格参数和正确/错误密码验证、
  setup 状态机、进程内并发门、SQLite 原子二次初始化拒绝，以及 composition root 重启持久化；
- 初始化与初始会话同事务回滚、统一登录失败与虚拟 verifier、高熵 Cookie/CSRF 摘要、
  7 天绝对过期、auth version/禁用失败关闭、last-seen、退出撤销、清理、Cookie 属性，以及
  登录和会话跨 composition root 重启；
- 认证 HTTP status/setup/login/session/logout、严格 JSON、完整同源 Origin、重复 Cookie
  拒绝、业务 API 默认认证、状态修改 CSRF、统一 no-store/错误映射、Secure Cookie 真实 TLS
  判定，以及有界并发安全的直连 peer 固定窗口限流；
- 真实 composition root HTTP 流覆盖首次 setup、认证 status/session、跨进程重启恢复、
  大小写兼容重新登录、CSRF logout、过期 Cookie 和撤销后 `session_expired`；
- S1-105 安全矩阵覆盖五类认证操作的内部错误脱敏、CSRF 失败不触发退出、绝对期限前
  1 ms/等于期限边界、8 路真实 HTTP 并发 setup 仅一个成功、重复 setup 关闭、未知账号与
  错误密码外部响应一致，以及 32 路真实 HTTP/SQLite session 并发读取与 last-seen 更新；
- generation 的失败、取消、离线、受控重启、原子 finalize 回滚、活动扫描竞争与 complete/cancel 竞态；
- version 3→4 扫描契约迁移、queued/running/terminal phase 字段、默认 24 小时/关闭 typed
  schedule、heartbeat/lease/attempt 边界、每 run 50 个 issue group 上限和非法值拒绝；
- S2-102 通用 jobs worker 的恢复先于领取、全局最多 2 个并发、容量释放后继续消费、
  heartbeat/lease 与 durable cancel 传播；SQLite 公平领取、revision 稳定 heartbeat、
  第二次 requeue/第三次 interrupted，以及正式 composition 自动完成 creation scan；
- S2-103 真实文件 walker 与 production creation worker 覆盖根、空、隐藏、嵌套和系统跳过
  目录，分别记录跳过目录/文件，并发布逐级直接/递归计数；128 层链、同库循环、跨库
  目录/资产损坏、当前代次条目指向同库陈旧目录等损坏在 stale cleanup 前失败关闭且不
  丢失当前行或影响另一媒体库。
- S2-104 以真实 version 5 catalog 验证 migration 6 的 source fingerprint 回填，并通过
  真实文件 walker、SQLite 与 production creation worker 验证固定格式候选、同路径 ID/
  fingerprint 保持、纳秒 mtime 变化失效、重命名新 ID、processed counter 和成功后的
  stale 收敛；架构测试禁止复制候选注册表、fingerprint 编码或 stale SQL owner。
- S2-105 覆盖取消、offline、根 identity 变化、部分目录不可读与 nested mount 的稳定
  terminal code，验证所有非成功代次保留最后可靠索引；真实 production composition
  验证启动时按 ID 分页 admission/coalesce `startup` scan，并验证启动后才到期的 lease
  仍由周期恢复重新排队且最终收敛。架构测试固定 startup admission、恢复循环与 SQLite
  keyset owner。
- S2-106 覆盖唯一 256 active/256 batch 边界、跨 SQLite 连接最后一个 admission 名额
  竞态、2-worker 三媒体库公平消费、深目录/损坏拓扑和真实 SQLite 满页；失败记录稳定
  `database_unavailable`、保留最后可靠 generation，解除限制后可完整恢复。
- S2-107 覆盖扫描历史的库绑定防篡改 keyset cursor、详情 ETag/304、安全 issue、queued/
  running 协作取消、设置 If-Match，以及默认 24 小时/可关闭 scheduler；真实 composition
  通过认证 HTTP 消费设置、历史、详情和 terminal 取消。
- S3-001 契约测试固定 indexed root 的公开映射、root-to-current breadcrumb（1～2049 项）、
  direct/recursive 目录 scope、自然名称/修改时间排序 tuple、reliable generation-bound
  cursor、跨库目录 404、offline preserved index 和 browse 请求不遍历文件系统。S3-002 的
  catalog/SQLite 测试进一步固定自然数字排序、完整 tuple keyset、query fingerprint、
  跨 scope/generation 失效、offline availability、migration 6→7 回填和 context
  cancellation；architecture test 阻止生产 SQLite 浏览查询使用 `OFFSET`。
- S3-104 前端状态测试固定首屏 skeleton、普通 empty 与 offline-empty 的互斥、首屏错误
  重试、下一页错误保留已加载项目，以及 pending-only 2.5 秒轮询和 4 页请求上限。真实认证
  Chromium 链继续使用后端 ready WebP，同时用受控契约响应覆盖 pending→failed、空、
  错误恢复和 offline；每个稳定状态检查无页面横向溢出及 axe serious/critical。
- S3-105 共享预览测试固定图片/原生视频分支、基本信息、前后项/关闭、加载失败降级和
  360～620px 键盘宽度边界；真实认证 Chromium 链使用原内容 endpoint 验证图片预览、
  按钮禁用、separator 调宽和关闭。视频 Range/codec/离线/删除矩阵仍归 S4-007～009。
- S3-106 交互测试固定选择与预览的独立状态、未固定单击跟随、固定后单击仅选择和双击
  切换、取消固定立即跟随、固定状态文案及 Escape。真实认证 Chromium 链验证固定态只
  存在一个活动媒体、关闭后虚拟滚动锚点与卡片语义按钮焦点恢复；播放资源容量预算仍归
  S3-107，完整 Range/codec/离线/删除矩阵仍归 S4-007～009。
- S4-007 以唯一纯策略测试固定 source availability 优先于 probe/playback、GIF 不被阻断、
  ready thumbnail poster；共享预览/查看器组件测试固定原生视频属性、状态卡、重试与导航，
  页面测试固定 offline 和 `asset_not_found`。1440×900 原型/实现同状态视觉对照及 390×844
  状态操作无遮挡证据进入 `web/design-qa.md`。
- S4-008 的 Desktop Chrome 与 Pixel 5 项目固定查看器关闭初始焦点、工具条焦点下的
  `I`/方向键/Escape、原生视频控件冲突保护、触摸信息开关和离线重试。真实合成 MP4
  由受控 endpoint 返回 `206 Content-Range`，测试必须观察浏览器发出的
  `Range: bytes=…`；unsupported codec、offline、deleted 继续分别验证无播放器、有效
  重试与无效重试。两个视口都检查页面无横向溢出及 axe serious/critical 为零。
  Firefox、Safari/WebKit 具体发布版本和真机性能归 Stage 5 发布矩阵。
- S4-009 把一次性真实 SQLite/Go/只读媒体链延伸为搜索 → 非模态预览 → 完整查看器 →
  原搜索 URL 与卡片焦点恢复；同一链继续验证固定预览筛选保留、scope 历史、主题、
  responsive、overflow 和 axe，并形成 Stage 4 Integrated Done。
- S3-107 容量测试使用 100,000 个稳定资产 ID 验证默认视口挂载不超过 64 个 DOM 项、
  首屏不预取、末端仅允许一个在途 cursor 请求、远距离虚拟锚点恢复和 12 帧焦点重试。
- Stage 5 的 `make test-browser-capacity` 使用同一 100,000 项工作台主档，在 Chromium、
  Firefox 与 WebKit 记录连续滚动 FPS、帧间隔 P95、浏览器进程树 RSS 和挂载 DOM，并
  强制 `S5-005B` 预算；它不替代真实浏览器、读屏或移动物理设备签署。
  原生视频 rerender 必须卸载旧节点且只留下一个 video。组件工作台同一主档在真实
  1280×720 Chromium 中记录首屏 42 项、末端 40 项、`aria-posinset` 99,961～100,000
  与末项焦点；代表性低性能客户端 FPS/RSS 仍由 Stage 5 发布 Gate 固定。
- S3-108 真实纵向 E2E 从一次性 setup/建库/扫描进入 browse，覆盖真实 WebP/原内容、
  direct/recursive/source link、URL 历史恢复、grid/masonry 记忆、移动目录抽屉、
  非模态预览调宽/固定/双击/Escape/焦点，以及分页错误保留已加载项并只在显式重试后
  续页。受控 skeleton、pending→failed、首屏错误、empty、offline 与稳定浅/深主题均
  检查无页面横向溢出和 axe serious/critical。

缓存、扫描调度和 fuzz 仍是目标项；认证的故障、并发和时间矩阵已由 S1-106 Gate 复核为
Backend Ready。媒体库的安全目录 cursor、生命周期、路径故障矩阵、重启移除和逐字节原媒体
不变已由 S2-007 Gate 复核为 Backend Ready。S2-102 已接入生产扫描 worker，S2-103～106
已完成目录/计数、媒体增量收敛、故障/重启恢复、容量矩阵与扫描 Backend Ready；
S3-004 已增加 production govips/FFmpeg adapter、派生键、原子缓存发布、SQLite 状态和
真实 scanner→source→cache→database 组合测试；S3-005 已增加 durable lease/retry/
公平领取、fingerprint 原子失效、90%→80% LRU、512 MiB 余量和真实 worker lifecycle；
S3-006 已增加 256 MiB 图片、100 MP/32,768 px、govips runtime、FFmpeg 单线程/
进程组取消/工具输出 cap、取消安全点和真实 8 MiB tmpfs `ENOSPC` 矩阵；2026-08-06 回归将
视频源上限从 4 GiB 提高到 1 TiB，覆盖 4 GiB 以上实际工具调用、`source_too_large`、
probe/poster 独立 60 秒预算、未知 duration 以及损坏/缺少 moov/工具故障分类。资产/缩略图 HTTP、
Stage 1 认证、Stage 2 媒体库/扫描、Stage 3 浏览/非模态预览和 Stage 4
搜索/完整查看器浏览器流程已通过各自 Integrated Done Gate；发布网络、存储、最终
浏览器/真机和候选镜像边界仍在 Stage 5。

### 前端单元与组件测试

- URL 状态与路由往返：媒体库、目录、递归、搜索、过滤、排序。
- 目录树、面包屑、媒体卡片、扫描状态、离线提示和删除确认。
- 单管理员初始化/登录/过期流程，中英切换和浏览器语言默认值。
- 键盘导航、焦点恢复、可访问名称、动态状态公告和 reduced-motion 分支。
- Query 缓存更新、分页去重、错误重试边界和取消过期请求。
- 瀑布流尺寸估算与虚拟化适配逻辑，不以大量脆弱 DOM 快照代替行为断言。
- 查看器适应视口、缩放/平移、1:1、全屏与基本信息；验证没有完整 EXIF、显式下载按钮或移动滑动手势。

### 集成测试

放在 `tests/integration`，使用临时目录、真实 SQLite 和可控的媒体工具替身或合成 fixture：

- 创建媒体库、首次扫描、再次扫描、删除与离线恢复的完整后端路径。
- version 2→3 追加迁移、库/首次 queued scan/幂等记录同事务 commit/rollback、唯一
  creation scan/active removal、ETag revision、摘要长度与至少 24 小时幂等保留。
- 已认证目录选择器的真实 composition、直接目录/普通文件过滤、隐藏与 Unicode 名称、
  numeric natural order、默认/最大页面、有界 `limit+1` 选择、opaque query-bound cursor
  的续页/篡改/跨 scope 拒绝，以及 symlink/mount/unavailable 公开错误映射。
- 唯一媒体库名称、两个独立 Store 并发名称/根冲突仅一方成功、只允许改名、拒绝根路径
  修改、目录选择器相同/祖先/后代冲突标注与权威快照失败关闭，以及 24 小时可配置调度与
  协作取消。
- 正式认证 HTTP/composition/SQLite/removal worker 删除链路，比较删除前后的完整 synthetic
  媒体树 entry、模式、symlink target 和普通文件完整字节，并验证应用缓存清理与 SQLite
  bounded cleanup 在重启后幂等续作。
- generation 批量 upsert 与仅在完整成功后清理旧记录。
- 进程中断后的 `running` 任务恢复和原子缓存写入。
- migration 10 搜索键/FTS 升级回填、Unicode NFKC/full fold、中文/英文/组合字符/变音符号、
  1～2 字符和标点字面查询、三种 scope、kind/mtime 半开区间、离线索引、自然/mtime
  keyset、query/generation/global-revision cursor 与请求取消；S4-002 已覆盖正确性主矩阵，
  S4-003 已在 100k/10k macOS 与 Linux 2 CPU/4 GiB 档覆盖扫描并发、FTS/短词/全局、
  两页 keyset、取消/先完成收敛、integrity rebuild 和失败关闭。
- 真实文件变化：新增、修改、删除、重命名、权限错误、深目录和空目录。
- 所有可读目录的直接/递归计数，以及隐藏项、系统派生目录和回收站的跳过清单/统计。
- HTTP 条件请求、`HEAD`、合法/非法 Range、`416`、客户端取消。
- libvips/FFprobe/FFmpeg 的输入大小、像素炸弹、超时、进程树取消、工具输出和并发限制；
  与 CLI 的调用使用参数数组而非 shell，Linux tmpfs 注入真实 `ENOSPC`。图片矩阵还要用
  探测到的真实格式而非扩展名分支，分别覆盖非 JPEG 100 MP、真实 JPEG ≤100 MP 原路径、
  100～180 MP shrink-on-load、超过 180 MP、0 字节、截断数据和未知工具故障；诊断断言复用
  既有 error/stage/reason/tool，不要求 API、schema、migration 或 transform version 变化。
- JPEG 容错增量矩阵必须覆盖：正常 JPEG 严格路径输出不变；真实且≤100 MP 的
  `premature end` / `incomplete scan` / `corrupt JPEG data` 输入只在严格失败后
  单次容错，有效 WebP 进入 ready/succeeded 并在成功 attempt 中记录内部
  `output_validation / decode_recovered / libvips` warning；伪装 JPG、0 字节、超过
  100 MP、未知错误和容错无效产物仍失败。验证原文件 hash/mtime 不变、warning 不会
  出现在 failed-only 列表，且前端只把计数描述为“本轮尝试”，不声称具有全部终身历史。
- MPEG-TS 增量矩阵必须用最终镜像验证：已按 `.mp4/.mov/.mkv/.avi` 收录、真实容器
  为 `mpegts` 的受控 fixture 可完成 probe/poster/storyboard，`playback_status=unknown`；
  `.ts/.mts/.m2ts` 仍不进入索引合同。未允许容器、无效 TS、无视频流和解码器缺失
  仍失败，并证明原文件不变、不从派生成功推断浏览器可播放。
- 工具输出矩阵必须证明退出 0 且 stdout 产物有效时，超过 64 KiB 的 stderr 只被截断而不
  误判失败；同时保留非零退出、无效/超限 stdout、脱敏日志与 attempt 不含原始 stderr 的反例。
- 数据库迁移从每个已发布版本前进，并验证失败时不部分就绪。
- `POST-MVP-3` 诊断/恢复覆盖 newest-first 有界列表、library-relative path、transient/
  permanent 重试策略、每 job 最近 10 次 attempt 保留、FFmpeg 输出到稳定原因的脱敏分类、
  旧失败无详情降级，以及消息 failure watermark 对确认和新失败的重新触发；
  重新排队、permanent 保留、256 admission 上限、CSRF/认证和安全错误；release-info 覆盖
  stable semver、draft/prerelease 忽略、5 秒超时、1 MiB cap、六小时缓存与无缓存降级；系统
  日志覆盖分级、白名单字段、5,000 条保留、keyset 分页和认证；Web 覆盖媒体库扫描记录/处理
  结果空错状态与结果分页、系统日志筛选、关于更新状态，以及消息中心派生任务 1.5 秒轮询、
  聚合计数、Escape/外点/清除完成而保留 active/attention。

POST-MVP-2 自动发现的 WCH-S2 后端证据还必须覆盖：

- fresh schema 与 migration 11→12、CHECK/FK/唯一键、运行 lease 恢复以及设置 ETag；
- create/close-write/move-in/rename/delete、新建空目录、慢写稳定窗口和事件合并；
- watcher 删除与 full-scan stale cleanup 均在索引删除前登记 grid/storyboard 缓存删除；
  failed/cancelled/offline 扫描不得登记，提交后缓存 worker 被主动唤醒并可幂等重试；
- running 期间的新事件递增水位且不能被旧 claim 删除，完整扫描与定向校准按库互斥；
- 父目录不可完整枚举、root replacement、symlink、nested mount、unmount、overflow、
  watch `ENOSPC`、强杀、重启和 full-scan 最终收敛；
- 原生 linux/amd64 与 linux/arm64 的 `openat2`/inotify 行为；模拟架构的 `ENOSYS`
  只证明失败关闭；
- 1 万目录 watch、10 万媒体和 100k burst 下的 RSS、FD、队列、SQLite 写放大、跨库公平、
  自动发现 P95 及浏览/搜索 P95；
- 认证 `GET /api/v1/catalog/state` 的 `200/304/401/429/500`、ETag/no-store、错误脱敏和
  revision 与搜索 cursor revision 的独立性。
- 可见媒体库/浏览/搜索页面每 5 秒条件检查、隐藏暂停、`304` 不刷新、revision 变化只裁剪并
  重取 active scope 第一页，以及自动发现状态/净化原因的中英文呈现。

WCH-S1 冻结的进程内上限是：每实例最多 32,768 个目录 watch、8,192 个未合并事件、
4,096 个 dirty 目录、全局最多 2 个定向执行、每库最多 1 个；基础 debounce 750ms，
持续事件最迟 5s admission，单次直接目录枚举最多 2,048 entries/批次。触及任一上限时不得
无界扩容，而是把受影响媒体库标记 degraded 并合并完整扫描。WCH-S2 可以基于证据下调这些
上限，提升必须重新评审资源预算。

测试只使用临时 `/library` 等价目录，绝不读取开发者或 CI 宿主机的真实照片。

当前 `tests/integration` 已使用 `t.TempDir()` 和真实文件 SQLite 覆盖首次递归扫描、空目录、
直接/递归计数、格式及系统目录跳过、失败/取消/离线/根替换与 A → B → A 替换保留、后续
成功收敛和跨媒体库隔离。测试专用 HTTP capability seam 还覆盖 opaque asset ID 到
`internal/files` 的 GET、HEAD、条件请求、单 Range、416、路径攻击和错误脱敏。FS-03 另以
运行时合成 fixture 调用真实 FFmpeg CLI。自动发现另以独立领取子进程被操作系统强杀、
跨 SQLite 重开 lease 恢复、真实 watcher→catalog 纵向链和受控 catalog state 故障注入
覆盖恢复及 HTTP 边界；磁盘满和备份恢复仍由既有 release 证据负责，不把它们重复归入 WCH。
带 `linux && fsboundary` tag 的隔离高权限探针已在原生 Linux amd64/arm64 覆盖同设备、
跨设备和 self-bind mount，现同时断言目录选择 adapter 不进入挂载内容并返回稳定
`mount_boundary`；WCH 增加的同类探针进一步验证运行期 bind mount 覆盖时定向校准失败关闭、
旧索引保留以及真实 unmount 后 durable 重试收敛。当前 WCH 新实现已通过 Linux/arm64 与
本机 race；QEMU amd64 因 `openat2` 不可用按安全边界失败关闭，仍不能替代原生 amd64。

### 浏览器端到端测试

放在 `tests/e2e`，优先保持少而关键：

1. 首次进入 → 原子创建单管理员 → 登录 → 创建媒体库 → 看见扫描状态 → 浏览第一批媒体。
2. 切换目录与递归模式 → 刷新/复制 URL → 状态仍可恢复。
3. 在当前目录、当前媒体库和全部媒体库之间搜索/过滤/排序 → 翻页或滚动 → 打开查看器 → 返回后恢复位置与焦点。
4. 媒体库离线 → 保留索引提示 → 挂载恢复并重新扫描。
5. 删除媒体库 → 明确非破坏性确认 → 配置消失且 fixture 原文件仍存在。
6. 统一 BrandMark、桌面固定侧栏、移动目录抽屉、网格/瀑布流切换与键盘关键路径；
   检查品牌矢量标识、favicon、中英界面、浅色/深色、reduced motion 与高对比模式。

当前 `make test-web-e2e` 已固定 Stage 1～4 的一次性真实后端 Chromium/Pixel 5 成功链。测试镜像以
`libvips` build tag 和 Debian libvips runtime 构建，使用只读合成 JPEG 验证扫描后的
ready WebP、grid/masonry 切换与偏好恢复；评审后的契约响应只补齐运行、取消、部分不可读
和离线等无法稳定定时制造的 UI 状态。它同时断言长名称/路径、同操作快速双击只提交一次、
对话框焦点恢复、390/768/1024/1440px 横向溢出和 axe serious/critical；搜索、预览和
查看器已在 Stage 3～4 扩展同一入口。Stage 5 的 `make test-web-release-e2e` 另以完全
模拟的稳定 API 状态运行 Firefox、WebKit 和 Linux Chromium 视觉回归，避免多引擎争用
唯一首次管理员 fixture；真实 MP4 Range/206 仍由 Chromium 负责，三个桌面引擎共同验证
键盘/焦点、unsupported codec、offline/deleted、overflow 和 axe serious/critical。
7. 会话过期、退出、CSRF 拒绝和再次访问受保护媒体，不泄露内容或敏感状态。

端到端测试不能通过固定长时间 sleep 等待扫描，应轮询可观察状态并设置明确超时。

#### FTR-UIF-001 一致性 Gate

[FTR-UIF-001](features/frontend-prototype-fidelity.md)在原有功能 E2E 之上增加独立阻断矩阵：

- 原型和生产使用同一 fixture、语言、主题、交互状态与
  1440×900、1265×800、768×1024、390×844 视口；
- 每个页面保存原型图、生产图和组合比较，P0/P1/P2 必须全部关闭，主要区域几何偏差不超过
  2px；基线更新必须解释来源，不能批量无理由接受；
- Linux-owned 确定性视觉回归覆盖单一 Header、管理四页、浏览顶部/底部、Search、预览与
  Viewer；动态区域只做最小遮罩；
- 真实纵向链覆盖资料/密码、建库/扫描、目录全量过滤、搜索/预览/Viewer、缓存清理和重新
  登录；另跑 Chromium、Firefox、WebKit、axe、键盘、触摸、forced-colors 与 reduced-motion；
- 100k 媒体/10k 目录验证稳定 cursor、有界 DOM、目录 query P95、滚动 FPS/RSS、缓存清理
  写放大和并发浏览；全链前后校验只读媒体 sentinel。

详细任务与证据归档入口见
[FTR-UIF-001 开发清单](features/frontend-prototype-fidelity-task-list.md)。`UIF-S4` 已通过，
但最终 digest、物理辅助功能和供应链 Gate 仍阻断发布，不能仅凭截图宣布可发布。

[UIF-S3 Consumer/UI Ready](gates/MVP-2026-07-23/uif-s3-consumer-ui-ready.md)已固定
`web/qa/visual-reference-manifest.json`：12 个生产页面必须映射到原型、生产路由、状态与
确定性 Storybook/component/E2E fixture；`npm run check:visual-references` 已进入前端总检查。
S3 同时通过双语双主题四档、键盘/触摸、axe、forced-colors/reduced-motion、六种浏览器/
输入 profile 与三引擎 100k 虚拟化预算。它只授权进入 Integrated Slice；后续逐页 comparison、
Linux 基线、真实纵向链和候选复验由 UIF-401～406 分别完成，不能倒写为 S3 当时已经完成。

`UIF-401` 已把该 manifest 的 12 个页面在同一 `1280 × 720` CSS 视口、简体中文、深色主题
下渲染为原型截图、真实生产路由截图和逐页组合比较；证据与比较历史保存在
[`docs/evidence/uif-401`](evidence/uif-401/README.md)。原型中明确属于 Post-MVP 的任务中心、
缺失缓存补齐、全部重建和系统维护不计作当前生产缺失，也不得用静态控件或 mock 补齐；
`UIF-402` 已把当前 MVP 页面区域转成 11 张 Linux-owned 稳定基线：单一 Header、管理四页、
Browse 顶部/底部/预览、Search、可用图片 Viewer 和既有离线 Viewer。测试固定
`1280 × 800`、English、dark、reduced-motion、认证/媒体合同与仓库 synthetic image，
不遮罩任何动态区域；Linux 生成后无更新复跑 `9 passed`。基线清单、fixture 边界与命令见
[`docs/evidence/uif-402`](evidence/uif-402/README.md)。

`UIF-408` 又在 `390×844 / 768×1024 / 1265×800 / 1440×900` 为同一 12 页分别保存
48 张原型图、48 张真实生产路由图和 12 张成对审阅图，确认无横向溢出、P0/P1/P2 或延期
P3；这组逐页断点证据不与 UIF-317 双语双主题共享状态矩阵或 UIF-401 共同 1280 比较混淆。
同轮 browser release E2E、Chrome Stable、三引擎容量、生产容器和 RC readiness 复验见
[`docs/evidence/uif-408`](evidence/uif-408/README.md)。

`UIF-403` 已把 setup/login、账户改名/改密、建库/扫描、当前目录 `q`、Search、预览/
Viewer、扫描与缓存设置、真实 cache cleanup、logout/login 和安全移除串成同一真实容器
纵向链。`tests/e2e/web_auth.sh` 在运行前后比较 `/library` 全部路径和原文件 SHA-256；
Chromium 结果为 `6 passed / 3 applicable skips`。步骤和只读证据见
[`docs/evidence/uif-403`](evidence/uif-403/README.md)。

`UIF-404` 复验 Chromium/Firefox/WebKit、axe、键盘、Pixel 5 touch、Chrome forced-colors
与 reduced-motion，并明确真实读屏、缩放、OS 高对比和物理设备仍归 S5-006B；
`S5-006B` 后续增加 200% 浏览器缩放的确定性等效重排护栏：固定 `1280×800` 桌面在
200% 下的有效 `640×400` CSS 视口，分别验证 Chromium、Firefox、WebKit、品牌
Chrome Stable 与 forced-colors 的媒体卡焦点入口、查看器主焦点、缩放/信息/关闭控件、
无页面横向溢出及 axe serious/critical 为零。该自动化只防止产品重排回归，不替代真实
品牌 Firefox、物理浏览器缩放、读屏、触摸或移动设备签署。随后
[`S5-006B Chrome 200% 物理浏览器证据`](evidence/s5-006b/README.md)在 Google Chrome
151 / macOS 26.6 的原生 `200%` 页面缩放下完成扫描、浏览、预览、Viewer、`I` 信息、
1:1、媒体放大/缩小与 `Escape` 返回，并复核只读挂载和媒体 SHA-256；它只关闭 Chrome
桌面缩放子项。随后
[`S5-006B Firefox 真实浏览器证据`](evidence/s5-006c/README.md)以 Mozilla 官方 Firefox
153.0.1 的临时隔离 profile 完成相同真实候选的首次设置、只读扫描、`?q=jpg`、图片
预览/Viewer、MP4 实际播放，并在原生 `200%`/`400%` 下复验扫描状态、响应式浏览、
当前目录过滤、预览、Viewer 与快捷键；Firefox DMG 来源、SHA-256、代码签名和 Apple 公证均已验证。两份物理浏览器
证据仍不替代读屏、物理触控、移动设备、Safari 缩放或最终跨设备视觉签署；
`UIF-405` 在最新共享集合上复验三引擎 100k 滚动/DOM/FPS/RSS，以及 10k 目录/100k 文件
扫描期 2,353 次浏览和搜索并发；`UIF-406` 又原样运行 fmt、architecture、generation、
lint、unit、integration 和生产容器 E2E 七项完整仓库验证。证据分别见
[`UIF-404`](evidence/uif-404/README.md)、[`UIF-405`](evidence/uif-405/README.md)和
[`UIF-406`](evidence/uif-406/README.md)。

`UIF-S1` 已增加可重复的
[10k 直接子目录过滤 spike](spikes/uif-001-directory-filter.md)：查询先使用 parent-scoped
browse index，再执行 capability-derived Unicode 搜索键上的 literal `instr`；常规测试以
100ms P95 为跨环境护栏。S2 已关闭 Backend Ready；UIF-405 又在完整 10k/100k 档复验
扫描期间浏览/搜索并发、数据库大小和零预算违规。

### 容器与发布测试

- 最终镜像按 ADR-0012 以 root 运行，媒体目录 `:ro`，Docker 创建的 root-owned 数据目录可写。
- `linux/amd64` 与 `linux/arm64` 运行相同的 smoke suite。
- PID 1 收到终止信号时停止接收请求、取消任务、提交安全状态并按时退出。
- readiness 与 liveness 在迁移、扫描、单库离线和数据库不可写时语义正确。
- Compose 启动、反向代理 Range、备份/恢复、向前迁移和失败回滚演练。
- 磁盘已满、缓存不可写、媒体挂载断连与容器重启不会修改 fixture 原件。

## 风险覆盖矩阵

| 风险 | 单元 | 集成 | E2E / 容器 | 发布阻断 |
| --- | --- | --- | --- | --- |
| 路径遍历或符号链接逃逸 | 是 | 是 | 容器抽样 | 是 |
| 删除/修改原媒体 | 服务行为 | 只读 fixture 校验 | 删除库流程 | 是 |
| 扫描失败误清索引 | 状态机 | 故障注入 | 离线恢复 | 是 |
| 游标重复、漏项或死循环 | 是 | 数据变化场景 | 长滚动抽样 | 是 |
| 损坏媒体耗尽资源 | 限制逻辑 | 真实工具超时 | 容器资源限制 | 是 |
| Range/视频取消错误 | handler | HTTP 流 | 浏览器播放 | 是 |
| 可访问性回退 | 组件 | — | 键盘/自动审计 | 严重项阻断 |
| 迁移/备份不可恢复 | 迁移测试 | 数据快照 | 恢复演练 | 是 |
| 大库性能失控 | 算法基准 | 合成规模测试 | 浏览器性能 | 超过已确认预算时阻断 |
| storyboard 抽帧/backfill/hover 资源失控 | 采样与状态机 | FFmpeg/SQLite/jobs | 三浏览器虚拟 hover | `Post-MVP/1` 阻断 |

更完整的产品与工程风险见[风险登记](risk-register.md)。

## Fixture 设计

`tests/fixtures` 只保存小型、合成或许可明确的数据，并带一份 manifest 描述预期：

- JPEG、PNG、WebP、GIF，以及 MP4、MOV、MKV、AVI 中的最小有效样例。
- 扩展名与魔数不一致、截断、畸形、超大像素声明、长动画元数据。
- 横竖方向、透明度、Unicode/emoji/组合字符、大小写近似和超长文件名。
- 深目录、空目录、隐藏目录、系统垃圾目录和权限受限子树。
- 浏览器兼容与不兼容的视频编码样例；容器可索引与编码可直放必须分别断言。
- `FTR-VID-001` 使用 2s、4s、10s、10min、2h 的横/竖屏、VFR、长 GOP、片头黑屏、
  损坏和不可 seek 视频；manifest 固定期望帧数、时间窗口、layout 和允许降级。
- SVG、HEIC/HEIF、AVIF 与 RAW 的非契约样例，验证不会被误报为 MVP 支持。

符号链接、权限、断连和磁盘已满应在测试运行时动态构造，避免提交平台相关链接。Windows 文件系统不属于容器运行目标，但跨平台开发工具应明确哪些测试只在 Linux 运行。

## 性能与容量验证

主要容量验收档已经确认为约 10 万媒体、1 万目录、4 GiB 内存的四核 NAS/家庭服务器。
FS-04 当前在 Linux/arm64 Docker Desktop VM 与 tmpfs 上完成混合宽度、最大深度 32 的
扫描/索引目标档；首次记录 generation 扫描/finalize 为 10.449 秒，扫描期间库内页读取
P95 为 3.193 ms，采样 Go heap 峰值约 39.2 MB。后续三档趋势记录 Linux `VmHWM` RSS，
并以 `stage0-comparable-v1` 提供同环境回归护栏。测试还对账根 recursive count、全树 direct count
及选定 32 层链的每一级聚合。独立的 1,000 层 SQLite-only 档在同一受限 Linux 容器中
finalize 为 147 ms；它不创建宿主深目录，只证明目录 rollup 算法而不证明文件系统遍历。
该环境不是代表性 NAS 存储，且未包含媒体、FTS、正式 HTTP 或前端，因此这些数字不是发布
SLA，也不能代替完整容量门槛。

S5-005A 已补充 `tests/release/capacity_smoke.sh`：以真实生产候选、有效独立 PNG、
管理员会话、后台 libvips 缩略图和正式浏览/搜索 API 运行同一 100k/10k 主档。本机
Docker Desktop linux/arm64、4 CPU/4 GiB 记录扫描 189 秒，浏览/库内搜索/全局搜索 P95
为 57.311/45.602/72.586 ms，容器 memory peak 350,633,984 B。该档发现并修复 session
touch 绕过 SQLite 单写门导致的后台媒体写入竞争。原生 amd64/arm64 CI 保留 180 秒扫描
回归护栏和 JSON artifact 契约；本机 bind-mount 档使用 240 秒护栏。本轮按操作者决定，
以指定原生 amd64 服务器与本机原生 arm64 结果作为实际架构证据。

S5-005D 已在 4 CPU/4 GiB 原生 amd64 执行同一 100k/10k 后端目标档：首次扫描
148.293 秒，修复后的两次纯无变化重扫为 100.303/105.548 秒，恢复 94,234 个媒体任务
调度后重扫为 140.190 秒。逐请求认证读取保留，但 `last_seen` 审计写入限制为最多每
30 秒一次；unchanged fingerprint 不再重复失效 ready 派生状态或写已有媒体任务。
扫描并发浏览 P95 从约 555 ms 降至 6.116 ms；跨审计写入边界为 6.614 ms。

同一原生 amd64 目标档随后在 6,202,317 ms 内完成全部 100,000 个媒体任务，0 个
terminal failure，容器峰值内存 176,287,744 B。在线把 cache 配额降至 4 MiB 时发现并
修复 cache 删除绕过单写门和逐文件事务写放大；最终 cache 从高水位收敛至
3,355,440 B（80% 上限 3,355,443 B），0 pending deletion，容器持续 healthy、OOM=0，
原媒体哨兵不变。容量回归护栏固定为全量派生不超过 2.5 小时、峰值不超过 1.5 GiB、
terminal failure 为零并正确收敛到低水位；它们不是用户可见 SLA。

目录计数实现不得回到每目录扫描全部资产或展开 asset×ancestor 的复杂度。当前基线在
SQLite O(D) 临时工作表中以最多 500 个叶节点为一批向父目录传播，每条目录边只处理一次；
现有索引下按 O(A + D log D) 评估，Go 侧不加载完整目录树。循环、无效根、缺失/跨库关系，
或当前代次条目指向同库陈旧目录，必须使同一 finalize 事务回滚，不能发布部分计数或让
stale cleanup 级联删除另一媒体库或当前代次条目。

至少测量：

- 首次扫描与无变化复扫的文件/秒、峰值 RSS、SQLite 写入时间和磁盘增长。
- 缩略图吞吐、单任务峰值内存、缓存命中率和不同并发下的尾延迟。
- 目录树首屏、媒体列表 API p50/p95、搜索和游标翻页延迟。
- 浏览器首屏可交互时间、滚动掉帧、DOM 节点数、图片解码内存和查看器切换。
- 服务扫描期间 API 延迟与取消响应，证明后台任务不会饿死交互请求。

建立小/中/目标上限三档合成数据集，其中目标档固定为上述规模，并在[可行性研究](feasibility-study.md)的 spike 后把实测结果与延迟预算写回本文。

### Post-MVP 视频故事板验证计划

[FTR-VID-001](features/video-storyboard-preview.md)在 `VSP-S0/S1` 先通过 spike 固定预算，
并以 service 测试固定普通视频及 4K/大文件均优先走 10 帧路径、超时后才降级 4 帧；
SQLite claim 测试固定全局最多一个 storyboard 运行且另一 worker 仍可领取 grid，fallback
再次超时必须终态结束而非原样重试。预算测试固定普通 4K 的 3 分钟、8 GiB 边界的
4 分钟、fallback 的 2 分钟与大文件总计 6 分钟硬上限。
再进入后端。后端证据必须覆盖采样纯函数、真实 FFmpeg、只追加 migration 升级、job
优先级/公平/重启、source fingerprint CAS、原子发布、ENOSPC、cache missing、认证 HTTP、
原件 hash/mtime 不变和目标容量 backfill。

2026-07-29 的
[VSP-S2 Backend Evidence Ready](gates/POST-MVP-1/vsp-s2-backend-evidence-ready.md)
已通过双架构生产 FFmpeg、真实认证生产镜像纵向链、cache repair、故障矩阵，以及
Linux 四核/4 GiB 的 100k/10k、10% 视频档。随后
[VSP-S3 Consumer/UI Ready](gates/POST-MVP-1/vsp-s3-consumer-ui-ready.md)已覆盖生成
client、300ms hover intent、4/10 帧布局、decode failure、同页单活动动画、虚拟回收、
页面隐藏、touch/键盘/reduced-motion、grid/masonry、组件工作台、forced-colors 和
Chromium/Firefox/WebKit，并以 100-video 三引擎档验证活动数、FPS 与 RSS。
[VSP-301 真实产品纵向链](gates/POST-MVP-1/vsp-301-product-vertical.md)随后以生产镜像从
扫描/poster/storyboard ready 贯通真实登录、浏览与搜索 hover、预览焦点恢复和 cache
202→200。[VSP-302 目标平台与资源复验](gates/POST-MVP-1/vsp-302-target-platform.md)
已经建立原生 amd64/arm64 candidate jobs、结构化 artifact 与
`make verify-storyboard-evidence` 成对校验；它会拒绝 source commit、实际架构、FFmpeg、
workflow run/attempt、fixture、5×2/10 帧布局、decoded pixel hash、cache repair 或资源
限制漂移，并要求数字 run ID 与 RFC3339 生成时间。成功后 workflow 还上传聚合 image
digest 和全部检查结果的 paired summary；
Gate 必须等同一提交的原生 workflow 实际成功后才能签署。

2026-08-12 回归在该纵向链上追加：计划点无帧后的 `±1s` 且不跨相邻采样中线的有界邻帧、邻帧耗尽后的同 job
10→4 帧、四帧无帧耗尽永久 `media_processing_failed`、预算耗尽永久
`media_processing_timeout`、1080p/10 帧 45 秒、4K fallback
120 秒和两次合计不超过 5 分钟。候选镜像还必须证明 FFmpeg 实际链接/启用 external
libdav1d 并成功处理合成 AV1 派生资源；浏览器测试仍以原生 `loadeddata`/`error` 结果判定
播放能力，不从 poster/storyboard 成功推断直放。升级恢复测试通过“补齐缺失”重排受影响失败，
同时验证既有 ready 派生、schema/transform version 与原媒体 hash/mtime 不变，真实损坏仍失败。

`docs/releases/POST-MVP-1-readiness.json` 进一步汇总 VSP Gate、`VSP-AC-001～008` 和
R-018。`make storyboard-readiness-check` 不只检查证据路径存在，还要求每个验收项的
稳定事实锚点确实出现在对应 Gate 记录中，并与 feature spec、S4 聚合表及任务勾选一致；
`make storyboard-ready` 在任一 Gate、验收或风险未处置时失败关闭。

## 安全验证

- 路径输入表：绝对路径、`..`、双重编码、NUL、不同分隔符、Unicode 归一化和 symlink race。
- HTTP：认证绕过、CSRF、开放重定向、可信代理头、限流、错误信息与安全响应头。
- 媒体：像素炸弹、损坏容器、探测超时、命令参数注入和主动内容同源执行。
- 依赖：Go/npm 系统依赖漏洞扫描、镜像 SBOM、第三方许可证检查和固定构建来源。
  `S5-007A/G` 已对候选建立固定 digest Syft/Trivy 与 CI artifact；供应链 job 在原生
  amd64/arm64 分别执行 `all` 策略，并由 `make verify-supply-chain-evidence` 拒绝缺失
  架构、不同 commit/run/attempt、非零 High/Critical、SPDX 漂移、错误 GLib 版本或
  被移除包重新进入闭包。只有成对 summary 通过才构成 S5-007B 自动化证据。
- 日志：令牌、Cookie、宿主机路径、SQL 和原始 stderr 不得出现在正常或故障日志中。

认证范围已经确认；上述单管理员安全测试是稳定版发布阻断项，不能用“以后补”作为公网发布依据。

## 本地验证与合并门槛

为节省 GitHub Actions 配额，PR 与 `main` push 不运行自动测试。提交或合并前由开发者在
本地运行统一入口，名称与根 `AGENTS.md` 对齐：

```sh
make fmt
make arch-check
make generate-check
make lint
make test
make test-integration
make test-e2e
```

当前 `Makefile` 已存在 `fmt`、`fmt-check`、`arch-check`、`contract-check`、`lint`、`test`、
`test-race`、`test-integration` 和显式 `spike-capacity`；`arch-check` 验证 Go 依赖方向与
禁止的通用包。它还在 intelligent-media Release No-Go 时强制 production reviewed AI catalog 为空，允许 ADR-0014
已接受的 fail-closed semantic search composition，并继续禁止未获 release Gate 的 face runtime/route；
Gate 转换必须同步更新该检查。`contract-check` 固定使用 `kin-openapi v0.142.0` 和纯 Go ECMAScript
pattern 编译器，每次禁用 Go test 缓存，强制完整解析 YAML、解析本地引用、执行 OpenAPI
结构/pattern 验证，并用 AST/Schema 与跨源检查固定认证、错误、健康状态、分页、路径、
Range，以及 scanner/migration 的 durable admission、phase/counters、issues、cancel、
lease/recovery、schedule 与资源上限契约；这不是完整领域实现一致性证明，生产 worker、
故障和容量证据仍须在 Backend Gate 补齐。运行时不依赖 Ruby、Node 或网络。权威 OpenAPI 还通过了
Redocly 外部交叉验证；当前只有两条 health endpoint 未声明虚构 4xx 响应的规则 warning，
没有结构错误。`generate-check` 同时验证固定版本 sqlc 和 OpenAPI TypeScript 生成无漂移，
`web-check`、摘要锁和语义兼容入口已在本地通过。`test-e2e` 已使用测试专用容器启动真实
`cmd/foliopath`，覆盖 root runtime、固定 volumes、health、默认 401、重复 migration、SIGTERM
和媒体只读 sentinel。前端组件/token、认证与媒体库/扫描浏览器产品 E2E 已可执行；浏览、
搜索、预览和查看器继续按对应前端 Stage 补齐。

每次本地验证至少要求：

- 格式、架构依赖、静态检查、生成一致性和单元测试通过。
- 受影响的集成/组件测试通过；数据库、路径或扫描改动运行完整相关矩阵。
- OpenAPI、迁移、设计文档与实现没有可检测漂移。
- 新增依赖通过许可证与安全检查。
- 不允许以重试掩盖不稳定测试；先隔离并登记 owner 与修复条件。

发布候选额外要求：

- 全量集成、E2E、双架构容器和恢复演练通过。
- Docker Hub 发布只允许由 `main` push、`vMAJOR.MINOR.PATCH` tag 或显式手动运行触发；
  PR 不得读取发布凭据或推送镜像。发布的 OCI index 必须同时包含 `linux/amd64`
  与 `linux/arm64`。
- Release Please 只在 `main` push 上创建或更新 Release PR，并由同一 workflow 自动 squash
  merge 后完成 GitHub Release；用户更新日志必须保留带 Emoji 的“新功能、改进、修复、
  注意事项”分类，并隐藏纯技术提交。提交者必须在 push 前确认适用 Release Gate；自动化
  不替代 Go/No-Go 判断。
- 发布 workflow 不包含 SSH、实例地址或远程部署 action；版本和 Docker 发布不得隐式更新
  任何实际运行实例。
- `make test-release-image` 必须在原生 linux/amd64 与 linux/arm64 runner 对同一提交
  成功；它覆盖真实 SPA、MVP 媒体 probe/poster、损坏输入、Compose 安全约束、代理
  失败关闭、HSTS、健康检查、SIGTERM、离线恢复、强杀恢复、数据盘满和损坏数据库。
- 两个原生 job 必须上传 `release-image-<sha>-<arch>` artifact，并以
  `make verify-release-image-evidence` 证明 release、commit、workflow run、实际架构、
  digest/size 和 passed smoke 一致；不能拼接不同 run 的单架构结果。
- 没有未处置的高危安全问题或会修改原媒体的缺陷。
- 性能没有超过已确认预算，或退化已明确接受。
- 支持格式、部署参数、迁移和已知限制与 README/发布说明一致。

## 当前可执行检查

当前可执行：

- `go test ./...`
- `go test -race ./...`
- `make fmt` / `make fmt-check`
- `make arch-check`
- `make release-docs-check`（README/部署/Compose/.env/Dockerfile/媒体格式与候选限制防漂移）
- `make release-readiness-check`（校验当前 RC Gate/risk 快照；准确的 No-Go 也应通过）
- `make release-ready`（实际 RC promotion 门；只要有 Gate/risk 未处置就必须非零失败）
- `make verify-release-image-evidence EVIDENCE_DIR=... RELEASE_SHA=...`（成对校验原生
  amd64/arm64 候选 JSON，拒绝 commit/run/架构/结果不一致）
- `make contract-check`
- `make generate-check`
- `make web-check`
- `make openapi-lint`
- `make compatibility-check OPENAPI_BASELINE=api/openapi.yaml`
- `make lint`
- `make test`
- `make test-race`
- `make test-libvips`（在 Dockerfile 固定的 libvips 8.16.1 构建环境中运行带
  `libvips` tag 的真实格式、超大 JPEG shrink-on-load、方向与资源泄漏回归）
- `make test-integration`
- `make test-e2e`（真实后端进程的测试专用容器 smoke；不是浏览器或发布镜像验收）
- `make test-web-e2e`（一次性真实后端的 Stage 1～4 Chromium 产品 E2E 与媒体矩阵）
- `make test-web-release-e2e`（Linux 发布候选 Firefox/WebKit 稳定状态与 Chromium
  视觉基线；需要锁定的三个 Playwright browser runner）
- `npm --prefix web run check`
- `npm --prefix web run build`
- `make spike-capacity`（显式重型目标档）
- `git diff --check`
- 仓库内 Markdown 相对链接检查
- 可用时运行 Markdown linter
- 人工核对 README、PRD、ADR、API、部署与安全语义

POST-MVP-5 另提供手动
[`Intelligent media native evidence`](../.github/workflows/intelligent-media-native.yml) workflow，使用
GitHub-hosted 原生 linux/amd64 与 linux/arm64 runner 执行同一 source SHA 的常规仓库检查、production
libvips、两库 order-first 等价矩阵、强制 10k/100k 容量基线，以及固定 YuNet/AuraFace/公开 JPEG 的
production-boundary candidate preflight。候选步骤逐项校验外部文件 SHA，在断网只读容器中运行，只上传
hash、candidate count 和单向量化指纹；它明确不提供最终模型、质量或合规批准。workflow 不在
`push`/`pull_request` 上自动运行，不使用 QEMU，也不上传模型、媒体、数据库或 `/library`、`/app/data`
内容。任一检查失败时 job 保持失败，仅通过 `if: always()` 上传有界诊断 artifact；不得用 artifact 已上传
冒充检查成功。

workflow 文件和架构测试存在只证明“原生执行入口可审计”。只有两个目标架构在同一 source SHA 上实际
成功运行，且 artifact 中 identity、step outcomes、质量/RSS/数值、索引重建和容量结果满足对应 Gate，
才能作为 `INT-403` 或 S2 Backend/Release 证据。当前尚无该远程运行，因此双架构状态仍为未完成；已知
`make spike-capacity` 的 production keyset 超预算会使工作流失败关闭，而不是被候选查询矩阵掩盖。
2026-09-01 本机 production 有序扫描已以 133.637 ms 通过冻结的 250 ms 预算；该本机结果不替代 workflow
要求的同 SHA 原生双架构实际运行。资产搜索游标从 v2 起携带经 HMAC 认证、同时绑定规范化查询与
catalog revision/generation 的分类计数；后续页复用该计数，避免为同一稳定快照重复执行全量 FTS count。

下载两个 artifact 后必须执行：

```sh
make verify-intelligent-media-native-evidence \
  EVIDENCE_DIR=... RELEASE_SHA=... \
  WORKFLOW_RUN_ID=... WORKFLOW_RUN_ATTEMPT=... \
  SUMMARY_FILE=...
```

workflow 的 paired job 已自动调用该入口。verifier 拒绝缺失/重复架构、不同 commit/run/attempt、非原生
runner identity、QEMU 及任一步骤非 success，并严格验证两架构 candidate record 的同模型/runtime commit/
fixture/candidate count 和非批准 flags，仅在全通过时发布 paired summary。两架构量化指纹允许不同，不会
被 baseline verifier 误升格为数值容差结论。它不读取日志猜测质量结论。最终模型获准并在两个原生 job
真实生成 `model-evidence.json` 后，必须改用：

```sh
make verify-intelligent-media-native-model-evidence \
  EVIDENCE_DIR=... RELEASE_SHA=... \
  WORKFLOW_RUN_ID=... WORKFLOW_RUN_ATTEMPT=... \
  SUMMARY_FILE=...
```

严格模式额外验证同一 model package/质量集/fixture、跨架构 Top-20 集合、`1e-3` 数值容差、最终容量/RSS/
延迟/空间预算、索引重建/重启与 runtime failure matrix。baseline workflow 当前不运行未获准最终模型，
因此默认 paired summary 不能关闭 `INT-403`；只有严格模式的真实双架构结果才可进入 Gate 复审。严格
模式还会实际读取并哈希 quality summary、Top-20、numeric、runtime、index 报告，拒绝路径逃逸、symlink、
非普通文件和内容篡改；合法审核数据本体不进入 artifact，只由 governed manifest hash 绑定。

前端 import/token lint、Storybook/组件、认证、媒体库/扫描、浏览/预览、搜索与查看器
视觉/E2E 已可执行；
只读发布 volume/运行期 unmount、最终 Safari/物理辅助功能与移动设备发布签署、代表性客户端性能、
双架构发布镜像和恢复演练仍不可执行或尚不存在；搜索功能正确性、
旧库回填、认证 HTTP、真实 composition、100k 容量、扫描并发、取消和 rebuild 已由
S4-002～003 执行并汇总为 Backend Ready。定义好的 CI
执行现有 Go、双架构 openat2/mount、HTTP harness 或 tmpfs 容量检查不能替代这些缺失门槛。架构检查
的完整状态与最晚落地阶段见[架构适配度检查](architecture/fitness-functions.md)。

## POST-MVP-5 revision 1 验证合同

S1 冻结测试判定，不把现有隔离 spike 冒充生产证据：

- 合同：OpenAPI 结构/语义兼容、错误枚举、ETag/幂等/CSRF、生成客户端确定性及中英文 fixture。
- 数据：fresh/upgrade/restart/故障回滚、FK/cascade、active generation 唯一、cursor 跨代拒绝、bounded
  transaction、库移除和源 fingerprint 失效。
- 模型边界：未知/篡改/错架构/错大小/external-data、symlink/hardlink/special file/nested mount、来源
  可写、TOCTOU、取消、ENOSPC、强杀和 direct 消失/同 hash 恢复。
- runtime adapter：默认构建稳定 unavailable；显式 `onnxruntime` 构建只接受 1.28.0，验证两张 split
  graph 的名称、dtype、固定 shape、线程/arena 配置、错误脱敏和所有失败路径资源释放。分别记录
  session load 前/后的协作取消与不可中断 `CreateSession` 的实际上界，不把 goroutine timeout 当作
  native 取消证据；native amd64/arm64 均为发布门槛。
- 质量集：由产品/QA/ML 提供合法、代表项目人物套图分布且不进入仓库的图片与中英文查询标注；冻结
  指标为 Top-20 query success ≥ 85%，并分别报告中文/英文、近景/全身、室内/室外与小样本置信区间。
  低于门槛删除 Slice B，不用合成向量、公开抓取人脸图片或调低指标替代。
- 性能：4 CPU/4 GiB、10k 目录/100k 媒体；整进程 RSS ≤ 3.2 GiB，语义查询 P95 ≤ 750 ms、
  P99 ≤ 1.5 s，ordinary browse P95 退化 ≤ 20%，可重建 AI 派生增量 ≤ 500 MiB（不含模型）。
- 跨架构：同一审核模型与 fixture 在 native linux/amd64、linux/arm64 的 ranking Top-20 集合一致；
  score 允许绝对误差 ≤ 1e-3，若排序在 cutoff 附近受误差影响则以冻结 tie fixture 单独裁决。QEMU
  结果不能替代 native release evidence。
- 可靠性：旧 active generation 在 activation/cancel/crash/盘满失败后仍可查询；模型 unavailable 只
  阻断语义请求；clear/rebuild/library removal 前后原媒体 sentinel 完全不变。
- UI：五种语义空/错误状态、覆盖率、stale cursor refresh、模型 managed/direct 选择、双语双主题、
  390/768/1265/1440、键盘与屏幕阅读器。静态原型只作设计输入，不算生产验收。

所有质量 fixture 必须记录来源、授权、用途、保留期限和删除责任人；不得提交真人生物特征 ground
truth。S1 不新增重复合成 benchmark，只有能改变候选或合同的测试才可重新打开 spike。

## POST-MVP-5 revision 2 C+D+E 验证合同

S3 消费者验证在后端合同测试之上增加：生成客户端 adapter 的路径/游标/revision/CSRF/幂等断言，搜索
模式 URL 恢复与语义游标失效，标签 suggestion/curation 双 revision 冲突，人物重名/移动/移除/合并，
单图粗略框与键盘等价列表，以及中英文隐私文案。Storybook 使用全局 axe error policy，并在
390/768/1265/1440、light/dark、zh-CN/en 和 reduced-motion 矩阵复核；通过 S3 不解除 S4 的模型、质量、
native 双架构、联合容量、供应链和 owner 批准门禁。

`make test-intelligent-media-offline` 提供 `INT-410` 的本地拓扑回归：真实候选容器使用
`--network none`、只读 rootfs、`/library:ro` 与 `/models:ro`，完成管理员初始化、空 reviewed catalog/
空 candidate scan、重启及两个来源 sentinel hash 对照。HTTP 辅助容器只加入应用的无外网 network
namespace，不创建可出站网络。该检查证明无模型时核心服务可用且不写模型/媒体来源，但它是单一
linux/arm64 本地候选，不替代最终发行源、最终模型或 native 双架构的安装/升级/恢复验收。

S4 浏览器自动化使用 `make test-web-release-e2e` 与 `make test-browser-capacity`。前者覆盖 Firefox/WebKit
的键盘、焦点、降级、重排和 storyboard 行为，后者对 Chromium/Firefox/WebKit 的 100k 虚拟集合强制
FPS、P95 frame interval、process-tree RSS 与 mounted-item 上限。Playwright WebKit、viewport/touch 模拟和
axe 分别不能替代 retail Safari、物理输入与真实读屏验收；报告必须明确保留这些人工 Gate。

Retail Safari 必须另用实际 Safari 应用复核 DOM 键盘顺序、预览/查看器快捷键、退出焦点恢复与浏览器
真实缩放；Playwright WebKit 不替代该步骤。macOS 在“按下 Tab 键高亮显示网页上的每一项”关闭时使用
`Option+Tab` 是 Safari 的等价完整键盘遍历路径。该检查仍不能替代触屏设备或真实读屏操作。

`make test-intelligent-media-privacy` 是 `INT-406` 的工程回归入口：使用敏感 canary 验证结构化日志脱敏、
face/semantic/diagnostic HTTP 安全投影、face capability 与日志/诊断依赖隔离、derived/manual clear 分类边界，
并要求每个 SQLite 连接启用 `secure_delete=ON`。文件级用例先证明 canary 已写入，再删除、truncate WAL，
最后复核活动 DB/WAL/SHM 无残留。该入口不宣称清除 operator backup、storage snapshot、文件系统 journal
或合同明确保留的应用状态，也不替代 privacy/compliance/security owner 发布批准。

`INT-119` 冻结以下验收面，但不把尚未取得的外部数据、native 主机、模型许可或最终镜像写成通过：

- C 合同/集成覆盖 vocabulary revision、Top-5/finite confidence、generation/source 失效、同分 tag ID、
  人工 tag 优先、100 项批量上限、逐项 conflict、accept 原子性、dismiss 跨重建保留及独立 review clear。
  合法代表性词表集冻结每 tag precision/recall、宏平均与人工接受率；产品/ML/QA 在 S2B Gate 前签署阈值，
  未签署或未达阈值时只保留手工标签，不展示模型建议。
- D 使用至少 100 个合法代表性 MP4/MOV/MKV，覆盖 2 秒～2 小时、4/10 帧、运动/静态、室内/室外及
  中英文查询。冻结 Top-20 success ≥80%，验证 max-frame、ordinal/asset tie、partial/mixed plan 排除、
  fallback coverage、storyboard eviction/source change/cancel/crash；不得重新抽帧或用合成向量代替质量集。

S2B 审核集到位后必须执行：

```sh
make verify-intelligent-media-quality \
  QUALITY_INPUT=... DATASET_MANIFEST=... RELEASE_SHA=... SUMMARY_FILE=...
```

该入口核对 governed manifest 实际 hash、非 synthetic ordinary-media 准入、100-video 覆盖、三方批准引用，
并从 counts/result IDs 重算 tag 与 video 指标；不能直接信任输入中的 pass 标志。真实媒体/标注和批准材料
不进仓库，验证报告只记录有界 ID、hash、计数和聚合指标。

S2C 经治理的人脸 ground truth 到位后必须独立执行：

```sh
make verify-intelligent-media-face-quality \
  FACE_QUALITY_INPUT=... DATASET_MANIFEST=... RELEASE_SHA=... SUMMARY_FILE=...
```

该入口拒绝公开许可冒充生物特征授权，要求 schema v2 `biometric-ground-truth`、书面授权、隐私复核、
至少 50 个 opaque identity 且每个至少 20 张图，并逐项核对 detection、verification 和 cluster label 与
governed manifest 一致。验证器重算 detector recall、verification recall/FPR、anonymous core/edge precision、
肤色呈现/年龄呈现/光照/遮挡/多人图片切片；每个维度至少两个有效分组、每组至少 20 个 expected face。
通过判定使用 Wilson 95% 下界（FPR 使用上界），core precision 下界未达 99.5% 时强制失败；未知/单一分组、
小样本全正确、输入中的 pass 字段或降低阈值均不能绕过。

最终供应链材料到位后还必须执行：

```sh
make verify-intelligent-media-supply-chain \
  SUPPLY_CHAIN_INPUT=... RELEASE_SHA=... SUMMARY_FILE=...
```

该入口实际读取并哈希 catalog、最终 model package、component notices、native amd64/arm64 SBOM、
provenance、signature-verification、vulnerability report 与可选 VEX；拒绝路径逃逸/symlink、不完整 SBOM、
未验证签名、缺再分发批准，或存在 Critical/High 却没有 VEX 和 security approval 的输入。通过只证明
证据文件与 manifest 一致；security、compliance、release、inference owner 仍须审阅并签署。
schema v2 还要求五个唯一角色：inference runtime、semantic tokenizer/model、face detector/embedder；组件名或
角色重复、任一 notices/再分发批准缺失都失败关闭。

四个入口各自产生真实 summary 后，最终 S2 复审前还必须执行：

```sh
make verify-intelligent-media-s2-evidence \
  QUALITY_SUMMARY=... FACE_QUALITY_SUMMARY=... \
  NATIVE_SUMMARY=... SUPPLY_CHAIN_SUMMARY=... \
  RELEASE_SHA=... SUMMARY_FILE=...
```

聚合器要求 C/D quality、E face quality、native 与 supply-chain 使用同一 source commit、同一 model package，
并逐架构匹配 strict native 与 supply-chain 的最终
image digest；baseline native summary 的 `finalModelEvidence=false` 会被拒绝。聚合 summary 记录四个
输入文件的实际 SHA-256，但不替代任何 owner 的签署或原始 verifier。
原生 identity/outcomes/final-model evidence、供应链 manifest 和聚合 summary 共用严格 JSON owner，拒绝
duplicate key、unknown field 与 trailing value，并用 16 MiB 上限和 open 前后 file identity 复核拒绝 symlink/
非普通文档；artifact 相对路径和内容 hash 检查仍由各 evidence verifier 持有。
可能包含逐项结果的 quality/face-quality input 及其 governed dataset/model manifest 使用 256 MiB 上限，
同样只接受 non-symlink regular file、在 open 前后复核 identity 并执行 bounded read；其 strict decoder
递归拒绝 duplicate JSON key。

`make spike-ai` 还执行不依赖 OpenCV 权重的 E 功能冒烟安全合同，覆盖显式 operator authorization、
采样/文件大小上限、media-root 和子项 symlink 拒绝、分组均衡选择及候选模型 size/hash 绑定。显式授权的
本地真实图片功能运行只允许输出聚合 decode/detect/align/embed 计数和延迟，不保存 path、crop 或 embedding，
且不能替代下面的 ground truth、质量、隐私、双架构或发布验证。

S2C candidate production boundary 另以 build tags `libvips onnxruntime` 在 native Linux 运行：固定 libvips
闭包从只读公开 JPEG 解码，YuNet/AuraFace 只从 `/proc/self/fd` 加载，并覆盖尺寸/格式上限、BGR detector
tensor、12-output ABI、NMS、五点 alignment、AuraFace preprocess 与有限非零 embedding。该 smoke 只证明
原生组合可执行；没有 governed face-level label 时不得计算或宣称 recall、FPR、core precision 或偏差通过。

face format v3 的定向回归还覆盖 detector/embedder/threshold 三文件完整性、purpose/version confusion、严格
threshold JSON、有限 detector 输出、有限非零 512 维 embedding、session 关闭，以及 activation 事务只替换
旧 face generation、不修改 semantic active pointer、不提前切库。应用使用无 face runtime 的 composition，
证明模型缺失时稳定 `model_unavailable`。

同一候选镜像还必须在 `--cpus 4 --memory 4g`、断网、只读 rootfs 下运行 100k × 512 合成人脸聚类。
每侧写出严格 `face-capacity.json`，绑定 native identity、候选镜像、512 维，以及两种 100,000 成员极端：
50,000 paired core cluster 与 100,000 edge-only singleton；同时记录总耗时、Go memory sys 和不含向量的
确定性结果指纹。paired verifier 要求两架构指纹一致，并要求
`identityGroundTruth=false`、`qualityGate=false`。它只验证候选生成/聚类容量，不替代真实质量、完整 SQLite/
HTTP/浏览并发或下述最终联合容量。

聚类回归还必须包含 transitive bridge：固定 `A/B` 与 `B/C` 均达到 core threshold、`A/C` 未达到，断言
smallest-ID anchor coherence 只允许 A/B 成为 core，C 最多为 edge；输入换序结果不变。真实扩大样本若产生
跨来源巨簇，必须先按同一规则重算并保留失败记录，不能靠提高展示层阈值隐藏。

- E 的真实 ground truth 由隐私 owner 批准且不进仓库/普通 CI。冻结 anonymous core precision ≥99.5%，
  分开报告 recall、edge、肤色/年龄呈现/光照/遮挡/多人图片切片与置信区间；不达 precision 时整组操作
  失败关闭并降为逐脸/小组，仍不达安全可用性则删除 E。禁止从公开网页抓取真人图片补数。
- E 集成覆盖默认关闭/告知、多人逐 face、manual assignment/exclusion/cannot-link、named person 不自动合并、
  merge 原子冲突、split、guarded undo、generation reconciliation/needs_review、derived/manual clear 分离、
  backup/restore、日志/诊断/错误脱敏和 `/library` sentinel SHA-256/mtime 不变。
- native linux/amd64 与 arm64 对同一审核模型/fixture 比较 C/D ranking Top-20 集合、E detection/cluster
  decision 与 score 绝对误差 ≤1e-3；cutoff tie 使用冻结 fixture。两架构都必须执行 runtime load、取消、
  强杀、ENOSPC、source offline、session unload 与长周期泄漏；QEMU 不算 release evidence。
- 联合容量档为 4 CPU/4 GiB、100k media/10k directories，混合 ordinary browse、A/B image semantic、C tag、
  D video 与 E face。整进程 RSS ≤3.2 GiB、ordinary browse P95 退化 ≤20%，队列有界且 browsing 优先；
  每 slice 另报告吞吐、DB/WAL/cache 增量与恢复时间，不用平均值隐藏长尾。
- 最终双架构 digest 校验 model/runtime hash/license、SBOM、VEX、notices、provenance、漏洞处置、只读 volume、
  非 root、backup/restore/upgrade/rollback。任何外部 evidence 缺失均保持对应 S2 Gate No-Go。

本阶段可由 contract/architecture test 验证 OpenAPI、生成确定性和生产 composition fail-closed；production
migration/handler/worker 测试只有 S1R2 Gate Go 后才允许出现。测试报告必须逐条写已执行/未执行和环境，
不得把 static prototype、synthetic face state machine 或 arm64 单机数据描述为发布证明。
