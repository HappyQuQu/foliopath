# FolioPath 安全模型

## 范围和信任边界

FolioPath 读取用户提供的目录和媒体文件，并通过 Web 服务展示内容。即使媒体目录只读，路径、文件名、元数据和媒体编码仍是不可信输入。主要边界包括：

- 浏览器与 HTTP API 之间的网络边界。
- `/library` 只读媒体根目录与容器其他文件系统之间的路径边界。
- Go 进程与 libvips、`ffprobe`、FFmpeg 等原生媒体解析器之间的进程/库边界。
- 可重建媒体索引与未来用户配置、认证和分享数据之间的数据边界。

首版安全目标是防止越界读取、未授权网络暴露、媒体解析导致的无界资源消耗，以及扫描故障造成的索引破坏。只读挂载只能防止修改原媒体，不能替代访问控制和输入验证。

## 文件系统边界

- `/library` 是唯一允许创建媒体库、也是唯一允许的媒体挂载目标。它自身可以是
  一个只读 mount，但其后代不得嵌套 volume、bind mount 或其他 mount point。
  多个 UI 媒体库必须对应这个单一根中的普通目录。
- API 不接受任意宿主机路径或容器绝对路径；公开读取接口使用媒体库、目录和媒体 ID。
- 用户选择的路径先转换为相对于 `/library` 的规范形式。拒绝绝对路径、NUL、`.`/`..` 越界、编码绕过和平台不支持的路径表示。
- Linux 上不能把“先检查、后打开”作为安全边界。`internal/files` 必须从已锚定的
  `/library` 目录文件描述符出发，以 `openat2` 的
  `RESOLVE_BENEATH | RESOLVE_NO_SYMLINKS | RESOLVE_NO_XDEV` 原子解析并打开实际
  文件或目录；只做字符串前缀、realpath 或 device/inode 比较均不足够。
- Linux 内核或 seccomp/LSM 不支持所需 `openat2` 行为时必须失败关闭，媒体根不得
  进入可用状态，且不得回退到 `os.Root` 或较弱的用户态检查。非 Linux adapter
  仅提供开发证据，不声明同等级的 mount-boundary 保证。
- 扫描默认不跟随目录符号链接。任何允许符号链接的新设计必须先处理逃逸、循环和检查/使用竞态，并记录 ADR。
- 管理 UI 只显示 allowed-root-relative 库根和安全的 `/library/...` 容器标签；浏览 UI 显示媒体库内
  相对路径。错误响应和日志不得泄露宿主机路径、数据库位置或容器内部敏感路径。
- 目录选择器必须先取得权威媒体库快照，再把相同、祖先和后代目录标为不可选择；无法读取
  或无法规范化该快照时失败关闭，不能在不知道重叠状态时继续枚举成可选目录。

若媒体位于多个独立宿主机卷，只能由部署者先提供一个对容器表现为单一文件系统、
没有后代 mount crossing 的呈现根，再一次性只读挂入 `/library`。FolioPath 不
承诺具体 union 或聚合技术；`NO_XDEV` 拒绝仍是应用的最终边界。完整决策见
[ADR-0009](adr/0009-linux-openat2-single-media-root.md)。

纯相对路径词法规则集中在 `internal/pathpolicy`，实际文件系统身份、解析和打开集中
在 `internal/files`；其他包不得自行拼接或打开真实媒体路径。

`S2-002` 已把已认证目录选择器接到该生产边界：handler 只传 allowed-root-relative
`parent`，`internal/library` 拥有自然排序、页面上限和 query-bound opaque cursor，
`internal/files` 从锚定根按批次读取直接子目录并把 symlink、unreadable 与 mount boundary
映射为有限原因码。游标使用随机进程密钥的 AES-GCM，不包含明文父路径或最后目录名；错误
响应不包含绝对根、errno 或底层路径。

### 自动发现事件边界

Linux watcher 的名称、cookie、顺序和事件类型全部是不可信 hint。adapter 只能产出媒体库
ID、规范相对目录/名称和稳定事件类别；它不能返回给 API 的绝对路径，也不能直接修改索引。
每次定向校准与新 watch 注册仍必须经 `internal/pathpolicy` 和 `internal/files` 的
`openat2(RESOLVE_BENEATH | RESOLVE_NO_SYMLINKS | RESOLVE_NO_XDEV)` 锚定边界重新确认。

`delete`、`move-out`、`unmount`、overflow 或 watch invalidation 本身没有删除资格。只有库根
身份未变且父目录完整、安全枚举成功后，scanner 才能删除明确缺失的直接子项；否则保留旧
索引并进入 degraded/offline。持久任务、日志和 API 错误不得包含宿主路径、容器绝对路径、
原始 errno 或内核消息。`GET /api/v1/catalog/state` 需要现有管理员 session，只泄露一个
单调通知 revision，并使用 `no-store`。

`S2-105` 把相同失败关闭边界接入生产扫描：媒体库根离线、根 symlink、root identity
变化、后代 mount crossing、部分目录权限失败和一般 I/O 只映射为 OpenAPI
`ScanFailureCode` 中的稳定码。除完整成功扫描外均不得运行 stale cleanup；公开状态和
SQLite 不保存 errno、绝对路径或任意内部错误文本。

## 网络和认证

开发预览版在内建认证完成前默认绑定 `127.0.0.1`，或通过可信认证反向代理访问；不得将其直接暴露到局域网或互联网。内建认证完成后，受信局域网可通过 HTTP 直连，但不提供匿名模式；公网或不可信网络的 TLS 与访问控制由部署者负责。详见 [ADR-0005](adr/0005-built-in-single-admin-auth.md)与 [ADR-0010](adr/0010-authenticated-lan-http.md)。

稳定版内建认证至少需要：

- 首次管理员初始化流程，且不能保留公开默认密码；初始化完成后必须原子关闭再次创建管理员的入口。
- 使用适合密码存储的算法、会话过期和安全 Cookie。
- 对所有状态修改请求实施 CSRF 防护或等价的同源令牌设计。
- 登录、扫描启动、目录浏览和媒体读取的速率/并发限制；退出、改密或安全事件能够撤销会话。
- 分享链接使用高熵、可撤销、可过期的令牌，并明确媒体库与目录范围。

MVP 没有分享链接；上条只约束未来功能。除健康检查与受限的初始化/登录端点外，媒体、搜索、扫描、媒体库和设置 API 均要求有效会话。

`S1-101` 已冻结认证 HTTP 与持久化边界：数据库唯一 `singleton_key` 阻止第二个管理员；
密码只保存带 scheme/参数的 verifier；随机会话 Cookie 与 CSRF 值只保存 32 字节摘要；
session 具有绝对过期、撤销和 `auth_version`。认证 JSON 不允许缓存，setup/login 校验同源
`Origin`，状态修改同时校验 Cookie 与 CSRF。

`S1-102` 已实现密码存储与首次初始化领域边界：使用 Argon2id v19、64 MiB、3 次迭代、
4 lanes、16 字节密码学随机 salt 和 32 字节派生 key；验证器只接受这一组已知参数和严格
长度，不接受调用方提供成本参数。该选择采用
[RFC 9106 的内存受限推荐配置](https://www.rfc-editor.org/rfc/rfc9106.html#section-7.4)和
[Go `x/crypto/argon2` 的 Argon2id 指引](https://pkg.go.dev/golang.org/x/crypto/argon2)。
用户名使用 NFKC 和 Unicode full case folding 形成唯一比较键；创建前进程内串行，最终由
SQLite 写事务与 singleton 约束原子关闭再次初始化。日志和错误不包含密码或 verifier。
首次管理员密码接受 8～128 个 Unicode 字符并拒绝控制字符；不强制大小写、数字或特殊
字符组合。服务端 `internal/auth` 是该规则的最终 owner，前端长度提示只用于即时反馈。
该可用性调整记录于
[FIX-2026-07-29](changes/FIX-2026-07-29-admin-password-minimum.md)，不改变 Argon2id、
登录限流、错误脱敏或会话边界。

`S1-103` 已实现服务端会话：

- 每次初始化或登录使用 Go `crypto/rand` 发放两个独立的 32-byte 随机秘密；复合
  `foliopath_session` Cookie 为 host-only、`Path=/`、HttpOnly、SameSite=Strict，并在经
  验证的 HTTPS 传输上设置 Secure。浏览器脚本只收到 CSRF 部分，不能从它还原 Cookie。
- SQLite 只保存整个 Cookie 和 CSRF 部分各自的 SHA-256 摘要；不存在可直接重放的明文列。
- 固定 7 天服务端绝对期限，不因活动滑动续期；重新认证发放全新 Cookie/CSRF。退出立即
  写入撤销时间，`auth_version` 变化或管理员禁用也使会话失败关闭。
- 过期和撤销记录保留 24 小时用于稳定识别过期状态，随后在创建新会话时有界清理。
- 未知账号、非法用户名和错误密码统一失败；未知账号仍执行虚拟 Argon2id 校验，降低账号
  枚举的时间差。

该设计满足 [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
关于 CSPRNG、高熵标识、服务端绝对过期和退出撤销的要求；7 天短于
[NIST SP 800-63B AAL1](https://pages.nist.gov/800-63-4/sp800-63b.html#reauthentication-1)
建议的 30 天总体上限。随机源使用
[Go `crypto/rand`](https://pkg.go.dev/crypto/rand)。

`S1-104` 已实现认证 HTTP handler 和集中 middleware：

- 仅 health 与精确匹配的 auth status/setup/login 操作匿名；其余 `/api/v1` 请求必须先
  通过唯一 Cookie 的服务端 session 验证，未实现的业务路径也不会在认证前泄露存在性。
- 所有状态修改在执行业务 handler 前常量时间比较 session-bound `X-CSRF-Token`；
  setup/login 尚无 session，因此在解析 JSON 凭据前对实际请求 scheme/host/port 执行
  完整 Origin 同源比较。缺失、`null`、多值、userinfo、path 和不等端口均失败关闭。
- 所有认证 JSON 与公共错误使用 `Cache-Control: no-store`；JSON 只接受
  `application/json`、单个值、已知字段和最多 4 KiB。
- setup/login 每个直连 peer 每分钟 10 次，status/session 120 次，logout 60 次；内存
  bucket 上限 4096，满载失败关闭。限流和同源判断只使用直连地址与真实 TLS，不信任客户端
  转发头。

这符合 [OWASP CSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
关于有状态同步 token、自定义 header、同源 Origin 完整比较和缺失时阻断的建议。`S1-105`
已验证五类认证操作的错误脱敏、CSRF 短路、未知账号/错误密码一致响应、绝对过期边界、
并发 setup 单胜者和并发 session/SQLite 更新；`S1-106` 已通过
[认证 Backend Evidence Ready](gates/MVP-2026-07-23/s1-auth-backend-ready.md)，允许正式
前端连接真实认证 API，但不放宽回环监听、可信代理或发布限制。

`S5-003` 已实现反向代理信任边界。默认直连模式清除并忽略所有客户端转发声明，使用实际
peer、Host 与 TLS 状态。非回环监听可直接提供经认证的 LAN HTTP；配置
`FOLIOPATH_TRUSTED_PROXIES` 的显式 IP CIDR 后进入代理专用模式，只有匹配的直连 peer 才能提交
单跳、单值的 `X-Forwarded-Proto: https`、`X-Forwarded-Host` 和数值
`X-Forwarded-For`。缺失、多值、逗号链、非 HTTPS、非法 authority/IP 或同时提交标准
`Forwarded` 均失败关闭。直接 HTTP 或验证后的 HTTPS transport 都是 Origin 与认证限流
身份的唯一来源；Secure Cookie 和 HSTS 仅由 HTTPS transport 启用。
客户端 key 的唯一输入。

所有 HTTP 响应统一设置限制型 CSP、禁止 frame、`nosniff`、`no-referrer` 和禁用相机/
位置/麦克风的 Permissions Policy；只有验证后的 HTTPS transport 返回一年 HSTS。
部署者仍必须用宿主机端口绑定/firewall 阻止代理旁路，CIDR 本身不替代网络隔离。

Stage 5 候选使用固定 digest 的 Go 1.26.5 trixie build layer 与 Debian 13
distroless runtime；生产 final stage 不含 shell、curl、tar 或包管理器。
固定来源和升级安全修复不能替代镜像审查：
[S5-007 候选供应链 Gate](gates/MVP-2026-07-23/s5-supply-chain-candidate.md)记录的
本机 arm64 修复候选已为 `0 Critical / 0 High`，但发布前仍必须从干净提交在最终原生
双架构 digest 上复扫并完成 provenance 与安全/合规签署。

## 媒体解析

- 按允许的媒体类型和文件魔数识别输入，不能只信任扩展名或客户端 MIME。
- MVP 允许列表为 JPEG、PNG、WebP、GIF，以及 MP4、MOV、MKV；SVG、HEIC/HEIF、AVIF 与 RAW 不进入首版媒体处理契约。容器允许不代表浏览器编码可直放。
- Post-MVP/1 通过 CR-2026-010 将 AVI 加入同一扩展名/MIME allowlist、FFmpeg 资源预算
  和原内容 Range 边界；AVI 容器不构成浏览器可播放承诺。
- libvips、`ffprobe` 和 FFmpeg 操作必须具有超时、取消、输入大小限制和全局并发上限。
- 限制图片像素数、动画帧数、媒体探测时间和输出尺寸，防止像素炸弹及 CPU、内存或磁盘耗尽。
- 调用媒体 CLI 时使用独立参数数组，不拼接 shell 命令；媒体文件名不得成为可解释的命令片段。
- SVG、HTML 或其他主动内容不得作为可信同源页面直接内联。缩略图响应使用明确 MIME、`X-Content-Type-Options: nosniff` 和适当的内容安全策略。
- 文件名、路径、EXIF 和视频标签在进入 HTML、日志或错误消息前必须按上下文编码。

媒体处理任务写入受控缓存目录，使用随机临时文件和原子替换。缓存键不能直接包含未经处理的路径片段。

`S3-007` 的缩略图读取仍经过集中 session middleware；API 只接受 opaque asset ID，
handler 不接收缓存路径、不访问 SQLite/文件系统，也不调用媒体工具。delivery capability
只打开数据库记录的 `/app/data/cache` 相对路径，返回 WebP、ETag、private immutable cache、
`nosniff` 与限制型 CSP。缓存缺失或长度异常只撤销派生 ready 并重排 durable job；
公开错误不包含 cache path，离线源返回稳定 `source_offline`。

`S3-006` 已把 MVP 解析预算固定为：图片编码最大 256 MiB、视频最大 4 GiB、单边最大
32,768 px、总解码像素最大 100 MP、工具 stdout 最大 8 MiB、stderr 最大 64 KiB。
应用显式启动 govips，固定 native concurrency 1、64 MiB/32 entry cache 和 0 cached files；
两个媒体 worker 是进程级唯一任务并发。FFmpeg/ffprobe 使用单 decoder/filter thread、
15 秒超时和独立进程组，取消会杀整个组而不是只杀直接子进程。libvips 仍是进程内 native
调用，不能承诺在任意 C 调用中间抢占或隔离 native crash；当前在求值前拒绝超限输入，并在
返回后的第一个安全点重新检查取消。改变到进程隔离需要先接受 ADR。

真实 Linux 8 MiB tmpfs `ENOSPC` fixture 已确认缓存发布失败不留下临时文件、不提交 ready，
释放空间后可恢复；长期 WAL/temp 同卷竞争和最终卷容量仍由 Release Gate 验证。

### Post-MVP 视频故事板处理

[FTR-VID-001](features/video-storyboard-preview.md)不新增信任边界，但把一次 poster 抽取
扩展为最多 10 个目标时间点的派生处理。`VSP-S1/S2` 已固定并验证总 job 超时、
seek/输出上限、峰值 RSS、worker 优先级和 backfill admission。它继续使用
`internal/files` 的锚定只读 FD、参数数组、单 decoder/filter thread、进程组取消、原子缓存
发布、认证 delivery 和错误脱敏。不得用完整顺序解码长视频、API 线程现场生成、任意路径或
更高无界并发实现故事板。

`VSP-S2 Backend Evidence Ready` 已 Go；本节仍不改变 MVP 已验证的 15 秒 poster 预算，
完整 feature 发布继续由 `VSP-S4` 阻断。

## 容器和持久数据

- 只把一个媒体 volume 挂到 `/library:ro`，不在其下创建子挂载。根据
  [ADR-0012](adr/0012-root-runtime-bind-data.md)，应用默认以 root 运行以兼容自动创建的
  root-owned 数据 bind；这是已接受的扩大风险，不放宽媒体路径或写入边界。
- 最终镜像只包含运行所需的 Go 服务、libvips、FFmpeg、证书和必要运行库。
- 默认丢弃不需要的 Linux capabilities，并启用 `no-new-privileges`；在兼容的部署环境中使用只读根文件系统和独立可写 `/app/data`。
- 默认 seccomp/LSM 策略必须允许应用使用所需 `openat2` flags。生产容器不需要
  mount capability；`CAP_SYS_ADMIN` 只授予隔离测试容器以构造边界探针。
- `/app/data` 可由 Docker 自动创建为 root-owned bind 目录，并避免放在不可靠的网络文件系统上。
- 数据库迁移、磁盘已满、断电和进程终止必须有恢复测试。缩略图缓存耗尽不能破坏数据库或原媒体。
- 发布镜像使用不可变版本标签，并为 amd64/arm64 维护依赖与许可证清单。

## 日志和错误

- API 使用稳定、可公开的错误码；客户端不接收 SQL、堆栈或媒体工具原始 stderr。
- 日志默认记录媒体库 ID、任务 ID 和相对路径的安全摘要，不记录认证秘密、分享令牌或宿主机绝对路径。
- 媒体库创建/移除的 `Idempotency-Key` 只以 SHA-256 摘要持久化，不记录明文或原始请求体。
- 媒体库 removal capability 只清理 SQLite、durable jobs 和 `/app/data` 派生缓存；其端口
  不提供打开、写入、移动、改名或删除 `/library` 原件的能力。
- 对扫描失败、路径拒绝、媒体解析超时、队列深度、缓存占用和磁盘余量提供可观察信息。
- 同一损坏媒体的自动重试必须有上限和退避，避免永久占满工作队列。

S4-005B 已验证生产原媒体 route 只接受资产 ID，并经 session、SQLite 索引、
`internal/media` fingerprint 校验和 `internal/files.Root.Open` 读取；poisoned traversal、
源变化/缺失/offline、错误脱敏、16-stream admission 与 cooperative cancellation 均有回归
证据。Linux arm64 production route/nested mount 已通过；当前 amd64 QEMU 缺失 `openat2`
时应用在启动阶段失败关闭，不能用该模拟环境代替恢复后的 native amd64 PR CI。

## 必需测试

`FTR-UIF-001` 已补充并通过以下安全证据：

- 管理员改密验证当前密码、限流、CSRF、`no-store`、请求体上限和统一凭据错误；成功时当前
  session 保留、其他 session 在同一事务语义中撤销，事务失败不得留下半更新状态；
- 目录 `q` 只查询可靠索引并只接受资源 ID、规范文本和受控枚举，不接受路径或触发文件系统
  遍历；错误与日志不得泄露宿主机路径或 SQL；
- 缓存清理只调用 thumbnail/cache owner，只删除 `/app/data` 下可重建派生物；不接受路径，
  盘满、权限失败、重启和重复请求都必须保留配置、索引和原媒体；
- 合成媒体 sentinel 在账户、目录过滤和缓存清理纵向链前后逐项、逐字节一致。

账户事务、目录索引边界和 cache owner 证据由
[UIF-S2 Backend Evidence Ready](gates/MVP-2026-07-23/uif-s2-backend-evidence-ready.md)持有；
真实浏览器链在只读 `/library` 前后比较全部路径与 SHA-256，见
[`UIF-403`](evidence/uif-403/README.md)。这不关闭 S5-006B 的物理辅助功能，也不替代最终
发布镜像、网络拓扑与供应链安全签署。

- `..`、绝对路径、双重 URL 编码、NUL 和符号链接逃逸。
- Linux 同设备、跨设备与 self-bind 子挂载拒绝，以及 `openat2`/resolve flags
  被内核或 seccomp 阻断时的失败关闭。
- 目录选择器未认证短路、恶意 parent、文件过滤、隐藏/Unicode 目录、自然分页、游标篡改/
  跨 scope 复用、symlink 和 nested mount 原因码及公开错误脱敏。
- 相同、祖先和后代媒体库重叠，以及两个数据库连接并发创建时只能有一个成功。
- 挂载离线、权限错误、扫描中断和扫描期间路径变化。
- 畸形图片、像素炸弹、超大输入/尺寸、长动画、损坏视频、工具输出超限、超时进程树和
  native 调用返回后的取消安全点。
- HTTP Range、条件请求、无效 Range 和取消播放。
- 管理员初始化竞态、未认证访问、登录限流、会话过期/撤销、退出、CSRF 和反向代理配置。
- root runtime、单一只读媒体根挂载、无 mount capability、真实 tmpfs 磁盘已满和数据库恢复的
  Docker 集成测试；非 Linux 结果不能代替 Linux mount-boundary 证据。
